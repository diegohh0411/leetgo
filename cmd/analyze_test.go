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
