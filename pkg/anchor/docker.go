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

func processFromCommand(node *Node) (string, error) {
	if node.CommandType != CommandFrom {
		return "", fmt.Errorf("Node is not a FROM command")
	}
	for i := range node.Entries {
		entry := node.Entries[i]
		if entry.Type != EntryCommand {
			continue
		}

		commandSplit := strings.Split(entry.Value, " ")
		if len(commandSplit) < 2 {
			return "", fmt.Errorf("FROM command is missing image name")
		}

		if len(commandSplit) == 4 {
			color.Blue("Parsing %s image...", commandSplit[3])
		} else {
			color.Blue("Parsing the final image...")
		}

		image := commandSplit[1]
		image = strings.TrimSpace((image))
		digest, err := crane.Digest(image)
		if err != nil {
			return "", err
		}

		entry.Value = strings.Replace((entry.Value), image, fmt.Sprintf("%s@%s", image, digest), 1)
		node.Entries[i] = entry
		fmt.Printf("\t⚓Anchored %s to %s\n", image, digest)
		// FROM command can only be one line, exit here
		return image, nil
	}
	return "", fmt.Errorf("Node did not contain a FROM command")
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

func processRunCommand(ctx context.Context, node *Node, architecture string, image string) error {
	if node.CommandType != CommandRun {
		return fmt.Errorf("Node is not a RUN command")
	}

	packageNames := parseCommand(node.Command)
	if len(packageNames) == 0 {
		return nil
	}
	packageMap, err := fetchPackageVersions(ctx, packageNames, architecture, image)
	if err != nil {
		return err
	}
	appendPackageVersions(node, packageMap, architecture)
	return nil
}

func appendPackageVersions(node *Node, packageMap map[string]string, architecture string) {
	aptGet := false
	install := false
	dpkgSet := false
	for i := range node.Entries {
		entry := node.Entries[i]
		if entry.Type != EntryCommand {
			continue
		}

		elements := strings.Split(entry.Value, " ")
		for j := range elements {
			if elements[j] == "apt-get" {
				aptGet = true
			}
			if aptGet && elements[j] == "install" {
				install = true

			}
			if aptGet && install {
				if _, ok := packageMap[elements[j]]; ok {
					elements[j] = fmt.Sprintf("%s=%s", elements[j], packageMap[elements[j]])
				}
			}
			if strings.TrimSpace(elements[i]) == "&&" {
				aptGet = false
				install = false
			}
		}
		entry.Value = strings.Join(elements, " ")
		if !dpkgSet {
			dpkgSet = true
			// since we know we have packages, that must mean we have apt-get install as part of this command
			// so we append the architecture and update to the beginning
			if entry.Beginning {
				s := fmt.Sprintf("RUN dpkg --add-architecture %s && apt-get update &&", architecture)
				entry.Value = strings.Replace(entry.Value,
					"RUN",
					s,
					1,
				)
			} else {
				entry.Value = fmt.Sprintf(
					// leading space is intentional to separate commands
					" dpkg --add-architecture %s && apt-get update \\\n &&%s",
					architecture,
					entry.Value,
				)
			}
		}
		node.Entries[i] = entry
	}
}

func Process(ctx context.Context, nodes []Node, architecture string) error {
	image := ""
	var err error
	for _, node := range nodes {
		switch node.CommandType {
		case CommandFrom:
			image, err = processFromCommand(&node)
			if err != nil {
				return err
			}
		case CommandRun:
			err := processRunCommand(ctx, &node, architecture, image)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
