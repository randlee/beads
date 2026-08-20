package main

import (
	"testing"

	"github.com/steveyegge/beads/internal/config"
)

// resetConfigForPrefixTest gives each prefix-filter test a clean config plane,
// so a list.prefix value set by one test cannot leak into another. The
// ResetForTesting/Initialize pair is the same reset configureDirectoryLabel
// uses; it is required because config state is package-global.
func resetConfigForPrefixTest(t *testing.T) {
	t.Helper()
	config.ResetForTesting()
	t.Cleanup(config.ResetForTesting)
	if err := config.Initialize(); err != nil {
		t.Fatalf("config.Initialize: %v", err)
	}
}

// --- bd list ---

func TestListPrefixFlagMapsToRequest(t *testing.T) {
	in, err := gatherListInput(newListFlagsCommand(t, "--prefix", "skillrx-"))
	if err != nil {
		t.Fatalf("gatherListInput(--prefix): %v", err)
	}
	if in.Prefix != "skillrx-" {
		t.Errorf("in.Prefix = %q, want skillrx-", in.Prefix)
	}
}

func TestListPrefixFromConfigDefault(t *testing.T) {
	resetConfigForPrefixTest(t)
	config.Set("list.prefix", "skillrx-")

	in, err := gatherListInput(newListFlagsCommand(t))
	if err != nil {
		t.Fatalf("gatherListInput: %v", err)
	}
	if in.Prefix != "skillrx-" {
		t.Errorf("in.Prefix = %q, want skillrx- from list.prefix config", in.Prefix)
	}
}

func TestListPrefixFromEnv(t *testing.T) {
	resetConfigForPrefixTest(t)
	t.Setenv("BD_LIST_PREFIX", "skillrx-")

	in, err := gatherListInput(newListFlagsCommand(t))
	if err != nil {
		t.Fatalf("gatherListInput: %v", err)
	}
	if in.Prefix != "skillrx-" {
		t.Errorf("in.Prefix = %q, want skillrx- from BD_LIST_PREFIX env", in.Prefix)
	}
}

func TestListPrefixAllOverridesConfig(t *testing.T) {
	resetConfigForPrefixTest(t)
	config.Set("list.prefix", "skillrx-")

	in, err := gatherListInput(newListFlagsCommand(t, "--all"))
	if err != nil {
		t.Fatalf("gatherListInput(--all): %v", err)
	}
	if in.Prefix != "" {
		t.Errorf("in.Prefix = %q, want empty (--all bypasses list.prefix)", in.Prefix)
	}
}

func TestListPrefixLegacyNoFilter(t *testing.T) {
	resetConfigForPrefixTest(t)

	in, err := gatherListInput(newListFlagsCommand(t))
	if err != nil {
		t.Fatalf("gatherListInput: %v", err)
	}
	if in.Prefix != "" {
		t.Errorf("in.Prefix = %q, want empty (legacy: no prefix filter)", in.Prefix)
	}
}

// --- bd ready ---

func TestReadyPrefixFlagMapsToFilter(t *testing.T) {
	got := runGatherReadyInput(t, newReadyFlagsCommand(t, "--prefix", "skillrx-"), nil)
	if got.err != nil {
		t.Fatalf("gatherReadyInput(--prefix): %v", got.err)
	}
	if got.in.Prefix != "skillrx-" {
		t.Errorf("in.Prefix = %q, want skillrx-", got.in.Prefix)
	}
	if got.in.filter.IDPrefix != "skillrx-" {
		t.Errorf("filter.IDPrefix = %q, want skillrx-", got.in.filter.IDPrefix)
	}
}

func TestReadyPrefixFromConfigDefault(t *testing.T) {
	resetConfigForPrefixTest(t)
	config.Set("list.prefix", "skillrx-")

	got := runGatherReadyInput(t, newReadyFlagsCommand(t), nil)
	if got.err != nil {
		t.Fatalf("gatherReadyInput: %v", got.err)
	}
	if got.in.Prefix != "skillrx-" {
		t.Errorf("in.Prefix = %q, want skillrx- from list.prefix config", got.in.Prefix)
	}
	if got.in.filter.IDPrefix != "skillrx-" {
		t.Errorf("filter.IDPrefix = %q, want skillrx- from list.prefix config", got.in.filter.IDPrefix)
	}
}

func TestReadyPrefixLegacyNoFilter(t *testing.T) {
	resetConfigForPrefixTest(t)

	got := runGatherReadyInput(t, newReadyFlagsCommand(t), nil)
	if got.err != nil {
		t.Fatalf("gatherReadyInput: %v", got.err)
	}
	if got.in.Prefix != "" {
		t.Errorf("in.Prefix = %q, want empty (legacy)", got.in.Prefix)
	}
	if got.in.filter.IDPrefix != "" {
		t.Errorf("filter.IDPrefix = %q, want empty (legacy)", got.in.filter.IDPrefix)
	}
}
