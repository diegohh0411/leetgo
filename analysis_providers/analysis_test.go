// analysis_providers/analysis_test.go
package analysis_providers

import (
	"testing"
)

type mockAnalyzer struct {
	name string
}

func (m *mockAnalyzer) Analyze(ctx AnalysisContext) (string, error) {
	return "analyzed", nil
}

func (m *mockAnalyzer) Name() string {
	return m.name
}

func TestRegisterAndGetAnalyzer(t *testing.T) {
	registries = map[string]func(config map[string]any) Analyzer{}

	Register("test-analyzer", func(config map[string]any) Analyzer {
		return &mockAnalyzer{name: "test-analyzer"}
	})

	got, err := Get("test-analyzer", nil)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Name() != "test-analyzer" {
		t.Errorf("Name() = %q, want %q", got.Name(), "test-analyzer")
	}
}

func TestGetUnknownAnalyzer(t *testing.T) {
	registries = map[string]func(config map[string]any) Analyzer{}

	_, err := Get("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
}
