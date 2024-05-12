package anchor

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version string // Used to store the tag via LDFLAGS
var commit string  // Used to store the short commit hash via LDFLAGS

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of anchor",
	RunE: func(cmd *cobra.Command, args []string) error {
		if version == "" {
			version = "development"
		}
		if commit == "" {
			commit = "unknown"
		}
		fmt.Printf("version: %s (commit %s)\n", version, commit)
		return nil
	},
}
