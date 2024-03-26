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
	fmt.Println(node.Value)
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
	} else if node.Next != nil {
		parseNode(node.Next)
	}

	for _, child := range node.Children {
		parseNode(child)
	}
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
