package cmd

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root command for the nota CLI
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "nota",
		Short: "Personal knowledge management system",
		Long:  "Nota Orbis - Personal knowledge management system with PARA-inspired structure and AI-driven workflows",
	}

	rootCmd.AddCommand(NewInitCmd())
	rootCmd.AddCommand(NewHwCmd())
	rootCmd.AddCommand(NewVersionCmd())

	return rootCmd
}
