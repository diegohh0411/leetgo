# Leetgo Transcribe + Analyze Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `leetgo transcribe` and `leetgo analyze` commands with pluggable provider architecture, shipping ElevenLabs STT and Claude CLI analysis as defaults.

**Architecture:** Two top-level provider packages (`stt_providers/`, `analysis_providers/`) each define an interface and a registry. Command files in `cmd/` resolve the problem directory, gather files, and call providers. Post-recording prompts chain transcribe → analyze after `leetgo record`.

**Tech Stack:** Go, Cobra, bubbletea, ElevenLabs HTTP API, Claude CLI subprocess, `httptest` for mocked API tests.

**Spec:** `docs/superpowers/specs/2026-04-04-leetgo-transcribe-analyze-design.md`

**Branch:** `feat/audio-recording` at `/home/pi/pnyc/opensource/leetgo`

---

## File Structure

| File | Responsibility |
|------|---------------|
| `stt_providers/stt.go` | `Transcriber` interface + registry |
| `stt_providers/stt_test.go` | Registry unit tests |
| `stt_providers/elevenlabs/elevenlabs.go` | ElevenLabs HTTP API transcriber |
| `stt_providers/elevenlabs/elevenlabs_test.go` | Request building, response parsing, error handling (mocked HTTP) |
| `analysis_providers/analysis.go` | `Analyzer` interface, `AnalysisContext` struct + registry |
| `analysis_providers/analysis_test.go` | Registry unit tests |
| `analysis_providers/claude/claude.go` | Claude CLI subprocess analyzer |
| `analysis_providers/claude/claude_test.go` | Preflight check, prompt building, arg construction tests |
| `cmd/transcribe.go` | Cobra command: QID parsing, attempt scanning, provider dispatch |
| `cmd/transcribe_test.go` | Attempt scanning logic, force flag tests |
| `cmd/analyze.go` | Cobra command: context gathering, prompt building, provider dispatch |
| `cmd/analyze_test.go` | Context gathering, prompt formatting tests |
| `cmd/recorder_tui.go` | Modified: post-recording prompts for transcribe/analyze |
| `cmd/root.go` | Modified: register `transcribeCmd` and `analyzeCmd` |
| `config/config.go` | Modified: add `Audio` field to `Config` struct |

---

### Task 1: STT Provider Interface + Registry

**Files:**
- Create: `stt_providers/stt.go`
- Create: `stt_providers/stt_test.go`

- [ ] **Step 1: Write the failing test**

```go
// stt_providers/stt_test.go
package stt_providers

import (
	"testing"
)

type mockTranscriber struct {
	name string
}

func (m *mockTranscriber) Transcribe(audioPath string) (string, error) {
	return "transcribed: " + audioPath, nil
}

func (m *mockTranscriber) Name() string {
	return m.name
}

func TestRegisterAndGetTranscriber(t *testing.T) {
	// Reset registry for test isolation
	registries = map[string]func(config map[string]any) Transcriber{}

	Register("test-stt", func(config map[string]any) Transcriber {
		return &mockTranscriber{name: "test-stt"}
	})

	got, err := Get("test-stt", nil)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Name() != "test-stt" {
		t.Errorf("Name() = %q, want %q", got.Name(), "test-stt")
	}
}

func TestGetUnknownTranscriber(t *testing.T) {
	registries = map[string]func(config map[string]any) Transcriber{}

	_, err := Get("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/pi/pnyc/opensource/leetgo && go test ./stt_providers/ -v -run TestRegisterAndGetTranscriber`
Expected: FAIL — package doesn't exist yet

- [ ] **Step 3: Write minimal implementation**

```go
// stt_providers/stt.go
package stt_providers

import (
	"fmt"
	"sort"
	"strings"
)

// Transcriber converts an audio file to text.
type Transcriber interface {
	Transcribe(audioPath string) (string, error)
	Name() string
}

// registries maps provider names to factory functions.
var registries = map[string]func(config map[string]any) Transcriber{}

// Register adds a transcriber factory under the given name.
func Register(name string, factory func(config map[string]any) Transcriber) {
	registries[name] = factory
}

// Get creates a transcriber by name, passing config to the factory.
func Get(name string, config map[string]any) (Transcriber, error) {
	factory, ok := registries[name]
	if !ok {
		available := make([]string, 0, len(registries))
		for k := range registries {
			available = append(available, k)
		}
		sort.Strings(available)
		return nil, fmt.Errorf("unknown STT provider %q (available: %s)", name, strings.Join(available, ", "))
	}
	return factory(config), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/pi/pnyc/opensource/leetgo && go test ./stt_providers/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /home/pi/pnyc/opensource/leetgo
git add stt_providers/stt.go stt_providers/stt_test.go
git commit -m "feat(transcribe): add STT provider interface and registry"
```

---

### Task 2: Analysis Provider Interface + Registry

**Files:**
- Create: `analysis_providers/analysis.go`
- Create: `analysis_providers/analysis_test.go`

- [ ] **Step 1: Write the failing test**

```go
// analysis_providers/analysis_test.go
package analysis_providers

import (
	"testing"
)

type mockAnalyzer struct {
	name string
}

func (m *mockAnalyzer) Analyze(ctx AnalysisContext) (string, error) {
	return "analyzed", nil
}

func (m *mockAnalyzer) Name() string {
	return m.name
}

func TestRegisterAndGetAnalyzer(t *testing.T) {
	registries = map[string]func(config map[string]any) Analyzer{}

	Register("test-analyzer", func(config map[string]any) Analyzer {
		return &mockAnalyzer{name: "test-analyzer"}
	})

	got, err := Get("test-analyzer", nil)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Name() != "test-analyzer" {
		t.Errorf("Name() = %q, want %q", got.Name(), "test-analyzer")
	}
}

func TestGetUnknownAnalyzer(t *testing.T) {
	registries = map[string]func(config map[string]any) Analyzer{}

	_, err := Get("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/pi/pnyc/opensource/leetgo && go test ./analysis_providers/ -v -run TestRegisterAndGetAnalyzer`
Expected: FAIL — package doesn't exist yet

- [ ] **Step 3: Write minimal implementation**

```go
// analysis_providers/analysis.go
package analysis_providers

import (
	"fmt"
	"sort"
	"strings"
)

// AnalysisContext holds all input data for an analysis request.
type AnalysisContext struct {
	Question    string   // problem description (markdown)
	Solution    string   // latest solution source code
	Transcripts []string // contents of all attempt-N.md files
}

// Analyzer produces structured analysis from problem context.
type Analyzer interface {
	Analyze(ctx AnalysisContext) (string, error)
	Name() string
}

// registries maps provider names to factory functions.
var registries = map[string]func(config map[string]any) Analyzer{}

// Register adds an analyzer factory under the given name.
func Register(name string, factory func(config map[string]any) Analyzer) {
	registries[name] = factory
}

// Get creates an analyzer by name, passing config to the factory.
func Get(name string, config map[string]any) (Analyzer, error) {
	factory, ok := registries[name]
	if !ok {
		available := make([]string, 0, len(registries))
		for k := range registries {
			available = append(available, k)
		}
		sort.Strings(available)
		return nil, fmt.Errorf("unknown analysis provider %q (available: %s)", name, strings.Join(available, ", "))
	}
	return factory(config), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/pi/pnyc/opensource/leetgo && go test ./analysis_providers/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /home/pi/pnyc/opensource/leetgo
git add analysis_providers/analysis.go analysis_providers/analysis_test.go
git commit -m "feat(analyze): add analysis provider interface and registry"
```

---

### Task 3: ElevenLabs STT Provider

**Files:**
- Create: `stt_providers/elevenlabs/elevenlabs.go`
- Create: `stt_providers/elevenlabs/elevenlabs_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// stt_providers/elevenlabs/elevenlabs_test.go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/pi/pnyc/opensource/leetgo && go test ./stt_providers/elevenlabs/ -v`
Expected: FAIL — package doesn't exist yet

- [ ] **Step 3: Write the implementation**

```go
// stt_providers/elevenlabs/elevenlabs.go
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

	body, _ := io.ReadAll(resp.Body)

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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/pi/pnyc/opensource/leetgo && go test ./stt_providers/elevenlabs/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /home/pi/pnyc/opensource/leetgo
git add stt_providers/elevenlabs/elevenlabs.go stt_providers/elevenlabs/elevenlabs_test.go
git commit -m "feat(transcribe): add ElevenLabs STT provider"
```

---

### Task 4: Claude CLI Analysis Provider

**Files:**
- Create: `analysis_providers/claude/claude.go`
- Create: `analysis_providers/claude/claude_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// analysis_providers/claude/claude_test.go
package claude

import (
	"strings"
	"testing"

	analysis "github.com/j178/leetgo/analysis_providers"
)

func TestBuildPrompt(t *testing.T) {
	ctx := analysis.AnalysisContext{
		Question:    "Two Sum problem description",
		Solution:    "func twoSum(nums []int, target int) []int {",
		Transcripts: []string{"I started by thinking about brute force", "Then optimized with a hashmap"},
	}

	prompt := buildPrompt(ctx)

	if !strings.Contains(prompt, "Two Sum problem description") {
		t.Error("prompt should contain the question")
	}
	if !strings.Contains(prompt, "func twoSum") {
		t.Error("prompt should contain the solution")
	}
	if !strings.Contains(prompt, "I started by thinking about brute force") {
		t.Error("prompt should contain transcripts")
	}
	if !strings.Contains(prompt, "Then optimized with a hashmap") {
		t.Error("prompt should contain all transcripts")
	}
	if !strings.Contains(prompt, "What I did well") {
		t.Error("prompt should contain analysis structure request")
	}
}

func TestBuildArgs(t *testing.T) {
	args := buildArgs("test prompt", "sonnet")

	found := false
	for i, a := range args {
		if a == "--model" && i+1 < len(args) && args[i+1] == "sonnet" {
			found = true
		}
	}
	if !found {
		t.Error("args should contain --model sonnet")
	}

	hasPFlag := false
	for _, a := range args {
		if a == "-p" {
			hasPFlag = true
		}
	}
	if !hasPFlag {
		t.Error("args should contain -p flag")
	}
}

func TestNewFactory(t *testing.T) {
	config := map[string]any{
		"model": "sonnet",
	}
	a := NewFactory(config)
	if a.Name() != "claude" {
		t.Errorf("Name() = %q, want %q", a.Name(), "claude")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/pi/pnyc/opensource/leetgo && go test ./analysis_providers/claude/ -v`
Expected: FAIL — package doesn't exist yet

- [ ] **Step 3: Write the implementation**

```go
// analysis_providers/claude/claude.go
package claude

import (
	"fmt"
	"os/exec"
	"strings"

	analysis "github.com/j178/leetgo/analysis_providers"
)

func init() {
	analysis.Register("claude", NewFactory)
}

// Analyzer implements analysis.Analyzer using the Claude CLI.
type Analyzer struct {
	model string
}

// NewFactory creates a Claude analyzer from config.
func NewFactory(config map[string]any) analysis.Analyzer {
	model := "sonnet"
	if v, ok := config["model"].(string); ok && v != "" {
		model = v
	}
	return &Analyzer{model: model}
}

func (a *Analyzer) Name() string {
	return "claude"
}

// CheckInstalled verifies that the claude CLI is available.
func CheckInstalled() error {
	_, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI is not installed or not on PATH.\n\nInstall it from: https://docs.anthropic.com/en/docs/claude-code")
	}
	return nil
}

// Analyze sends the analysis context to Claude CLI and returns the response.
func (a *Analyzer) Analyze(ctx analysis.AnalysisContext) (string, error) {
	if err := CheckInstalled(); err != nil {
		return "", err
	}

	prompt := buildPrompt(ctx)
	args := buildArgs(prompt, a.model)

	cmd := exec.Command("claude", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude CLI failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("claude CLI error: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// buildPrompt constructs the analysis prompt from context.
func buildPrompt(ctx analysis.AnalysisContext) string {
	transcripts := strings.Join(ctx.Transcripts, "\n\n---\n\n")

	return fmt.Sprintf(`Analyze this Leetcode problem solution based on my voice notes. Keep it brief - 2-3 paragraphs max.

PROBLEM:
%s

MY SOLUTION (latest attempt):
%s

MY VOICE NOTES:
%s

Provide:
1. Brief overview of how the problem went
2. What I did well
3. What I struggled with / areas to improve
4. Improvement guide: if the solution was unsolved, suboptimal, or inefficient, provide a concrete guide on how to solve or optimize it. Include the key algorithm/data structure to use, time/space complexity, and a brief pseudocode outline of the improved approach. If the solution is already optimal, skip this section.

Focus on identifying strengths, weaknesses, and actionable feedback for future practice.`, ctx.Question, ctx.Solution, transcripts)
}

// buildArgs constructs the claude CLI argument list.
func buildArgs(prompt, model string) []string {
	return []string{"--model", model, "-p", prompt, "--output-format", "text"}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/pi/pnyc/opensource/leetgo && go test ./analysis_providers/claude/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /home/pi/pnyc/opensource/leetgo
git add analysis_providers/claude/claude.go analysis_providers/claude/claude_test.go
git commit -m "feat(analyze): add Claude CLI analysis provider"
```

---

### Task 5: Audio Config Section

**Files:**
- Modify: `config/config.go` — add `Audio` field to `Config` struct (around line 38–47)

- [ ] **Step 1: Add config types**

Add these types and fields to `config/config.go`. Insert the new types after the `Editor` struct (around line 59). Add `Audio AudioConfig` field to the `Config` struct.

```go
// Add to Config struct (after Editor field, around line 46):
	Audio       AudioConfig    `yaml:"audio" mapstructure:"audio"`

// Add these new types after the Editor struct (around line 59):

type AudioConfig struct {
	Transcribe TranscribeConfig `yaml:"transcribe" mapstructure:"transcribe"`
	Analyze    AnalyzeConfig    `yaml:"analyze" mapstructure:"analyze"`
}

type TranscribeConfig struct {
	Provider   string                 `yaml:"provider" mapstructure:"provider"`
	ElevenLabs map[string]any         `yaml:"elevenlabs" mapstructure:"elevenlabs"`
}

type AnalyzeConfig struct {
	Provider string            `yaml:"provider" mapstructure:"provider"`
	Claude   map[string]any    `yaml:"claude" mapstructure:"claude"`
}
```

- [ ] **Step 2: Verify build compiles**

Run: `cd /home/pi/pnyc/opensource/leetgo && go build ./...`
Expected: compiles with no errors

- [ ] **Step 3: Commit**

```bash
cd /home/pi/pnyc/opensource/leetgo
git add config/config.go
git commit -m "feat(config): add audio config section for transcribe/analyze providers"
```

---

### Task 6: Transcribe Command

**Files:**
- Create: `cmd/transcribe.go`
- Create: `cmd/transcribe_test.go`
- Modify: `cmd/root.go` — add `transcribeCmd` to commands slice

- [ ] **Step 1: Write the failing tests**

```go
// cmd/transcribe_test.go
package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindUntranscribed(t *testing.T) {
	tests := []struct {
		name      string
		create    []string // files to create in temp dir
		wantCount int      // expected number of untranscribed files
		wantFirst string   // expected base name of first result (empty if none)
	}{
		{
			name:      "empty directory",
			create:    nil,
			wantCount: 0,
		},
		{
			name:      "one mp3 no transcript",
			create:    []string{"attempt-1.mp3"},
			wantCount: 1,
			wantFirst: "attempt-1.mp3",
		},
		{
			name:      "one mp3 with transcript",
			create:    []string{"attempt-1.mp3", "attempt-1.md"},
			wantCount: 0,
		},
		{
			name:      "three mp3s two transcripts",
			create:    []string{"attempt-1.mp3", "attempt-1.md", "attempt-2.mp3", "attempt-2.md", "attempt-3.mp3"},
			wantCount: 1,
			wantFirst: "attempt-3.mp3",
		},
		{
			name:      "non-attempt files ignored",
			create:    []string{"notes.txt", "solution.cpp", "attempt-1.mp3", "attempt-1.md"},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.create {
				os.WriteFile(filepath.Join(dir, f), []byte{}, 0o644)
			}

			got := findUntranscribed(dir)
			if len(got) != tt.wantCount {
				t.Fatalf("findUntranscribed() returned %d files, want %d", len(got), tt.wantCount)
			}
			if tt.wantFirst != "" && len(got) > 0 {
				if got[0] != tt.wantFirst {
					t.Errorf("first result = %q, want %q", got[0], tt.wantFirst)
				}
			}
		})
	}
}

func TestFindAllAudio(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "attempt-1.mp3"), []byte{}, 0o644)
	os.WriteFile(filepath.Join(dir, "attempt-2.mp3"), []byte{}, 0o644)
	os.WriteFile(filepath.Join(dir, "attempt-1.md"), []byte{}, 0o644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte{}, 0o644)

	got := findAllAudio(dir)
	if len(got) != 2 {
		t.Fatalf("findAllAudio() returned %d files, want 2", len(got))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/pi/pnyc/opensource/leetgo && go test ./cmd/ -v -run TestFindUntranscribed`
Expected: FAIL — functions not defined yet

- [ ] **Step 3: Write the implementation**

```go
// cmd/transcribe.go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/j178/leetgo/config"
	"github.com/j178/leetgo/lang"
	"github.com/j178/leetgo/leetcode"

	_ "github.com/j178/leetgo/stt_providers/elevenlabs"
	stt "github.com/j178/leetgo/stt_providers"
)

var transcribeForce bool

var transcribeCmd = &cobra.Command{
	Use:   "transcribe qid",
	Short: "Transcribe voice note recordings for a problem",
	Args:  cobra.ExactArgs(1),
	RunE:  runTranscribe,
}

func init() {
	transcribeCmd.Flags().BoolVarP(&transcribeForce, "force", "f", false, "re-transcribe all recordings")
}

// audioFileRe matches attempt-N.mp3 filenames.
var audioFileRe = regexp.MustCompile(`^attempt-(\d+)\.mp3$`)

// findUntranscribed returns mp3 filenames that don't have a matching .md transcript.
func findUntranscribed(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	// Build set of transcribed attempt numbers
	transcribed := map[int]bool{}
	for _, e := range entries {
		if m := audioFileRe.FindStringSubmatch(e.Name()); m != nil {
			mdName := fmt.Sprintf("attempt-%s.md", m[1])
			if _, err := os.Stat(filepath.Join(dir, mdName)); err == nil {
				n, _ := strconv.Atoi(m[1])
				transcribed[n] = true
			}
		}
	}

	var result []string
	for _, e := range entries {
		m := audioFileRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		if !transcribed[n] {
			result = append(result, e.Name())
		}
	}

	sort.Slice(result, func(i, j int) bool {
		ni, _ := strconv.Atoi(audioFileRe.FindStringSubmatch(result[i])[1])
		nj, _ := strconv.Atoi(audioFileRe.FindStringSubmatch(result[j])[1])
		return ni < nj
	})

	return result
}

// findAllAudio returns all attempt-N.mp3 filenames in dir.
func findAllAudio(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var result []string
	for _, e := range entries {
		if audioFileRe.MatchString(e.Name()) {
			result = append(result, e.Name())
		}
	}

	sort.Slice(result, func(i, j int) bool {
		ni, _ := strconv.Atoi(audioFileRe.FindStringSubmatch(result[i])[1])
		nj, _ := strconv.Atoi(audioFileRe.FindStringSubmatch(result[j])[1])
		return ni < nj
	})

	return result
}

func runTranscribe(cmd *cobra.Command, args []string) error {
	// Parse QID and resolve problem directory.
	c := leetcode.NewClient(leetcode.ReadCredentials())
	qs, err := leetcode.ParseQID(args[0], c)
	if err != nil {
		return err
	}
	if len(qs) > 1 {
		return fmt.Errorf("multiple questions found")
	}

	result, err := lang.GeneratePathsOnly(qs[0])
	if err != nil {
		return err
	}
	outDir := result.TargetDir()

	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		return fmt.Errorf("problem directory %q does not exist — run `leetgo pick` first", outDir)
	}

	// Find audio files to transcribe.
	var audioFiles []string
	if transcribeForce {
		audioFiles = findAllAudio(outDir)
	} else {
		audioFiles = findUntranscribed(outDir)
	}

	if len(audioFiles) == 0 {
		fmt.Println("All transcripts up to date.")
		return nil
	}

	// Get the transcriber from config.
	cfg := config.Get()
	providerName := cfg.Audio.Transcribe.Provider
	if providerName == "" {
		providerName = "elevenlabs"
	}

	var providerConfig map[string]any
	if providerName == "elevenlabs" {
		providerConfig = cfg.Audio.Transcribe.ElevenLabs
	}

	provider, err := stt.Get(providerName, providerConfig)
	if err != nil {
		return err
	}

	// Transcribe each file.
	transcribed := 0
	for _, audioFile := range audioFiles {
		audioPath := filepath.Join(outDir, audioFile)
		fmt.Printf("Transcribing %s...\n", audioFile)

		text, err := provider.Transcribe(audioPath)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}

		if text == "" {
			fmt.Printf("  Warning: transcript for %s appears empty\n", audioFile)
		}

		// Derive output filename: attempt-N.mp3 → attempt-N.md
		m := audioFileRe.FindStringSubmatch(audioFile)
		mdName := fmt.Sprintf("attempt-%s.md", m[1])
		mdPath := filepath.Join(outDir, mdName)

		if err := os.WriteFile(mdPath, []byte(text), 0o644); err != nil {
			fmt.Printf("  Error writing %s: %v\n", mdName, err)
			continue
		}

		fmt.Printf("  ✓ %s\n", mdName)
		transcribed++
	}

	fmt.Printf("Transcribed %d file(s).\n", transcribed)
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/pi/pnyc/opensource/leetgo && go test ./cmd/ -v -run "TestFindUntranscribed|TestFindAllAudio"`
Expected: PASS

- [ ] **Step 5: Register the command in root.go**

Add `transcribeCmd` to the `commands` slice in `cmd/root.go` (after `recordCmd`):

```go
recordCmd,
transcribeCmd,
```

- [ ] **Step 6: Verify build compiles**

Run: `cd /home/pi/pnyc/opensource/leetgo && go build ./...`
Expected: compiles with no errors

- [ ] **Step 7: Commit**

```bash
cd /home/pi/pnyc/opensource/leetgo
git add cmd/transcribe.go cmd/transcribe_test.go cmd/root.go
git commit -m "feat(transcribe): add leetgo transcribe command with attempt scanning"
```

---

### Task 7: Analyze Command

**Files:**
- Create: `cmd/analyze.go`
- Create: `cmd/analyze_test.go`
- Modify: `cmd/root.go` — add `analyzeCmd` to commands slice

- [ ] **Step 1: Write the failing tests**

```go
// cmd/analyze_test.go
package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadTranscripts(t *testing.T) {
	dir := t.TempDir()

	// Create some transcript files
	os.WriteFile(filepath.Join(dir, "attempt-1.md"), []byte("First attempt notes"), 0o644)
	os.WriteFile(filepath.Join(dir, "attempt-2.md"), []byte("Second attempt notes"), 0o644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("Not a transcript"), 0o644)

	transcripts := readTranscripts(dir)
	if len(transcripts) != 2 {
		t.Fatalf("readTranscripts() returned %d, want 2", len(transcripts))
	}
	if transcripts[0] != "First attempt notes" {
		t.Errorf("transcripts[0] = %q, want %q", transcripts[0], "First attempt notes")
	}
	if transcripts[1] != "Second attempt notes" {
		t.Errorf("transcripts[1] = %q, want %q", transcripts[1], "Second attempt notes")
	}
}

func TestReadTranscriptsEmpty(t *testing.T) {
	dir := t.TempDir()
	transcripts := readTranscripts(dir)
	if len(transcripts) != 0 {
		t.Fatalf("expected 0 transcripts in empty dir, got %d", len(transcripts))
	}
}

func TestReadLatestSolution(t *testing.T) {
	dir := t.TempDir()

	// No solution files
	_, err := readLatestSolution(dir)
	if err == nil {
		t.Fatal("expected error when no solution file exists")
	}

	// Create a solution file
	os.WriteFile(filepath.Join(dir, "solution.cpp"), []byte("int main() {}"), 0o644)
	content, err := readLatestSolution(dir)
	if err != nil {
		t.Fatalf("readLatestSolution() error = %v", err)
	}
	if content != "int main() {}" {
		t.Errorf("got %q, want %q", content, "int main() {}")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/pi/pnyc/opensource/leetgo && go test ./cmd/ -v -run "TestReadTranscripts|TestReadLatestSolution"`
Expected: FAIL — functions not defined yet

- [ ] **Step 3: Write the implementation**

```go
// cmd/analyze.go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/j178/leetgo/config"
	"github.com/j178/leetgo/lang"
	"github.com/j178/leetgo/leetcode"

	_ "github.com/j178/leetgo/analysis_providers/claude"
	analysis "github.com/j178/leetgo/analysis_providers"
)

var analyzeForce bool

var analyzeCmd = &cobra.Command{
	Use:   "analyze qid",
	Short: "Analyze problem solution using AI",
	Args:  cobra.ExactArgs(1),
	RunE:  runAnalyze,
}

func init() {
	analyzeCmd.Flags().BoolVarP(&analyzeForce, "force", "f", false, "overwrite existing analysis")
}

// transcriptRe matches attempt-N.md filenames.
var transcriptRe = regexp.MustCompile(`^attempt-(\d+)\.md$`)

// readTranscripts reads all attempt-N.md files from dir, sorted by attempt number.
func readTranscripts(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	type numbered struct {
		n      int
		path   string
	}
	var files []numbered
	for _, e := range entries {
		m := transcriptRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		files = append(files, numbered{n: n, path: filepath.Join(dir, e.Name())})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].n < files[j].n
	})

	var transcripts []string
	for _, f := range files {
		data, err := os.ReadFile(f.path)
		if err != nil {
			continue
		}
		transcripts = append(transcripts, string(data))
	}
	return transcripts
}

// readLatestSolution reads the most recently modified code file from dir.
// Looks for common source extensions: .cpp, .py, .go, .java, .rs, .js, .ts
func readLatestSolution(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var best os.FileInfo
	bestPath := ""
	for _, e := range entries {
		ext := filepath.Ext(e.Name())
		switch ext {
		case ".cpp", ".py", ".go", ".java", ".rs", ".js", ".ts", ".c", ".cs":
			info, err := e.Info()
			if err != nil {
				continue
			}
			if best == nil || info.ModTime().After(best.ModTime()) {
				best = info
				bestPath = filepath.Join(dir, e.Name())
			}
		}
	}

	if bestPath == "" {
		return "", fmt.Errorf("no solution file found in %s", dir)
	}

	data, err := os.ReadFile(bestPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	// Parse QID and resolve problem directory.
	c := leetcode.NewClient(leetcode.ReadCredentials())
	qs, err := leetcode.ParseQID(args[0], c)
	if err != nil {
		return err
	}
	if len(qs) > 1 {
		return fmt.Errorf("multiple questions found")
	}

	result, err := lang.GeneratePathsOnly(qs[0])
	if err != nil {
		return err
	}
	outDir := result.TargetDir()

	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		return fmt.Errorf("problem directory %q does not exist — run `leetgo pick` first", outDir)
	}

	// Check for existing analysis.
	analysisPath := filepath.Join(outDir, "analysis.md")
	if !analyzeForce {
		if _, err := os.Stat(analysisPath); err == nil {
			return fmt.Errorf("analysis already exists. Use --force to overwrite.")
		}
	}

	// Gather context.
	question := qs[0].GetFormattedContent()
	solution, err := readLatestSolution(outDir)
	if err != nil {
		return fmt.Errorf("failed to read solution: %w", err)
	}

	transcripts := readTranscripts(outDir)
	if len(transcripts) == 0 {
		return fmt.Errorf("no transcripts found. Run `leetgo transcribe %s` first.", args[0])
	}

	// Get the analyzer from config.
	cfg := config.Get()
	providerName := cfg.Audio.Analyze.Provider
	if providerName == "" {
		providerName = "claude"
	}

	var providerConfig map[string]any
	if providerName == "claude" {
		providerConfig = cfg.Audio.Analyze.Claude
	}

	provider, err := analysis.Get(providerName, providerConfig)
	if err != nil {
		return err
	}

	fmt.Println("Analyzing...")

	ctx := analysis.AnalysisContext{
		Question:    question,
		Solution:    solution,
		Transcripts: transcripts,
	}

	text, err := provider.Analyze(ctx)
	if err != nil {
		return err
	}

	if err := os.WriteFile(analysisPath, []byte(text), 0o644); err != nil {
		return fmt.Errorf("failed to write analysis: %w", err)
	}

	fmt.Printf("✓ Saved %s\n", analysisPath)
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/pi/pnyc/opensource/leetgo && go test ./cmd/ -v -run "TestReadTranscripts|TestReadLatestSolution"`
Expected: PASS

- [ ] **Step 5: Register the command in root.go**

Add `analyzeCmd` to the `commands` slice in `cmd/root.go` (after `transcribeCmd`):

```go
transcribeCmd,
analyzeCmd,
```

- [ ] **Step 6: Verify build compiles**

Run: `cd /home/pi/pnyc/opensource/leetgo && go build ./...`
Expected: compiles with no errors

- [ ] **Step 7: Commit**

```bash
cd /home/pi/pnyc/opensource/leetgo
git add cmd/analyze.go cmd/analyze_test.go cmd/root.go
git commit -m "feat(analyze): add leetgo analyze command with context gathering"
```

---

### Task 8: Post-Recording Prompts in TUI

**Files:**
- Modify: `cmd/recorder_tui.go` — add post-recording transcribe/analyze prompts

- [ ] **Step 1: Modify the recorder model**

The post-recording prompts happen outside the bubbletea TUI (after it exits). Modify `runRecorderTUI` in `cmd/recorder_tui.go` to return the saved filepath so the caller can offer prompts. Then modify `runRecord` in `cmd/record.go` to handle the post-recording flow.

First, change `runRecorderTUI` to return the outputPath along with error:

In `cmd/recorder_tui.go`, update the `runRecorderTUI` function (around line 73):

```go
// runRecorderTUI launches the bubbletea recording interface.
// Returns the output path if recording was saved, empty string if cancelled.
func runRecorderTUI(outDir, filename, outputPath string) (string, error) {
	m := &recorderModel{
		outDir:     outDir,
		filename:   filename,
		outputPath: outputPath,
		canPause:   detectPlatform().canPause,
		width:      80,
	}
	m.bands = make([]float64, numBandsForWidth(m.width))
	p := tea.NewProgram(m)
	_, err := p.Run()
	if err != nil {
		return "", err
	}

	if m.status == statusDone {
		return outputPath, nil
	}
	return "", nil
}
```

Then update `cmd/record.go` — modify `runRecord` to handle post-recording prompts:

```go
func runRecord(cmd *cobra.Command, args []string) error {
	// Check ffmpeg first — fail fast with install instructions.
	if err := checkFFmpeg(); err != nil {
		return err
	}

	// Parse the question ID (supports "219", "contains-duplicate-ii", "today", etc.)
	c := leetcode.NewClient(leetcode.ReadCredentials())
	qs, err := leetcode.ParseQID(args[0], c)
	if err != nil {
		return err
	}
	if len(qs) > 1 {
		return fmt.Errorf("multiple questions found")
	}

	// Resolve the problem output directory.
	result, err := lang.GeneratePathsOnly(qs[0])
	if err != nil {
		return err
	}
	outDir := result.TargetDir()

	// Ensure the output directory exists.
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		return fmt.Errorf("problem directory %q does not exist — run `leetgo pick` first", outDir)
	}

	// Determine the next attempt number.
	attempt := nextAttemptNumber(outDir, recordForce)
	outputPath := fmt.Sprintf("%s/attempt-%d.mp3", outDir, attempt)
	filename := fmt.Sprintf("attempt-%d.mp3", attempt)

	// Launch the recording TUI.
	savedPath, err := runRecorderTUI(outDir, filename, outputPath)
	if err != nil {
		return err
	}

	// If recording was saved, offer post-recording workflow.
	if savedPath != "" {
		fmt.Println() // blank line after TUI
		if promptYesNo("Transcribe now?", true) {
			if err := transcribeFile(savedPath, outDir); err != nil {
				fmt.Printf("Transcription error: %v\n", err)
			} else {
				mdName := fmt.Sprintf("attempt-%d.md", attempt)
				fmt.Printf("✓ Transcribed → %s\n", mdName)

				if promptYesNo("Analyze?", true) {
					if err := runAnalysis(qs[0], outDir); err != nil {
						fmt.Printf("Analysis error: %v\n", err)
					}
				}
			}
		}
	}

	return nil
}
```

Now add the helper functions to `cmd/record.go`:

```go
import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/j178/leetgo/config"
	"github.com/j178/leetgo/lang"
	"github.com/j178/leetgo/leetcode"

	_ "github.com/j178/leetgo/stt_providers/elevenlabs"
	stt "github.com/j178/leetgo/stt_providers"
	_ "github.com/j178/leetgo/analysis_providers/claude"
	analysis "github.com/j178/leetgo/analysis_providers"
)

// promptYesNo asks a yes/no question. defaultYes controls the default.
func promptYesNo(question string, defaultYes bool) bool {
	suffix := " [Y/n]"
	if !defaultYes {
		suffix = " [y/N]"
	}
	fmt.Printf("%s%s ", question, suffix)

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		return defaultYes
	}
	return input == "y" || input == "yes"
}

// transcribeFile transcribes a single audio file using the configured provider.
func transcribeFile(audioPath, outDir string) error {
	cfg := config.Get()
	providerName := cfg.Audio.Transcribe.Provider
	if providerName == "" {
		providerName = "elevenlabs"
	}

	var providerConfig map[string]any
	if providerName == "elevenlabs" {
		providerConfig = cfg.Audio.Transcribe.ElevenLabs
	}

	provider, err := stt.Get(providerName, providerConfig)
	if err != nil {
		return err
	}

	text, err := provider.Transcribe(audioPath)
	if err != nil {
		return err
	}

	// Derive md filename from mp3 filename.
	base := filepath.Base(audioPath)
	m := attemptRe.FindStringSubmatch(base)
	if m == nil {
		return fmt.Errorf("unexpected audio filename: %s", base)
	}
	mdPath := filepath.Join(outDir, fmt.Sprintf("attempt-%s.md", m[1]))

	return os.WriteFile(mdPath, []byte(text), 0o644)
}

// runAnalysis runs the analysis pipeline for a problem.
func runAnalysis(q *leetcode.QuestionData, outDir string) error {
	cfg := config.Get()
	providerName := cfg.Audio.Analyze.Provider
	if providerName == "" {
		providerName = "claude"
	}

	var providerConfig map[string]any
	if providerName == "claude" {
		providerConfig = cfg.Audio.Analyze.Claude
	}

	provider, err := analysis.Get(providerName, providerConfig)
	if err != nil {
		return err
	}

	solution, err := readLatestSolution(outDir)
	if err != nil {
		return err
	}

	transcripts := readTranscripts(outDir)
	if len(transcripts) == 0 {
		return fmt.Errorf("no transcripts found")
	}

	ctx := analysis.AnalysisContext{
		Question:    q.GetFormattedContent(),
		Solution:    solution,
		Transcripts: transcripts,
	}

	text, err := provider.Analyze(ctx)
	if err != nil {
		return err
	}

	analysisPath := filepath.Join(outDir, "analysis.md")
	fmt.Println("Analyzing...")
	return os.WriteFile(analysisPath, []byte(text), 0o644)
}
```

Note: `readLatestSolution` and `readTranscripts` are defined in `cmd/analyze.go` — they're in the same package so they're accessible.

Also need to add `"path/filepath"` to the imports in `cmd/record.go`.

- [ ] **Step 2: Verify build compiles**

Run: `cd /home/pi/pnyc/opensource/leetgo && go build ./...`
Expected: compiles with no errors

- [ ] **Step 3: Run all tests**

Run: `cd /home/pi/pnyc/opensource/leetgo && go test ./cmd/ ./stt_providers/... ./analysis_providers/... -v`
Expected: all tests PASS

- [ ] **Step 4: Commit**

```bash
cd /home/pi/pnyc/opensource/leetgo
git add cmd/record.go cmd/recorder_tui.go
git commit -m "feat(record): add post-recording transcribe/analyze prompts"
```

---

### Task 9: End-to-End Verification

**Files:** None (verification only)

- [ ] **Step 1: Run all tests across all packages**

Run: `cd /home/pi/pnyc/opensource/leetgo && go test ./...`
Expected: all tests PASS (no failures)

- [ ] **Step 2: Verify the binary builds and commands are registered**

Run: `cd /home/pi/pnyc/opensource/leetgo && go build -o leetgo . && ./leetgo help`
Expected: help output shows `transcribe` and `analyze` commands

- [ ] **Step 3: Verify transcribe subcommand help**

Run: `cd /home/pi/pnyc/opensource/leetgo && ./leetgo transcribe --help`
Expected: shows usage with `--force` flag

- [ ] **Step 4: Verify analyze subcommand help**

Run: `cd /home/pi/pnyc/opensource/leetgo && ./leetgo analyze --help`
Expected: shows usage with `--force` flag

- [ ] **Step 5: Commit (only if any fixes were needed)**

```bash
cd /home/pi/pnyc/opensource/leetgo
git add -A
git commit -m "fix: address issues found during end-to-end verification"
```
