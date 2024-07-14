package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/songstitch/anchor/pkg/anchor"
)

type Options struct {
	Architectures []string
	OutputFile    string
	InputFile     string
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		return err
	}
	return nil
}

func init() {
	rootCmd.PersistentFlags().
		StringP("input", "i", "Dockerfile.template", "Dockerfile to anchor")
	rootCmd.PersistentFlags().
		StringP("output", "o", "Dockerfile", "Name of the output dockerfile. If using multiple architectures, the architecture will be appended to the output file name")
	rootCmd.PersistentFlags().
		StringP("architectures", "a", "", "Comma delimited list of architectures to anchor: \"amd64\" and \"arm64\" are supported. If the flag is not used, the system architecture will be used")
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
		ctx, cancel := context.WithCancel(context.Background())
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			cancel()
			os.Exit(1)
		}()

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
		if architectures == "" {
			architectures, err = getArchitecture()
			if err != nil {
				return err
			}
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

		for _, architecture := range options.Architectures {
			content, err := os.Open(options.InputFile)
			if err != nil {
				return err
			}
			nodes := anchor.Parse(content)
			defer content.Close()
			color.Cyan("Anchoring to architecture: %s\n", architecture)
			err = anchor.Process(ctx, nodes, architecture)
			if err != nil {
				return err
			}

			outputName := options.OutputFile
			if appendArch {
				outputName = fmt.Sprintf("%s.%s", outputName, architecture)
			}

			if dryRun {
				color.Green("Generated anchored Dockerfile\n")
				nodes.Write(os.Stdout)
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
					nodes.Write(os.Stdout)
					return fmt.Errorf("exiting without writing file")
				}
			}

			f, err := os.Create(outputName)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}
			defer f.Close()

			w := bufio.NewWriter(f)
			nodes.Write(w)
			w.Flush()
			color.Green("Generated anchored Dockerfile: %s", absPath)
		}
		return nil
	},
}

func getArchitecture() (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64", nil
	case "arm64":
		return "arm64", nil
	default:
		return "unknown", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}
}
