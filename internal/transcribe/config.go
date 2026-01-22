// Package transcribe provides audio transcription configuration and services.
package transcribe

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/TechnicallyShaun/nota-orbis/internal/vault"
)

// ConfigFileName is the name of the transcription config file within .nota
const ConfigFileName = "transcribe.json"

// Default values for optional configuration fields
const (
	DefaultArchiveDir              = "~/.nota/archive/audio"
	DefaultStabilizationIntervalMs = 2000
	DefaultStabilizationChecks     = 3
	DefaultLanguage                = "auto"
	DefaultModel                   = "base"
	DefaultMaxFileSizeMB           = 100
	DefaultRetryCount              = 3
)

// DefaultWatchPatterns are the default file patterns to watch
var DefaultWatchPatterns = []string{"*.m4a", "*.mp3", "*.wav"}

// Config represents the transcription service configuration
type Config struct {
	WatchDir                  string   `json:"watch_dir"`
	APIURL                    string   `json:"api_url"`
	OutputDir                 string   `json:"output_dir"`
	TemplatePath              *string  `json:"template_path"`
	ArchiveDir                string   `json:"archive_dir"`
	WatchPatterns             []string `json:"watch_patterns"`
	StabilizationIntervalMs   int      `json:"stabilization_interval_ms"`
	StabilizationChecks       int      `json:"stabilization_checks"`
	Language                  string   `json:"language"`
	Model                     string   `json:"model"`
	MaxFileSizeMB             int      `json:"max_file_size_mb"`
	RetryCount                int      `json:"retry_count"`
}

// Validation errors
var (
	ErrWatchDirRequired  = errors.New("watch_dir is required")
	ErrAPIURLRequired    = errors.New("api_url is required")
	ErrOutputDirRequired = errors.New("output_dir is required")
)

// Load reads the transcription configuration from the vault's .nota/transcribe.json file.
// It uses vault.FindVaultRoot to locate the vault, then reads and parses the config.
// Paths containing ~ are expanded to the user's home directory.
func Load() (*Config, error) {
	vaultRoot, err := vault.FindVaultRoot()
	if err != nil {
		return nil, err
	}
	return LoadFromVault(vaultRoot)
}

// LoadFromVault reads the transcription configuration from a specific vault path.
// Paths containing ~ are expanded to the user's home directory.
func LoadFromVault(vaultRoot string) (*Config, error) {
	configPath := filepath.Join(vaultRoot, vault.VaultMarkerDir, ConfigFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	cfg.expandPaths()
	return &cfg, nil
}

// Save writes the configuration to the vault's .nota/transcribe.json file.
// It uses vault.FindVaultRoot to locate the vault.
// The file is created with 0644 permissions.
func (c *Config) Save() error {
	vaultRoot, err := vault.FindVaultRoot()
	if err != nil {
		return err
	}
	return c.SaveToVault(vaultRoot)
}

// SaveToVault writes the configuration to a specific vault path.
// The file is created with 0644 permissions.
func (c *Config) SaveToVault(vaultRoot string) error {
	configPath := filepath.Join(vaultRoot, vault.VaultMarkerDir, ConfigFileName)

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// Validate checks that all required fields are present.
// Returns an error if any required field is missing or empty.
func (c *Config) Validate() error {
	if c.WatchDir == "" {
		return ErrWatchDirRequired
	}
	if c.APIURL == "" {
		return ErrAPIURLRequired
	}
	if c.OutputDir == "" {
		return ErrOutputDirRequired
	}
	return nil
}

// ApplyDefaults sets default values for optional fields that are empty or zero.
// Call this after creating a new Config to ensure all optional fields have sensible defaults.
func (c *Config) ApplyDefaults() {
	if c.ArchiveDir == "" {
		c.ArchiveDir = DefaultArchiveDir
	}
	if len(c.WatchPatterns) == 0 {
		c.WatchPatterns = DefaultWatchPatterns
	}
	if c.StabilizationIntervalMs == 0 {
		c.StabilizationIntervalMs = DefaultStabilizationIntervalMs
	}
	if c.StabilizationChecks == 0 {
		c.StabilizationChecks = DefaultStabilizationChecks
	}
	if c.Language == "" {
		c.Language = DefaultLanguage
	}
	if c.Model == "" {
		c.Model = DefaultModel
	}
	if c.MaxFileSizeMB == 0 {
		c.MaxFileSizeMB = DefaultMaxFileSizeMB
	}
	if c.RetryCount == 0 {
		c.RetryCount = DefaultRetryCount
	}
}

// expandPaths expands ~ to the user's home directory in path fields.
func (c *Config) expandPaths() {
	c.WatchDir = expandTilde(c.WatchDir)
	c.OutputDir = expandTilde(c.OutputDir)
	c.ArchiveDir = expandTilde(c.ArchiveDir)
	if c.TemplatePath != nil {
		expanded := expandTilde(*c.TemplatePath)
		c.TemplatePath = &expanded
	}
}

// expandTilde expands ~ at the beginning of a path to the user's home directory.
func expandTilde(path string) string {
	if path == "" {
		return path
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
