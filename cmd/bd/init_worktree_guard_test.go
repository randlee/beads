package main

// Regression tests for contessa-09i (P1): `bd init` inside a git worktree
// silently created `.beads/` at the MAIN repo root (exit 0, "initialized
// successfully"), landing data at a path the operator never chose. The
// worktree fallback is for attaching to an EXISTING shared store; init must
// hard-fail when it would create one.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steveyegge/beads/internal/git"
)

// setupWorktreeRepo creates main-repo + linked worktree, returns both paths.
func setupWorktreeRepo(t *testing.T) (mainDir, wtDir string) {
	t.Helper()
	tmp := t.TempDir()
	mainDir = filepath.Join(tmp, "main-repo")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatal(err)
	}
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	run(mainDir, "init")
	run(mainDir, "config", "user.email", "test@example.com")
	run(mainDir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(mainDir, "README.md"), []byte("# t\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run(mainDir, "add", "README.md")
	run(mainDir, "commit", "-m", "init")
	wtDir = filepath.Join(tmp, "worktree")
	run(mainDir, "worktree", "add", "-b", "feature", wtDir)
	t.Cleanup(func() {
		cmd := exec.Command("git", "worktree", "remove", "--force", wtDir)
		cmd.Dir = mainDir
		_ = cmd.Run()
	})
	return mainDir, wtDir
}

// TestGuardWorktreeInit_RefusesWhenNoSharedStore is the direct unit contract:
// missing fallback dir -> error naming contessa-09i; existing -> nil.
func TestGuardWorktreeInit_RefusesWhenNoSharedStore(t *testing.T) {
	tmp := t.TempDir()

	missing := filepath.Join(tmp, "no-such-repo", ".beads")
	err := guardWorktreeInit(missing)
	if err == nil {
		t.Fatalf("guardWorktreeInit(%q) = nil, want refusal", missing)
	}
	for _, want := range []string{"refusing to initialize in git worktree", "contessa-09i", "BEADS_DIR"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err.Error(), want)
		}
	}
	// Must not have created anything.
	if _, statErr := os.Stat(missing); statErr == nil {
		t.Errorf("guardWorktreeInit created %q", missing)
	}

	existing := filepath.Join(tmp, "has-repo", ".beads")
	if err := os.MkdirAll(existing, 0755); err != nil {
		t.Fatal(err)
	}
	if err := guardWorktreeInit(existing); err != nil {
		t.Errorf("guardWorktreeInit(%q) = %v, want nil (attach to existing shared store)", existing, err)
	}
}

// TestInitCommand_WorktreeHardFail is the end-to-end regression: run the real
// init command from inside a worktree with no shared store and assert it
// fails with the guard error and writes nothing to the main repo root.
func TestInitCommand_WorktreeHardFail(t *testing.T) {
	skipIfNoDolt(t)
	mainDir, wtDir := setupWorktreeRepo(t)

	// Global git context is process-wide and sync.Once-cached; reset around chdir.
	origDBPath := dbPath
	origStore := store
	defer func() {
		if store != nil && store != origStore {
			store.Close()
		}
		store = origStore
		dbPath = origDBPath
		git.ResetCaches()
	}()
	dbPath = ""
	store = nil
	t.Setenv("BEADS_DIR", "")
	os.Unsetenv("BEADS_DIR")

	t.Chdir(wtDir)
	git.ResetCaches()

	rootCmd.SetArgs([]string{"init", "--prefix", "wt", "--non-interactive", "--quiet"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("init in worktree with no shared store: err = nil, want refusal (contessa-09i)")
	}
	if !strings.Contains(err.Error(), "refusing to initialize in git worktree") {
		t.Fatalf("init in worktree: err = %v, want worktree refusal", err)
	}
	// The bug's signature: .beads/ silently created at the MAIN repo root.
	if _, statErr := os.Stat(filepath.Join(mainDir, ".beads")); statErr == nil {
		t.Error("contessa-09i regression: init created .beads at main repo root from a worktree")
	}
	if _, statErr := os.Stat(filepath.Join(wtDir, ".beads")); statErr == nil {
		t.Error("init created .beads in the worktree despite refusing")
	}
}
