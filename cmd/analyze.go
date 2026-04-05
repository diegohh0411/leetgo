package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/j178/leetgo/config"
	"github.com/j178/leetgo/lang"
	"github.com/j178/leetgo/leetcode"

	analysis "github.com/j178/leetgo/analysis_providers"
	_ "github.com/j178/leetgo/analysis_providers/claude"
)

var analyzeForce bool

var analyzeCmd = &cobra.Command{
	Use:   "analyze qid",
	Short: "Analyze problem solution using AI",
	Args:  cobra.ExactArgs(1),
	RunE:  runAnalyze,
}

func init() {
	analyzeCmd.Flags().BoolVarP(&analyzeForce, "force", "f", false, "overwrite existing analysis")
}

// transcriptRe matches attempt-N.md filenames.
var transcriptRe = regexp.MustCompile(`^attempt-(\d+)\.md$`)

// readTranscripts reads all attempt-N.md files from dir, sorted by attempt number.
func readTranscripts(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	type numbered struct {
		n    int
		path string
	}
	var files []numbered
	for _, e := range entries {
		m := transcriptRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		files = append(files, numbered{n: n, path: filepath.Join(dir, e.Name())})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].n < files[j].n
	})

	var transcripts []string
	for _, f := range files {
		data, err := os.ReadFile(f.path)
		if err != nil {
			continue
		}
		transcripts = append(transcripts, string(data))
	}
	return transcripts
}

// readLatestSolution reads the most recently modified code file from dir.
// Looks for common source extensions: .cpp, .py, .go, .java, .rs, .js, .ts
func readLatestSolution(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var best os.FileInfo
	bestPath := ""
	for _, e := range entries {
		ext := filepath.Ext(e.Name())
		switch ext {
		case ".cpp", ".py", ".go", ".java", ".rs", ".js", ".ts", ".c", ".cs":
			info, err := e.Info()
			if err != nil {
				continue
			}
			if best == nil || info.ModTime().After(best.ModTime()) {
				best = info
				bestPath = filepath.Join(dir, e.Name())
			}
		}
	}

	if bestPath == "" {
		return "", fmt.Errorf("no solution file found in %s", dir)
	}

	data, err := os.ReadFile(bestPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	// Parse QID and resolve problem directory.
	c := leetcode.NewClient(leetcode.ReadCredentials())
	qs, err := leetcode.ParseQID(args[0], c)
	if err != nil {
		return err
	}
	if len(qs) > 1 {
		return fmt.Errorf("multiple questions found")
	}

	result, err := lang.GeneratePathsOnly(qs[0])
	if err != nil {
		return err
	}
	outDir := result.TargetDir()

	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		return fmt.Errorf("problem directory %q does not exist — run `leetgo pick` first", outDir)
	}

	// Check for existing analysis.
	analysisPath := filepath.Join(outDir, "analysis.md")
	if !analyzeForce {
		if _, err := os.Stat(analysisPath); err == nil {
			return fmt.Errorf("analysis already exists. Use --force to overwrite.")
		}
	}

	// Gather context.
	question := qs[0].GetFormattedContent()
	solution, err := readLatestSolution(outDir)
	if err != nil {
		return fmt.Errorf("failed to read solution: %w", err)
	}

	transcripts := readTranscripts(outDir)
	if len(transcripts) == 0 {
		return fmt.Errorf("no transcripts found. Run `leetgo transcribe %s` first.", args[0])
	}

	// Get the analyzer from config.
	cfg := config.Get()
	providerName := cfg.Audio.Analyze.Provider
	if providerName == "" {
		providerName = "claude"
	}

	var providerConfig map[string]any
	if providerName == "claude" {
		providerConfig = cfg.Audio.Analyze.Claude
	}

	provider, err := analysis.Get(providerName, providerConfig)
	if err != nil {
		return err
	}

	fmt.Println("Analyzing...")

	ctx := analysis.AnalysisContext{
		Question:    question,
		Solution:    solution,
		Transcripts: transcripts,
	}

	text, err := provider.Analyze(ctx)
	if err != nil {
		return err
	}

	if err := os.WriteFile(analysisPath, []byte(text), 0o644); err != nil {
		return fmt.Errorf("failed to write analysis: %w", err)
	}

	fmt.Printf("✓ Saved %s\n", analysisPath)
	return nil
}
