package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/TechnicallyShaun/nota-orbis/internal/transcribe"
	"github.com/TechnicallyShaun/nota-orbis/internal/transcribe/pidfile"
	"github.com/TechnicallyShaun/nota-orbis/internal/transcribe/status"
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

	cmd.AddCommand(NewTranscribeConfigCmd(nil, false))
	cmd.AddCommand(newTranscribeStartCmd())
	cmd.AddCommand(newTranscribeStopCmd())
	cmd.AddCommand(newTranscribeStatusCmd())

	return cmd
}

// NewTranscribeConfigCmd creates the config subcommand
func NewTranscribeConfigCmd(prompter Prompter, advanced bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure transcription service",
		Long:  "Interactive configuration for the transcription service",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use provided prompter or create stdin prompter
			p := prompter
			if p == nil {
				p = NewStdinPrompter()
			}

			advancedFlag, _ := cmd.Flags().GetBool("advanced")
			return runTranscribeConfig(cmd, p, advancedFlag || advanced)
		},
	}

	cmd.Flags().Bool("advanced", false, "Prompt for advanced configuration options")

	return cmd
}

func runTranscribeConfig(cmd *cobra.Command, prompter Prompter, advanced bool) error {
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

	// Apply defaults for advanced settings first
	cfg.ApplyDefaults()

	// If advanced mode, prompt for advanced settings
	if advanced {
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Advanced Settings (press Enter to accept defaults)")
		fmt.Fprintln(out, "-------------------------------------------------")

		// Stabilization interval
		stabInterval, err := prompter.Prompt(fmt.Sprintf("Stabilization interval in ms [default: %d]: ", transcribe.DefaultStabilizationIntervalMs))
		if err != nil {
			return err
		}
		if stabInterval != "" {
			if val, err := strconv.Atoi(stabInterval); err == nil && val > 0 {
				cfg.StabilizationIntervalMs = val
			}
		}

		// Stabilization checks
		stabChecks, err := prompter.Prompt(fmt.Sprintf("Stabilization checks [default: %d]: ", transcribe.DefaultStabilizationChecks))
		if err != nil {
			return err
		}
		if stabChecks != "" {
			if val, err := strconv.Atoi(stabChecks); err == nil && val > 0 {
				cfg.StabilizationChecks = val
			}
		}

		// Language
		language, err := prompter.Prompt(fmt.Sprintf("Language [default: %s]: ", transcribe.DefaultLanguage))
		if err != nil {
			return err
		}
		if language != "" {
			cfg.Language = language
		}

		// Model
		model, err := prompter.Prompt(fmt.Sprintf("Model [default: %s]: ", transcribe.DefaultModel))
		if err != nil {
			return err
		}
		if model != "" {
			cfg.Model = model
		}

		// Max file size
		maxFileSize, err := prompter.Prompt(fmt.Sprintf("Max file size in MB [default: %d]: ", transcribe.DefaultMaxFileSizeMB))
		if err != nil {
			return err
		}
		if maxFileSize != "" {
			if val, err := strconv.Atoi(maxFileSize); err == nil && val > 0 {
				cfg.MaxFileSizeMB = val
			}
		}

		// Retry count
		retryCount, err := prompter.Prompt(fmt.Sprintf("Retry count [default: %d]: ", transcribe.DefaultRetryCount))
		if err != nil {
			return err
		}
		if retryCount != "" {
			if val, err := strconv.Atoi(retryCount); err == nil && val >= 0 {
				cfg.RetryCount = val
			}
		}

		// Watch patterns
		defaultPatterns := strings.Join(transcribe.DefaultWatchPatterns, ",")
		watchPatterns, err := prompter.Prompt(fmt.Sprintf("Watch patterns (comma-separated) [default: %s]: ", defaultPatterns))
		if err != nil {
			return err
		}
		if watchPatterns != "" {
			patterns := strings.Split(watchPatterns, ",")
			cfg.WatchPatterns = make([]string, 0, len(patterns))
			for _, p := range patterns {
				p = strings.TrimSpace(p)
				if p != "" {
					cfg.WatchPatterns = append(cfg.WatchPatterns, p)
				}
			}
		}
	}

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
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start transcription service",
		Long: `Start the transcription service.

The service watches for audio files and automatically transcribes them using
a whisper-asr-webservice instance. Configuration is read from .nota/transcribe.json
in the current vault.

Use --daemon to run in the background. The service runs until stopped with
'nota transcribe stop' or interrupted with Ctrl+C/SIGTERM.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			daemon, _ := cmd.Flags().GetBool("daemon")
			daemonChild, _ := cmd.Flags().GetBool("daemon-child")

			if daemon {
				return runDaemon(cmd)
			}

			if daemonChild {
				// Running as daemon child - write PID file
				if err := pidfile.Write(os.Getpid()); err != nil {
					return fmt.Errorf("write PID file: %w", err)
				}
			}

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

			if !daemonChild {
				fmt.Fprintln(cmd.OutOrStdout(), "Starting transcription service...")
				fmt.Fprintf(cmd.OutOrStdout(), "Watching: %s\n", cfg.WatchDir)
				fmt.Fprintf(cmd.OutOrStdout(), "Output:   %s\n", cfg.OutputDir)
				fmt.Fprintln(cmd.OutOrStdout(), "Press Ctrl+C to stop")
				fmt.Fprintln(cmd.OutOrStdout())
			}

			err = svc.Run(context.Background())

			// Clean up PID file if we were a daemon child
			if daemonChild {
				pidfile.Remove()
			}

			return err
		},
	}

	cmd.Flags().Bool("daemon", false, "Run in background as daemon")
	cmd.Flags().Bool("daemon-child", false, "Internal flag for daemon child process")
	cmd.Flags().MarkHidden("daemon-child")

	return cmd
}

// runDaemon spawns a daemon child process
func runDaemon(cmd *cobra.Command) error {
	// Check if already running
	running, pid, err := pidfile.IsRunning()
	if err != nil {
		return fmt.Errorf("check running status: %w", err)
	}
	if running {
		return fmt.Errorf("transcription service is already running (PID %d)", pid)
	}

	// Clean up stale PID file if any
	pidfile.CleanStale()

	// Find vault root for the child process
	vaultRoot, err := vault.FindVaultRoot()
	if err != nil {
		return fmt.Errorf("not in a vault: %w", err)
	}

	// Get executable path
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}

	// Open log file for stdout/stderr
	logPath, err := status.TodayLogPath()
	if err != nil {
		return fmt.Errorf("get log path: %w", err)
	}

	// Ensure log directory exists
	logDir := logPath[:len(logPath)-len("/transcribe-2006-01-02.log")+1]
	if idx := strings.LastIndex(logPath, "/"); idx >= 0 {
		logDir = logPath[:idx]
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	// Spawn child process
	childCmd := exec.Command(exe, "transcribe", "start", "--daemon-child")
	childCmd.Env = append(os.Environ(), vault.EnvVaultRoot+"="+vaultRoot)
	childCmd.Stdout = logFile
	childCmd.Stderr = logFile
	childCmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	// Open /dev/null for stdin
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		logFile.Close()
		return fmt.Errorf("open /dev/null: %w", err)
	}
	childCmd.Stdin = devNull

	if err := childCmd.Start(); err != nil {
		devNull.Close()
		logFile.Close()
		return fmt.Errorf("start daemon: %w", err)
	}

	childPID := childCmd.Process.Pid

	// Close file handles - child has them now
	devNull.Close()
	logFile.Close()

	// Write PID file for the child
	if err := pidfile.Write(childPID); err != nil {
		// Try to kill the child if we can't write PID file
		childCmd.Process.Kill()
		return fmt.Errorf("write PID file: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Transcription service started (PID %d)\n", childPID)
	fmt.Fprintf(cmd.OutOrStdout(), "Logs: %s\n", logPath)

	return nil
}

// newTranscribeStopCmd creates the transcribe stop command
func newTranscribeStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the transcription service daemon",
		Long:  "Gracefully stops the background transcription service.",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			// Check if running
			running, pid, err := pidfile.IsRunning()
			if err != nil {
				return fmt.Errorf("check running status: %w", err)
			}

			if !running {
				if pid > 0 {
					// Stale PID file
					pidfile.Remove()
					fmt.Fprintln(out, "Transcription service is not running (cleaned stale PID file)")
				} else {
					fmt.Fprintln(out, "Transcription service is not running")
				}
				return nil
			}

			fmt.Fprintf(out, "Stopping transcription service (PID %d)...\n", pid)

			// Send SIGTERM
			process, err := os.FindProcess(pid)
			if err != nil {
				return fmt.Errorf("find process: %w", err)
			}

			if err := process.Signal(syscall.SIGTERM); err != nil {
				return fmt.Errorf("send SIGTERM: %w", err)
			}

			// Wait for graceful shutdown (5 seconds)
			stopped := false
			for i := 0; i < 50; i++ { // 50 * 100ms = 5s
				time.Sleep(100 * time.Millisecond)
				running, _, _ = pidfile.IsRunning()
				if !running {
					stopped = true
					break
				}
			}

			if !stopped {
				// Force kill
				fmt.Fprintln(out, "Graceful shutdown timed out, sending SIGKILL...")
				if err := process.Signal(syscall.SIGKILL); err != nil {
					return fmt.Errorf("send SIGKILL: %w", err)
				}
				// Wait a bit for SIGKILL to take effect
				time.Sleep(500 * time.Millisecond)
			}

			// Remove PID file
			if err := pidfile.Remove(); err != nil {
				return fmt.Errorf("remove PID file: %w", err)
			}

			fmt.Fprintln(out, "Transcription service stopped")
			return nil
		},
	}
}

// newTranscribeStatusCmd creates the transcribe status command
func newTranscribeStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show transcription service status",
		Long:  "Shows the current status of the transcription service daemon.",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			// Check if running
			running, pid, err := pidfile.IsRunning()
			if err != nil {
				return fmt.Errorf("check running status: %w", err)
			}

			if !running {
				fmt.Fprintln(out, "Status: not running")
				return nil
			}

			fmt.Fprintf(out, "Status: running (pid %d)\n", pid)

			// Try to load config to show watch directory
			cfg, err := transcribe.Load()
			if err == nil {
				fmt.Fprintf(out, "Watching: %s\n", cfg.WatchDir)
			}

			// Parse today's stats
			stats, err := status.ParseTodayStats()
			if err != nil {
				// Don't fail if we can't parse stats
				return nil
			}

			if stats.LastProcessed != nil {
				fmt.Fprintf(out, "Last processed: %s (%s)\n",
					status.FormatTimestamp(stats.LastProcessed.Timestamp),
					status.BaseName(stats.LastProcessed.Path))
			}

			fmt.Fprintf(out, "Files processed today: %d\n", stats.FilesProcessed)
			fmt.Fprintf(out, "Errors today: %d\n", stats.Errors)

			return nil
		},
	}
}
