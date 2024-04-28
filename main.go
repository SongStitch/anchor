package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/spf13/cobra"
)

type Options struct {
	Architectures []string
	OutputFile    string
	InputFile     string
}

var image string

var rootCmd = &cobra.Command{
	Use:   "dockerlock",
	Short: "dockerlock is a tool to lock down Dockerfiles to specific versions",
	Long:  "dockerlock is a tool to lock down Dockerfiles to specific versions for their base images and all packages.",
	Run: func(cmd *cobra.Command, args []string) {
		output, err := cmd.Flags().GetString("output")
		if err != nil {
			panic(err)
		}
		architectures, err := cmd.Flags().GetString("architectures")
		if err != nil {
			panic(err)
		}
		input, err := cmd.Flags().GetString("input")
		if err != nil {
			panic(err)
		}

		options := Options{
			Architectures: strings.Split(architectures, ","),
			OutputFile:    output,
			InputFile:     input,
		}
		appendArch := len(options.Architectures) > 1

		for _, architecture := range options.Architectures {
			content, err := os.Open(options.InputFile)
			if err != nil {
				panic(err)
			}

			defer content.Close()
			result, err := parser.Parse(content)
			if err != nil {
				panic(err)
			}

			node := result.AST
			printNode(node)

			parseNode(node, architecture)

			var builder strings.Builder
			writeDockerfile(&builder, node, true)
			outputName := options.OutputFile
			if appendArch {
				outputName = fmt.Sprintf("%s.%s", outputName, architecture)
			}
      log.Printf("Writing to %s", outputName)
			err = os.WriteFile(outputName, []byte(builder.String()), 0600)
			if err != nil {
				panic(err)
			}
		}
	},
}

func main() {
	rootCmd.PersistentFlags().
		StringP("input", "i", "Dockerfile.template", "Dockerfile to lock down")
	rootCmd.PersistentFlags().
		StringP("output", "o", "Dockerfile", "Name of the output dockerfile. If using multiple architectures, the architecture will be appended to the output file name.")
	rootCmd.PersistentFlags().
		StringP("architectures", "a", "arm64", "Comma separated list of architectures to lock down to")
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}

func writeDockerfile(builder *strings.Builder, node *parser.Node, useOriginal bool) {
	if node.Value == "FROM" || node.Value == "RUN" {
		useOriginal = false
	}
	if useOriginal {
		builder.WriteString(node.Original)
	} else {
		builder.WriteString(node.Value)
	}
	for _, child := range node.Children {
		writeDockerfile(builder, child, useOriginal)
		builder.WriteString("\n")
	}

	if node.Next != nil {
		builder.WriteString(" ")
		writeDockerfile(builder, node.Next, useOriginal)
	}
}

func printNode(node *parser.Node) {
	for _, child := range node.Children {
		printNode(child)
	}

	if node.Next != nil {
		printNode(node.Next)
	}
}

func parseNode(node *parser.Node, architecture string) {
	if node == nil {
		return
	}

	if node.Value == "FROM" {
		image = attachDockerSha(node.Next)
	} else if node.Value == "RUN" {
		parseRunCommand(node.Next, architecture)
	} else if node.Next != nil {
		parseNode(node.Next, architecture)
	}

	for _, child := range node.Children {
		parseNode(child, architecture)
	}
}

func parseRunCommand(node *parser.Node, architecture string) {
	if node == nil {
		return
	}

	commands := strings.Split(node.Value, "&&")
	for i := range commands {
		packageNames := parseCommand(commands[i])
		if len(packageNames) == 0 {
			continue
		}
		packageMap := fetchPackageVersions(packageNames, architecture)
		elements := strings.Split(commands[i], " ")
		for j := range elements {
			if _, ok := packageMap[elements[j]]; ok {
				elements[j] = fmt.Sprintf(
					"%s:%s=%s",
					elements[j],
					architecture,
					packageMap[elements[j]],
				)
			}
		}
		commands[i] = strings.Join(elements, " ")
		commands[i] = fmt.Sprintf(
			"dpkg --add-architecture %s && apt-get update && %s",
			architecture,
			commands[i],
		)
	}

	node.Value = strings.Join(commands, "&&")
}

func parseCommand(command string) []string {
	components := strings.Split(command, " ")
	var stripped []string
	for _, part := range components {
		if part == "" {
			continue
		}
		if !strings.HasPrefix(part, "-") {
			stripped = append(stripped, part)
		}
	}
	if len(stripped) < 3 {
		return []string{}
	}
	var packages []string
	for i, part := range stripped {
		if i == 0 {
			if part != "apt-get" {
				return []string{}
			} else {
				continue
			}
		}
		if i == 1 {
			if part != "install" {
				return []string{}
			} else {
				continue
			}
		}
		packages = append(packages, part)
	}

	return packages
}

func fetchPackageVersions(packages []string, architecture string) map[string]string {
	fmt.Println(image)
	var b bytes.Buffer
	command := "dpkg --add-architecture " + architecture + " && apt-get update && apt-cache show --"
	for _, pkg := range packages {
		command += " " + pkg + ":" + architecture
	}
	c := exec.Command("docker", "run", "--rm", image, "bash", "-c", command)
	c.Stdout = &b
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		log.Fatalf("Failed to run command: %v", err)
	}
	versions := parsePackageVersions(b.String())
	return versions
}

func parsePackageVersions(s string) map[string]string {
	versions := make(map[string]string)
	currentPackage := ""
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "Package:") {
			currentPackage = strings.Split(line, ": ")[1]
			continue
		}
		if strings.HasPrefix(line, "Version:") {
			if currentPackage == "" {
				log.Fatalf("Version found before package, offending line: %s", line)
			}
			versions[currentPackage] = strings.Split(line, ": ")[1]
			currentPackage = ""
		}
	}
	return versions
}

func attachDockerSha(node *parser.Node) string {
	if node == nil {
		return ""
	}
	digest, err := crane.Digest(node.Value)
	if err != nil {
		panic(err)
	}
	node.Value = fmt.Sprintf("%s@%s", node.Value, digest)
	return node.Value
}
