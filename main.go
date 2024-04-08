package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func main() {
	content, err := os.Open("Dockerfile.template")
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

	parseNode(node)

	var builder strings.Builder
	writeDockerfile(&builder, node)
	os.WriteFile("Dockerfile", []byte(builder.String()), 0644)
}

func writeDockerfile(builder *strings.Builder, node *parser.Node) {
	builder.WriteString(node.Value)
	for _, child := range node.Children {
		writeDockerfile(builder, child)
		builder.WriteString("\n")
	}

	if node.Next != nil {
		builder.WriteString(" ")
		writeDockerfile(builder, node.Next)
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

func parseNode(node *parser.Node) {
	if node == nil {
		return
	}

	if node.Value == "FROM" {
		attachDockerSha(node.Next)
	} else if node.Value == "RUN" {
		parseRunCommand(node.Next)
	} else if node.Next != nil {
		parseNode(node.Next)
	}

	for _, child := range node.Children {
		parseNode(child)
	}
}

func parseRunCommand(node *parser.Node) {
	if node == nil {
		return
	}

	commands := strings.Split(node.Value, "&&")
	for i := range commands {
		packageNames := parseCommand(commands[i])
		if len(packageNames) == 0 {
			continue
		}
		packageMap := fetchPackageVersions(packageNames)
		elements := strings.Split(commands[i], " ")
		for j := range elements {
			if _, ok := packageMap[elements[j]]; ok {
				elements[j] = fmt.Sprintf("%s=%s", elements[j], packageMap[elements[j]])
			}
		}
		commands[i] = strings.Join(elements, " ")
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

func fetchPackageVersions(packages []string) map[string]string {
	// TODO: Actually fetch package versions
	versionMap := make(map[string]string)
	for _, pkg := range packages {
		versionMap[pkg] = "latest"
	}

	return versionMap
}

func attachDockerSha(node *parser.Node) {
	if node == nil {
		return
	}
	digest, err := crane.Digest(node.Value)
	if err != nil {
		panic(err)
	}
	node.Value = fmt.Sprintf("%s@%s", node.Value, digest)
}
