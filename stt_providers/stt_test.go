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
