package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Attempt struct {
	Date   string `yaml:"date"`
	Rating int    `yaml:"rating"`
}

type ProblemHistory struct {
	Title         string    `yaml:"title,omitempty"`
	Difficulty    string    `yaml:"difficulty,omitempty"`
	Box           int       `yaml:"box"`
	StreakPerfect int       `yaml:"streak_perfect"`
	LastReview    string    `yaml:"last_review"`
	Attempts      []Attempt `yaml:"attempts"`
}

type History struct {
	Problems map[string]*ProblemHistory `yaml:"problems"`
}

func LoadHistory(path string) (*History, error) {
	h := &History{Problems: make(map[string]*ProblemHistory)}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return h, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return h, nil
	}
	err = yaml.Unmarshal(data, h)
	if err != nil {
		return nil, err
	}
	if h.Problems == nil {
		h.Problems = make(map[string]*ProblemHistory)
	}
	return h, nil
}

func SaveHistory(path string, h *History) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(h)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LogAttempt records an attempt and updates box/streak.
// For a new problem, it starts at box 1 then applies movement rules.
func LogAttempt(h *History, id, title, difficulty string, rating int, date string) {
	p, exists := h.Problems[id]
	if !exists {
		p = &ProblemHistory{
			Title:      title,
			Difficulty: difficulty,
			Box:        MinBox,
		}
		h.Problems[id] = p
	}
	if title != "" {
		p.Title = title
		p.Difficulty = difficulty
	}

	p.Attempts = append(p.Attempts, Attempt{Date: date, Rating: rating})
	p.LastReview = date

	p.Box = NextBox(p.Box, rating)
	p.StreakPerfect = UpdateStreak(p.StreakPerfect, rating)

	// Mastery: 3 consecutive perfect ratings → jump to max box
	if p.StreakPerfect >= MasteryStreak {
		p.Box = MaxBox
	}
}
