package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/TechnicallyShaun/nota-orbis/internal/vault"
	"github.com/spf13/cobra"
)

// NewInitCmd creates the init command
func NewInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a new vault",
		Long:  "Initialize a new vault in the current directory using the directory name as the vault name",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			name := filepath.Base(cwd)

			result, err := vault.Init(cwd, name)
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
