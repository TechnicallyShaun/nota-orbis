package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/TechnicallyShaun/nota-orbis/internal/transcribe"
	"github.com/TechnicallyShaun/nota-orbis/internal/vault"
	"github.com/spf13/cobra"
)

// Prompter defines the interface for reading user input
type Prompter interface {
	Prompt(prompt string) (string, error)
}

// StdinPrompter reads from stdin
type StdinPrompter struct {
	reader *bufio.Reader
}

// NewStdinPrompter creates a prompter that reads from stdin
func NewStdinPrompter() *StdinPrompter {
	return &StdinPrompter{reader: bufio.NewReader(os.Stdin)}
}

// Prompt displays a prompt and reads user input
func (p *StdinPrompter) Prompt(prompt string) (string, error) {
	fmt.Print(prompt)
	input, err := p.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// ReaderPrompter reads from a provided reader (for testing)
type ReaderPrompter struct {
	reader *bufio.Reader
}

// NewReaderPrompter creates a prompter that reads from the provided reader
func NewReaderPrompter(r io.Reader) *ReaderPrompter {
	return &ReaderPrompter{reader: bufio.NewReader(r)}
}

// Prompt reads input from the reader
func (p *ReaderPrompter) Prompt(prompt string) (string, error) {
	input, err := p.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// NewTranscribeCmd creates the transcribe command group
func NewTranscribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transcribe",
		Short: "Manage audio transcription service",
		Long:  "Commands for configuring and managing the audio transcription service",
	}

	cmd.AddCommand(NewTranscribeConfigCmd(nil))
	cmd.AddCommand(newTranscribeStartCmd())
	cmd.AddCommand(newTranscribeStopCmd())

	return cmd
}

// NewTranscribeConfigCmd creates the config subcommand
func NewTranscribeConfigCmd(prompter Prompter) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Configure transcription service",
		Long:  "Interactive configuration for the transcription service",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use provided prompter or create stdin prompter
			p := prompter
			if p == nil {
				p = NewStdinPrompter()
			}

			return runTranscribeConfig(cmd, p)
		},
	}
}

func runTranscribeConfig(cmd *cobra.Command, prompter Prompter) error {
	// Find vault root first
	vaultRoot, err := vault.FindVaultRoot()
	if err != nil {
		return fmt.Errorf("not in a vault: %w", err)
	}

	out := cmd.OutOrStdout()

	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Transcription Service Configuration")
	fmt.Fprintln(out, "===================================")
	fmt.Fprintln(out, "")

	// Prompt for watch_dir (required)
	watchDir, err := promptRequired(prompter, "Watch folder [required]: ")
	if err != nil {
		return err
	}

	// Prompt for api_url (required)
	apiURL, err := promptRequired(prompter, "Transcription API URL [required]: ")
	if err != nil {
		return err
	}

	// Prompt for output_dir (required)
	outputDir, err := promptRequired(prompter, "Output location (inbox) [required]: ")
	if err != nil {
		return err
	}

	// Prompt for template_path (optional)
	templatePath, err := prompter.Prompt("Template file [optional, Enter to skip]: ")
	if err != nil {
		return err
	}

	// Prompt for archive_dir (optional with default)
	archiveDir, err := prompter.Prompt(fmt.Sprintf("Audio archive location [default: %s]: ", transcribe.DefaultArchiveDir))
	if err != nil {
		return err
	}
	if archiveDir == "" {
		archiveDir = transcribe.DefaultArchiveDir
	}

	// Build config
	cfg := &transcribe.Config{
		WatchDir:   watchDir,
		APIURL:     apiURL,
		OutputDir:  outputDir,
		ArchiveDir: archiveDir,
	}

	// Set template path if provided
	if templatePath != "" {
		cfg.TemplatePath = &templatePath
	}

	// Apply defaults for advanced settings
	cfg.ApplyDefaults()

	// Validate
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Save to vault
	if err := cfg.SaveToVault(vaultRoot); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Show summary
	fmt.Fprintln(out, "")
	configPath := fmt.Sprintf("%s/.nota/transcribe.json", vaultRoot)
	fmt.Fprintf(out, "Configuration saved to %s\n", configPath)

	return nil
}

// promptRequired prompts for a required field, returning an error if empty
func promptRequired(prompter Prompter, prompt string) (string, error) {
	value, err := prompter.Prompt(prompt)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("value is required")
	}
	return value, nil
}

// newTranscribeStartCmd creates the transcribe start command
func newTranscribeStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start transcription service in foreground mode",
		Long: `Start the transcription service in foreground mode.

The service watches for audio files and automatically transcribes them using
a whisper-asr-webservice instance. Configuration is read from .nota/transcribe.json
in the current vault.

The service runs until interrupted with Ctrl+C or SIGTERM.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration from vault
			cfg, err := transcribe.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Create and run service
			svc, err := transcribe.NewService(cfg)
			if err != nil {
				return fmt.Errorf("create service: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Starting transcription service...")
			fmt.Fprintf(cmd.OutOrStdout(), "Watching: %s\n", cfg.WatchDir)
			fmt.Fprintf(cmd.OutOrStdout(), "Output:   %s\n", cfg.OutputDir)
			fmt.Fprintln(cmd.OutOrStdout(), "Press Ctrl+C to stop")
			fmt.Fprintln(cmd.OutOrStdout())

			return svc.Run(context.Background())
		},
	}
}

// stopTimeout is the maximum time to wait for graceful shutdown before sending SIGKILL
const stopTimeout = 10 * time.Second

// ErrNotRunning indicates the transcription service is not running
var ErrNotRunning = errors.New("transcription service is not running")

// ErrStaleProcess indicates the PID file exists but the process is not running
var ErrStaleProcess = errors.New("stale PID file (process not running)")

// newTranscribeStopCmd creates the transcribe stop command
func newTranscribeStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the transcription service",
		Long: `Stop the transcription service.

Reads the PID from ~/.nota/transcribe.pid and sends SIGTERM for graceful shutdown.
If the process doesn't exit within 10 seconds, SIGKILL is sent to force termination.
The PID file is removed after the process exits.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTranscribeStop(cmd)
		},
	}
}

// runTranscribeStop stops the transcription service
func runTranscribeStop(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	pidPath := transcribe.PidFilePath()

	// Read PID file
	pid, err := readPidFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotRunning
		}
		return fmt.Errorf("read PID file: %w", err)
	}

	// Check if process is running
	process, err := os.FindProcess(pid)
	if err != nil {
		// On Unix, FindProcess always succeeds, so this shouldn't happen
		return fmt.Errorf("find process: %w", err)
	}

	// Check if process exists by sending signal 0
	if err := process.Signal(syscall.Signal(0)); err != nil {
		// Process doesn't exist - clean up stale PID file
		if removeErr := os.Remove(pidPath); removeErr != nil && !os.IsNotExist(removeErr) {
			fmt.Fprintf(out, "Warning: failed to remove stale PID file: %v\n", removeErr)
		}
		return ErrStaleProcess
	}

	fmt.Fprintf(out, "Stopping transcription service (PID %d)...\n", pid)

	// Send SIGTERM for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("send SIGTERM: %w", err)
	}

	// Wait for process to exit with timeout
	exited := waitForExit(pid, stopTimeout)

	if !exited {
		// Process didn't exit gracefully, send SIGKILL
		fmt.Fprintln(out, "Process did not exit gracefully, sending SIGKILL...")
		if err := process.Signal(syscall.SIGKILL); err != nil {
			// Process may have exited between check and kill
			if !errors.Is(err, os.ErrProcessDone) {
				return fmt.Errorf("send SIGKILL: %w", err)
			}
		}
		// Wait a bit more for SIGKILL to take effect
		waitForExit(pid, 2*time.Second)
	}

	// Remove PID file
	if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(out, "Warning: failed to remove PID file: %v\n", err)
	}

	fmt.Fprintln(out, "Transcription service stopped")
	return nil
}

// readPidFile reads the PID from the specified file
func readPidFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	if pid <= 0 {
		return 0, fmt.Errorf("invalid PID: %d", pid)
	}

	return pid, nil
}

// waitForExit polls until the process exits or timeout is reached
func waitForExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	pollInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		process, err := os.FindProcess(pid)
		if err != nil {
			return true // Process gone
		}

		// Check if process still exists
		if err := process.Signal(syscall.Signal(0)); err != nil {
			return true // Process gone
		}

		time.Sleep(pollInterval)
	}

	return false // Timeout
}
