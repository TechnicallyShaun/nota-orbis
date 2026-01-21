package vault

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInit_CreatesVaultDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := Init(tmpDir, "test-vault")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if result.AlreadyExisted {
		t.Error("expected AlreadyExisted to be false for new vault")
	}

	notaDir := filepath.Join(tmpDir, ".nota")
	if _, err := os.Stat(notaDir); os.IsNotExist(err) {
		t.Errorf("expected .nota directory to exist")
	}
}

func TestInit_CreatesVaultJson(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := Init(tmpDir, "test-vault")
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

	result, err := Init(tmpDir, "test-vault")
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

	if len(result.FoldersCreated) != len(expectedFolders) {
		t.Errorf("expected %d folders created, got %d", len(expectedFolders), len(result.FoldersCreated))
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

	_, err := Init(tmpDir, "test-vault")
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

	_, err := Init(tmpDir, "test-vault")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify the original folder still exists with marker
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Errorf("expected marker file to still exist in Projects folder")
	}
}

func TestInit_IdempotentWhenVaultExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .nota directory with vault.json to simulate existing vault
	notaDir := filepath.Join(tmpDir, ".nota")
	if err := os.Mkdir(notaDir, 0755); err != nil {
		t.Fatalf("failed to create .nota directory: %v", err)
	}
	vaultJSON := filepath.Join(notaDir, "vault.json")
	if err := os.WriteFile(vaultJSON, []byte(`{"name":"original","created_at":"2024-01-01T00:00:00Z","version":"1.0"}`), 0644); err != nil {
		t.Fatalf("failed to create vault.json: %v", err)
	}

	result, err := Init(tmpDir, "test-vault")
	if err != nil {
		t.Fatalf("Init should succeed on existing vault, got: %v", err)
	}

	if !result.AlreadyExisted {
		t.Error("expected AlreadyExisted to be true")
	}

	// Verify original vault.json is preserved (not overwritten)
	data, err := os.ReadFile(vaultJSON)
	if err != nil {
		t.Fatalf("failed to read vault.json: %v", err)
	}
	var metadata VaultMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("failed to unmarshal vault.json: %v", err)
	}
	if metadata.Name != "original" {
		t.Errorf("expected vault.json name to be 'original', got '%s'", metadata.Name)
	}
}

func TestInit_ErrorWhenNameEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := Init(tmpDir, "")
	if err != ErrNameEmpty {
		t.Errorf("expected ErrNameEmpty, got: %v", err)
	}
}

func TestInit_VaultJsonHasCorrectSchema(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := Init(tmpDir, "my-test-vault")
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

func TestInit_CreatesMissingFoldersOnExistingVault(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .nota directory with vault.json to simulate existing vault
	notaDir := filepath.Join(tmpDir, ".nota")
	if err := os.Mkdir(notaDir, 0755); err != nil {
		t.Fatalf("failed to create .nota directory: %v", err)
	}
	vaultJSON := filepath.Join(notaDir, "vault.json")
	if err := os.WriteFile(vaultJSON, []byte(`{"name":"test","created_at":"2024-01-01T00:00:00Z","version":"1.0"}`), 0644); err != nil {
		t.Fatalf("failed to create vault.json: %v", err)
	}

	// Create only some PARA folders
	os.Mkdir(filepath.Join(tmpDir, "Inbox"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "Projects"), 0755)

	result, err := Init(tmpDir, "test-vault")
	if err != nil {
		t.Fatalf("Init should succeed, got: %v", err)
	}

	if !result.AlreadyExisted {
		t.Error("expected AlreadyExisted to be true")
	}

	// Should have created the 4 missing folders
	expectedCreated := []string{"Journal", "Areas", "Resources", "Archive"}
	if len(result.FoldersCreated) != len(expectedCreated) {
		t.Errorf("expected %d folders created, got %d: %v", len(expectedCreated), len(result.FoldersCreated), result.FoldersCreated)
	}

	// Verify all folders now exist
	allFolders := []string{"Inbox", "Journal", "Projects", "Areas", "Resources", "Archive"}
	for _, folder := range allFolders {
		folderPath := filepath.Join(tmpDir, folder)
		if _, err := os.Stat(folderPath); os.IsNotExist(err) {
			t.Errorf("expected %s folder to exist", folder)
		}
	}
}

func TestInit_ErrorWhenCannotCreateNotaDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Make the directory read-only so .nota cannot be created
	if err := os.Chmod(tmpDir, 0555); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}
	defer os.Chmod(tmpDir, 0755)

	_, err := Init(tmpDir, "test-vault")
	if err == nil {
		t.Error("expected error when .nota directory cannot be created")
	}
}

func TestInit_ErrorWhenCannotReadExistingFolders(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .nota so Init skips creating it
	notaDir := filepath.Join(tmpDir, ".nota")
	if err := os.Mkdir(notaDir, 0755); err != nil {
		t.Fatalf("failed to create .nota directory: %v", err)
	}
	vaultJSON := filepath.Join(notaDir, "vault.json")
	if err := os.WriteFile(vaultJSON, []byte(`{"name":"test","created_at":"2024-01-01T00:00:00Z","version":"1.0"}`), 0644); err != nil {
		t.Fatalf("failed to create vault.json: %v", err)
	}

	// Make the vault directory unreadable (can't list contents)
	if err := os.Chmod(tmpDir, 0000); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}
	defer os.Chmod(tmpDir, 0755)

	_, err := Init(tmpDir, "test-vault")
	if err == nil {
		t.Error("expected error when directory contents cannot be read")
	}
}

func TestInit_ErrorWhenCannotCreateParaFolder(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .nota so Init skips creating it
	notaDir := filepath.Join(tmpDir, ".nota")
	if err := os.Mkdir(notaDir, 0755); err != nil {
		t.Fatalf("failed to create .nota directory: %v", err)
	}
	vaultJSON := filepath.Join(notaDir, "vault.json")
	if err := os.WriteFile(vaultJSON, []byte(`{"name":"test","created_at":"2024-01-01T00:00:00Z","version":"1.0"}`), 0644); err != nil {
		t.Fatalf("failed to create vault.json: %v", err)
	}

	// Create a file named "Inbox" to block folder creation
	inboxPath := filepath.Join(tmpDir, "Inbox")
	if err := os.WriteFile(inboxPath, []byte("blocker"), 0644); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}

	_, err := Init(tmpDir, "test-vault")
	if err == nil {
		t.Error("expected error when PARA folder cannot be created")
	}
}
