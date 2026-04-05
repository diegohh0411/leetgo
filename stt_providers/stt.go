package stt_providers

import (
	"fmt"
	"sort"
	"strings"
)

// Transcriber converts an audio file to text.
type Transcriber interface {
	Transcribe(audioPath string) (string, error)
	Name() string
}

// registries maps provider names to factory functions.
var registries = map[string]func(config map[string]any) Transcriber{}

// Register adds a transcriber factory under the given name.
func Register(name string, factory func(config map[string]any) Transcriber) {
	registries[name] = factory
}

// Get creates a transcriber by name, passing config to the factory.
func Get(name string, config map[string]any) (Transcriber, error) {
	factory, ok := registries[name]
	if !ok {
		available := make([]string, 0, len(registries))
		for k := range registries {
			available = append(available, k)
		}
		sort.Strings(available)
		return nil, fmt.Errorf("unknown STT provider %q (available: %s)", name, strings.Join(available, ", "))
	}
	return factory(config), nil
}
