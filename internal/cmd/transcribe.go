package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

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
