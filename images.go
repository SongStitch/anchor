package main

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

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
