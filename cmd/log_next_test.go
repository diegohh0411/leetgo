package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/j178/leetgo/config"
)

func TestLogAndNextIntegration(t *testing.T) {
	dir := t.TempDir()
	historyPath := filepath.Join(dir, "history.yaml")

	// Log two problems
	h := &config.History{Problems: make(map[string]*config.ProblemHistory)}
	config.LogAttempt(h, "532", "K-diff Pairs", "Medium", 3, "2026-04-01")
	config.LogAttempt(h, "1", "Two Sum", "Easy", 5, "2026-04-01")

	err := config.SaveHistory(historyPath, h)
	if err != nil {
		t.Fatalf("SaveHistory failed: %v", err)
	}

	// Load and verify
	loaded, err := config.LoadHistory(historyPath)
	if err != nil {
		t.Fatalf("LoadHistory failed: %v", err)
	}

	if len(loaded.Problems) != 2 {
		t.Fatalf("expected 2 problems, got %d", len(loaded.Problems))
	}

	// Problem 532: rating 3 stays at box 1
	p532 := loaded.Problems["532"]
	if p532.Box != 1 {
		t.Errorf("532: expected box 1 (rating 3 stays), got %d", p532.Box)
	}

	// Problem 1: rating 5 moves from box 1 to box 2
	p1 := loaded.Problems["1"]
	if p1.Box != 2 {
		t.Errorf("1: expected box 2 (rating 5 moves up), got %d", p1.Box)
	}
	if p1.StreakPerfect != 1 {
		t.Errorf("1: expected streak 1, got %d", p1.StreakPerfect)
	}

	// Verify YAML file exists and is readable
	data, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("failed to read history file: %v", err)
	}
	if len(data) == 0 {
		t.Error("history file is empty")
	}
}

func TestMasteryFlow(t *testing.T) {
	h := &config.History{Problems: make(map[string]*config.ProblemHistory)}

	// Three consecutive 5-star ratings
	config.LogAttempt(h, "1", "Two Sum", "Easy", 5, "2026-04-01")
	config.LogAttempt(h, "1", "Two Sum", "Easy", 5, "2026-04-02")
	config.LogAttempt(h, "1", "Two Sum", "Easy", 5, "2026-04-03")

	p := h.Problems["1"]
	if p.StreakPerfect != config.MasteryStreak {
		t.Errorf("expected streak %d (mastered), got %d", config.MasteryStreak, p.StreakPerfect)
	}
	if p.Box != config.MaxBox {
		t.Errorf("expected box %d (mastered), got %d", config.MaxBox, p.Box)
	}

	// A non-perfect rating resets streak but box stays at max (already capped)
	config.LogAttempt(h, "1", "Two Sum", "Easy", 4, "2026-04-04")
	if p.StreakPerfect != 0 {
		t.Errorf("expected streak 0 after non-perfect, got %d", p.StreakPerfect)
	}
	if p.Box != config.MaxBox {
		t.Errorf("expected box %d (already max, rating 4 moves up but capped), got %d", config.MaxBox, p.Box)
	}
}
