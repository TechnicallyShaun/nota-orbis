package cmd

import (
	"fmt"

	"github.com/TechnicallyShaun/nota-orbis/internal/vault"
	"github.com/spf13/cobra"
)

// NewHsCmd creates the hs (hello shaun) command
func NewHsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hs",
		Short: "Print hello shaun",
		Long:  "Print hello shaun when inside a vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := vault.FindVaultRoot()
			if err != nil {
				return ErrNotAVault
			}

			fmt.Fprintln(cmd.OutOrStdout(), "hello shaun")
			return nil
		},
	}
}
