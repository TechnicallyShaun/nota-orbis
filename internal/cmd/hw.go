package cmd

import (
	"errors"
	"fmt"

	"github.com/TechnicallyShaun/nota-orbis/internal/vault"
	"github.com/spf13/cobra"
)

// ErrNotAVault is returned when the hw command is run outside a vault.
var ErrNotAVault = errors.New("not a nota vault (run nota init to create one)")

// NewHwCmd creates the hw (hello world) command
func NewHwCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hw",
		Short: "Print hello world",
		Long:  "Print hello world when inside a vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := vault.FindVaultRoot()
			if err != nil {
				return ErrNotAVault
			}

			fmt.Fprintln(cmd.OutOrStdout(), "hello world")
			return nil
		},
	}
}
