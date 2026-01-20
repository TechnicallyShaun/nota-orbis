package cmd

import (
	"testing"
)

func TestNewRootCmd(t *testing.T) {
	rootCmd := NewRootCmd()

	if rootCmd.Use != "nota" {
		t.Errorf("expected Use to be 'nota', got '%s'", rootCmd.Use)
	}

	// Verify subcommands are registered
	subcommands := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		subcommands[cmd.Use] = true
	}

	expected := []string{"init", "hw", "version"}
	for _, name := range expected {
		if !subcommands[name] {
			t.Errorf("expected subcommand '%s' to be registered", name)
		}
	}
}
