package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
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
var writeOutput bool

var rootCmd = &cobra.Command{
	Use:           "dockerlock",
	Short:         "dockerlock is a tool to lock Dockerfiles to specific versions",
	Long:          "dockerlock is a tool to lock Dockerfiles to specific versions for their base images and packages.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {

		if isDockerInstalled() {
			if !isDockerRunning() {
				return fmt.Errorf("Docker is not running")
			}
		} else {
			return fmt.Errorf("Docker is not installed")
		}

		output, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}
		architectures, err := cmd.Flags().GetString("architectures")
		if err != nil {
			return err
		}
		input, err := cmd.Flags().GetString("input")
		if err != nil {
			return err
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
				return err
			}

			defer content.Close()
			result, err := parser.Parse(content)
			if err != nil {
				return err
			}

			node := result.AST
			printNode(node)

			color.Yellow("Locking to architecture: %s\n", architecture)
			err = parseNode(node, architecture)
			if err != nil {
				return err
			}

			var builder strings.Builder
			writeDockerfile(&builder, node, true)
			outputName := options.OutputFile
			if appendArch {
				outputName = fmt.Sprintf("%s.%s", outputName, architecture)
			}
			if writeOutput {
				absPath, err := filepath.Abs(outputName)
				if err != nil {
					return err
				}
				err = os.WriteFile(outputName, []byte(builder.String()), 0600)
				if err != nil {
					return err
				}
				color.Green("Generated pinned Dockerfile: %s", absPath)
			} else {
				color.Green("Generated pinned Dockerfile\n")
				fmt.Println(builder.String())
			}
		}
		return nil
	},
}

func main() {
	rootCmd.PersistentFlags().
		StringP("input", "i", "Dockerfile.template", "Dockerfile to lock")
	rootCmd.PersistentFlags().
		StringP("output", "o", "Dockerfile", "Name of the output dockerfile. If using multiple architectures, the architecture will be appended to the output file name.")
	rootCmd.PersistentFlags().
		StringP("architectures", "a", "arm64", "Comma delimited list of architectures to lock")
	rootCmd.Flags().
		BoolVarP(&writeOutput, "write", "w", false, "Write the Dockerfile to the output file")
	err := rootCmd.Execute()
	if err != nil {
		color.Red("%s", err)
		os.Exit(1)
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

func parseNode(node *parser.Node, architecture string) error {
	if node == nil {
		return nil
	}

	if node.Value == "FROM" {
		var err error
		image, err = attachDockerSha(node.Next)
		if err != nil {
			return err
		}
	} else if node.Value == "RUN" {
		err := parseRunCommand(node.Next, architecture)
		if err != nil {
			return err
		}
	} else if node.Next != nil {
		parseNode(node.Next, architecture)
	}

	for _, child := range node.Children {
		parseNode(child, architecture)
	}
	return nil
}

func parseRunCommand(node *parser.Node, architecture string) error {
	if node == nil {
		return nil
	}

	commands := strings.Split(node.Value, "&&")
	for i := range commands {
		packageNames := parseCommand(commands[i])
		if len(packageNames) == 0 {
			continue
		}
		packageMap, err := fetchPackageVersions(packageNames, architecture)
		if err != nil {
			return err
		}
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
	return nil
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

func fetchPackageVersions(packages []string, architecture string) (map[string]string, error) {
	var b bytes.Buffer
	command := "dpkg --add-architecture " + architecture + " && apt-get update && apt-cache show --"
	for _, pkg := range packages {
		command += " " + pkg + ":" + architecture
	}
	c := exec.Command("docker", "run", "--rm", image, "bash", "-c", command)
	c.Stdout = &b
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return nil, fmt.Errorf("failed to run command: %w", err)
	}
	versions, err := parsePackageVersions(b.String())
	if err != nil {
		return nil, err
	}
	return versions, nil
}

func parsePackageVersions(s string) (map[string]string, error) {
	versions := make(map[string]string)
	currentPackage := ""
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "Package:") {
			currentPackage = strings.Split(line, ": ")[1]
			continue
		}
		if strings.HasPrefix(line, "Version:") {
			if currentPackage == "" {
				return nil, fmt.Errorf("version found before package, offending line: %s", line)
			}
			versions[currentPackage] = strings.Split(line, ": ")[1]
			fmt.Printf(
				"\tLocked %s to %s\n",
				currentPackage,
				versions[currentPackage],
			)
			currentPackage = ""
		}
	}
	return versions, nil
}

func attachDockerSha(node *parser.Node) (string, error) {
	if node.Next != nil && strings.ToLower(node.Next.Value) == "as" {
		color.Blue("Parsing %s image...", node.Next.Next.Value)
	} else {
		color.Blue("Parsing the final image")
	}
	if node == nil {
		return "", nil
	}
	digest, err := crane.Digest(node.Value)
	if err != nil {
		return digest, err
	}
	fmt.Printf("\tLocked %s to %s\n", node.Value, digest)
	node.Value = fmt.Sprintf("%s@%s", node.Value, digest)
	return node.Value, nil
}

func isDockerInstalled() bool {
	cmd := exec.Command("docker", "--version")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func isDockerRunning() bool {
	cmd := exec.Command("docker", "info")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	// Check if output contains information indicating Docker is running
	return strings.Contains(string(output), "Server:")
}
