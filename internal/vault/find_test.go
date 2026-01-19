package vault

import (
	"os"
	"path/filepath"
	"testing"
)

// createVault creates a valid vault structure in the given directory.
func createVault(t *testing.T, dir string) {
	t.Helper()
	notaDir := filepath.Join(dir, VaultMarkerDir)
	if err := os.MkdirAll(notaDir, 0755); err != nil {
		t.Fatalf("failed to create .nota dir: %v", err)
	}
	configPath := filepath.Join(notaDir, VaultConfigFile)
	if err := os.WriteFile(configPath, []byte(`{"name": "test-vault"}`), 0644); err != nil {
		t.Fatalf("failed to create vault.json: %v", err)
	}
}

func TestFindVaultRoot_InVaultRoot(t *testing.T) {
	// Create temp vault
	tmpDir := t.TempDir()
	createVault(t, tmpDir)

	// Change to vault root
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	root, err := FindVaultRoot()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if root != tmpDir {
		t.Errorf("expected root %q, got %q", tmpDir, root)
	}
}

func TestFindVaultRoot_InSubdirectory(t *testing.T) {
	// Create temp vault with subdirectory
	tmpDir := t.TempDir()
	createVault(t, tmpDir)
	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)

	// Change to subdirectory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(subDir)

	root, err := FindVaultRoot()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if root != tmpDir {
		t.Errorf("expected root %q, got %q", tmpDir, root)
	}
}

func TestFindVaultRoot_InDeeplyNestedSubdirectory(t *testing.T) {
	// Create temp vault with deeply nested subdirectory
	tmpDir := t.TempDir()
	createVault(t, tmpDir)
	deepDir := filepath.Join(tmpDir, "a", "b", "c", "d", "e")
	os.MkdirAll(deepDir, 0755)

	// Change to deeply nested subdirectory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(deepDir)

	root, err := FindVaultRoot()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if root != tmpDir {
		t.Errorf("expected root %q, got %q", tmpDir, root)
	}
}

func TestFindVaultRoot_NotInVault(t *testing.T) {
	// Create temp directory without vault structure
	tmpDir := t.TempDir()

	// Change to non-vault directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	_, err := FindVaultRoot()
	if err != ErrNotInVault {
		t.Errorf("expected ErrNotInVault, got: %v", err)
	}
}

func TestFindVaultRoot_FallbackToEnvVar(t *testing.T) {
	// Create temp vault
	tmpDir := t.TempDir()
	createVault(t, tmpDir)

	// Create a non-vault directory to run from
	nonVaultDir := t.TempDir()

	// Change to non-vault directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(nonVaultDir)

	// Set env var to vault
	os.Setenv(EnvVaultRoot, tmpDir)
	defer os.Unsetenv(EnvVaultRoot)

	root, err := FindVaultRoot()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if root != tmpDir {
		t.Errorf("expected root %q, got %q", tmpDir, root)
	}
}

func TestFindVaultRoot_EnvVarInvalid(t *testing.T) {
	// Create a non-vault directory
	tmpDir := t.TempDir()

	// Change to non-vault directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Set env var to invalid path (non-vault directory)
	os.Setenv(EnvVaultRoot, tmpDir)
	defer os.Unsetenv(EnvVaultRoot)

	_, err := FindVaultRoot()
	if err != ErrNotInVault {
		t.Errorf("expected ErrNotInVault, got: %v", err)
	}
}

func TestIsVault_ValidVault(t *testing.T) {
	tmpDir := t.TempDir()
	createVault(t, tmpDir)

	if !IsVault(tmpDir) {
		t.Error("expected IsVault to return true for valid vault")
	}
}

func TestIsVault_MissingVaultJson(t *testing.T) {
	tmpDir := t.TempDir()
	// Create .nota dir but no vault.json
	notaDir := filepath.Join(tmpDir, VaultMarkerDir)
	os.MkdirAll(notaDir, 0755)

	if IsVault(tmpDir) {
		t.Error("expected IsVault to return false when vault.json is missing")
	}
}

func TestIsVault_InvalidJson(t *testing.T) {
	tmpDir := t.TempDir()
	notaDir := filepath.Join(tmpDir, VaultMarkerDir)
	os.MkdirAll(notaDir, 0755)
	// Write invalid JSON
	configPath := filepath.Join(notaDir, VaultConfigFile)
	os.WriteFile(configPath, []byte(`{invalid json`), 0644)

	if IsVault(tmpDir) {
		t.Error("expected IsVault to return false for invalid JSON")
	}
}

func TestIsVault_MissingNotaDir(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create .nota directory

	if IsVault(tmpDir) {
		t.Error("expected IsVault to return false when .nota dir is missing")
	}
}
