package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInitCmd_RequiresNameArgument(t *testing.T) {
	cmd := NewInitCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when no name argument provided")
	}
}

func TestInitCmd_InitializesVault(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	cmd := NewInitCmd()
	cmd.SetArgs([]string{"test-vault"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	notaDir := filepath.Join(tmpDir, ".nota")
	if _, err := os.Stat(notaDir); os.IsNotExist(err) {
		t.Error("expected .nota directory to be created")
	}
}

func TestInitCmd_PrintsSuccessMessage(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	var buf bytes.Buffer
	cmd := NewInitCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"my-vault"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if output != "Initialized vault 'my-vault'\n" {
		t.Errorf("expected success message, got: %q", output)
	}
}

func TestInitCmd_ReturnsErrorWhenVaultExists(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	// Create .nota directory to simulate existing vault
	os.Mkdir(filepath.Join(tmpDir, ".nota"), 0755)

	cmd := NewInitCmd()
	cmd.SetArgs([]string{"test-vault"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when vault already exists")
	}
}
