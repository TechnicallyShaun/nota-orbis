package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Level represents a log severity level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value any
}

// String creates a string field
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an integer field
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Duration creates a duration field
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}

// Logger handles structured logging
type Logger interface {
	Info(msg string, fields ...Field)
	Error(msg string, err error, fields ...Field)
	Debug(msg string, fields ...Field)
	Close() error
}

// Config configures the logger
type Config struct {
	// LogDir is the directory where log files are stored (default: ~/.nota/logs)
	LogDir string
	// Prefix is the log file prefix (e.g., "transcribe" produces transcribe-YYYY-MM-DD.log)
	Prefix string
	// RetentionDays is the number of days to retain old log files (default: 30)
	RetentionDays int
	// Component is the component name shown in brackets (e.g., "[watcher]")
	Component string
	// MinLevel is the minimum log level to write (default: LevelInfo)
	MinLevel Level
	// minLevelSet tracks whether MinLevel was explicitly configured
	minLevelSet bool
}

// WithMinLevel returns a copy of Config with the specified minimum log level
func (c Config) WithMinLevel(level Level) Config {
	c.MinLevel = level
	c.minLevelSet = true
	return c
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	return Config{
		LogDir:        filepath.Join(homeDir, ".nota", "logs"),
		Prefix:        "transcribe",
		RetentionDays: 30,
		Component:     "",
		MinLevel:      LevelInfo,
	}
}

// FileLogger implements Logger with daily file rotation
type FileLogger struct {
	config      Config
	mu          sync.Mutex
	file        *os.File
	currentDate string
}

// New creates a new FileLogger with the given configuration
func New(config Config) (*FileLogger, error) {
	if config.LogDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		config.LogDir = filepath.Join(homeDir, ".nota", "logs")
	}
	if config.Prefix == "" {
		config.Prefix = "transcribe"
	}
	if config.RetentionDays <= 0 {
		config.RetentionDays = 30
	}
	if !config.minLevelSet {
		config.MinLevel = LevelInfo
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logger := &FileLogger{
		config: config,
	}

	// Open initial log file
	if err := logger.rotateIfNeeded(); err != nil {
		return nil, err
	}

	// Clean up old log files
	if err := logger.cleanOldLogs(); err != nil {
		// Log cleanup errors but don't fail initialization
		logger.writeLog(LevelError, "failed to clean old logs", err)
	}

	return logger, nil
}

// Info logs an informational message
func (l *FileLogger) Info(msg string, fields ...Field) {
	l.log(LevelInfo, msg, nil, fields...)
}

// Error logs an error message
func (l *FileLogger) Error(msg string, err error, fields ...Field) {
	l.log(LevelError, msg, err, fields...)
}

// Debug logs a debug message
func (l *FileLogger) Debug(msg string, fields ...Field) {
	l.log(LevelDebug, msg, nil, fields...)
}

// Close closes the logger and its underlying file
func (l *FileLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// WithComponent returns a new logger with the specified component name
func (l *FileLogger) WithComponent(component string) *FileLogger {
	newConfig := l.config
	newConfig.Component = component
	return &FileLogger{
		config:      newConfig,
		file:        l.file,
		currentDate: l.currentDate,
	}
}

func (l *FileLogger) log(level Level, msg string, err error, fields ...Field) {
	if level < l.config.MinLevel {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if rotateErr := l.rotateIfNeeded(); rotateErr != nil {
		// If rotation fails, try to write to stderr
		fmt.Fprintf(os.Stderr, "log rotation failed: %v\n", rotateErr)
		return
	}

	l.writeLog(level, msg, err, fields...)
}

func (l *FileLogger) writeLog(level Level, msg string, err error, fields ...Field) {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	var sb strings.Builder
	sb.WriteString(timestamp)
	sb.WriteString(" ")
	sb.WriteString(fmt.Sprintf("%-5s", level.String()))
	sb.WriteString(" ")

	if l.config.Component != "" {
		sb.WriteString("[")
		sb.WriteString(l.config.Component)
		sb.WriteString("] ")
	}

	sb.WriteString(msg)

	if err != nil {
		sb.WriteString(" error=")
		sb.WriteString(err.Error())
	}

	for _, f := range fields {
		sb.WriteString(" ")
		sb.WriteString(f.Key)
		sb.WriteString("=")
		sb.WriteString(formatValue(f.Value))
	}

	sb.WriteString("\n")

	if l.file != nil {
		l.file.WriteString(sb.String())
	}
}

func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		if strings.ContainsAny(val, " \t\n") {
			return fmt.Sprintf("%q", val)
		}
		return val
	case time.Duration:
		return val.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (l *FileLogger) rotateIfNeeded() error {
	today := time.Now().UTC().Format("2006-01-02")

	if l.currentDate == today && l.file != nil {
		return nil
	}

	// Close existing file
	if l.file != nil {
		l.file.Close()
		l.file = nil
	}

	// Open new file for today
	filename := fmt.Sprintf("%s-%s.log", l.config.Prefix, today)
	filepath := filepath.Join(l.config.LogDir, filename)

	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	l.file = file
	l.currentDate = today

	return nil
}

func (l *FileLogger) cleanOldLogs() error {
	entries, err := os.ReadDir(l.config.LogDir)
	if err != nil {
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	prefix := l.config.Prefix + "-"
	cutoff := time.Now().UTC().AddDate(0, 0, -l.config.RetentionDays)

	var toDelete []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".log") {
			continue
		}

		// Extract date from filename: prefix-YYYY-MM-DD.log
		dateStr := strings.TrimPrefix(name, prefix)
		dateStr = strings.TrimSuffix(dateStr, ".log")

		logDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue // Skip files that don't match the expected pattern
		}

		if logDate.Before(cutoff) {
			toDelete = append(toDelete, filepath.Join(l.config.LogDir, name))
		}
	}

	// Sort oldest first for consistent deletion order
	sort.Strings(toDelete)

	for _, path := range toDelete {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove old log file %s: %w", path, err)
		}
	}

	return nil
}

// LogPath returns the path to the current log file
func (l *FileLogger) LogPath() string {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Name()
	}

	today := time.Now().UTC().Format("2006-01-02")
	filename := fmt.Sprintf("%s-%s.log", l.config.Prefix, today)
	return filepath.Join(l.config.LogDir, filename)
}
