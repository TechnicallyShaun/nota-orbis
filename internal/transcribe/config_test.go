package transcribe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func setupTestVault(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	notaDir := filepath.Join(tmpDir, ".nota")
	if err := os.Mkdir(notaDir, 0755); err != nil {
		t.Fatalf("failed to create .nota directory: %v", err)
	}
	vaultJSON := filepath.Join(notaDir, "vault.json")
	if err := os.WriteFile(vaultJSON, []byte(`{"name":"test","created_at":"2024-01-01T00:00:00Z","version":"1.0"}`), 0644); err != nil {
		t.Fatalf("failed to create vault.json: %v", err)
	}
	return tmpDir
}

func TestLoadFromVault_Success(t *testing.T) {
	vaultRoot := setupTestVault(t)

	cfg := &Config{
		WatchDir:  "/mnt/sync/voice-notes",
		APIURL:    "http://nas:9000/asr",
		OutputDir: "/home/user/vault/Inbox",
	}

	configPath := filepath.Join(vaultRoot, ".nota", ConfigFileName)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	loaded, err := LoadFromVault(vaultRoot)
	if err != nil {
		t.Fatalf("LoadFromVault failed: %v", err)
	}

	if loaded.WatchDir != cfg.WatchDir {
		t.Errorf("expected WatchDir %q, got %q", cfg.WatchDir, loaded.WatchDir)
	}
	if loaded.APIURL != cfg.APIURL {
		t.Errorf("expected APIURL %q, got %q", cfg.APIURL, loaded.APIURL)
	}
	if loaded.OutputDir != cfg.OutputDir {
		t.Errorf("expected OutputDir %q, got %q", cfg.OutputDir, loaded.OutputDir)
	}
}

func TestLoadFromVault_FileNotFound(t *testing.T) {
	vaultRoot := setupTestVault(t)

	_, err := LoadFromVault(vaultRoot)
	if err == nil {
		t.Error("expected error when config file does not exist")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected not exist error, got: %v", err)
	}
}

func TestLoadFromVault_InvalidJSON(t *testing.T) {
	vaultRoot := setupTestVault(t)

	configPath := filepath.Join(vaultRoot, ".nota", ConfigFileName)
	if err := os.WriteFile(configPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	_, err := LoadFromVault(vaultRoot)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadFromVault_ExpandsTildePaths(t *testing.T) {
	vaultRoot := setupTestVault(t)

	cfg := &Config{
		WatchDir:   "~/sync/voice-notes",
		APIURL:     "http://nas:9000/asr",
		OutputDir:  "~/vault/Inbox",
		ArchiveDir: "~/.nota/archive/audio",
	}

	configPath := filepath.Join(vaultRoot, ".nota", ConfigFileName)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	loaded, err := LoadFromVault(vaultRoot)
	if err != nil {
		t.Fatalf("LoadFromVault failed: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	expectedWatchDir := filepath.Join(home, "sync/voice-notes")
	if loaded.WatchDir != expectedWatchDir {
		t.Errorf("expected WatchDir %q, got %q", expectedWatchDir, loaded.WatchDir)
	}

	expectedOutputDir := filepath.Join(home, "vault/Inbox")
	if loaded.OutputDir != expectedOutputDir {
		t.Errorf("expected OutputDir %q, got %q", expectedOutputDir, loaded.OutputDir)
	}

	expectedArchiveDir := filepath.Join(home, ".nota/archive/audio")
	if loaded.ArchiveDir != expectedArchiveDir {
		t.Errorf("expected ArchiveDir %q, got %q", expectedArchiveDir, loaded.ArchiveDir)
	}
}

func TestLoadFromVault_ExpandsTildeTemplatePath(t *testing.T) {
	vaultRoot := setupTestVault(t)

	templatePath := "~/templates/voice-note.md"
	cfg := &Config{
		WatchDir:     "/mnt/sync",
		APIURL:       "http://nas:9000/asr",
		OutputDir:    "/home/user/vault/Inbox",
		TemplatePath: &templatePath,
	}

	configPath := filepath.Join(vaultRoot, ".nota", ConfigFileName)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	loaded, err := LoadFromVault(vaultRoot)
	if err != nil {
		t.Fatalf("LoadFromVault failed: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	if loaded.TemplatePath == nil {
		t.Fatal("expected TemplatePath to be non-nil")
	}

	expectedTemplatePath := filepath.Join(home, "templates/voice-note.md")
	if *loaded.TemplatePath != expectedTemplatePath {
		t.Errorf("expected TemplatePath %q, got %q", expectedTemplatePath, *loaded.TemplatePath)
	}
}

func TestSaveToVault_Success(t *testing.T) {
	vaultRoot := setupTestVault(t)

	cfg := &Config{
		WatchDir:                "/mnt/sync/voice-notes",
		APIURL:                  "http://nas:9000/asr",
		OutputDir:               "/home/user/vault/Inbox",
		ArchiveDir:              "~/.nota/archive/audio",
		WatchPatterns:           []string{"*.m4a", "*.mp3"},
		StabilizationIntervalMs: 2000,
		StabilizationChecks:     3,
		Language:                "auto",
		Model:                   "base",
		MaxFileSizeMB:           100,
		RetryCount:              3,
	}

	if err := cfg.SaveToVault(vaultRoot); err != nil {
		t.Fatalf("SaveToVault failed: %v", err)
	}

	configPath := filepath.Join(vaultRoot, ".nota", ConfigFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read saved config: %v", err)
	}

	var saved Config
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("failed to unmarshal saved config: %v", err)
	}

	if saved.WatchDir != cfg.WatchDir {
		t.Errorf("expected WatchDir %q, got %q", cfg.WatchDir, saved.WatchDir)
	}
	if saved.APIURL != cfg.APIURL {
		t.Errorf("expected APIURL %q, got %q", cfg.APIURL, saved.APIURL)
	}
}

func TestSaveToVault_FilePermissions(t *testing.T) {
	vaultRoot := setupTestVault(t)

	cfg := &Config{
		WatchDir:  "/mnt/sync",
		APIURL:    "http://nas:9000/asr",
		OutputDir: "/home/user/vault/Inbox",
	}

	if err := cfg.SaveToVault(vaultRoot); err != nil {
		t.Fatalf("SaveToVault failed: %v", err)
	}

	configPath := filepath.Join(vaultRoot, ".nota", ConfigFileName)
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}

	expectedPerm := os.FileMode(0644)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("expected permissions %o, got %o", expectedPerm, info.Mode().Perm())
	}
}

func TestValidate_Success(t *testing.T) {
	cfg := &Config{
		WatchDir:  "/mnt/sync/voice-notes",
		APIURL:    "http://nas:9000/asr",
		OutputDir: "/home/user/vault/Inbox",
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate failed for valid config: %v", err)
	}
}

func TestValidate_MissingWatchDir(t *testing.T) {
	cfg := &Config{
		APIURL:    "http://nas:9000/asr",
		OutputDir: "/home/user/vault/Inbox",
	}

	err := cfg.Validate()
	if err != ErrWatchDirRequired {
		t.Errorf("expected ErrWatchDirRequired, got: %v", err)
	}
}

func TestValidate_MissingAPIURL(t *testing.T) {
	cfg := &Config{
		WatchDir:  "/mnt/sync/voice-notes",
		OutputDir: "/home/user/vault/Inbox",
	}

	err := cfg.Validate()
	if err != ErrAPIURLRequired {
		t.Errorf("expected ErrAPIURLRequired, got: %v", err)
	}
}

func TestValidate_MissingOutputDir(t *testing.T) {
	cfg := &Config{
		WatchDir: "/mnt/sync/voice-notes",
		APIURL:   "http://nas:9000/asr",
	}

	err := cfg.Validate()
	if err != ErrOutputDirRequired {
		t.Errorf("expected ErrOutputDirRequired, got: %v", err)
	}
}

func TestApplyDefaults_SetsAllDefaults(t *testing.T) {
	cfg := &Config{
		WatchDir:  "/mnt/sync/voice-notes",
		APIURL:    "http://nas:9000/asr",
		OutputDir: "/home/user/vault/Inbox",
	}

	cfg.ApplyDefaults()

	if cfg.ArchiveDir != DefaultArchiveDir {
		t.Errorf("expected ArchiveDir %q, got %q", DefaultArchiveDir, cfg.ArchiveDir)
	}
	if len(cfg.WatchPatterns) != len(DefaultWatchPatterns) {
		t.Errorf("expected %d WatchPatterns, got %d", len(DefaultWatchPatterns), len(cfg.WatchPatterns))
	}
	if cfg.StabilizationIntervalMs != DefaultStabilizationIntervalMs {
		t.Errorf("expected StabilizationIntervalMs %d, got %d", DefaultStabilizationIntervalMs, cfg.StabilizationIntervalMs)
	}
	if cfg.StabilizationChecks != DefaultStabilizationChecks {
		t.Errorf("expected StabilizationChecks %d, got %d", DefaultStabilizationChecks, cfg.StabilizationChecks)
	}
	if cfg.Language != DefaultLanguage {
		t.Errorf("expected Language %q, got %q", DefaultLanguage, cfg.Language)
	}
	if cfg.Model != DefaultModel {
		t.Errorf("expected Model %q, got %q", DefaultModel, cfg.Model)
	}
	if cfg.MaxFileSizeMB != DefaultMaxFileSizeMB {
		t.Errorf("expected MaxFileSizeMB %d, got %d", DefaultMaxFileSizeMB, cfg.MaxFileSizeMB)
	}
	if cfg.RetryCount != DefaultRetryCount {
		t.Errorf("expected RetryCount %d, got %d", DefaultRetryCount, cfg.RetryCount)
	}
}

func TestApplyDefaults_PreservesExistingValues(t *testing.T) {
	cfg := &Config{
		WatchDir:                "/mnt/sync/voice-notes",
		APIURL:                  "http://nas:9000/asr",
		OutputDir:               "/home/user/vault/Inbox",
		ArchiveDir:              "/custom/archive",
		WatchPatterns:           []string{"*.ogg"},
		StabilizationIntervalMs: 5000,
		StabilizationChecks:     5,
		Language:                "en",
		Model:                   "large",
		MaxFileSizeMB:           200,
		RetryCount:              5,
	}

	cfg.ApplyDefaults()

	if cfg.ArchiveDir != "/custom/archive" {
		t.Errorf("expected ArchiveDir to be preserved, got %q", cfg.ArchiveDir)
	}
	if len(cfg.WatchPatterns) != 1 || cfg.WatchPatterns[0] != "*.ogg" {
		t.Errorf("expected WatchPatterns to be preserved, got %v", cfg.WatchPatterns)
	}
	if cfg.StabilizationIntervalMs != 5000 {
		t.Errorf("expected StabilizationIntervalMs to be preserved, got %d", cfg.StabilizationIntervalMs)
	}
	if cfg.StabilizationChecks != 5 {
		t.Errorf("expected StabilizationChecks to be preserved, got %d", cfg.StabilizationChecks)
	}
	if cfg.Language != "en" {
		t.Errorf("expected Language to be preserved, got %q", cfg.Language)
	}
	if cfg.Model != "large" {
		t.Errorf("expected Model to be preserved, got %q", cfg.Model)
	}
	if cfg.MaxFileSizeMB != 200 {
		t.Errorf("expected MaxFileSizeMB to be preserved, got %d", cfg.MaxFileSizeMB)
	}
	if cfg.RetryCount != 5 {
		t.Errorf("expected RetryCount to be preserved, got %d", cfg.RetryCount)
	}
}

func TestExpandTilde_PlainTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	result := expandTilde("~")
	if result != home {
		t.Errorf("expected %q, got %q", home, result)
	}
}

func TestExpandTilde_TildeWithPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	result := expandTilde("~/some/path")
	expected := filepath.Join(home, "some/path")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExpandTilde_NoTilde(t *testing.T) {
	input := "/absolute/path"
	result := expandTilde(input)
	if result != input {
		t.Errorf("expected %q, got %q", input, result)
	}
}

func TestExpandTilde_TildeInMiddle(t *testing.T) {
	input := "/path/to/~something"
	result := expandTilde(input)
	if result != input {
		t.Errorf("expected %q, got %q (tilde in middle should not expand)", input, result)
	}
}

func TestExpandTilde_EmptyString(t *testing.T) {
	result := expandTilde("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestConfig_RoundTrip(t *testing.T) {
	vaultRoot := setupTestVault(t)

	templatePath := "/path/to/template.md"
	original := &Config{
		WatchDir:                "/mnt/sync/voice-notes",
		APIURL:                  "http://nas:9000/asr",
		OutputDir:               "/home/user/vault/Inbox",
		TemplatePath:            &templatePath,
		ArchiveDir:              "/custom/archive",
		WatchPatterns:           []string{"*.m4a", "*.mp3", "*.wav", "*.ogg"},
		StabilizationIntervalMs: 3000,
		StabilizationChecks:     4,
		Language:                "en",
		Model:                   "medium",
		MaxFileSizeMB:           150,
		RetryCount:              5,
	}

	if err := original.SaveToVault(vaultRoot); err != nil {
		t.Fatalf("SaveToVault failed: %v", err)
	}

	loaded, err := LoadFromVault(vaultRoot)
	if err != nil {
		t.Fatalf("LoadFromVault failed: %v", err)
	}

	if loaded.WatchDir != original.WatchDir {
		t.Errorf("WatchDir mismatch: expected %q, got %q", original.WatchDir, loaded.WatchDir)
	}
	if loaded.APIURL != original.APIURL {
		t.Errorf("APIURL mismatch: expected %q, got %q", original.APIURL, loaded.APIURL)
	}
	if loaded.OutputDir != original.OutputDir {
		t.Errorf("OutputDir mismatch: expected %q, got %q", original.OutputDir, loaded.OutputDir)
	}
	if loaded.TemplatePath == nil || *loaded.TemplatePath != templatePath {
		t.Errorf("TemplatePath mismatch: expected %q, got %v", templatePath, loaded.TemplatePath)
	}
	if loaded.ArchiveDir != original.ArchiveDir {
		t.Errorf("ArchiveDir mismatch: expected %q, got %q", original.ArchiveDir, loaded.ArchiveDir)
	}
	if len(loaded.WatchPatterns) != len(original.WatchPatterns) {
		t.Errorf("WatchPatterns length mismatch: expected %d, got %d", len(original.WatchPatterns), len(loaded.WatchPatterns))
	}
	if loaded.StabilizationIntervalMs != original.StabilizationIntervalMs {
		t.Errorf("StabilizationIntervalMs mismatch: expected %d, got %d", original.StabilizationIntervalMs, loaded.StabilizationIntervalMs)
	}
	if loaded.StabilizationChecks != original.StabilizationChecks {
		t.Errorf("StabilizationChecks mismatch: expected %d, got %d", original.StabilizationChecks, loaded.StabilizationChecks)
	}
	if loaded.Language != original.Language {
		t.Errorf("Language mismatch: expected %q, got %q", original.Language, loaded.Language)
	}
	if loaded.Model != original.Model {
		t.Errorf("Model mismatch: expected %q, got %q", original.Model, loaded.Model)
	}
	if loaded.MaxFileSizeMB != original.MaxFileSizeMB {
		t.Errorf("MaxFileSizeMB mismatch: expected %d, got %d", original.MaxFileSizeMB, loaded.MaxFileSizeMB)
	}
	if loaded.RetryCount != original.RetryCount {
		t.Errorf("RetryCount mismatch: expected %d, got %d", original.RetryCount, loaded.RetryCount)
	}
}

func TestConfig_NullTemplatePath(t *testing.T) {
	vaultRoot := setupTestVault(t)

	original := &Config{
		WatchDir:     "/mnt/sync",
		APIURL:       "http://nas:9000/asr",
		OutputDir:    "/home/user/vault/Inbox",
		TemplatePath: nil,
	}

	if err := original.SaveToVault(vaultRoot); err != nil {
		t.Fatalf("SaveToVault failed: %v", err)
	}

	loaded, err := LoadFromVault(vaultRoot)
	if err != nil {
		t.Fatalf("LoadFromVault failed: %v", err)
	}

	if loaded.TemplatePath != nil {
		t.Errorf("expected TemplatePath to be nil, got %v", loaded.TemplatePath)
	}
}
