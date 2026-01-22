package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew_CreatesLogDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	logger, err := New(Config{
		LogDir: logDir,
		Prefix: "test",
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer logger.Close()

	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Errorf("expected log directory to exist")
	}
}

func TestNew_CreatesLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	logger, err := New(Config{
		LogDir: logDir,
		Prefix: "test",
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer logger.Close()

	today := time.Now().UTC().Format("2006-01-02")
	expectedPath := filepath.Join(logDir, "test-"+today+".log")

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected log file to exist at %s", expectedPath)
	}
}

func TestNew_DefaultsApplied(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	logger, err := New(Config{
		LogDir: logDir,
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer logger.Close()

	// Check default prefix is "transcribe"
	today := time.Now().UTC().Format("2006-01-02")
	expectedPath := filepath.Join(logDir, "transcribe-"+today+".log")

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected log file with default prefix at %s", expectedPath)
	}
}

func TestFileLogger_Info(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	logger, err := New(Config{
		LogDir: logDir,
		Prefix: "test",
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	logger.Info("test message")
	logger.Close()

	content := readLogFile(t, logDir, "test")

	if !strings.Contains(content, "INFO") {
		t.Errorf("expected log to contain INFO level")
	}
	if !strings.Contains(content, "test message") {
		t.Errorf("expected log to contain message")
	}
}

func TestFileLogger_Error(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	logger, err := New(Config{
		LogDir: logDir,
		Prefix: "test",
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	testErr := os.ErrNotExist
	logger.Error("something failed", testErr)
	logger.Close()

	content := readLogFile(t, logDir, "test")

	if !strings.Contains(content, "ERROR") {
		t.Errorf("expected log to contain ERROR level")
	}
	if !strings.Contains(content, "something failed") {
		t.Errorf("expected log to contain message")
	}
	if !strings.Contains(content, "error=") {
		t.Errorf("expected log to contain error field")
	}
}

func TestFileLogger_Debug(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	logger, err := New(Config{
		LogDir: logDir,
		Prefix: "test",
	}.WithMinLevel(LevelDebug))
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	logger.Debug("debug info")
	logger.Close()

	content := readLogFile(t, logDir, "test")

	if !strings.Contains(content, "DEBUG") {
		t.Errorf("expected log to contain DEBUG level")
	}
	if !strings.Contains(content, "debug info") {
		t.Errorf("expected log to contain message")
	}
}

func TestFileLogger_DebugFilteredByDefault(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	logger, err := New(Config{
		LogDir: logDir,
		Prefix: "test",
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	logger.Debug("debug info")
	logger.Close()

	content := readLogFile(t, logDir, "test")

	if strings.Contains(content, "DEBUG") {
		t.Errorf("expected DEBUG to be filtered out by default")
	}
}

func TestFileLogger_WithFields(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	logger, err := New(Config{
		LogDir: logDir,
		Prefix: "test",
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	logger.Info("processing file",
		String("file", "test.m4a"),
		Int64("size", 2400000),
		Duration("elapsed", 5*time.Second),
	)
	logger.Close()

	content := readLogFile(t, logDir, "test")

	if !strings.Contains(content, "file=test.m4a") {
		t.Errorf("expected log to contain file field")
	}
	if !strings.Contains(content, "size=2400000") {
		t.Errorf("expected log to contain size field")
	}
	if !strings.Contains(content, "elapsed=5s") {
		t.Errorf("expected log to contain elapsed field")
	}
}

func TestFileLogger_WithComponent(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	logger, err := New(Config{
		LogDir:    logDir,
		Prefix:    "test",
		Component: "watcher",
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	logger.Info("file detected")
	logger.Close()

	content := readLogFile(t, logDir, "test")

	if !strings.Contains(content, "[watcher]") {
		t.Errorf("expected log to contain component")
	}
}

func TestFileLogger_LogFormat(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	logger, err := New(Config{
		LogDir:    logDir,
		Prefix:    "test",
		Component: "watcher",
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	logger.Info("file detected", String("file", "meeting-notes.m4a"))
	logger.Close()

	content := readLogFile(t, logDir, "test")

	// Example expected format: 2026-01-22T14:30:00Z INFO  [watcher] file detected file=meeting-notes.m4a
	// Check timestamp format (RFC3339)
	if !strings.Contains(content, "T") || !strings.Contains(content, "Z") {
		t.Errorf("expected RFC3339 timestamp format")
	}
	if !strings.Contains(content, "INFO") {
		t.Errorf("expected INFO level")
	}
	if !strings.Contains(content, "[watcher]") {
		t.Errorf("expected component in brackets")
	}
	if !strings.Contains(content, "file detected") {
		t.Errorf("expected message")
	}
	if !strings.Contains(content, "file=meeting-notes.m4a") {
		t.Errorf("expected field")
	}
}

func TestFileLogger_CleanOldLogs(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	// Create log directory
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log dir: %v", err)
	}

	// Create old log files (older than 30 days)
	oldDate := time.Now().UTC().AddDate(0, 0, -35).Format("2006-01-02")
	oldLogPath := filepath.Join(logDir, "test-"+oldDate+".log")
	if err := os.WriteFile(oldLogPath, []byte("old log"), 0644); err != nil {
		t.Fatalf("failed to create old log: %v", err)
	}

	// Create recent log file (within retention)
	recentDate := time.Now().UTC().AddDate(0, 0, -5).Format("2006-01-02")
	recentLogPath := filepath.Join(logDir, "test-"+recentDate+".log")
	if err := os.WriteFile(recentLogPath, []byte("recent log"), 0644); err != nil {
		t.Fatalf("failed to create recent log: %v", err)
	}

	logger, err := New(Config{
		LogDir:        logDir,
		Prefix:        "test",
		RetentionDays: 30,
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer logger.Close()

	// Old log should be deleted
	if _, err := os.Stat(oldLogPath); !os.IsNotExist(err) {
		t.Errorf("expected old log file to be deleted")
	}

	// Recent log should still exist
	if _, err := os.Stat(recentLogPath); os.IsNotExist(err) {
		t.Errorf("expected recent log file to still exist")
	}
}

func TestFileLogger_CustomRetention(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	// Create log directory
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log dir: %v", err)
	}

	// Create log files at various ages
	for i := 1; i <= 10; i++ {
		date := time.Now().UTC().AddDate(0, 0, -i).Format("2006-01-02")
		logPath := filepath.Join(logDir, "test-"+date+".log")
		if err := os.WriteFile(logPath, []byte("log"), 0644); err != nil {
			t.Fatalf("failed to create log: %v", err)
		}
	}

	// Set retention to 5 days
	logger, err := New(Config{
		LogDir:        logDir,
		Prefix:        "test",
		RetentionDays: 5,
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer logger.Close()

	// Count remaining log files
	entries, _ := os.ReadDir(logDir)
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "test-") && strings.HasSuffix(e.Name(), ".log") {
			count++
		}
	}

	// Should have 5 days + today's log = 6 files max
	// But we created logs for days 1-10, and today's gets created, so we should have:
	// - days 1-5 kept (5 files from our creation)
	// - today's new file (1 file)
	// - days 6-10 deleted
	if count > 6 {
		t.Errorf("expected at most 6 log files, got %d", count)
	}
}

func TestFileLogger_LogPath(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	logger, err := New(Config{
		LogDir: logDir,
		Prefix: "test",
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer logger.Close()

	logPath := logger.LogPath()
	today := time.Now().UTC().Format("2006-01-02")

	expectedPath := filepath.Join(logDir, "test-"+today+".log")
	if logPath != expectedPath {
		t.Errorf("expected LogPath() = %s, got %s", expectedPath, logPath)
	}
}

func TestFileLogger_WithComponentMethod(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	logger, err := New(Config{
		LogDir: logDir,
		Prefix: "test",
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer logger.Close()

	watcherLogger := logger.WithComponent("watcher")
	watcherLogger.Info("file detected")

	content := readLogFile(t, logDir, "test")

	if !strings.Contains(content, "[watcher]") {
		t.Errorf("expected log to contain component from WithComponent")
	}
}

func TestFieldHelpers(t *testing.T) {
	tests := []struct {
		name     string
		field    Field
		wantKey  string
		wantType string
	}{
		{"String", String("key", "value"), "key", "string"},
		{"Int", Int("count", 42), "count", "int"},
		{"Int64", Int64("bytes", 1024), "bytes", "int64"},
		{"Float64", Float64("ratio", 3.14), "ratio", "float64"},
		{"Duration", Duration("elapsed", time.Second), "elapsed", "duration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field.Key != tt.wantKey {
				t.Errorf("expected key %s, got %s", tt.wantKey, tt.field.Key)
			}
			if tt.field.Value == nil {
				t.Errorf("expected non-nil value")
			}
		})
	}
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelError, "ERROR"},
		{Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level.String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestFormatValue_QuotesSpaces(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	logger, err := New(Config{
		LogDir: logDir,
		Prefix: "test",
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	logger.Info("test", String("msg", "hello world"))
	logger.Close()

	content := readLogFile(t, logDir, "test")

	// Value with spaces should be quoted
	if !strings.Contains(content, `msg="hello world"`) {
		t.Errorf("expected quoted value with spaces, got: %s", content)
	}
}

func TestNew_ErrorOnInvalidLogDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	tmpDir := t.TempDir()

	// Create a file where we want the log directory to be
	blockingFile := filepath.Join(tmpDir, "logs")
	if err := os.WriteFile(blockingFile, []byte("blocker"), 0644); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}

	_, err := New(Config{
		LogDir: blockingFile,
		Prefix: "test",
	})
	if err == nil {
		t.Error("expected error when log directory cannot be created")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Prefix != "transcribe" {
		t.Errorf("expected default prefix 'transcribe', got '%s'", config.Prefix)
	}
	if config.RetentionDays != 30 {
		t.Errorf("expected default retention 30 days, got %d", config.RetentionDays)
	}
	if config.MinLevel != LevelInfo {
		t.Errorf("expected default min level INFO")
	}
	if !strings.Contains(config.LogDir, ".nota") || !strings.Contains(config.LogDir, "logs") {
		t.Errorf("expected default log dir to contain .nota/logs, got %s", config.LogDir)
	}
}

// Helper to read log file content
func readLogFile(t *testing.T, logDir, prefix string) string {
	t.Helper()

	today := time.Now().UTC().Format("2006-01-02")
	logPath := filepath.Join(logDir, prefix+"-"+today+".log")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	return string(content)
}
