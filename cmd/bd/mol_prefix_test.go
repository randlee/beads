package main

import (
	"testing"

	"github.com/steveyegge/beads/internal/config"
)

// TestOverlayMolPrefix covers the front-door prefix composition for pour/wisp:
// workspace issue-prefix + mol/wisp suffix. Mirrors TestOverlayYAMLPrefix for
// the create path, plus the --global and trailing-dash/whitespace edge cases.
func TestOverlayMolPrefix(t *testing.T) {
	tests := []struct {
		name       string
		global     bool
		yamlPrefix string
		suffix     string
		want       string
	}{
		{"workspace prefix composes mol", false, "ap", "mol", "ap-mol"},
		{"workspace prefix composes wisp", false, "ap", "wisp", "ap-wisp"},
		{"trailing dash trimmed", false, "ap-", "mol", "ap-mol"},
		{"surrounding whitespace trimmed", false, "  ap  ", "mol", "ap-mol"},
		{"no workspace prefix falls back to empty", false, "", "mol", ""},
		{"whitespace-only prefix treated as empty", false, "   ", "mol", ""},
		{"global ignores workspace prefix", true, "ap", "mol", ""},
		{"global with empty prefix", true, "", "wisp", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.ResetForTesting()
			_ = config.Initialize()
			config.Set("issue-prefix", tt.yamlPrefix)
			oldGlobal := globalFlag
			globalFlag = tt.global
			t.Cleanup(func() {
				config.ResetForTesting()
				globalFlag = oldGlobal
			})

			if got := overlayMolPrefix(tt.suffix); got != tt.want {
				t.Errorf("overlayMolPrefix(%q) = %q, want %q (global=%t, yaml=%q)",
					tt.suffix, got, tt.want, tt.global, tt.yamlPrefix)
			}
		})
	}
}
