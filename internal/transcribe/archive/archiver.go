// Package archive provides file archival for the transcription service.
package archive

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// Archiver moves processed files to archive.
type Archiver interface {
	Archive(ctx context.Context, sourcePath, archiveDir string) error
}

// ErrSourceNotFound is returned when the source file does not exist.
var ErrSourceNotFound = errors.New("source file not found")

// FileArchiver implements Archiver by copying files to an archive directory
// and deleting the original on success.
type FileArchiver struct{}

// NewFileArchiver creates a new FileArchiver.
func NewFileArchiver() *FileArchiver {
	return &FileArchiver{}
}

// Archive copies the source file to archiveDir, preserving the original filename.
// If archiveDir does not exist, it is created with 0755 permissions.
// The original file is deleted only after successful copy.
// Returns an error if the copy fails; the original is not deleted in that case.
func (a *FileArchiver) Archive(ctx context.Context, sourcePath, archiveDir string) error {
	// Check context before starting
	if err := ctx.Err(); err != nil {
		return err
	}

	// Verify source file exists
	srcInfo, err := os.Stat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrSourceNotFound
		}
		return err
	}

	// Create archive directory if it doesn't exist
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return err
	}

	// Determine destination path (preserve original filename)
	filename := filepath.Base(sourcePath)
	destPath := filepath.Join(archiveDir, filename)

	// Copy file to archive
	if err := copyFile(sourcePath, destPath, srcInfo.Mode()); err != nil {
		return err
	}

	// Check context before deleting original
	if err := ctx.Err(); err != nil {
		return err
	}

	// Delete original only after successful copy
	if err := os.Remove(sourcePath); err != nil {
		return err
	}

	return nil
}

// copyFile copies src to dst, preserving the file mode.
func copyFile(src, dst string, mode os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Ensure data is flushed to disk
	return dstFile.Sync()
}
