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

func TestInitCmd_IdempotentWhenVaultExists(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	// Create .nota directory and PARA folders to simulate fully initialized vault
	os.Mkdir(filepath.Join(tmpDir, ".nota"), 0755)
	for _, folder := range []string{"Inbox", "Journal", "Projects", "Areas", "Resources", "Archive"} {
		os.Mkdir(filepath.Join(tmpDir, folder), 0755)
	}

	var buf bytes.Buffer
	cmd := NewInitCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"test-vault"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error for idempotent init, got: %v", err)
	}

	output := buf.String()
	if output != "Vault already initialized\n" {
		t.Errorf("expected 'Vault already initialized' message, got: %q", output)
	}
}

func TestInitCmd_CreatesMissingFoldersOnExistingVault(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	// Create .nota directory but no PARA folders
	os.Mkdir(filepath.Join(tmpDir, ".nota"), 0755)

	var buf bytes.Buffer
	cmd := NewInitCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"test-vault"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	expected := "Vault already initialized. Created missing folders: [Inbox Journal Projects Areas Resources Archive]\n"
	if output != expected {
		t.Errorf("expected %q, got: %q", expected, output)
	}
}
