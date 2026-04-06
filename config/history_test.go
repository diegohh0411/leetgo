package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadHistoryEmpty(t *testing.T) {
	h, err := LoadHistory("/nonexistent/path/history.yaml")
	if err != nil {
		t.Fatalf("LoadHistory should not error on missing file: %v", err)
	}
	if len(h.Problems) != 0 {
		t.Errorf("expected empty problems, got %d", len(h.Problems))
	}
}

func TestSaveAndLoadHistory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.yaml")

	h := &History{
		Problems: map[string]*ProblemHistory{
			"532": {
				Title:         "K-diff Pairs in an Array",
				Difficulty:    "Medium",
				Box:           2,
				StreakPerfect: 1,
				LastReview:    "2026-04-06",
				Attempts: []Attempt{
					{Date: "2026-04-06", Rating: 5},
				},
			},
		},
	}

	err := SaveHistory(path, h)
	if err != nil {
		t.Fatalf("SaveHistory failed: %v", err)
	}

	loaded, err := LoadHistory(path)
	if err != nil {
		t.Fatalf("LoadHistory failed: %v", err)
	}
	if len(loaded.Problems) != 1 {
		t.Fatalf("expected 1 problem, got %d", len(loaded.Problems))
	}
	p := loaded.Problems["532"]
	if p.Title != "K-diff Pairs in an Array" {
		t.Errorf("expected title 'K-diff Pairs in an Array', got %q", p.Title)
	}
	if p.Box != 2 {
		t.Errorf("expected box 2, got %d", p.Box)
	}
	if p.StreakPerfect != 1 {
		t.Errorf("expected streak 1, got %d", p.StreakPerfect)
	}
	if len(p.Attempts) != 1 {
		t.Fatalf("expected 1 attempt, got %d", len(p.Attempts))
	}
}

func TestLogAttempt(t *testing.T) {
	h := &History{
		Problems: make(map[string]*ProblemHistory),
	}

	LogAttempt(h, "532", "K-diff Pairs", "Medium", 4, "2026-04-06")

	p := h.Problems["532"]
	if p == nil {
		t.Fatal("problem 532 not found after logging")
	}
	if p.Box != 2 {
		t.Errorf("expected box 2 (new problem starts at 1, rating 4 moves up), got %d", p.Box)
	}
	if p.StreakPerfect != 0 {
		t.Errorf("expected streak 0 (rating 4), got %d", p.StreakPerfect)
	}

	// Second attempt with rating 5
	LogAttempt(h, "532", "K-diff Pairs", "Medium", 5, "2026-04-07")
	if p.Box != 3 {
		t.Errorf("expected box 3, got %d", p.Box)
	}
	if p.StreakPerfect != 1 {
		t.Errorf("expected streak 1, got %d", p.StreakPerfect)
	}
	if len(p.Attempts) != 2 {
		t.Errorf("expected 2 attempts, got %d", len(p.Attempts))
	}
}

func TestLogAttemptMastery(t *testing.T) {
	h := &History{
		Problems: map[string]*ProblemHistory{
			"1": {
				Title:         "Two Sum",
				Difficulty:    "Easy",
				Box:           2,
				StreakPerfect: 2,
				LastReview:    "2026-04-05",
				Attempts:      []Attempt{{Date: "2026-04-05", Rating: 5}},
			},
		},
	}

	LogAttempt(h, "1", "Two Sum", "Easy", 5, "2026-04-06")

	p := h.Problems["1"]
	if p.StreakPerfect != 3 {
		t.Errorf("expected streak 3 (mastered), got %d", p.StreakPerfect)
	}
	if p.Box != MaxBox {
		t.Errorf("expected box %d (mastered jumps to max), got %d", MaxBox, p.Box)
	}
}

func TestSaveHistoryCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "history.yaml")

	h := &History{Problems: map[string]*ProblemHistory{}}
	err := SaveHistory(path, h)
	if err != nil {
		t.Fatalf("SaveHistory failed: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected file to be created")
	}
}
