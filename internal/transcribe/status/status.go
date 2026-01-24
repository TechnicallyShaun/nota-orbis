// Package status provides log parsing for transcription service status display.
package status

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Stats holds parsed statistics from the log file.
type Stats struct {
	FilesProcessed int
	Errors         int
	LastProcessed  *ProcessedFile
}

// ProcessedFile holds information about the last processed file.
type ProcessedFile struct {
	Timestamp time.Time
	Path      string
	Output    string
}

// logDir returns the default log directory path
func logDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".nota", "logs"), nil
}

// TodayLogPath returns the path to today's transcribe log file.
func TodayLogPath() (string, error) {
	dir, err := logDir()
	if err != nil {
		return "", err
	}
	today := time.Now().UTC().Format("2006-01-02")
	return filepath.Join(dir, "transcribe-"+today+".log"), nil
}

// ParseTodayStats parses today's log file and returns statistics.
// Returns empty stats if the log file doesn't exist.
func ParseTodayStats() (*Stats, error) {
	logPath, err := TodayLogPath()
	if err != nil {
		return nil, err
	}
	return ParseLogFile(logPath)
}

// ParseLogFile parses a log file and returns statistics.
// Returns empty stats if the file doesn't exist.
func ParseLogFile(path string) (*Stats, error) {
	stats := &Stats{}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return stats, nil
		}
		return nil, err
	}
	defer file.Close()

	// Regex patterns for parsing log lines
	// Format: 2026-01-22T14:30:00Z INFO  [pipeline] file processing complete path=/path/to/file output=/path/to/output elapsed=1.5s
	completedPattern := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z)\s+INFO\s+\[pipeline\]\s+file processing complete\s+path=(\S+)\s+output=(\S+)`)
	errorPattern := regexp.MustCompile(`\s+ERROR\s+`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check for completed files
		if matches := completedPattern.FindStringSubmatch(line); matches != nil {
			stats.FilesProcessed++
			timestamp, err := time.Parse(time.RFC3339, matches[1])
			if err == nil {
				stats.LastProcessed = &ProcessedFile{
					Timestamp: timestamp,
					Path:      unquoteIfNeeded(matches[2]),
					Output:    unquoteIfNeeded(matches[3]),
				}
			}
		}

		// Check for errors
		if errorPattern.MatchString(line) {
			stats.Errors++
		}
	}

	return stats, scanner.Err()
}

// unquoteIfNeeded removes surrounding quotes from a string if present.
func unquoteIfNeeded(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// FormatTimestamp formats a timestamp for display.
func FormatTimestamp(t time.Time) string {
	return t.Local().Format("2006-01-02T15:04:05")
}

// BaseName returns just the filename from a path.
func BaseName(path string) string {
	return filepath.Base(strings.TrimSuffix(path, "/"))
}
