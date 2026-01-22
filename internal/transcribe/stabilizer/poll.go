package stabilizer

import (
	"context"
	"errors"
	"os"
	"time"
)

// ErrStabilizationTimeout is returned when the file does not stabilize within the timeout.
var ErrStabilizationTimeout = errors.New("stabilization timeout: file did not stabilize in time")

// PollStabilizer implements Stabilizer using polling.
type PollStabilizer struct {
	// Interval is the duration between file size checks.
	Interval time.Duration

	// Checks is the number of consecutive stable checks required.
	Checks int

	// Timeout is the maximum duration to wait for stabilization.
	// If zero, no timeout is applied (relies on context).
	Timeout time.Duration
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
//
// The method respects both the provided context and the Timeout field. If Timeout
// is set and the context has no deadline, a timeout context is created internally.
// Returns ErrStabilizationTimeout if the internal timeout expires before stability is achieved.
func (s *PollStabilizer) WaitForStable(ctx context.Context, path string) error {
	// Apply timeout if configured and context has no deadline
	usingInternalTimeout := false
	if s.Timeout > 0 {
		_, hasDeadline := ctx.Deadline()
		if !hasDeadline {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, s.Timeout)
			defer cancel()
			usingInternalTimeout = true
		}
	}

	var lastSize int64 = -1
	stableCount := 0

	for stableCount < s.Checks {
		select {
		case <-ctx.Done():
			if usingInternalTimeout && errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return ErrStabilizationTimeout
			}
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
