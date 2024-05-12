package anchor

import (
	"fmt"
	"os/exec"
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
	fmt.Printf("\tAnchored %s to %s\n", node.Value, digest)
	node.Value = fmt.Sprintf("%s@%s", node.Value, digest)
	return node.Value, nil
}

func WriteDockerfile(builder *strings.Builder, node *parser.Node, useOriginal bool) {
	if node.Value == "FROM" || node.Value == "RUN" {
		useOriginal = false
	}
	if useOriginal {
		builder.WriteString(node.Original)
	} else {
		splits := strings.Split(node.Value, "  ")
		splitsTrimmed := []string{}
		for _, split := range splits {
			if split != "" {
				splitsTrimmed = append(splitsTrimmed, strings.TrimSpace(split))
			}
		}
		s := strings.Join(splitsTrimmed, " \\\n    ")
		builder.WriteString(s)
	}
	for _, child := range node.Children {
		WriteDockerfile(builder, child, useOriginal)
		builder.WriteString("\n\n")
	}

	if node.Next != nil {
		builder.WriteString(" ")
		WriteDockerfile(builder, node.Next, useOriginal)
	}
}

func PrintNode(node *parser.Node) {
	for _, child := range node.Children {
		PrintNode(child)
	}

	if node.Next != nil {
		PrintNode(node.Next)
	}
}

func ParseNode(node *parser.Node, architecture string, image *string) error {
	if node == nil {
		return nil
	}

	if node.Value == "FROM" {
		var err error
		newImage, err := attachDockerSha(node.Next)
		if err != nil {
			return err
		}
		*image = newImage
	} else if node.Value == "RUN" {
		err := parseRunCommand(node.Next, architecture, *image)
		if err != nil {
			return err
		}
	} else if node.Next != nil {
		ParseNode(node.Next, architecture, image)
	}

	for _, child := range node.Children {
		ParseNode(child, architecture, image)
	}
	return nil
}

func IsDockerInstalled() bool {
	cmd := exec.Command("docker", "--version")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func IsDockerRunning() bool {
	cmd := exec.Command("docker", "info")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	// Check if output contains information indicating Docker is running
	return strings.Contains(string(output), "Server:")
}
