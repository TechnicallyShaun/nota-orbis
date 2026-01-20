package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestHelloShaun_InsideVault(t *testing.T) {
	tmpDir := t.TempDir()
	createTestVault(t, tmpDir)

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	var buf bytes.Buffer
	cmd := NewHsCmd()
	cmd.SetOut(&buf)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if output != "hello shaun\n" {
		t.Errorf("expected 'hello shaun\\n', got: %q", output)
	}
}

func TestHelloShaun_OutsideVault(t *testing.T) {
	tmpDir := t.TempDir()
	// No vault created - just an empty directory

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	cmd := NewHsCmd()
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when outside vault")
	}

	if err.Error() != "not a nota vault (run nota init to create one)" {
		t.Errorf("expected specific error message, got: %q", err.Error())
	}
}

func TestHelloShaun_InNestedSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	createTestVault(t, tmpDir)

	// Create nested subdirectory
	nestedDir := filepath.Join(tmpDir, "projects", "my-project", "src")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested directory: %v", err)
	}

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(nestedDir)

	var buf bytes.Buffer
	cmd := NewHsCmd()
	cmd.SetOut(&buf)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if output != "hello shaun\n" {
		t.Errorf("expected 'hello shaun\\n', got: %q", output)
	}
}
