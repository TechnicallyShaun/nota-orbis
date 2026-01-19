// Package vault provides vault detection and management functionality.
package vault

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// ErrNotInVault is returned when the current directory is not within a vault.
var ErrNotInVault = errors.New("not in a vault")

// VaultMarkerDir is the directory that marks a vault root.
const VaultMarkerDir = ".nota"

// VaultConfigFile is the configuration file within the marker directory.
const VaultConfigFile = "vault.json"

// EnvVaultRoot is the environment variable for overriding vault root detection.
const EnvVaultRoot = "NOTA_VAULT_ROOT"

// IsVault checks if the given path is a valid vault root.
// A valid vault has a .nota directory containing a valid vault.json file.
func IsVault(path string) bool {
	notaDir := filepath.Join(path, VaultMarkerDir)
	configPath := filepath.Join(notaDir, VaultConfigFile)

	// Check .nota directory exists
	info, err := os.Stat(notaDir)
	if err != nil || !info.IsDir() {
		return false
	}

	// Check vault.json exists and is valid JSON
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return false
	}

	return true
}

// FindVaultRoot finds the root of the vault containing the current working directory.
// It walks up the directory tree looking for a .nota/vault.json file.
// If NOTA_VAULT_ROOT is set and points to a valid vault, it takes precedence.
// Returns ErrNotInVault if no vault is found.
func FindVaultRoot() (string, error) {
	// Check environment variable first
	if envRoot := os.Getenv(EnvVaultRoot); envRoot != "" {
		absPath, err := filepath.Abs(envRoot)
		if err != nil {
			return "", ErrNotInVault
		}
		if IsVault(absPath) {
			return absPath, nil
		}
		// Env var set but invalid - return error
		return "", ErrNotInVault
	}

	// Walk up from current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return FindVaultRootFrom(cwd)
}

// FindVaultRootFrom finds the root of the vault containing the given path.
// It walks up the directory tree looking for a .nota/vault.json file.
// Returns ErrNotInVault if no vault is found.
func FindVaultRootFrom(startPath string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", err
	}

	current := absPath
	for {
		if IsVault(current) {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			return "", ErrNotInVault
		}
		current = parent
	}
}
