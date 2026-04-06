package config

const (
	MinBox        = 1
	MaxBox        = 5
	MasteryStreak = 3
	MaxRating     = 5
	MinRating     = 1
)

// NextBox returns the new box after a review with the given rating.
// Rating 4-5: move up one box (capped at 5).
// Rating 3: stay in current box.
// Rating 1-2: drop to box 1.
func NextBox(current, rating int) int {
	switch {
	case rating >= 4:
		if current < MaxBox {
			return current + 1
		}
		return MaxBox
	case rating == 3:
		return current
	default:
		return MinBox
	}
}

// UpdateStreak returns the new perfect-streak count after a review.
// Increments on rating 5, resets on anything else, caps at MasteryStreak.
func UpdateStreak(current, rating int) int {
	if rating == MaxRating {
		if current < MasteryStreak {
			return current + 1
		}
		return MasteryStreak
	}
	return 0
}

// IntervalForBox returns the review interval in days for a given box and streak.
// Mastered problems (streak == MasteryStreak) use the mastered interval.
func IntervalForBox(box, streakPerfect int, intervals []int, masteredInterval int) int {
	if streakPerfect >= MasteryStreak {
		return masteredInterval
	}
	idx := box - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(intervals) {
		idx = len(intervals) - 1
	}
	return intervals[idx]
}
