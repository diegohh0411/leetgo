# Leetgo Transcribe + Analyze Feature Design

**Date**: 2026-04-04
**Branch**: `feat/audio-recording` (extends existing recording feature)
**Depends on**: `leetgo record` (already implemented)

## Overview

Two new commands — `leetgo transcribe` and `leetgo analyze` — that complete the
record → transcribe → analyze workflow. After recording a voice note during
problem-solving, the user can transcribe it to text (via ElevenLabs API) and
then get AI-powered analysis (via Claude CLI). Both steps are also offered as
post-recording prompts for a smooth inline experience.

Provider architecture is pluggable: ship with ElevenLabs + Claude CLI defaults,
but interfaces allow adding Whisper, OpenRouter, or any other provider without
modifying existing code.

## Commands

### `leetgo transcribe <qid> [--force]`

Finds all `attempt-*.mp3` recordings in the problem directory that don't yet
have a matching `attempt-*.md` transcript, and transcribes them.

- `--force` / `-f`: Re-transcribe all audio files, overwriting existing
  transcripts
- Default provider: `elevenlabs` (ElevenLabs API)
- Output: `attempt-N.md` for each `attempt-N.mp3`

Behavior:
1. Parse QID, resolve problem directory
2. Scan for `attempt-*.mp3` files
3. Filter to those without matching `.md` (or all with `--force`)
4. For each: run provider, write transcript
5. Summary: "Transcribed 2 files" or "All transcripts up to date"

### `leetgo analyze <qid> [--force]`

Gathers problem description, solution code, and all transcripts, sends to LLM
for structured analysis.

- `--force` / `-f`: Overwrite existing `analysis.md`
- Default provider: `claude` (Claude Code CLI)
- Output: `analysis.md` in the problem directory

Behavior:
1. Parse QID, resolve problem directory
2. Read problem description from leetgo's cached question data
3. Read latest solution file
4. Read all `attempt-*.md` transcripts
5. Build prompt with structured template
6. Run provider, write `analysis.md`

### Post-Recording Prompts

After `leetgo record` ends successfully, the TUI offers:

```
✓ Saved attempt-1.mp3 (02:34)
Transcribe now? [Y/n]:
```

If yes, runs transcription with a spinner, then:

```
✓ Transcribed → attempt-1.md
Analyze? [Y/n]:
```

If yes, runs analysis and shows result. User can decline at either step and
run the commands separately later.

## Provider Architecture

### Package Structure

```
stt_providers/
  stt.go                   # Transcriber interface + registry
  stt_test.go              # Registry tests
  elevenlabs/
    elevenlabs.go          # ElevenLabs API transcriber implementation
    elevenlabs_test.go
  whisper/                 # future
analysis_providers/
  analysis.go              # Analyzer interface + registry
  analysis_test.go         # Registry tests
  claude/
    claude.go              # Claude CLI analyzer implementation
    claude_test.go
  openrouter/              # future
```

Adding a new provider = adding a sub-package under the appropriate top-level
directory + calling `RegisterTranscriber` or `RegisterAnalyzer` in an `init()`
function.

### Interfaces

```go
// Transcriber converts an audio file to text.
type Transcriber interface {
    Transcribe(audioPath string) (string, error)
    Name() string // e.g. "whisper", "elevenlabs"
}

// Analyzer produces structured analysis from problem context.
type Analyzer interface {
    Analyze(ctx AnalysisContext) (string, error)
    Name() string // e.g. "claude", "openrouter"
}

type AnalysisContext struct {
    Question    string   // problem description (HTML or plain text)
    Solution    string   // latest solution source code
    Transcripts []string // contents of all attempt-N.md files
}
```

### Registry

```go
func RegisterTranscriber(name string, factory func(config map[string]any) Transcriber)
func RegisterAnalyzer(name string, factory func(config map[string]any) Analyzer)
func GetTranscriber(name string, config map[string]any) Transcriber
func GetAnalyzer(name string, config map[string]any) Analyzer
```

### Default Providers

**ElevenLabs (transcription)**:
- Calls the ElevenLabs Speech-to-Text API via HTTP
- Endpoint: `POST https://api.elevenlabs.io/v1/speech-to-text`
- Sends the audio file as multipart form data
- Returns the transcribed text
- API key read from `leetgo.yaml` under `audio.transcribe.elevenlabs.api_key`
- Falls back to `ELEVENLABS_API_KEY` environment variable if not in config
- Pre-flight check: validate API key is present (config or env)
- Config: `api_key` (required), `model` (default: `scribe_v1`)

**Claude (analysis)**:
- Shells out to `claude` CLI with `-p` flag for prompt
- Command: `claude -p "<prompt>" --model <model> --output-format text`
- Pre-flight check: `claude --version` to verify installation
- Config: `model` (default: `sonnet`)

### Analysis Prompt Template

```
Analyze this Leetcode problem solution based on my voice notes. Keep it brief - 2-3 paragraphs max.

PROBLEM:
{question}

MY SOLUTION (latest attempt):
{solution}

MY VOICE NOTES:
{transcripts}

Provide:
1. Brief overview of how the problem went
2. What I did well
3. What I struggled with / areas to improve
4. Improvement guide: if the solution was unsolved, suboptimal, or inefficient,
   provide a concrete guide on how to solve or optimize it. Include the key
   algorithm/data structure to use, time/space complexity, and a brief pseudocode
   outline of the improved approach. If the solution is already optimal, skip this section.

Focus on identifying strengths, weaknesses, and actionable feedback for future practice.
```

## Configuration

New `audio` section in `leetgo.yaml`:

```yaml
audio:
  transcribe:
    provider: elevenlabs
    elevenlabs:
      api_key: your-api-key-here
      model: scribe_v1
  analyze:
    provider: claude
    claude:
      model: sonnet
```

All fields optional — defaults are used when absent. Unknown provider names
produce a clear error listing available providers.

## File Structure (New Files)

```
cmd/
  transcribe.go            # cobra command for transcribe
  transcribe_test.go       # tests for attempt scanning, force logic
  analyze.go               # cobra command for analyze
  analyze_test.go          # tests for context gathering, prompt building
stt_providers/
  stt.go                   # Transcriber interface + registry
  stt_test.go              # registry tests
  elevenlabs/
    elevenlabs.go          # ElevenLabs API transcriber
    elevenlabs_test.go     # API request building tests, response parsing tests
analysis_providers/
  analysis.go              # Analyzer interface + registry
  analysis_test.go         # registry tests
  claude/
    claude.go              # claude CLI analyzer
    claude_test.go         # preflight check tests, prompt building tests
```

Modified files:
- `cmd/recorder_tui.go` — post-recording prompts
- `cmd/root.go` — register new commands
- `config/` — add audio config section

## Error Handling

| Scenario | Behavior |
|----------|----------|
| `whisper` not installed | Error with install instructions: `pip install openai-whisper` |
| ElevenLabs API key missing | Error: "Set audio.transcribe.elevenlabs.api_key in leetgo.yaml or ELEVENLABS_API_KEY env var" |
| ElevenLabs API error (401, etc.) | Error: "ElevenLabs API error: <status> <message>" |
| `claude` not installed | Error with install instructions: download from claude.ai |
| No audio files found | "No recordings found for problem <qid>" |
| No transcripts (for analyze) | "No transcripts found. Run `leetgo transcribe <qid>` first." |
| Existing analysis.md (no --force) | "Analysis already exists. Use --force to overwrite." |
| Provider fails mid-run | Show error, don't delete existing files |
| Whisper produces empty transcript | Warning: "Transcript for attempt-N appears empty" |
| ElevenLabs returns empty transcript | Warning: "Transcript for attempt-N appears empty" |

## Testing Strategy

- **Unit tests**: Provider registry, attempt scanning, prompt building, arg
  construction for each provider
- **Integration tests**: Transcriber/Analyzer interfaces with mock implementations
- **ElevenLabs tests**: HTTP request construction, response parsing, error
  handling — use a mock HTTP server (`httptest.NewServer`) to avoid real API
  calls
- **No live API tests**: Claude provider tested with mocked `exec.Command` calls

## Scope

**In scope (this PR)**:
- `transcribe` command with ElevenLabs provider
- `analyze` command with Claude provider
- Provider interfaces and registry
- Post-recording prompts
- Config support

**Out of scope (future)**:
- Whisper local transcription provider
- OpenRouter or other LLM providers
- Streaming/progress display during transcription
- Audio playback command
- Multi-language transcript support
