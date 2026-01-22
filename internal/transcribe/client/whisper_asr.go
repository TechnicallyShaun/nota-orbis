// Package client provides transcription client implementations.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// TranscriptionClient sends audio and receives text.
type TranscriptionClient interface {
	Transcribe(ctx context.Context, audioPath string, opts TranscribeOptions) (*TranscriptionResult, error)
}

// TranscribeOptions configures the transcription request.
type TranscribeOptions struct {
	Language string
	Model    string
}

// TranscriptionResult contains the API response.
type TranscriptionResult struct {
	Text     string
	Language string
	Duration float64
}

// OutputFormat specifies the response format from the transcription API.
type OutputFormat string

const (
	OutputFormatText OutputFormat = "text"
	OutputFormatJSON OutputFormat = "json"
)

// DefaultTimeout is the default HTTP request timeout.
const DefaultTimeout = 5 * time.Minute

// WhisperASRClient implements TranscriptionClient for onerahmet/openai-whisper-asr-webservice.
type WhisperASRClient struct {
	baseURL    string
	httpClient *http.Client
	output     OutputFormat
}

// WhisperASROption configures the WhisperASRClient.
type WhisperASROption func(*WhisperASRClient)

// WithTimeout sets the HTTP request timeout.
func WithTimeout(d time.Duration) WhisperASROption {
	return func(c *WhisperASRClient) {
		c.httpClient.Timeout = d
	}
}

// WithOutputFormat sets the response format (text or json).
func WithOutputFormat(format OutputFormat) WhisperASROption {
	return func(c *WhisperASRClient) {
		c.output = format
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) WhisperASROption {
	return func(c *WhisperASRClient) {
		c.httpClient = client
	}
}

// NewWhisperASRClient creates a new client for the whisper-asr-webservice.
func NewWhisperASRClient(baseURL string, opts ...WhisperASROption) *WhisperASRClient {
	c := &WhisperASRClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		output: OutputFormatJSON,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Transcribe sends an audio file to the whisper-asr-webservice and returns the transcription.
func (c *WhisperASRClient) Transcribe(ctx context.Context, audioPath string, opts TranscribeOptions) (*TranscriptionResult, error) {
	// Open the audio file
	file, err := os.Open(audioPath)
	if err != nil {
		return nil, fmt.Errorf("open audio file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("audio_file", filepath.Base(audioPath))
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("copy audio data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	// Build request URL with query parameters
	reqURL, err := c.buildURL(opts)
	if err != nil {
		return nil, fmt.Errorf("build URL: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response based on output format
	return c.parseResponse(resp.Body)
}

func (c *WhisperASRClient) buildURL(opts TranscribeOptions) (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}

	// Ensure path ends with /asr
	if u.Path == "" || u.Path == "/" {
		u.Path = "/asr"
	}

	q := u.Query()
	q.Set("output", string(c.output))

	if opts.Language != "" && opts.Language != "auto" {
		q.Set("language", opts.Language)
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c *WhisperASRClient) parseResponse(body io.Reader) (*TranscriptionResult, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if c.output == OutputFormatText {
		return &TranscriptionResult{
			Text: string(data),
		}, nil
	}

	// Parse JSON response
	var resp whisperASRResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse JSON response: %w", err)
	}

	return &TranscriptionResult{
		Text:     resp.Text,
		Language: resp.Language,
	}, nil
}

// whisperASRResponse represents the JSON response from the whisper-asr-webservice.
type whisperASRResponse struct {
	Text     string `json:"text"`
	Language string `json:"language"`
}
