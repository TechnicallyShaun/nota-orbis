package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

// DefaultRetryCount is the default number of retry attempts.
const DefaultRetryCount = 3

// DefaultBaseDelay is the initial delay for exponential backoff.
const DefaultBaseDelay = 1 * time.Second

// RetryClient wraps a TranscriptionClient with retry logic and exponential backoff.
type RetryClient struct {
	client    TranscriptionClient
	maxRetry  int
	baseDelay time.Duration
	logger    *log.Logger
}

// RetryOption configures the RetryClient.
type RetryOption func(*RetryClient)

// WithRetryCount sets the maximum number of retry attempts.
func WithRetryCount(n int) RetryOption {
	return func(c *RetryClient) {
		c.maxRetry = n
	}
}

// WithBaseDelay sets the initial delay for exponential backoff.
func WithBaseDelay(d time.Duration) RetryOption {
	return func(c *RetryClient) {
		c.baseDelay = d
	}
}

// WithLogger sets a custom logger for retry attempts.
func WithLogger(l *log.Logger) RetryOption {
	return func(c *RetryClient) {
		c.logger = l
	}
}

// NewRetryClient creates a new RetryClient wrapping the given TranscriptionClient.
func NewRetryClient(client TranscriptionClient, opts ...RetryOption) *RetryClient {
	c := &RetryClient{
		client:    client,
		maxRetry:  DefaultRetryCount,
		baseDelay: DefaultBaseDelay,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Transcribe sends an audio file for transcription with retry logic.
// It retries on connection errors and 5xx responses, but not on 4xx client errors.
func (c *RetryClient) Transcribe(ctx context.Context, audioPath string, opts TranscribeOptions) (*TranscriptionResult, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetry; attempt++ {
		if attempt > 0 {
			delay := c.baseDelay * (1 << (attempt - 1)) // Exponential: 1s, 2s, 4s, 8s...
			c.logRetry(attempt, delay, lastErr)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		result, err := c.client.Transcribe(ctx, audioPath, opts)
		if err == nil {
			return result, nil
		}

		if !isRetryable(err) {
			return nil, err
		}

		lastErr = err
	}

	return nil, fmt.Errorf("transcription failed after %d retries: %w", c.maxRetry, lastErr)
}

// isRetryable determines if an error should trigger a retry.
// Returns true for connection errors and 5xx server errors.
// Returns false for 4xx client errors.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for context cancellation - not retryable
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check for network errors - retryable
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Check for connection refused/reset - retryable
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// Parse API error status codes
	errStr := err.Error()
	if strings.Contains(errStr, "API error: status ") {
		var status int
		if _, scanErr := fmt.Sscanf(errStr, "API error: status %d", &status); scanErr == nil {
			// 4xx client errors are not retryable
			if status >= 400 && status < 500 {
				return false
			}
			// 5xx server errors are retryable
			if status >= 500 && status < 600 {
				return true
			}
		}
	}

	// Connection errors in wrapped error messages - retryable
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "send request:") {
		return true
	}

	// Default: don't retry unknown errors
	return false
}

func (c *RetryClient) logRetry(attempt int, delay time.Duration, err error) {
	if c.logger != nil {
		c.logger.Printf("retry attempt %d/%d after %v: %v", attempt, c.maxRetry, delay, err)
	}
}
