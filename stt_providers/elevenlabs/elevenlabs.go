package elevenlabs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	stt "github.com/j178/leetgo/stt_providers"
)

const defaultBaseURL = "https://api.elevenlabs.io"

func init() {
	stt.Register("elevenlabs", NewFactory)
}

// Transcriber implements stt.Transcriber using the ElevenLabs Speech-to-Text API.
type Transcriber struct {
	apiKey  string
	model   string
	baseURL string
}

// NewFactory creates an ElevenLabs transcriber from config.
// Reads api_key from config, falls back to ELEVENLABS_API_KEY env var.
func NewFactory(config map[string]any) stt.Transcriber {
	apiKey := ""
	if v, ok := config["api_key"].(string); ok && v != "" {
		apiKey = v
	}
	if apiKey == "" {
		apiKey = os.Getenv("ELEVENLABS_API_KEY")
	}

	model := "scribe_v1"
	if v, ok := config["model"].(string); ok && v != "" {
		model = v
	}

	return &Transcriber{
		apiKey:  apiKey,
		model:   model,
		baseURL: defaultBaseURL,
	}
}

func (t *Transcriber) Name() string {
	return "elevenlabs"
}

func (t *Transcriber) Transcribe(audioPath string) (string, error) {
	if t.apiKey == "" {
		return "", fmt.Errorf("ElevenLabs API key not configured.\n\nSet audio.transcribe.elevenlabs.api_key in leetgo.yaml\nor set the ELEVENLABS_API_KEY environment variable.")
	}

	// Build multipart form request
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	file, err := os.Open(audioPath)
	if err != nil {
		return "", fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("failed to write audio data: %w", err)
	}

	_ = writer.WriteField("model_id", t.model)

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Send request
	url := t.baseURL + "/v1/speech-to-text"
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("xi-api-key", t.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ElevenLabs API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ElevenLabs API error: %d %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse ElevenLabs response: %w", err)
	}

	return result.Text, nil
}
