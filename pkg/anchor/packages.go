package anchor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func fetchPackageVersions(
	ctx context.Context, packages []string, architecture string, image string,
) (map[string]string, error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	command := "dpkg --add-architecture " + architecture + " && apt-get update && apt-cache show --"
	for _, pkg := range packages {
		command += " " + pkg + ":" + architecture
	}
	c := exec.CommandContext(
		ctx, "docker", "run", "--rm", image, "bash", "-c", command,
	) // #nosec G204
	c.Stdout = &stdoutBuf
	c.Stderr = &stderrBuf // Use a buffer to capture stderr output

	if err := c.Run(); err != nil {
		// Log stderr or handle it as needed
		fmt.Fprintf(os.Stderr, "error running command: %s\n", stderrBuf.String())
		return nil, fmt.Errorf("failed to run command: %w", err)
	}

	versions, err := parsePackageVersions(stdoutBuf.String())
	if err != nil {
		fmt.Printf("Error parsing versions: %s\n", stderrBuf.String())
		return nil, err
	}
	return versions, nil
}

func parsePackageVersions(s string) (map[string]string, error) {
	versions := make(map[string]string)
	currentPackage := ""
	color.Blue("\tParsing package versions...")

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
				"\t⚓Anchored %s to %s\n",
				currentPackage,
				versions[currentPackage],
			)
			currentPackage = ""
		}
	}
	return versions, nil
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

func parseRunCommand(
	ctx context.Context,
	node *parser.Node,
	architecture string,
	image string,
) error {
	if node == nil {
		return nil
	}

	commands := strings.Split(node.Value, "&&")
	for i := range commands {
		packageNames := parseCommand(commands[i])
		if len(packageNames) == 0 {
			continue
		}
		packageMap, err := fetchPackageVersions(ctx, packageNames, architecture, image)
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
			// leading space is intentional to separate commands
			" dpkg --add-architecture %s && apt-get update && %s",
			architecture,
			commands[i],
		)
	}

	node.Value = strings.Join(commands, "&&")
	return nil
}
