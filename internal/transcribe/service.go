// Package transcribe provides the main transcription service orchestrator.
package transcribe

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/TechnicallyShaun/nota-orbis/internal/transcribe/archiver"
	"github.com/TechnicallyShaun/nota-orbis/internal/transcribe/client"
	"github.com/TechnicallyShaun/nota-orbis/internal/transcribe/logging"
	"github.com/TechnicallyShaun/nota-orbis/internal/transcribe/stabilizer"
	"github.com/TechnicallyShaun/nota-orbis/internal/transcribe/watcher"
	"github.com/TechnicallyShaun/nota-orbis/internal/transcribe/writer"
)

// Service orchestrates the transcription pipeline.
type Service struct {
	config     *Config
	logger     *logging.FileLogger
	watcher    *watcher.InotifyWatcher
	stabilizer *stabilizer.PollStabilizer
	client     *client.WhisperASRClient
	writer     *writer.SimpleWriter
	archiver   *archiver.SimpleArchiver

	wg       sync.WaitGroup
	stopCh   chan struct{}
	eventsCh <-chan watcher.FileEvent
}

// NewService creates a new transcription service with all components initialized.
func NewService(cfg *Config) (*Service, error) {
	// Apply defaults for optional fields
	cfg.ApplyDefaults()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Initialize logger
	logConfig := logging.DefaultConfig()
	logConfig.Component = "service"
	logger, err := logging.New(logConfig)
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}

	// Initialize file watcher
	fw, err := watcher.NewInotifyWatcher()
	if err != nil {
		logger.Close()
		return nil, fmt.Errorf("create watcher: %w", err)
	}

	// Initialize stabilizer
	interval := time.Duration(cfg.StabilizationIntervalMs) * time.Millisecond
	stab := stabilizer.NewPollStabilizer(interval, cfg.StabilizationChecks)

	// Initialize transcription client
	tc := client.NewWhisperASRClient(cfg.APIURL)

	// Initialize output writer
	ow := writer.NewSimpleWriter()

	// Initialize archiver
	arch := archiver.NewSimpleArchiver()

	return &Service{
		config:     cfg,
		logger:     logger,
		watcher:    fw,
		stabilizer: stab,
		client:     tc,
		writer:     ow,
		archiver:   arch,
		stopCh:     make(chan struct{}),
	}, nil
}

// Run starts the transcription service and blocks until stopped.
// It handles SIGINT and SIGTERM for graceful shutdown.
func (s *Service) Run(ctx context.Context) error {
	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start file watcher
	s.logger.Info("starting transcription service",
		logging.String("watch_dir", s.config.WatchDir),
		logging.String("api_url", s.config.APIURL),
		logging.String("output_dir", s.config.OutputDir),
	)

	events, err := s.watcher.Watch(ctx, s.config.WatchDir, s.config.WatchPatterns)
	if err != nil {
		return fmt.Errorf("start watcher: %w", err)
	}
	s.eventsCh = events

	s.logger.Info("watching for files",
		logging.String("patterns", fmt.Sprintf("%v", s.config.WatchPatterns)),
	)

	// Main event loop
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("context cancelled, shutting down")
			return s.shutdown()

		case sig := <-sigCh:
			s.logger.Info("received signal, shutting down",
				logging.String("signal", sig.String()),
			)
			cancel()
			return s.shutdown()

		case event, ok := <-events:
			if !ok {
				s.logger.Info("watcher channel closed")
				return s.shutdown()
			}
			s.handleFileEvent(ctx, event)
		}
	}
}

// handleFileEvent processes a single file through the transcription pipeline.
func (s *Service) handleFileEvent(ctx context.Context, event watcher.FileEvent) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.processFile(ctx, event)
	}()
}

// processFile runs the full transcription pipeline for a single file.
func (s *Service) processFile(ctx context.Context, event watcher.FileEvent) {
	fileLogger := s.logger.WithComponent("pipeline")
	startTime := time.Now()

	fileLogger.Info("processing file",
		logging.String("path", event.Path),
		logging.Int64("size", event.Size),
	)

	// Check file size
	maxSize := int64(s.config.MaxFileSizeMB) * 1024 * 1024
	if event.Size > maxSize {
		fileLogger.Error("file too large, skipping", nil,
			logging.String("path", event.Path),
			logging.Int64("size", event.Size),
			logging.Int64("max_size", maxSize),
		)
		return
	}

	// Step 1: Wait for file to stabilize
	fileLogger.Debug("waiting for file to stabilize",
		logging.String("path", event.Path),
	)

	if err := s.stabilizer.WaitForStable(ctx, event.Path); err != nil {
		fileLogger.Error("stabilization failed", err,
			logging.String("path", event.Path),
		)
		return
	}

	fileLogger.Debug("file stabilized",
		logging.String("path", event.Path),
	)

	// Step 2: Transcribe the file
	fileLogger.Info("sending for transcription",
		logging.String("path", event.Path),
	)

	opts := client.TranscribeOptions{
		Language: s.config.Language,
		Model:    s.config.Model,
	}

	var result *client.TranscriptionResult
	var transcribeErr error

	for attempt := 1; attempt <= s.config.RetryCount; attempt++ {
		result, transcribeErr = s.client.Transcribe(ctx, event.Path, opts)
		if transcribeErr == nil {
			break
		}

		if attempt < s.config.RetryCount {
			fileLogger.Error("transcription failed, retrying", transcribeErr,
				logging.String("path", event.Path),
				logging.Int("attempt", attempt),
				logging.Int("max_attempts", s.config.RetryCount),
			)
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	if transcribeErr != nil {
		fileLogger.Error("transcription failed after retries", transcribeErr,
			logging.String("path", event.Path),
			logging.Int("attempts", s.config.RetryCount),
		)
		return
	}

	fileLogger.Info("transcription complete",
		logging.String("path", event.Path),
		logging.String("language", result.Language),
	)

	// Step 3: Write output
	writeOpts := writer.OutputOptions{
		OutputDir:  s.config.OutputDir,
		SourceFile: event.Path,
		Timestamp:  event.Timestamp,
	}
	if s.config.TemplatePath != nil {
		writeOpts.TemplatePath = *s.config.TemplatePath
	}

	outputPath, err := s.writer.Write(ctx, result.Text, writeOpts)
	if err != nil {
		fileLogger.Error("failed to write output", err,
			logging.String("path", event.Path),
		)
		return
	}

	fileLogger.Info("output written",
		logging.String("source", event.Path),
		logging.String("output", outputPath),
	)

	// Step 4: Archive the original file
	if err := s.archiver.Archive(ctx, event.Path, s.config.ArchiveDir); err != nil {
		fileLogger.Error("failed to archive file", err,
			logging.String("path", event.Path),
		)
		return
	}

	elapsed := time.Since(startTime)
	fileLogger.Info("file processing complete",
		logging.String("path", event.Path),
		logging.String("output", outputPath),
		logging.Duration("elapsed", elapsed),
	)
}

// shutdown performs graceful shutdown of the service.
func (s *Service) shutdown() error {
	close(s.stopCh)

	// Stop the watcher
	if err := s.watcher.Stop(); err != nil {
		s.logger.Error("error stopping watcher", err)
	}

	// Wait for in-flight file processing to complete
	s.logger.Info("waiting for in-flight processing to complete")
	s.wg.Wait()

	// Close the logger
	s.logger.Info("transcription service stopped")
	return s.logger.Close()
}

// Stop signals the service to stop.
func (s *Service) Stop() {
	close(s.stopCh)
}
