package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersion_OutputsVersionString(t *testing.T) {
	// Save original values
	origVersion := Version
	origCommit := Commit
	defer func() {
		Version = origVersion
		Commit = origCommit
	}()

	Version = "1.2.3"
	Commit = "abc123"

	var buf bytes.Buffer
	cmd := NewVersionCmd()
	cmd.SetOut(&buf)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "1.2.3") {
		t.Errorf("expected output to contain version '1.2.3', got: %q", output)
	}
}

func TestVersion_IncludesCommitHash(t *testing.T) {
	// Save original values
	origVersion := Version
	origCommit := Commit
	defer func() {
		Version = origVersion
		Commit = origCommit
	}()

	Version = "2.0.0"
	Commit = "def456789"

	var buf bytes.Buffer
	cmd := NewVersionCmd()
	cmd.SetOut(&buf)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "def456789") {
		t.Errorf("expected output to contain commit hash 'def456789', got: %q", output)
	}
}
