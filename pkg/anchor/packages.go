package anchor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/fatih/color"
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

			if _, ok := versions[currentPackage]; ok {
        // We have already seen this package, so we can skip it
        currentPackage = ""
				continue
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
	commands := strings.Split(command, "&&")
	packages := []string{}
	for _, c := range commands {
		components := strings.Split(c, " ")
		var stripped []string
		for _, part := range components {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if part == "\\" {
				continue
			}
			if !strings.HasPrefix(part, "-") {
				stripped = append(stripped, part)
			}
		}
		if len(stripped) < 3 {
			continue
		}
		for i, part := range stripped {
			if i == 0 {
				if part != "apt-get" {
					break
				} else {
					continue
				}
			}
			if i == 1 {
				if part != "install" {
					break
				} else {
					continue
				}
			}
			if !slices.Contains(packages, part) {
				packages = append(packages, part)
			}
		}
	}

	return packages
}
