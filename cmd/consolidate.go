package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/j178/leetgo/config"
	"github.com/j178/leetgo/lang"
)

var consolidateCmd = &cobra.Command{
	Use:   "consolidate",
	Short: "Consolidate all problem analyses into a master file",
	Args:  cobra.NoArgs,
	RunE:  runConsolidate,
}

// dateRe matches "Created by ... at YYYY/MM/DD HH:MM" in solution file headers.
var dateRe = regexp.MustCompile(`(\d{4}/\d{2}/\d{2}\s+\d{2}:\d{2})`)

type analysisEntry struct {
	dirName string
	content string
	date    time.Time
}

// parseDateFromSolution returns the file's last modification time, which
// reflects when the solution was actually worked on (not when it was scaffolded).
// Falls back to the header date if stat fails.
func parseDateFromSolution(path string) time.Time {
	info, err := os.Stat(path)
	if err == nil {
		return info.ModTime()
	}
	// Fallback: parse date from leetgo-generated header.
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}
	}
	lines := strings.SplitN(string(data), "\n", 5)
	for _, line := range lines {
		if m := dateRe.FindString(line); m != "" {
			t, err := time.Parse("2006/01/02 15:04", m)
			if err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

// findSolutionFile returns the path to the first solution file in dir.
func findSolutionFile(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		switch ext {
		case ".cpp", ".py", ".go", ".java", ".rs", ".js", ".ts", ".c", ".cs", ".rb", ".swift", ".kt":
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}

// collectAnalyses scans all language output directories for analysis.md files.
func collectAnalyses() ([]analysisEntry, error) {
	cfg := config.Get()
	root := cfg.ProjectRoot()

	// Collect unique output directories from all supported languages.
	seen := map[string]bool{}
	var outDirs []string
	for _, l := range lang.SupportedLangs {
		dir := filepath.Join(root, l.Slug())
		if !seen[dir] {
			seen[dir] = true
			outDirs = append(outDirs, dir)
		}
	}

	var entries []analysisEntry
	for _, outDir := range outDirs {
		dirs, err := os.ReadDir(outDir)
		if err != nil {
			continue // language dir doesn't exist, skip
		}
		for _, d := range dirs {
			if !d.IsDir() {
				continue
			}
			problemDir := filepath.Join(outDir, d.Name())
			analysisPath := filepath.Join(problemDir, "analysis.md")
			data, err := os.ReadFile(analysisPath)
			if err != nil {
				continue // no analysis
			}

			var date time.Time
			if sol := findSolutionFile(problemDir); sol != "" {
				date = parseDateFromSolution(sol)
			}

			entries = append(entries, analysisEntry{
				dirName: d.Name(),
				content: string(data),
				date:    date,
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].date.Before(entries[j].date)
	})

	return entries, nil
}

func runConsolidate(cmd *cobra.Command, args []string) error {
	entries, err := collectAnalyses()
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Println("No analyses found.")
		return nil
	}

	cfg := config.Get()
	masterPath := filepath.Join(cfg.ProjectRoot(), "analyses-master.md")

	var sb strings.Builder
	sb.WriteString("# Leetcode Analyses Master\n\n")
	sb.WriteString(fmt.Sprintf("Generated on %s\n\n", time.Now().Format("2006-01-02 15:04")))

	for _, e := range entries {
		dateStr := ""
		if !e.date.IsZero() {
			dateStr = fmt.Sprintf(" (%s)", e.date.Format("2006/01/02"))
		}
		sb.WriteString(fmt.Sprintf("## %s%s\n\n", e.dirName, dateStr))
		sb.WriteString(strings.TrimSpace(e.content))
		sb.WriteString("\n\n---\n\n")
	}

	if err := os.WriteFile(masterPath, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write master analysis: %w", err)
	}

	fmt.Printf("✓ Master analysis written to %s (%d problems)\n", masterPath, len(entries))
	return nil
}
