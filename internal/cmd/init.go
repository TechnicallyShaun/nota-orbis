package cmd

import (
	"fmt"

	"github.com/TechnicallyShaun/nota-orbis/internal/vault"
	"github.com/spf13/cobra"
)

// NewInitCmd creates the init command
func NewInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <name>",
		Short: "Initialize a new vault",
		Long:  "Initialize a new vault in the current directory with the specified name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if err := vault.Init(".", name); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Initialized vault '%s'\n", name)
			return nil
		},
	}
}
