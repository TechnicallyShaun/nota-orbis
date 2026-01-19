package vault

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInit_CreatesVaultDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	err := Init(tmpDir, "test-vault")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	notaDir := filepath.Join(tmpDir, ".nota")
	if _, err := os.Stat(notaDir); os.IsNotExist(err) {
		t.Errorf("expected .nota directory to exist")
	}
}

func TestInit_CreatesVaultJson(t *testing.T) {
	tmpDir := t.TempDir()

	err := Init(tmpDir, "test-vault")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	vaultJSONPath := filepath.Join(tmpDir, ".nota", "vault.json")
	if _, err := os.Stat(vaultJSONPath); os.IsNotExist(err) {
		t.Errorf("expected vault.json to exist")
	}
}

func TestInit_CreatesAllParaFolders(t *testing.T) {
	tmpDir := t.TempDir()

	err := Init(tmpDir, "test-vault")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	expectedFolders := []string{"Inbox", "Journal", "Projects", "Areas", "Resources", "Archive"}
	for _, folder := range expectedFolders {
		folderPath := filepath.Join(tmpDir, folder)
		if _, err := os.Stat(folderPath); os.IsNotExist(err) {
			t.Errorf("expected %s folder to exist", folder)
		}
	}
}

func TestInit_SkipsExistingFolderLowercase(t *testing.T) {
	tmpDir := t.TempDir()

	// Create lowercase "inbox" folder before init
	inboxPath := filepath.Join(tmpDir, "inbox")
	if err := os.Mkdir(inboxPath, 0755); err != nil {
		t.Fatalf("failed to create inbox folder: %v", err)
	}

	// Create a marker file inside to verify it's not replaced
	markerPath := filepath.Join(inboxPath, "marker.txt")
	if err := os.WriteFile(markerPath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create marker file: %v", err)
	}

	err := Init(tmpDir, "test-vault")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify the original lowercase folder still exists with marker
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Errorf("expected marker file to still exist in lowercase inbox folder")
	}

	// Verify no capitalized "Inbox" folder was created
	capitalizedInboxPath := filepath.Join(tmpDir, "Inbox")
	if _, err := os.Stat(capitalizedInboxPath); err == nil {
		t.Errorf("expected capitalized Inbox folder not to be created")
	}
}

func TestInit_SkipsExistingFolderCapitalized(t *testing.T) {
	tmpDir := t.TempDir()

	// Create capitalized "Projects" folder before init
	projectsPath := filepath.Join(tmpDir, "Projects")
	if err := os.Mkdir(projectsPath, 0755); err != nil {
		t.Fatalf("failed to create Projects folder: %v", err)
	}

	// Create a marker file inside to verify it's not replaced
	markerPath := filepath.Join(projectsPath, "marker.txt")
	if err := os.WriteFile(markerPath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create marker file: %v", err)
	}

	err := Init(tmpDir, "test-vault")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify the original folder still exists with marker
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Errorf("expected marker file to still exist in Projects folder")
	}
}

func TestInit_ErrorWhenVaultExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .nota directory to simulate existing vault
	notaDir := filepath.Join(tmpDir, ".nota")
	if err := os.Mkdir(notaDir, 0755); err != nil {
		t.Fatalf("failed to create .nota directory: %v", err)
	}

	err := Init(tmpDir, "test-vault")
	if err != ErrVaultExists {
		t.Errorf("expected ErrVaultExists, got: %v", err)
	}
}

func TestInit_ErrorWhenNameEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	err := Init(tmpDir, "")
	if err != ErrNameEmpty {
		t.Errorf("expected ErrNameEmpty, got: %v", err)
	}
}

func TestInit_VaultJsonHasCorrectSchema(t *testing.T) {
	tmpDir := t.TempDir()

	err := Init(tmpDir, "my-test-vault")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	vaultJSONPath := filepath.Join(tmpDir, ".nota", "vault.json")
	data, err := os.ReadFile(vaultJSONPath)
	if err != nil {
		t.Fatalf("failed to read vault.json: %v", err)
	}

	var metadata VaultMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("failed to unmarshal vault.json: %v", err)
	}

	if metadata.Name != "my-test-vault" {
		t.Errorf("expected name 'my-test-vault', got '%s'", metadata.Name)
	}

	if metadata.Version != "1.0" {
		t.Errorf("expected version '1.0', got '%s'", metadata.Version)
	}

	if metadata.CreatedAt == "" {
		t.Errorf("expected created_at to be set")
	}
}
