package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// createTestVault creates a minimal vault structure in the given directory
func createTestVault(t *testing.T, dir string) {
	t.Helper()
	notaDir := filepath.Join(dir, ".nota")
	if err := os.Mkdir(notaDir, 0755); err != nil {
		t.Fatalf("failed to create .nota directory: %v", err)
	}

	config := map[string]interface{}{"name": "test-vault"}
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal vault config: %v", err)
	}

	if err := os.WriteFile(filepath.Join(notaDir, "vault.json"), data, 0644); err != nil {
		t.Fatalf("failed to write vault.json: %v", err)
	}
}

func TestHelloWorld_InsideVault(t *testing.T) {
	tmpDir := t.TempDir()
	createTestVault(t, tmpDir)

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	var buf bytes.Buffer
	cmd := NewHwCmd()
	cmd.SetOut(&buf)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if output != "hello world\n" {
		t.Errorf("expected 'hello world\\n', got: %q", output)
	}
}

func TestHelloWorld_OutsideVault(t *testing.T) {
	tmpDir := t.TempDir()
	// No vault created - just an empty directory

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	cmd := NewHwCmd()
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when outside vault")
	}

	if err.Error() != "not a nota vault (run nota init to create one)" {
		t.Errorf("expected specific error message, got: %q", err.Error())
	}
}

func TestHelloWorld_InNestedSubdirectory(t *testing.T) {
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
	cmd := NewHwCmd()
	cmd.SetOut(&buf)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if output != "hello world\n" {
		t.Errorf("expected 'hello world\\n', got: %q", output)
	}
}
