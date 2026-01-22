// Package stabilizer provides file stability checking for the transcription service.
package stabilizer

import (
	"context"
	"os"
	"time"
)

// Stabilizer waits for a file to finish writing.
type Stabilizer interface {
	WaitForStable(ctx context.Context, path string) error
}

// PollStabilizer implements Stabilizer using polling.
type PollStabilizer struct {
	Interval time.Duration
	Checks   int
}

// NewPollStabilizer creates a new polling-based stabilizer.
func NewPollStabilizer(interval time.Duration, checks int) *PollStabilizer {
	return &PollStabilizer{
		Interval: interval,
		Checks:   checks,
	}
}

// WaitForStable waits until the file size remains constant for the configured
// number of consecutive checks.
func (s *PollStabilizer) WaitForStable(ctx context.Context, path string) error {
	var lastSize int64 = -1
	stableCount := 0

	for stableCount < s.Checks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.Interval):
		}

		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		currentSize := info.Size()
		if currentSize == lastSize {
			stableCount++
		} else {
			stableCount = 0
			lastSize = currentSize
		}
	}

	return nil
}
