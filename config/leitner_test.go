package config

import (
	"testing"
)

func TestNextBox(t *testing.T) {
	tests := []struct {
		name    string
		current int
		rating  int
		want    int
	}{
		{"rating 5 moves up", 1, 5, 2},
		{"rating 4 moves up", 2, 4, 3},
		{"rating 3 stays", 3, 3, 3},
		{"rating 2 drops to 1", 4, 2, 1},
		{"rating 1 drops to 1", 3, 1, 1},
		{"box 5 stays at 5 on rating 5", 5, 5, 5},
		{"box 1 stays at 1 on rating 1", 1, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NextBox(tt.current, tt.rating)
			if got != tt.want {
				t.Errorf("NextBox(%d, %d) = %d, want %d", tt.current, tt.rating, got, tt.want)
			}
		})
	}
}

func TestUpdateStreak(t *testing.T) {
	tests := []struct {
		name    string
		current int
		rating  int
		want    int
	}{
		{"perfect increments", 0, 5, 1},
		{"perfect increments from 2", 2, 5, 3},
		{"non-perfect resets", 1, 4, 0},
		{"non-perfect resets from 2", 2, 3, 0},
		{"streak caps at 3", 3, 5, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UpdateStreak(tt.current, tt.rating)
			if got != tt.want {
				t.Errorf("UpdateStreak(%d, %d) = %d, want %d", tt.current, tt.rating, got, tt.want)
			}
		})
	}
}

func TestIntervalForBox(t *testing.T) {
	intervals := []int{1, 3, 7, 14, 30}
	tests := []struct {
		name          string
		box           int
		streakPerfect int
		masteredIntvl int
		want          int
	}{
		{"box 1", 1, 0, 45, 1},
		{"box 3", 3, 0, 45, 7},
		{"box 5", 5, 0, 45, 30},
		{"mastered uses mastered interval", 5, 3, 45, 45},
		{"mastered in box 3 still uses mastered", 3, 3, 45, 45},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IntervalForBox(tt.box, tt.streakPerfect, intervals, tt.masteredIntvl)
			if got != tt.want {
				t.Errorf("IntervalForBox(%d, %d, ...) = %d, want %d", tt.box, tt.streakPerfect, got, tt.want)
			}
		})
	}
}
