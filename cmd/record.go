package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/j178/leetgo/lang"
	"github.com/j178/leetgo/leetcode"
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
	return runRecorderTUI(outDir, filename, outputPath)
}
