// analysis_providers/claude/claude_test.go
package claude

import (
	"strings"
	"testing"

	analysis "github.com/j178/leetgo/analysis_providers"
)

func TestBuildPrompt(t *testing.T) {
	ctx := analysis.AnalysisContext{
		Question:    "Two Sum problem description",
		Solution:    "func twoSum(nums []int, target int) []int {",
		Transcripts: []string{"I started by thinking about brute force", "Then optimized with a hashmap"},
	}

	prompt := buildPrompt(ctx)

	if !strings.Contains(prompt, "Two Sum problem description") {
		t.Error("prompt should contain the question")
	}
	if !strings.Contains(prompt, "func twoSum") {
		t.Error("prompt should contain the solution")
	}
	if !strings.Contains(prompt, "I started by thinking about brute force") {
		t.Error("prompt should contain transcripts")
	}
	if !strings.Contains(prompt, "Then optimized with a hashmap") {
		t.Error("prompt should contain all transcripts")
	}
	if !strings.Contains(prompt, "What I did well") {
		t.Error("prompt should contain analysis structure request")
	}
}

func TestBuildArgs(t *testing.T) {
	args := buildArgs("test prompt", "sonnet")

	found := false
	for i, a := range args {
		if a == "--model" && i+1 < len(args) && args[i+1] == "sonnet" {
			found = true
		}
	}
	if !found {
		t.Error("args should contain --model sonnet")
	}

	hasPFlag := false
	for _, a := range args {
		if a == "-p" {
			hasPFlag = true
		}
	}
	if !hasPFlag {
		t.Error("args should contain -p flag")
	}
}

func TestNewFactory(t *testing.T) {
	config := map[string]any{
		"model": "sonnet",
	}
	a := NewFactory(config)
	if a.Name() != "claude" {
		t.Errorf("Name() = %q, want %q", a.Name(), "claude")
	}
}
