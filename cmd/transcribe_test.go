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
