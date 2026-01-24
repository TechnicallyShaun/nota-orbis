// Package pidfile provides PID file management for daemon lifecycle.
package pidfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Common errors
var (
	ErrNoPIDFile       = errors.New("no PID file found")
	ErrInvalidPID      = errors.New("invalid PID in file")
	ErrProcessNotFound = errors.New("process not found")
)

const (
	pidFileName = "transcribe.pid"
	dirPerm     = 0755
	filePerm    = 0644
)

// Path returns the path to the PID file (~/.nota/transcribe.pid)
func Path() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".nota", pidFileName), nil
}

// Write creates the PID file with the given process ID.
// Creates parent directories if needed.
func Write(pid int) error {
	path, err := Path()
	if err != nil {
		return err
	}

	// Create parent directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Write PID to file
	content := strconv.Itoa(pid) + "\n"
	if err := os.WriteFile(path, []byte(content), filePerm); err != nil {
		return fmt.Errorf("write PID file: %w", err)
	}

	return nil
}

// Read reads the PID from the PID file.
// Returns ErrNoPIDFile if the file doesn't exist.
// Returns ErrInvalidPID if the file contains invalid data.
func Read() (int, error) {
	path, err := Path()
	if err != nil {
		return 0, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, ErrNoPIDFile
		}
		return 0, fmt.Errorf("read PID file: %w", err)
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		return 0, ErrInvalidPID
	}

	return pid, nil
}

// Remove deletes the PID file.
// Returns nil if the file doesn't exist.
func Remove() error {
	path, err := Path()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove PID file: %w", err)
	}

	return nil
}

// IsRunning checks if the process with the PID in the file is alive.
// Returns (running, pid, error).
// If there's no PID file, returns (false, 0, nil).
// If the PID file exists but the process is not running (stale), returns (false, pid, nil).
// If the process is running, returns (true, pid, nil).
func IsRunning() (bool, int, error) {
	pid, err := Read()
	if err != nil {
		if errors.Is(err, ErrNoPIDFile) {
			return false, 0, nil
		}
		return false, 0, err
	}

	// Check if process is alive using signal 0
	// This doesn't send a signal but checks if the process exists
	err = syscall.Kill(pid, 0)
	if err != nil {
		if errors.Is(err, syscall.ESRCH) {
			// Process not found - stale PID file
			return false, pid, nil
		}
		if errors.Is(err, syscall.EPERM) {
			// Permission denied means process exists but we can't signal it
			return true, pid, nil
		}
		return false, pid, fmt.Errorf("check process: %w", err)
	}

	return true, pid, nil
}

// CleanStale removes the PID file if it's stale (process not running).
// Returns true if a stale PID file was removed.
func CleanStale() (bool, error) {
	running, _, err := IsRunning()
	if err != nil {
		return false, err
	}

	if !running {
		// Check if file exists before trying to remove
		path, err := Path()
		if err != nil {
			return false, err
		}
		if _, err := os.Stat(path); err == nil {
			if err := Remove(); err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return false, nil
}
