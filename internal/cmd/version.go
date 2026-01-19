package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version and Commit are set at build time via ldflags
var (
	Version = "dev"
	Commit  = "unknown"
)

// NewVersionCmd creates the version command
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print the version and commit hash of the nota CLI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "nota version %s (commit: %s)\n", Version, Commit)
		},
	}
}
