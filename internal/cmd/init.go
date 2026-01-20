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

			result, err := vault.Init(".", name)
			if err != nil {
				return err
			}

			if result.AlreadyExisted {
				if len(result.FoldersCreated) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "Vault already initialized. Created missing folders: %v\n", result.FoldersCreated)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "Vault already initialized\n")
				}
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Initialized vault '%s'\n", name)
			}
			return nil
		},
	}
}
