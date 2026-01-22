// Package archiver provides file archiving capabilities for the transcription service.
package archiver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Archiver moves processed files to an archive location.
type Archiver interface {
	Archive(ctx context.Context, sourcePath, archiveDir string) error
}

// SimpleArchiver implements Archiver with basic file moving.
type SimpleArchiver struct{}

// NewSimpleArchiver creates a new simple archiver.
func NewSimpleArchiver() *SimpleArchiver {
	return &SimpleArchiver{}
}

// Archive moves a file from sourcePath to the archiveDir.
// Files are organized by date in subdirectories (YYYY/MM/DD).
func (a *SimpleArchiver) Archive(ctx context.Context, sourcePath, archiveDir string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Create date-based subdirectory
	now := time.Now()
	dateDir := filepath.Join(archiveDir, now.Format("2006"), now.Format("01"), now.Format("02"))

	if err := os.MkdirAll(dateDir, 0755); err != nil {
		return fmt.Errorf("create archive directory: %w", err)
	}

	// Generate destination path
	baseName := filepath.Base(sourcePath)
	destPath := filepath.Join(dateDir, baseName)

	// Handle filename collision by adding timestamp
	if _, err := os.Stat(destPath); err == nil {
		ext := filepath.Ext(baseName)
		nameWithoutExt := baseName[:len(baseName)-len(ext)]
		timestamp := now.Format("150405")
		destPath = filepath.Join(dateDir, fmt.Sprintf("%s-%s%s", nameWithoutExt, timestamp, ext))
	}

	// Move the file
	if err := os.Rename(sourcePath, destPath); err != nil {
		// If rename fails (cross-device), try copy and delete
		if err := copyAndDelete(sourcePath, destPath); err != nil {
			return fmt.Errorf("archive file: %w", err)
		}
	}

	return nil
}

// copyAndDelete copies a file and then deletes the original.
// Used when os.Rename fails due to cross-device link.
func copyAndDelete(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read source file: %w", err)
	}

	// Get original file permissions
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source file: %w", err)
	}

	if err := os.WriteFile(dst, data, info.Mode()); err != nil {
		return fmt.Errorf("write destination file: %w", err)
	}

	if err := os.Remove(src); err != nil {
		return fmt.Errorf("remove source file: %w", err)
	}

	return nil
}
