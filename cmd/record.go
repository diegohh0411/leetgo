package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/j178/leetgo/config"
	"github.com/j178/leetgo/lang"
	"github.com/j178/leetgo/leetcode"

	analysis "github.com/j178/leetgo/analysis_providers"
	_ "github.com/j178/leetgo/analysis_providers/claude"
	stt "github.com/j178/leetgo/stt_providers"
	_ "github.com/j178/leetgo/stt_providers/elevenlabs"
)

var recordForce bool

var recordCmd = &cobra.Command{
	Use:   "record qid",
	Short: "Record a voice note for a problem attempt",
	Args:  cobra.ExactArgs(1),
	RunE:  runRecord,
}

func init() {
	recordCmd.Flags().BoolVarP(&recordForce, "force", "f", false, "restart numbering from attempt-1")
}

// attemptRe matches filenames like "attempt-1.mp3", "attempt-12.mp3", etc.
var attemptRe = regexp.MustCompile(`^attempt-(\d+)\.mp3$`)

// nextAttemptNumber scans dir for existing attempt-N.mp3 files and returns
// the next number. If force is true, returns 1.
func nextAttemptNumber(dir string, force bool) int {
	if force {
		return 1
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 1
	}
	maxN := 0
	for _, e := range entries {
		m := attemptRe.FindStringSubmatch(e.Name())
		if m != nil {
			n, _ := strconv.Atoi(m[1])
			if n > maxN {
				maxN = n
			}
		}
	}
	return maxN + 1
}

func runRecord(cmd *cobra.Command, args []string) error {
	// Check ffmpeg first — fail fast with install instructions.
	if err := checkFFmpeg(); err != nil {
		return err
	}

	// Parse the question ID (supports "219", "contains-duplicate-ii", "today", etc.)
	c := leetcode.NewClient(leetcode.ReadCredentials())
	qs, err := leetcode.ParseQID(args[0], c)
	if err != nil {
		return err
	}
	if len(qs) > 1 {
		return fmt.Errorf("multiple questions found")
	}

	// Resolve the problem output directory.
	result, err := lang.GeneratePathsOnly(qs[0])
	if err != nil {
		return err
	}
	outDir := result.TargetDir()

	// Ensure the output directory exists.
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		return fmt.Errorf("problem directory %q does not exist — run `leetgo pick` first", outDir)
	}

	// Determine the next attempt number.
	attempt := nextAttemptNumber(outDir, recordForce)
	outputPath := fmt.Sprintf("%s/attempt-%d.mp3", outDir, attempt)
	filename := fmt.Sprintf("attempt-%d.mp3", attempt)

	// Launch the recording TUI.
	savedPath, err := runRecorderTUI(outDir, filename, outputPath)
	if err != nil {
		return err
	}

	// If recording was saved, offer post-recording workflow.
	if savedPath != "" {
		fmt.Println() // blank line after TUI
		if promptYesNo("Transcribe now?", true) {
			if err := transcribeFile(savedPath, outDir); err != nil {
				fmt.Printf("Transcription error: %v\n", err)
			} else {
				mdName := fmt.Sprintf("attempt-%d.md", attempt)
				fmt.Printf("✓ Transcribed → %s\n", mdName)

				if promptYesNo("Analyze?", true) {
					if err := runAnalysis(qs[0], outDir); err != nil {
						fmt.Printf("Analysis error: %v\n", err)
					}
				}
			}
		}
	}

	return nil
}

// promptYesNo asks a yes/no question. defaultYes controls the default.
func promptYesNo(question string, defaultYes bool) bool {
	suffix := " [Y/n]"
	if !defaultYes {
		suffix = " [y/N]"
	}
	fmt.Printf("%s%s ", question, suffix)

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		return defaultYes
	}
	return input == "y" || input == "yes"
}

// transcribeFile transcribes a single audio file using the configured provider.
func transcribeFile(audioPath, outDir string) error {
	cfg := config.Get()
	providerName := cfg.Audio.Transcribe.Provider
	if providerName == "" {
		providerName = "elevenlabs"
	}

	var providerConfig map[string]any
	if providerName == "elevenlabs" {
		providerConfig = cfg.Audio.Transcribe.ElevenLabs
	}

	provider, err := stt.Get(providerName, providerConfig)
	if err != nil {
		return err
	}

	text, err := provider.Transcribe(audioPath)
	if err != nil {
		return err
	}

	// Derive md filename from mp3 filename.
	base := filepath.Base(audioPath)
	m := attemptRe.FindStringSubmatch(base)
	if m == nil {
		return fmt.Errorf("unexpected audio filename: %s", base)
	}
	mdPath := filepath.Join(outDir, fmt.Sprintf("attempt-%s.md", m[1]))

	return os.WriteFile(mdPath, []byte(text), 0o644)
}

// runAnalysis runs the analysis pipeline for a problem.
func runAnalysis(q *leetcode.QuestionData, outDir string) error {
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

	solution, err := readLatestSolution(outDir)
	if err != nil {
		return err
	}

	transcripts := readTranscripts(outDir)
	if len(transcripts) == 0 {
		return fmt.Errorf("no transcripts found")
	}

	ctx := analysis.AnalysisContext{
		Question:    q.GetFormattedContent(),
		Solution:    solution,
		Transcripts: transcripts,
	}

	text, err := provider.Analyze(ctx)
	if err != nil {
		return err
	}

	analysisPath := filepath.Join(outDir, "analysis.md")
	fmt.Println("Analyzing...")
	return os.WriteFile(analysisPath, []byte(text), 0o644)
}
