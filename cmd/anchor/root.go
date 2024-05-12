package anchor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/spf13/cobra"

	"github.com/songstitch/anchor"
)

type Options struct {
	Architectures []string
	OutputFile    string
	InputFile     string
}

func init() {
	rootCmd.PersistentFlags().
		StringP("input", "i", "Dockerfile.template", "Dockerfile to anchor")
	rootCmd.PersistentFlags().
		StringP("output", "o", "Dockerfile", "Name of the output dockerfile. If using multiple architectures, the architecture will be appended to the output file name")
	rootCmd.PersistentFlags().
		StringP("architectures", "a", "arm64", "Comma delimited list of architectures to anchor")
	rootCmd.PersistentFlags().
		BoolP("dry-run", "", false, "Write the output to stdout instead of a file")
	rootCmd.PersistentFlags().
		BoolP("yes", "y", false, "Write the output to the file without confirmation when the file exists. This will overwrite the file")

}

var rootCmd = &cobra.Command{
	Use:           "anchor",
	Short:         "anchor is a tool to anchor Dockerfiles to specific versions",
	Long:          "anchor is a tool to anchor Dockerfiles to specific versions for their base images and packages.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {

		if anchor.IsDockerInstalled() {
			if !anchor.IsDockerRunning() {
				return fmt.Errorf("docker is not running")
			}
		} else {
			return fmt.Errorf("docker is not installed")
		}

		output, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}
		architectures, err := cmd.Flags().GetString("architectures")
		if err != nil {
			return err
		}
		input, err := cmd.Flags().GetString("input")
		if err != nil {
			return err
		}
		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			return err
		}
		yes, err := cmd.Flags().GetBool("yes")
		if err != nil {
			return err
		}

		options := Options{
			Architectures: strings.Split(architectures, ","),
			OutputFile:    output,
			InputFile:     input,
		}
		appendArch := len(options.Architectures) > 1

		content, err := os.Open(options.InputFile)
		if err != nil {
			return err
		}
		defer content.Close()
		lines := []string{}
		scanner := bufio.NewScanner(content)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			return err
		}
		for _, architecture := range options.Architectures {
			content, err := os.Open(options.InputFile)
			if err != nil {
				return err
			}
			defer content.Close()
			result, err := parser.Parse(content)
			if err != nil {
				return err
			}

			node := result.AST

			color.Cyan("Anchoring to architecture: %s\n", architecture)
			image := ""
			err = anchor.ParseNode(node, architecture, &image)
			if err != nil {
				return err
			}

			var builder strings.Builder
			anchor.WriteDockerfile(&builder, node, true, 0, lines)
			outputName := options.OutputFile
			if appendArch {
				outputName = fmt.Sprintf("%s.%s", outputName, architecture)
			}

			if dryRun {
				color.Green("Generated anchored Dockerfile\n")
				fmt.Println(builder.String())
				return nil
			}

			absPath, err := filepath.Abs(outputName)
			if err != nil {
				return err
			}
			if _, err := os.Stat(absPath); err == nil && !yes {
				color.Yellow("File %s already exists. Overwrite? (y/n)", absPath)
				reader := bufio.NewReader(os.Stdin)
				response, err := reader.ReadString('\n')
				if err != nil {
					return err
				}

				if strings.ToLower(response) != "y\n" {
					color.Green("Generated anchored Dockerfile\n")
					fmt.Println(builder.String())
					return fmt.Errorf("exiting without writing file")
				}
			}
			err = os.WriteFile(outputName, []byte(builder.String()), 0600)
			if err != nil {
				return err
			}
			color.Green("Generated anchored Dockerfile: %s", absPath)
		}
		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		color.Red("%s", err)
		os.Exit(1)
	}
}
