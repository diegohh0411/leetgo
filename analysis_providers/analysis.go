// analysis_providers/analysis.go
package analysis_providers

import (
	"fmt"
	"sort"
	"strings"
)

// AnalysisContext holds all input data for an analysis request.
type AnalysisContext struct {
	Question    string   // problem description (markdown)
	Solution    string   // latest solution source code
	Transcripts []string // contents of all attempt-N.md files
}

// Analyzer produces structured analysis from problem context.
type Analyzer interface {
	Analyze(ctx AnalysisContext) (string, error)
	Name() string
}

// registries maps provider names to factory functions.
var registries = map[string]func(config map[string]any) Analyzer{}

// Register adds an analyzer factory under the given name.
func Register(name string, factory func(config map[string]any) Analyzer) {
	registries[name] = factory
}

// Get creates an analyzer by name, passing config to the factory.
func Get(name string, config map[string]any) (Analyzer, error) {
	factory, ok := registries[name]
	if !ok {
		available := make([]string, 0, len(registries))
		for k := range registries {
			available = append(available, k)
		}
		sort.Strings(available)
		return nil, fmt.Errorf("unknown analysis provider %q (available: %s)", name, strings.Join(available, ", "))
	}
	return factory(config), nil
}
