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
	stt "github.com/j178/leetgo/stt_providers"
	_ "github.com/j178/leetgo/stt_providers/elevenlabs"
)

var transcribeForce bool

var transcribeCmd = &cobra.Command{
	Use:   "transcribe qid",
	Short: "Transcribe voice note recordings for a problem",
	Args:  cobra.ExactArgs(1),
	RunE:  runTranscribe,
}

func init() {
	transcribeCmd.Flags().BoolVarP(&transcribeForce, "force", "f", false, "re-transcribe all recordings")
}

// audioFileRe matches attempt-N.mp3 filenames.
var audioFileRe = regexp.MustCompile(`^attempt-(\d+)\.mp3$`)

// findUntranscribed returns mp3 filenames (base names) that don't have a matching .md transcript.
func findUntranscribed(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	// Build set of transcribed attempt numbers
	transcribed := map[int]bool{}
	for _, e := range entries {
		if m := audioFileRe.FindStringSubmatch(e.Name()); m != nil {
			mdName := fmt.Sprintf("attempt-%s.md", m[1])
			if _, err := os.Stat(filepath.Join(dir, mdName)); err == nil {
				n, _ := strconv.Atoi(m[1])
				transcribed[n] = true
			}
		}
	}

	var result []string
	for _, e := range entries {
		m := audioFileRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		if !transcribed[n] {
			result = append(result, e.Name())
		}
	}

	sort.Slice(result, func(i, j int) bool {
		ni, _ := strconv.Atoi(audioFileRe.FindStringSubmatch(result[i])[1])
		nj, _ := strconv.Atoi(audioFileRe.FindStringSubmatch(result[j])[1])
		return ni < nj
	})

	return result
}

// findAllAudio returns all attempt-N.mp3 filenames (base names) in dir.
func findAllAudio(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var result []string
	for _, e := range entries {
		if audioFileRe.MatchString(e.Name()) {
			result = append(result, e.Name())
		}
	}

	sort.Slice(result, func(i, j int) bool {
		ni, _ := strconv.Atoi(audioFileRe.FindStringSubmatch(result[i])[1])
		nj, _ := strconv.Atoi(audioFileRe.FindStringSubmatch(result[j])[1])
		return ni < nj
	})

	return result
}

func runTranscribe(cmd *cobra.Command, args []string) error {
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

	// Find audio files to transcribe.
	var audioFiles []string
	if transcribeForce {
		audioFiles = findAllAudio(outDir)
	} else {
		audioFiles = findUntranscribed(outDir)
	}

	if len(audioFiles) == 0 {
		fmt.Println("All transcripts up to date.")
		return nil
	}

	// Get the transcriber from config.
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

	// Transcribe each file.
	transcribed := 0
	for _, audioFile := range audioFiles {
		audioPath := filepath.Join(outDir, audioFile)
		fmt.Printf("Transcribing %s...\n", audioFile)

		text, err := provider.Transcribe(audioPath)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}

		if text == "" {
			fmt.Printf("  Warning: transcript for %s appears empty\n", audioFile)
		}

		// Derive output filename: attempt-N.mp3 → attempt-N.md
		m := audioFileRe.FindStringSubmatch(audioFile)
		if m == nil {
			fmt.Printf("  Error: unexpected audio filename %q\n", audioFile)
			continue
		}
		mdName := fmt.Sprintf("attempt-%s.md", m[1])
		mdPath := filepath.Join(outDir, mdName)

		if err := os.WriteFile(mdPath, []byte(text), 0o644); err != nil {
			fmt.Printf("  Error writing %s: %v\n", mdName, err)
			continue
		}

		fmt.Printf("  ✓ %s\n", mdName)
		transcribed++
	}

	fmt.Printf("Transcribed %d file(s).\n", transcribed)
	return nil
}
