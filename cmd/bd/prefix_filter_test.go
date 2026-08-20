package main

import (
	"context"
	"path/filepath"
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

// prefixStoreFixture installs a store with a known issue_prefix as the
// package-global store (the source the fallback reads), and restores the
// originals after the test.
func prefixStoreFixture(t *testing.T, prefix string) {
	t.Helper()
	originalStore, originalRootCtx := store, rootCtx
	t.Cleanup(func() { store, rootCtx = originalStore, originalRootCtx })

	rootCtx = context.Background()
	tmpDir := t.TempDir()
	testStore := newTestStoreWithPrefix(t, filepath.Join(tmpDir, "test.db"), prefix)
	store = testStore
}

func TestListPrefixFallsBackToIssuePrefix(t *testing.T) {
	resetConfigForPrefixTest(t)
	prefixStoreFixture(t, "skillrx")

	in, err := gatherListInput(newListFlagsCommand(t))
	if err != nil {
		t.Fatalf("gatherListInput: %v", err)
	}
	if in.Prefix != "skillrx-" {
		t.Errorf("in.Prefix = %q, want skillrx- (fallback to store issue_prefix)", in.Prefix)
	}
}

func TestListPrefixFlagBeatsIssuePrefix(t *testing.T) {
	resetConfigForPrefixTest(t)
	prefixStoreFixture(t, "skillrx")

	in, err := gatherListInput(newListFlagsCommand(t, "--prefix", "pater"))
	if err != nil {
		t.Fatalf("gatherListInput(--prefix): %v", err)
	}
	if in.Prefix != "pater-" {
		t.Errorf("in.Prefix = %q, want pater- (flag outranks issue_prefix)", in.Prefix)
	}
}

func TestListPrefixAllBeatsIssuePrefix(t *testing.T) {
	resetConfigForPrefixTest(t)
	prefixStoreFixture(t, "skillrx")

	in, err := gatherListInput(newListFlagsCommand(t, "--all"))
	if err != nil {
		t.Fatalf("gatherListInput(--all): %v", err)
	}
	if in.Prefix != "" {
		t.Errorf("in.Prefix = %q, want empty (--all bypasses issue_prefix)", in.Prefix)
	}
}

func TestNormalizePrefixFilter(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"skillrx", "skillrx-"},
		{"skillrx-", "skillrx-"},
		{"skillrx--", "skillrx-"},
		{"sc-dolt", "sc-dolt-"},
	}
	for _, c := range cases {
		if got := normalizePrefixFilter(c.in); got != c.want {
			t.Errorf("normalizePrefixFilter(%q) = %q, want %q", c.in, got, c.want)
		}
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

func TestReadyPrefixFallsBackToIssuePrefix(t *testing.T) {
	resetConfigForPrefixTest(t)
	prefixStoreFixture(t, "skillrx")

	got := runGatherReadyInput(t, newReadyFlagsCommand(t), nil)
	if got.err != nil {
		t.Fatalf("gatherReadyInput: %v", got.err)
	}
	if got.in.Prefix != "skillrx-" {
		t.Errorf("in.Prefix = %q, want skillrx- (fallback to store issue_prefix)", got.in.Prefix)
	}
	if got.in.filter.IDPrefix != "skillrx-" {
		t.Errorf("filter.IDPrefix = %q, want skillrx- (fallback to store issue_prefix)", got.in.filter.IDPrefix)
	}
}
