package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"testing"
	"time"
)

// mockClient is a test double for TranscriptionClient.
type mockClient struct {
	results []mockResult
	calls   int
}

type mockResult struct {
	result *TranscriptionResult
	err    error
}

func (m *mockClient) Transcribe(ctx context.Context, audioPath string, opts TranscribeOptions) (*TranscriptionResult, error) {
	if m.calls >= len(m.results) {
		return nil, errors.New("unexpected call")
	}
	r := m.results[m.calls]
	m.calls++
	return r.result, r.err
}

func TestRetryClient_SuccessFirstAttempt(t *testing.T) {
	mock := &mockClient{
		results: []mockResult{
			{result: &TranscriptionResult{Text: "hello"}, err: nil},
		},
	}

	client := NewRetryClient(mock)
	result, err := client.Transcribe(context.Background(), "test.wav", TranscribeOptions{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "hello" {
		t.Errorf("got %q, want %q", result.Text, "hello")
	}
	if mock.calls != 1 {
		t.Errorf("expected 1 call, got %d", mock.calls)
	}
}

func TestRetryClient_SuccessAfterRetry(t *testing.T) {
	mock := &mockClient{
		results: []mockResult{
			{err: errors.New("API error: status 503: service unavailable")},
			{err: errors.New("API error: status 502: bad gateway")},
			{result: &TranscriptionResult{Text: "success"}, err: nil},
		},
	}

	var logBuf bytes.Buffer
	logger := log.New(&logBuf, "", 0)

	client := NewRetryClient(mock,
		WithRetryCount(3),
		WithBaseDelay(1*time.Millisecond),
		WithLogger(logger),
	)

	result, err := client.Transcribe(context.Background(), "test.wav", TranscribeOptions{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "success" {
		t.Errorf("got %q, want %q", result.Text, "success")
	}
	if mock.calls != 3 {
		t.Errorf("expected 3 calls, got %d", mock.calls)
	}

	// Check that retries were logged
	logOutput := logBuf.String()
	if logOutput == "" {
		t.Error("expected retry log output")
	}
}

func TestRetryClient_NoRetryOn4xx(t *testing.T) {
	mock := &mockClient{
		results: []mockResult{
			{err: errors.New("API error: status 400: bad request")},
		},
	}

	client := NewRetryClient(mock,
		WithRetryCount(3),
		WithBaseDelay(1*time.Millisecond),
	)

	_, err := client.Transcribe(context.Background(), "test.wav", TranscribeOptions{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mock.calls != 1 {
		t.Errorf("expected 1 call (no retry on 4xx), got %d", mock.calls)
	}
}

func TestRetryClient_NoRetryOn404(t *testing.T) {
	mock := &mockClient{
		results: []mockResult{
			{err: errors.New("API error: status 404: not found")},
		},
	}

	client := NewRetryClient(mock,
		WithRetryCount(3),
		WithBaseDelay(1*time.Millisecond),
	)

	_, err := client.Transcribe(context.Background(), "test.wav", TranscribeOptions{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mock.calls != 1 {
		t.Errorf("expected 1 call (no retry on 404), got %d", mock.calls)
	}
}

func TestRetryClient_RetryOnConnectionError(t *testing.T) {
	mock := &mockClient{
		results: []mockResult{
			{err: fmt.Errorf("send request: %w", &net.OpError{Op: "dial", Err: errors.New("connection refused")})},
			{result: &TranscriptionResult{Text: "ok"}, err: nil},
		},
	}

	client := NewRetryClient(mock,
		WithRetryCount(3),
		WithBaseDelay(1*time.Millisecond),
	)

	result, err := client.Transcribe(context.Background(), "test.wav", TranscribeOptions{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "ok" {
		t.Errorf("got %q, want %q", result.Text, "ok")
	}
	if mock.calls != 2 {
		t.Errorf("expected 2 calls, got %d", mock.calls)
	}
}

func TestRetryClient_MaxRetriesExhausted(t *testing.T) {
	mock := &mockClient{
		results: []mockResult{
			{err: errors.New("API error: status 500: internal error")},
			{err: errors.New("API error: status 500: internal error")},
			{err: errors.New("API error: status 500: internal error")},
			{err: errors.New("API error: status 500: internal error")},
		},
	}

	client := NewRetryClient(mock,
		WithRetryCount(3),
		WithBaseDelay(1*time.Millisecond),
	)

	_, err := client.Transcribe(context.Background(), "test.wav", TranscribeOptions{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mock.calls != 4 { // 1 initial + 3 retries
		t.Errorf("expected 4 calls, got %d", mock.calls)
	}
	// Verify error message contains the original error
	if err.Error() != "transcription failed after 3 retries: API error: status 500: internal error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRetryClient_ContextCancellation(t *testing.T) {
	mock := &mockClient{
		results: []mockResult{
			{err: errors.New("API error: status 500: internal error")},
			{err: errors.New("API error: status 500: internal error")},
		},
	}

	client := NewRetryClient(mock,
		WithRetryCount(3),
		WithBaseDelay(100*time.Millisecond),
	)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after first failure during the delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := client.Transcribe(ctx, "test.wav", TranscribeOptions{})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestRetryClient_ExponentialBackoff(t *testing.T) {
	mock := &mockClient{
		results: []mockResult{
			{err: errors.New("API error: status 500: error")},
			{err: errors.New("API error: status 500: error")},
			{err: errors.New("API error: status 500: error")},
			{result: &TranscriptionResult{Text: "done"}, err: nil},
		},
	}

	baseDelay := 10 * time.Millisecond
	client := NewRetryClient(mock,
		WithRetryCount(3),
		WithBaseDelay(baseDelay),
	)

	start := time.Now()
	_, err := client.Transcribe(context.Background(), "test.wav", TranscribeOptions{})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected delays: 10ms + 20ms + 40ms = 70ms
	// Allow some margin for execution time
	expectedMin := 60 * time.Millisecond
	expectedMax := 150 * time.Millisecond

	if elapsed < expectedMin || elapsed > expectedMax {
		t.Errorf("elapsed time %v not in expected range [%v, %v]", elapsed, expectedMin, expectedMax)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"context canceled", context.Canceled, false},
		{"context deadline", context.DeadlineExceeded, false},
		{"400 bad request", errors.New("API error: status 400: bad request"), false},
		{"401 unauthorized", errors.New("API error: status 401: unauthorized"), false},
		{"403 forbidden", errors.New("API error: status 403: forbidden"), false},
		{"404 not found", errors.New("API error: status 404: not found"), false},
		{"422 unprocessable", errors.New("API error: status 422: unprocessable"), false},
		{"500 internal error", errors.New("API error: status 500: internal error"), true},
		{"502 bad gateway", errors.New("API error: status 502: bad gateway"), true},
		{"503 unavailable", errors.New("API error: status 503: service unavailable"), true},
		{"504 timeout", errors.New("API error: status 504: gateway timeout"), true},
		{"connection refused", errors.New("send request: connection refused"), true},
		{"connection reset", errors.New("connection reset by peer"), true},
		{"no such host", errors.New("no such host"), true},
		{"unknown error", errors.New("some random error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryable(tt.err)
			if got != tt.expected {
				t.Errorf("isRetryable(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestRetryClient_DefaultOptions(t *testing.T) {
	mock := &mockClient{
		results: []mockResult{
			{result: &TranscriptionResult{Text: "test"}, err: nil},
		},
	}

	client := NewRetryClient(mock)

	if client.maxRetry != DefaultRetryCount {
		t.Errorf("default maxRetry = %d, want %d", client.maxRetry, DefaultRetryCount)
	}
	if client.baseDelay != DefaultBaseDelay {
		t.Errorf("default baseDelay = %v, want %v", client.baseDelay, DefaultBaseDelay)
	}
}
