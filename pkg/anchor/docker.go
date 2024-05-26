package anchor

import (
	"context"
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
		color.Blue("Parsing the final image...")
	}
	if node == nil {
		return "", nil
	}
	digest, err := crane.Digest(node.Value)
	if err != nil {
		return digest, err
	}
	fmt.Printf("\t⚓Anchored %s to %s\n", node.Value, digest)
	node.Value = fmt.Sprintf("%s@%s", node.Value, digest)
	return node.Value, nil
}

func WriteDockerfile(
	builder *strings.Builder,
	node *parser.Node,
	useOriginal bool,
	currentLine int,
	lines []string,
) int {
	// this allows us to maintain things like comments and newlines in the original Dockerfile
	if node.StartLine != 0 && currentLine < node.StartLine {
		for i := currentLine + 1; i < node.StartLine; i++ {
			builder.WriteString(lines[i-1])
			builder.WriteString("\n")
		}
	}
	if currentLine == 0 {
		currentLine = 1
	} else if node.EndLine != 0 {
		currentLine = node.EndLine
	}
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
		currentLine = WriteDockerfile(builder, child, useOriginal, currentLine, lines)
		builder.WriteString("\n")
	}

	if node.Next != nil {
		if !useOriginal {
			builder.WriteString(" ")
		}
		currentLine = WriteDockerfile(builder, node.Next, useOriginal, currentLine, lines)
	}
	return currentLine
}

func ParseNode(ctx context.Context, node *parser.Node, architecture string, image *string) error {
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
		err := parseRunCommand(ctx, node.Next, architecture, *image)
		if err != nil {
			return err
		}
	} else if node.Next != nil {
		err := ParseNode(ctx, node.Next, architecture, image)
		if err != nil {
			return err
		}
	}

	for _, child := range node.Children {
		err := ParseNode(ctx, child, architecture, image)
		if err != nil {
			return err
		}
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
