package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersion_OutputsVersionString(t *testing.T) {
	// Save original values
	originalVersion := Version
	defer func() { Version = originalVersion }()

	// Set test version
	Version = "1.2.3"

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
	if !strings.Contains(output, "nota version") {
		t.Errorf("expected output to contain 'nota version', got: %q", output)
	}
}

func TestVersion_IncludesCommitHash(t *testing.T) {
	// Save original values
	originalCommit := Commit
	defer func() { Commit = originalCommit }()

	// Set test commit hash
	Commit = "abc123def"

	var buf bytes.Buffer
	cmd := NewVersionCmd()
	cmd.SetOut(&buf)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "abc123def") {
		t.Errorf("expected output to contain commit hash 'abc123def', got: %q", output)
	}
	if !strings.Contains(output, "commit:") {
		t.Errorf("expected output to contain 'commit:', got: %q", output)
	}
}
