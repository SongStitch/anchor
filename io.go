package main

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

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
