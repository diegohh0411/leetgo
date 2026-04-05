package claude

import (
	"fmt"
	"os/exec"
	"strings"

	analysis "github.com/j178/leetgo/analysis_providers"
)

func init() {
	analysis.Register("claude", NewFactory)
}

// Analyzer implements analysis.Analyzer using the Claude CLI.
type Analyzer struct {
	model string
}

// NewFactory creates a Claude analyzer from config.
func NewFactory(config map[string]any) analysis.Analyzer {
	model := "sonnet"
	if v, ok := config["model"].(string); ok && v != "" {
		model = v
	}
	return &Analyzer{model: model}
}

func (a *Analyzer) Name() string {
	return "claude"
}

// CheckInstalled verifies that the claude CLI is available.
func CheckInstalled() error {
	_, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI is not installed or not on PATH.\n\nInstall it from: https://docs.anthropic.com/en/docs/claude-code")
	}
	return nil
}

// Analyze sends the analysis context to Claude CLI and returns the response.
func (a *Analyzer) Analyze(ctx analysis.AnalysisContext) (string, error) {
	if err := CheckInstalled(); err != nil {
		return "", err
	}

	prompt := buildPrompt(ctx)
	args := buildArgs(prompt, a.model)

	cmd := exec.Command("claude", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude CLI failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("claude CLI error: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// buildPrompt constructs the analysis prompt from context.
func buildPrompt(ctx analysis.AnalysisContext) string {
	transcripts := strings.Join(ctx.Transcripts, "\n\n---\n\n")

	return fmt.Sprintf(`Analyze this LeetCode problem solution based on my voice notes. Keep it brief - 2-3 paragraphs max.

PROBLEM:
%s

MY SOLUTION (latest attempt):
%s

MY VOICE NOTES:
%s

Provide:
1. Brief overview of how the problem went
2. What I did well
3. What I struggled with / areas to improve
4. Improvement guide: if the solution was unsolved, suboptimal, or inefficient, provide a concrete guide on how to solve or optimize it. Include the key algorithm/data structure to use, time/space complexity, and a brief pseudocode outline of the improved approach. If the solution is already optimal, skip this section.

Focus on identifying strengths, weaknesses, and actionable feedback for future practice.`, ctx.Question, ctx.Solution, transcripts)
}

// buildArgs constructs the claude CLI argument list.
func buildArgs(prompt, model string) []string {
	return []string{"--model", model, "-p", prompt, "--output-format", "text"}
}
