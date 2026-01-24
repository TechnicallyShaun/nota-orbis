package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TechnicallyShaun/nota-orbis/internal/transcribe"
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

func TestTranscribeCmd_HasConfigSubcommand(t *testing.T) {
	cmd := NewTranscribeCmd()

	found := false
	for _, sub := range cmd.Commands() {
		if sub.Use == "config" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected transcribe command to have config subcommand")
	}
}

func TestTranscribeConfigCmd_SavesConfiguration(t *testing.T) {
	vaultRoot := setupTestVault(t)
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(vaultRoot)

	// Simulate user input
	input := "/mnt/sync/voice-notes\nhttp://nas:9000/asr\n/home/user/vault/Inbox\n\n\n"
	prompter := NewReaderPrompter(strings.NewReader(input))

	var buf bytes.Buffer
	cmd := NewTranscribeConfigCmd(prompter, false)
	cmd.SetOut(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify config file was created
	configPath := filepath.Join(vaultRoot, ".nota", "transcribe.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}

	var cfg transcribe.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("expected valid JSON config: %v", err)
	}

	if cfg.WatchDir != "/mnt/sync/voice-notes" {
		t.Errorf("expected WatchDir %q, got %q", "/mnt/sync/voice-notes", cfg.WatchDir)
	}
	if cfg.APIURL != "http://nas:9000/asr" {
		t.Errorf("expected APIURL %q, got %q", "http://nas:9000/asr", cfg.APIURL)
	}
	if cfg.OutputDir != "/home/user/vault/Inbox" {
		t.Errorf("expected OutputDir %q, got %q", "/home/user/vault/Inbox", cfg.OutputDir)
	}
}

func TestTranscribeConfigCmd_AppliesDefaultArchiveDir(t *testing.T) {
	vaultRoot := setupTestVault(t)
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(vaultRoot)

	// Simulate user input with empty archive dir
	input := "/mnt/sync/voice-notes\nhttp://nas:9000/asr\n/home/user/vault/Inbox\n\n\n"
	prompter := NewReaderPrompter(strings.NewReader(input))

	var buf bytes.Buffer
	cmd := NewTranscribeConfigCmd(prompter, false)
	cmd.SetOut(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	configPath := filepath.Join(vaultRoot, ".nota", "transcribe.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}

	var cfg transcribe.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("expected valid JSON config: %v", err)
	}

	if cfg.ArchiveDir != transcribe.DefaultArchiveDir {
		t.Errorf("expected ArchiveDir %q, got %q", transcribe.DefaultArchiveDir, cfg.ArchiveDir)
	}
}

func TestTranscribeConfigCmd_SavesCustomArchiveDir(t *testing.T) {
	vaultRoot := setupTestVault(t)
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(vaultRoot)

	// Simulate user input with custom archive dir
	input := "/mnt/sync/voice-notes\nhttp://nas:9000/asr\n/home/user/vault/Inbox\n\n/custom/archive\n"
	prompter := NewReaderPrompter(strings.NewReader(input))

	var buf bytes.Buffer
	cmd := NewTranscribeConfigCmd(prompter, false)
	cmd.SetOut(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	configPath := filepath.Join(vaultRoot, ".nota", "transcribe.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}

	var cfg transcribe.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("expected valid JSON config: %v", err)
	}

	if cfg.ArchiveDir != "/custom/archive" {
		t.Errorf("expected ArchiveDir %q, got %q", "/custom/archive", cfg.ArchiveDir)
	}
}

func TestTranscribeConfigCmd_SavesTemplatePath(t *testing.T) {
	vaultRoot := setupTestVault(t)
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(vaultRoot)

	// Simulate user input with template path
	input := "/mnt/sync/voice-notes\nhttp://nas:9000/asr\n/home/user/vault/Inbox\n/path/to/template.md\n\n"
	prompter := NewReaderPrompter(strings.NewReader(input))

	var buf bytes.Buffer
	cmd := NewTranscribeConfigCmd(prompter, false)
	cmd.SetOut(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	configPath := filepath.Join(vaultRoot, ".nota", "transcribe.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}

	var cfg transcribe.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("expected valid JSON config: %v", err)
	}

	if cfg.TemplatePath == nil {
		t.Fatal("expected TemplatePath to be non-nil")
	}
	if *cfg.TemplatePath != "/path/to/template.md" {
		t.Errorf("expected TemplatePath %q, got %q", "/path/to/template.md", *cfg.TemplatePath)
	}
}

func TestTranscribeConfigCmd_SkipsTemplatePath(t *testing.T) {
	vaultRoot := setupTestVault(t)
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(vaultRoot)

	// Simulate user input with empty template path
	input := "/mnt/sync/voice-notes\nhttp://nas:9000/asr\n/home/user/vault/Inbox\n\n\n"
	prompter := NewReaderPrompter(strings.NewReader(input))

	var buf bytes.Buffer
	cmd := NewTranscribeConfigCmd(prompter, false)
	cmd.SetOut(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	configPath := filepath.Join(vaultRoot, ".nota", "transcribe.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}

	var cfg transcribe.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("expected valid JSON config: %v", err)
	}

	if cfg.TemplatePath != nil {
		t.Errorf("expected TemplatePath to be nil, got %q", *cfg.TemplatePath)
	}
}

func TestTranscribeConfigCmd_RequiresWatchDir(t *testing.T) {
	vaultRoot := setupTestVault(t)
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(vaultRoot)

	// Simulate user input with empty watch dir
	input := "\nhttp://nas:9000/asr\n/home/user/vault/Inbox\n\n\n"
	prompter := NewReaderPrompter(strings.NewReader(input))

	var buf bytes.Buffer
	cmd := NewTranscribeConfigCmd(prompter, false)
	cmd.SetOut(&buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when watch dir is empty")
	}
}

func TestTranscribeConfigCmd_RequiresAPIURL(t *testing.T) {
	vaultRoot := setupTestVault(t)
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(vaultRoot)

	// Simulate user input with empty API URL
	input := "/mnt/sync/voice-notes\n\n/home/user/vault/Inbox\n\n\n"
	prompter := NewReaderPrompter(strings.NewReader(input))

	var buf bytes.Buffer
	cmd := NewTranscribeConfigCmd(prompter, false)
	cmd.SetOut(&buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when API URL is empty")
	}
}

func TestTranscribeConfigCmd_RequiresOutputDir(t *testing.T) {
	vaultRoot := setupTestVault(t)
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(vaultRoot)

	// Simulate user input with empty output dir
	input := "/mnt/sync/voice-notes\nhttp://nas:9000/asr\n\n\n\n"
	prompter := NewReaderPrompter(strings.NewReader(input))

	var buf bytes.Buffer
	cmd := NewTranscribeConfigCmd(prompter, false)
	cmd.SetOut(&buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when output dir is empty")
	}
}

func TestTranscribeConfigCmd_RequiresVault(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	// Simulate user input
	input := "/mnt/sync/voice-notes\nhttp://nas:9000/asr\n/home/user/vault/Inbox\n\n\n"
	prompter := NewReaderPrompter(strings.NewReader(input))

	var buf bytes.Buffer
	cmd := NewTranscribeConfigCmd(prompter, false)
	cmd.SetOut(&buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when not in a vault")
	}
}

func TestTranscribeConfigCmd_PrintsSummary(t *testing.T) {
	vaultRoot := setupTestVault(t)
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(vaultRoot)

	input := "/mnt/sync/voice-notes\nhttp://nas:9000/asr\n/home/user/vault/Inbox\n\n\n"
	prompter := NewReaderPrompter(strings.NewReader(input))

	var buf bytes.Buffer
	cmd := NewTranscribeConfigCmd(prompter, false)
	cmd.SetOut(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Transcription Service Configuration") {
		t.Error("expected output to contain header")
	}
	if !strings.Contains(output, "Configuration saved to") {
		t.Error("expected output to contain save confirmation")
	}
	if !strings.Contains(output, ".nota/transcribe.json") {
		t.Error("expected output to contain config path")
	}
}

func TestTranscribeConfigCmd_AppliesDefaults(t *testing.T) {
	vaultRoot := setupTestVault(t)
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(vaultRoot)

	input := "/mnt/sync/voice-notes\nhttp://nas:9000/asr\n/home/user/vault/Inbox\n\n\n"
	prompter := NewReaderPrompter(strings.NewReader(input))

	var buf bytes.Buffer
	cmd := NewTranscribeConfigCmd(prompter, false)
	cmd.SetOut(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	configPath := filepath.Join(vaultRoot, ".nota", "transcribe.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}

	var cfg transcribe.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("expected valid JSON config: %v", err)
	}

	// Verify defaults were applied
	if cfg.StabilizationIntervalMs != transcribe.DefaultStabilizationIntervalMs {
		t.Errorf("expected StabilizationIntervalMs %d, got %d", transcribe.DefaultStabilizationIntervalMs, cfg.StabilizationIntervalMs)
	}
	if cfg.StabilizationChecks != transcribe.DefaultStabilizationChecks {
		t.Errorf("expected StabilizationChecks %d, got %d", transcribe.DefaultStabilizationChecks, cfg.StabilizationChecks)
	}
	if cfg.Language != transcribe.DefaultLanguage {
		t.Errorf("expected Language %q, got %q", transcribe.DefaultLanguage, cfg.Language)
	}
	if cfg.Model != transcribe.DefaultModel {
		t.Errorf("expected Model %q, got %q", transcribe.DefaultModel, cfg.Model)
	}
	if cfg.MaxFileSizeMB != transcribe.DefaultMaxFileSizeMB {
		t.Errorf("expected MaxFileSizeMB %d, got %d", transcribe.DefaultMaxFileSizeMB, cfg.MaxFileSizeMB)
	}
	if cfg.RetryCount != transcribe.DefaultRetryCount {
		t.Errorf("expected RetryCount %d, got %d", transcribe.DefaultRetryCount, cfg.RetryCount)
	}
	if len(cfg.WatchPatterns) != len(transcribe.DefaultWatchPatterns) {
		t.Errorf("expected WatchPatterns to have %d items, got %d", len(transcribe.DefaultWatchPatterns), len(cfg.WatchPatterns))
	}
}

func TestTranscribeCmd_HasStopSubcommand(t *testing.T) {
	cmd := NewTranscribeCmd()

	found := false
	for _, sub := range cmd.Commands() {
		if sub.Use == "stop" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected transcribe command to have stop subcommand")
	}
}

func TestTranscribeCmd_HasStatusSubcommand(t *testing.T) {
	cmd := NewTranscribeCmd()

	found := false
	for _, sub := range cmd.Commands() {
		if sub.Use == "status" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected transcribe command to have status subcommand")
	}
}

func TestTranscribeStopCmd_NoDaemonRunning(t *testing.T) {
	// Use a temp HOME so we don't interfere with real PID files
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	var buf bytes.Buffer
	cmd := newTranscribeStopCmd()
	cmd.SetOut(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "not running") {
		t.Errorf("expected output to say 'not running', got: %s", output)
	}
}

func TestTranscribeStatusCmd_NoDaemonRunning(t *testing.T) {
	// Use a temp HOME so we don't interfere with real PID files
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	var buf bytes.Buffer
	cmd := newTranscribeStatusCmd()
	cmd.SetOut(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "not running") {
		t.Errorf("expected output to say 'not running', got: %s", output)
	}
}

func TestTranscribeConfigCmd_AdvancedPromptsForAllFields(t *testing.T) {
	vaultRoot := setupTestVault(t)
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(vaultRoot)

	// Simulate user input with advanced options
	// Basic: watch_dir, api_url, output_dir, template_path, archive_dir
	// Advanced: stab_interval, stab_checks, language, model, max_file_size, retry_count, watch_patterns
	input := "/mnt/sync/voice-notes\nhttp://nas:9000/asr\n/home/user/vault/Inbox\n\n\n" +
		"3000\n5\nen\nlarge\n200\n5\n*.m4a,*.wav\n"
	prompter := NewReaderPrompter(strings.NewReader(input))

	var buf bytes.Buffer
	cmd := NewTranscribeConfigCmd(prompter, true)
	cmd.SetOut(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check that advanced settings header was shown
	output := buf.String()
	if !strings.Contains(output, "Advanced Settings") {
		t.Error("expected output to contain 'Advanced Settings' header")
	}

	// Verify config has custom advanced values
	configPath := filepath.Join(vaultRoot, ".nota", "transcribe.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}

	var cfg transcribe.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("expected valid JSON config: %v", err)
	}

	if cfg.StabilizationIntervalMs != 3000 {
		t.Errorf("expected StabilizationIntervalMs 3000, got %d", cfg.StabilizationIntervalMs)
	}
	if cfg.StabilizationChecks != 5 {
		t.Errorf("expected StabilizationChecks 5, got %d", cfg.StabilizationChecks)
	}
	if cfg.Language != "en" {
		t.Errorf("expected Language 'en', got %q", cfg.Language)
	}
	if cfg.Model != "large" {
		t.Errorf("expected Model 'large', got %q", cfg.Model)
	}
	if cfg.MaxFileSizeMB != 200 {
		t.Errorf("expected MaxFileSizeMB 200, got %d", cfg.MaxFileSizeMB)
	}
	if cfg.RetryCount != 5 {
		t.Errorf("expected RetryCount 5, got %d", cfg.RetryCount)
	}
	if len(cfg.WatchPatterns) != 2 {
		t.Errorf("expected 2 watch patterns, got %d", len(cfg.WatchPatterns))
	}
}

func TestTranscribeConfigCmd_AdvancedAcceptsDefaults(t *testing.T) {
	vaultRoot := setupTestVault(t)
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(vaultRoot)

	// Simulate user input accepting all defaults (empty inputs)
	input := "/mnt/sync/voice-notes\nhttp://nas:9000/asr\n/home/user/vault/Inbox\n\n\n" +
		"\n\n\n\n\n\n\n"
	prompter := NewReaderPrompter(strings.NewReader(input))

	var buf bytes.Buffer
	cmd := NewTranscribeConfigCmd(prompter, true)
	cmd.SetOut(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify config has default values
	configPath := filepath.Join(vaultRoot, ".nota", "transcribe.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}

	var cfg transcribe.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("expected valid JSON config: %v", err)
	}

	// All should be defaults
	if cfg.StabilizationIntervalMs != transcribe.DefaultStabilizationIntervalMs {
		t.Errorf("expected default StabilizationIntervalMs, got %d", cfg.StabilizationIntervalMs)
	}
	if cfg.StabilizationChecks != transcribe.DefaultStabilizationChecks {
		t.Errorf("expected default StabilizationChecks, got %d", cfg.StabilizationChecks)
	}
	if cfg.Language != transcribe.DefaultLanguage {
		t.Errorf("expected default Language, got %q", cfg.Language)
	}
}

func TestTranscribeStartCmd_HasDaemonFlag(t *testing.T) {
	cmd := newTranscribeStartCmd()

	flag := cmd.Flags().Lookup("daemon")
	if flag == nil {
		t.Error("expected start command to have --daemon flag")
	}
}

func TestTranscribeConfigCmd_HasAdvancedFlag(t *testing.T) {
	cmd := NewTranscribeConfigCmd(nil, false)

	flag := cmd.Flags().Lookup("advanced")
	if flag == nil {
		t.Error("expected config command to have --advanced flag")
	}
}
