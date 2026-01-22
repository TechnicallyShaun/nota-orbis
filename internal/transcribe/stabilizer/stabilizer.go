// Package stabilizer provides file stability checking for the transcription service.
package stabilizer

import "context"

// Stabilizer waits for a file to finish writing.
type Stabilizer interface {
	WaitForStable(ctx context.Context, path string) error
}
