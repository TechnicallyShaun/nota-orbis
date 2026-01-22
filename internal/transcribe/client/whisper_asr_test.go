package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewWhisperASRClient(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		c := NewWhisperASRClient("http://localhost:9000")
		if c.baseURL != "http://localhost:9000" {
			t.Errorf("baseURL = %q, want %q", c.baseURL, "http://localhost:9000")
		}
		if c.output != OutputFormatJSON {
			t.Errorf("output = %q, want %q", c.output, OutputFormatJSON)
		}
		if c.httpClient.Timeout != DefaultTimeout {
			t.Errorf("timeout = %v, want %v", c.httpClient.Timeout, DefaultTimeout)
		}
	})

	t.Run("with custom timeout", func(t *testing.T) {
		c := NewWhisperASRClient("http://localhost:9000", WithTimeout(30*time.Second))
		if c.httpClient.Timeout != 30*time.Second {
			t.Errorf("timeout = %v, want %v", c.httpClient.Timeout, 30*time.Second)
		}
	})

	t.Run("with text output format", func(t *testing.T) {
		c := NewWhisperASRClient("http://localhost:9000", WithOutputFormat(OutputFormatText))
		if c.output != OutputFormatText {
			t.Errorf("output = %q, want %q", c.output, OutputFormatText)
		}
	})
}

func TestWhisperASRClient_buildURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		output  OutputFormat
		opts    TranscribeOptions
		want    string
	}{
		{
			name:    "base URL only",
			baseURL: "http://localhost:9000",
			output:  OutputFormatJSON,
			opts:    TranscribeOptions{},
			want:    "http://localhost:9000/asr?output=json",
		},
		{
			name:    "with language",
			baseURL: "http://localhost:9000",
			output:  OutputFormatJSON,
			opts:    TranscribeOptions{Language: "en"},
			want:    "http://localhost:9000/asr?language=en&output=json",
		},
		{
			name:    "with auto language",
			baseURL: "http://localhost:9000",
			output:  OutputFormatJSON,
			opts:    TranscribeOptions{Language: "auto"},
			want:    "http://localhost:9000/asr?output=json",
		},
		{
			name:    "text output format",
			baseURL: "http://localhost:9000",
			output:  OutputFormatText,
			opts:    TranscribeOptions{},
			want:    "http://localhost:9000/asr?output=text",
		},
		{
			name:    "base URL with trailing slash",
			baseURL: "http://localhost:9000/",
			output:  OutputFormatJSON,
			opts:    TranscribeOptions{},
			want:    "http://localhost:9000/asr?output=json",
		},
		{
			name:    "base URL with path",
			baseURL: "http://localhost:9000/api/v1/asr",
			output:  OutputFormatJSON,
			opts:    TranscribeOptions{},
			want:    "http://localhost:9000/api/v1/asr?output=json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewWhisperASRClient(tt.baseURL, WithOutputFormat(tt.output))
			got, err := c.buildURL(tt.opts)
			if err != nil {
				t.Fatalf("buildURL() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("buildURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWhisperASRClient_parseResponse(t *testing.T) {
	t.Run("JSON response", func(t *testing.T) {
		c := NewWhisperASRClient("http://localhost:9000", WithOutputFormat(OutputFormatJSON))
		body := strings.NewReader(`{"text":"Hello, world!","language":"en"}`)
		result, err := c.parseResponse(body)
		if err != nil {
			t.Fatalf("parseResponse() error = %v", err)
		}
		if result.Text != "Hello, world!" {
			t.Errorf("Text = %q, want %q", result.Text, "Hello, world!")
		}
		if result.Language != "en" {
			t.Errorf("Language = %q, want %q", result.Language, "en")
		}
	})

	t.Run("text response", func(t *testing.T) {
		c := NewWhisperASRClient("http://localhost:9000", WithOutputFormat(OutputFormatText))
		body := strings.NewReader("Hello, world!")
		result, err := c.parseResponse(body)
		if err != nil {
			t.Fatalf("parseResponse() error = %v", err)
		}
		if result.Text != "Hello, world!" {
			t.Errorf("Text = %q, want %q", result.Text, "Hello, world!")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		c := NewWhisperASRClient("http://localhost:9000", WithOutputFormat(OutputFormatJSON))
		body := strings.NewReader("not json")
		_, err := c.parseResponse(body)
		if err == nil {
			t.Error("parseResponse() expected error for invalid JSON")
		}
	})
}

func TestWhisperASRClient_Transcribe(t *testing.T) {
	// Create a temporary audio file for testing
	tmpDir := t.TempDir()
	audioFile := filepath.Join(tmpDir, "test.m4a")
	if err := os.WriteFile(audioFile, []byte("fake audio content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Run("successful JSON transcription", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method
			if r.Method != http.MethodPost {
				t.Errorf("Method = %q, want %q", r.Method, http.MethodPost)
			}

			// Verify content type is multipart
			contentType := r.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, "multipart/form-data") {
				t.Errorf("Content-Type = %q, want multipart/form-data", contentType)
			}

			// Verify output query param
			if r.URL.Query().Get("output") != "json" {
				t.Errorf("output = %q, want %q", r.URL.Query().Get("output"), "json")
			}

			// Verify audio file is present
			file, header, err := r.FormFile("audio_file")
			if err != nil {
				t.Errorf("FormFile error: %v", err)
			}
			defer file.Close()

			if header.Filename != "test.m4a" {
				t.Errorf("Filename = %q, want %q", header.Filename, "test.m4a")
			}

			// Read file content to verify
			content, _ := io.ReadAll(file)
			if string(content) != "fake audio content" {
				t.Errorf("File content = %q, want %q", string(content), "fake audio content")
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"text":"Transcribed text","language":"en"}`))
		}))
		defer server.Close()

		c := NewWhisperASRClient(server.URL)
		result, err := c.Transcribe(context.Background(), audioFile, TranscribeOptions{})
		if err != nil {
			t.Fatalf("Transcribe() error = %v", err)
		}

		if result.Text != "Transcribed text" {
			t.Errorf("Text = %q, want %q", result.Text, "Transcribed text")
		}
		if result.Language != "en" {
			t.Errorf("Language = %q, want %q", result.Language, "en")
		}
	})

	t.Run("successful text transcription", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("output") != "text" {
				t.Errorf("output = %q, want %q", r.URL.Query().Get("output"), "text")
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Transcribed text"))
		}))
		defer server.Close()

		c := NewWhisperASRClient(server.URL, WithOutputFormat(OutputFormatText))
		result, err := c.Transcribe(context.Background(), audioFile, TranscribeOptions{})
		if err != nil {
			t.Fatalf("Transcribe() error = %v", err)
		}

		if result.Text != "Transcribed text" {
			t.Errorf("Text = %q, want %q", result.Text, "Transcribed text")
		}
	})

	t.Run("with language option", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("language") != "de" {
				t.Errorf("language = %q, want %q", r.URL.Query().Get("language"), "de")
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"text":"Hallo Welt","language":"de"}`))
		}))
		defer server.Close()

		c := NewWhisperASRClient(server.URL)
		result, err := c.Transcribe(context.Background(), audioFile, TranscribeOptions{Language: "de"})
		if err != nil {
			t.Fatalf("Transcribe() error = %v", err)
		}

		if result.Text != "Hallo Welt" {
			t.Errorf("Text = %q, want %q", result.Text, "Hallo Welt")
		}
	})

	t.Run("API error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal server error"))
		}))
		defer server.Close()

		c := NewWhisperASRClient(server.URL)
		_, err := c.Transcribe(context.Background(), audioFile, TranscribeOptions{})
		if err == nil {
			t.Error("Transcribe() expected error for API error")
		}
		if !strings.Contains(err.Error(), "status 500") {
			t.Errorf("Error should contain status code: %v", err)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		c := NewWhisperASRClient("http://localhost:9000")
		_, err := c.Transcribe(context.Background(), "/nonexistent/file.m4a", TranscribeOptions{})
		if err == nil {
			t.Error("Transcribe() expected error for nonexistent file")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		c := NewWhisperASRClient(server.URL)
		_, err := c.Transcribe(ctx, audioFile, TranscribeOptions{})
		if err == nil {
			t.Error("Transcribe() expected error for cancelled context")
		}
	})
}

func TestTranscriptionClientInterface(t *testing.T) {
	// Verify WhisperASRClient implements TranscriptionClient
	var _ TranscriptionClient = (*WhisperASRClient)(nil)
}
