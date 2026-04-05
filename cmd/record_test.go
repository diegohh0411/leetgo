package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNextAttemptNumber(t *testing.T) {
	tests := []struct {
		name     string
		existing []string // filenames to create in the temp dir
		force    bool
		want     int
	}{
		{
			name:     "empty directory",
			existing: nil,
			force:    false,
			want:     1,
		},
		{
			name:     "one existing attempt",
			existing: []string{"attempt-1.mp3"},
			force:    false,
			want:     2,
		},
		{
			name:     "three existing attempts",
			existing: []string{"attempt-1.mp3", "attempt-2.mp3", "attempt-3.mp3"},
			force:    false,
			want:     4,
		},
		{
			name:     "gap in numbering",
			existing: []string{"attempt-1.mp3", "attempt-3.mp3"},
			force:    false,
			want:     4,
		},
		{
			name:     "force flag resets to 1",
			existing: []string{"attempt-1.mp3", "attempt-2.mp3"},
			force:    true,
			want:     1,
		},
		{
			name:     "non-matching files ignored",
			existing: []string{"question.md", "solution.cpp", "notes.mp3"},
			force:    false,
			want:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.existing {
				if err := os.WriteFile(filepath.Join(dir, f), []byte{}, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got := nextAttemptNumber(dir, tt.force)
			if got != tt.want {
				t.Errorf("nextAttemptNumber() = %d, want %d", got, tt.want)
			}
		})
	}
}
