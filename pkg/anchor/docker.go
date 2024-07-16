package anchor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/google/go-containerregistry/pkg/crane"
)

func processFromCommand(node *Node) (string, error) {
	if node.CommandType != CommandFrom {
		return "", fmt.Errorf("node is not a FROM command")
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
	return "", fmt.Errorf("node did not contain a FROM command")
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
		return fmt.Errorf("node is not a RUN command")
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
			if strings.TrimSpace(elements[j]) == "&&" {
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
				s := fmt.Sprintf(
					"RUN dpkg --add-architecture %s && apt-get update &&",
					architecture,
				)
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
