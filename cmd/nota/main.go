package main

import (
	"os"

	"github.com/TechnicallyShaun/nota-orbis/internal/cmd"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
