package elevenlabs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestTranscribeSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify API key header
		if r.Header.Get("xi-api-key") != "test-key" {
			t.Errorf("expected xi-api-key header 'test-key', got %q", r.Header.Get("xi-api-key"))
		}
		// Verify method
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		// Verify path
		if r.URL.Path != "/v1/speech-to-text" {
			t.Errorf("expected path /v1/speech-to-text, got %s", r.URL.Path)
		}
		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"text": "Hello world transcript"})
	}))
	defer server.Close()

	tr := &Transcriber{apiKey: "test-key", model: "scribe_v1", baseURL: server.URL}

	// Create a temp audio file
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "attempt-1.mp3")
	if err := os.WriteFile(audioPath, []byte("fake audio data"), 0o644); err != nil {
		t.Fatal(err)
	}

	text, err := tr.Transcribe(audioPath)
	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}
	if text != "Hello world transcript" {
		t.Errorf("Transcribe() = %q, want %q", text, "Hello world transcript")
	}
}

func TestTranscribeAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid api key"}`))
	}))
	defer server.Close()

	tr := &Transcriber{apiKey: "bad-key", model: "scribe_v1", baseURL: server.URL}

	dir := t.TempDir()
	audioPath := filepath.Join(dir, "attempt-1.mp3")
	os.WriteFile(audioPath, []byte("fake"), 0o644)

	_, err := tr.Transcribe(audioPath)
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
}

func TestTranscribeEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"text": ""})
	}))
	defer server.Close()

	tr := &Transcriber{apiKey: "test-key", model: "scribe_v1", baseURL: server.URL}

	dir := t.TempDir()
	audioPath := filepath.Join(dir, "attempt-1.mp3")
	os.WriteFile(audioPath, []byte("fake"), 0o644)

	text, err := tr.Transcribe(audioPath)
	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}
	if text != "" {
		t.Errorf("expected empty string for empty response, got %q", text)
	}
}

func TestNewFactory(t *testing.T) {
	config := map[string]any{
		"api_key": "my-key",
		"model":   "scribe_v1",
	}
	tr := NewFactory(config)
	if tr.Name() != "elevenlabs" {
		t.Errorf("Name() = %q, want %q", tr.Name(), "elevenlabs")
	}
}

func TestAPIKeyFromEnv(t *testing.T) {
	os.Setenv("ELEVENLABS_API_KEY", "env-key")
	defer os.Unsetenv("ELEVENLABS_API_KEY")

	config := map[string]any{} // no api_key in config
	tr := NewFactory(config)
	el := tr.(*Transcriber)
	if el.apiKey != "env-key" {
		t.Errorf("apiKey = %q, want %q from env", el.apiKey, "env-key")
	}
}

func TestNoAPIKey(t *testing.T) {
	os.Unsetenv("ELEVENLABS_API_KEY")

	config := map[string]any{}
	tr := NewFactory(config)
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "attempt-1.mp3")
	os.WriteFile(audioPath, []byte("fake"), 0o644)

	_, err := tr.Transcribe(audioPath)
	if err == nil {
		t.Fatal("expected error when no API key configured")
	}
}
