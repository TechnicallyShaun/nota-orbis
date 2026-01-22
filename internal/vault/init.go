package vault

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// VaultMetadata represents the contents of vault.json
type VaultMetadata struct {
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	Version   string `json:"version"`
}

// paraFolders defines the PARA+ folder structure
var paraFolders = []string{
	"Inbox",
	"Journal",
	"Projects",
	"Areas",
	"Resources",
	"Archive",
}

var (
	ErrVaultExists = errors.New("vault already exists")
	ErrNameEmpty   = errors.New("vault name cannot be empty")
)

// Init initializes a new vault at the given path with the specified name.
// It creates the .nota directory, vault.json metadata file, and PARA+ folders.
// Existing folders with matching names (case-insensitive) are skipped.
func Init(path, name string) error {
	if name == "" {
		return ErrNameEmpty
	}

	notaDir := filepath.Join(path, ".nota")

	// Check if vault already exists
	if _, err := os.Stat(notaDir); err == nil {
		return ErrVaultExists
	}

	// Create .nota directory
	if err := os.MkdirAll(notaDir, 0755); err != nil {
		return err
	}

	// Create vault.json
	metadata := VaultMetadata{
		Name:      name,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Version:   "1.0",
	}

	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	vaultJSONPath := filepath.Join(notaDir, "vault.json")
	if err := os.WriteFile(vaultJSONPath, metadataJSON, 0644); err != nil {
		return err
	}

	// Create PARA+ folders, skipping existing ones (case-insensitive)
	existingFolders, err := getExistingFolders(path)
	if err != nil {
		return err
	}

	for _, folder := range paraFolders {
		if folderExistsCaseInsensitive(folder, existingFolders) {
			continue
		}
		folderPath := filepath.Join(path, folder)
		if err := os.MkdirAll(folderPath, 0755); err != nil {
			return err
		}
	}

	return nil
}

// getExistingFolders returns a list of existing folder names in the given path
func getExistingFolders(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var folders []string
	for _, entry := range entries {
		if entry.IsDir() {
			folders = append(folders, entry.Name())
		}
	}
	return folders, nil
}

// folderExistsCaseInsensitive checks if a folder name exists in the list (case-insensitive)
func folderExistsCaseInsensitive(name string, existingFolders []string) bool {
	nameLower := strings.ToLower(name)
	for _, existing := range existingFolders {
		if strings.ToLower(existing) == nameLower {
			return true
		}
	}
	return false
}
