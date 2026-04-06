package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/j178/leetgo/config"
)

var flagCount int

func init() {
	nextCmd.Flags().IntVarP(&flagCount, "count", "n", 10, "number of problems to show")
}

type dueItem struct {
	id          string
	problem     *config.ProblemHistory
	dueDate     time.Time
	overdueDays int
}

var nextCmd = &cobra.Command{
	Use:     "next",
	Short:   "Show problems due for review based on spaced repetition",
	Example: "leetgo next\nleetgo next -n 5",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		historyPath := cfg.HistoryFile()

		h, err := config.LoadHistory(historyPath)
		if err != nil {
			return fmt.Errorf("failed to load history: %w", err)
		}

		if len(h.Problems) == 0 {
			cmd.Println("No problems logged yet. Use 'leetgo log <id> <rating>' to start tracking.")
			return nil
		}

		intervals := cfg.SpacedRepetition.Intervals
		masteredInterval := cfg.SpacedRepetition.MasteredInterval
		today := time.Now().Truncate(24 * time.Hour)

		var due []dueItem
		var nearest *dueItem

		for id, p := range h.Problems {
			lastReview, err := time.Parse("2006-01-02", p.LastReview)
			if err != nil {
				continue
			}
			intervalDays := config.IntervalForBox(p.Box, p.StreakPerfect, intervals, masteredInterval)
			dueDate := lastReview.AddDate(0, 0, intervalDays)
			overdue := int(today.Sub(dueDate).Hours() / 24)

			item := dueItem{
				id:          id,
				problem:     p,
				dueDate:     dueDate,
				overdueDays: overdue,
			}

			if overdue >= 0 {
				due = append(due, item)
			} else if nearest == nil || dueDate.Before(nearest.dueDate) {
				nearest = &item
			}
		}

		if len(due) == 0 {
			if nearest != nil {
				daysUntil := int(nearest.dueDate.Sub(today).Hours()/24) + 1
				name := nearest.id
				if nearest.problem.Title != "" {
					name = fmt.Sprintf("%s (%s)", nearest.id, nearest.problem.Title)
				}
				cmd.Printf("Nothing to review today. Next up: %s in %d day(s).\n", name, daysUntil)
			}
			return nil
		}

		// Sort by most overdue first
		sort.Slice(due, func(i, j int) bool {
			return due[i].overdueDays > due[j].overdueDays
		})

		if flagCount > 0 && len(due) > flagCount {
			due = due[:flagCount]
		}

		w := table.NewWriter()
		w.SetOutputMirror(cmd.OutOrStdout())
		w.SetStyle(table.StyleColoredDark)
		w.AppendHeader(table.Row{"#", "Problem", "Box", "Last", "Rating", "Due", "Streak"})
		w.SetColumnConfigs([]table.ColumnConfig{
			{Number: 2, WidthMax: 35},
		})

		for _, item := range due {
			p := item.problem
			lastRating := p.Attempts[len(p.Attempts)-1].Rating

			dueStr := "today"
			if item.overdueDays > 0 {
				dueStr = fmt.Sprintf("%dd ago", item.overdueDays)
			}

			title := item.id
			if p.Title != "" {
				title = p.Title
			}

			w.AppendRow(table.Row{
				item.id,
				title,
				fmt.Sprintf("%d/5", p.Box),
				p.LastReview,
				ratingStars(lastRating),
				dueStr,
				fmt.Sprintf("%d/%d", p.StreakPerfect, config.MasteryStreak),
			})
		}
		w.Render()

		return nil
	},
}

func ratingStars(rating int) string {
	filled := strings.Repeat("★", rating)
	empty := strings.Repeat("☆", config.MaxRating-rating)
	return filled + empty
}
