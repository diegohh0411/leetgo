package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/j178/leetgo/config"
	"github.com/j178/leetgo/leetcode"
)

var flagForce bool

func init() {
	logCmd.Flags().BoolVar(&flagForce, "force", false, "skip cache validation, log without problem metadata")
}

var logCmd = &cobra.Command{
	Use:     "log qid rating",
	Short:   "Log a problem attempt with a self-evaluated rating (1-5)",
	Example: "leetgo log 532 5\nleetgo log 999 3 --force",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		rating, err := strconv.Atoi(args[1])
		if err != nil || rating < config.MinRating || rating > config.MaxRating {
			return fmt.Errorf("rating must be between %d and %d", config.MinRating, config.MaxRating)
		}

		cfg := config.Get()
		historyPath := cfg.HistoryFile()

		var title, difficulty string
		if !flagForce {
			c := leetcode.NewClient(leetcode.ReadCredentials())
			q, err := leetcode.QuestionFromCacheByID(id, c)
			if err != nil {
				return fmt.Errorf("problem %s not found in cache, use --force to log anyway", id)
			}
			title = q.GetTitle()
			difficulty = q.Difficulty
		}

		h, err := config.LoadHistory(historyPath)
		if err != nil {
			return fmt.Errorf("failed to load history: %w", err)
		}

		date := time.Now().Format("2006-01-02")
		oldBox := config.MinBox
		if p, exists := h.Problems[id]; exists {
			oldBox = p.Box
		}

		config.LogAttempt(h, id, title, difficulty, rating, date)

		err = config.SaveHistory(historyPath, h)
		if err != nil {
			return fmt.Errorf("failed to save history: %w", err)
		}

		p := h.Problems[id]
		name := id
		if p.Title != "" {
			name = fmt.Sprintf("%s (%s)", id, p.Title)
		}
		log.Info(
			"logged attempt",
			"problem", name,
			"rating", fmt.Sprintf("%d/5", rating),
			"box", fmt.Sprintf("%d→%d", oldBox, p.Box),
			"streak", fmt.Sprintf("%d/%d", p.StreakPerfect, config.MasteryStreak),
		)

		return nil
	},
}
