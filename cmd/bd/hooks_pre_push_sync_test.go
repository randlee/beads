package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/steveyegge/beads/internal/config"
)

// TestShouldSyncOnPush exercises the config gates that decide whether the
// pre-push hook spawns `bd sync`. The subprocess shell-out itself is not
// unit-tested here — that path requires a working bd binary on PATH and is
// covered by the integration suite (mirrors TestImportJSONLForSync_GuardClauses).
func TestShouldSyncOnPush(t *testing.T) {
	// Isolate config from the repo's tracked .beads/config.yaml and initialize
	// viper so config.Set/GetString round-trip (be-yjp4z).
	t.Chdir(t.TempDir())
	t.Setenv("BEADS_DIR", "")
	t.Setenv("BEADS_TEST_IGNORE_REPO_CONFIG", "1")
	config.ResetForTesting()
	t.Cleanup(func() { config.ResetForTesting() })
	if err := config.Initialize(); err != nil {
		t.Fatalf("config.Initialize: %v", err)
	}

	t.Run("no remote is false", func(t *testing.T) {
		config.Set("sync.remote", "")
		config.Set("sync.git-remote", "")
		config.Set("dolt.local-only", false)
		if shouldSyncOnPush() {
			t.Fatal("expected false when no sync remote is configured")
		}
	})

	t.Run("sync.remote configured is true", func(t *testing.T) {
		config.Set("sync.remote", "https://doltremoteapi.dolthub.com/acme/beads")
		config.Set("sync.git-remote", "")
		config.Set("dolt.local-only", false)
		if !shouldSyncOnPush() {
			t.Fatal("expected true when sync.remote is configured")
		}
	})

	t.Run("sync.git-remote fallback is true", func(t *testing.T) {
		config.Set("sync.remote", "")
		config.Set("sync.git-remote", "git+ssh://git@example.com/acme/repo.git")
		config.Set("dolt.local-only", false)
		if !shouldSyncOnPush() {
			t.Fatal("expected true when only sync.git-remote is configured")
		}
	})

	t.Run("dolt.local-only disables", func(t *testing.T) {
		config.Set("sync.remote", "https://doltremoteapi.dolthub.com/acme/beads")
		config.Set("sync.git-remote", "")
		config.Set("dolt.local-only", true)
		if shouldSyncOnPush() {
			t.Fatal("expected false when dolt.local-only=true")
		}
	})

	t.Run("sync.pre-push=false disables even with remote", func(t *testing.T) {
		config.Set("sync.remote", "https://doltremoteapi.dolthub.com/acme/beads")
		config.Set("sync.git-remote", "")
		config.Set("dolt.local-only", false)
		config.Set("sync.pre-push", false)
		if shouldSyncOnPush() {
			t.Fatal("expected false when sync.pre-push=false")
		}
	})
}

// TestSyncBeadsOnPush_GuardClauses verifies the early-return paths so
// runPrePushHook never spawns a sync subprocess on a misconfigured or
// remote-less workspace.
func TestSyncBeadsOnPush_GuardClauses(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("BEADS_DIR", "")
	t.Setenv("BEADS_TEST_IGNORE_REPO_CONFIG", "1")
	config.ResetForTesting()
	t.Cleanup(func() { config.ResetForTesting() })
	if err := config.Initialize(); err != nil {
		t.Fatalf("config.Initialize: %v", err)
	}

	t.Run("no beads dir is a no-op", func(t *testing.T) {
		stderr := captureHookStderr(t, func() { syncBeadsOnPush() })
		if stderr != "" {
			t.Fatalf("expected silent no-op without a beads dir, got stderr: %q", stderr)
		}
	})

	t.Run("no remote is a no-op even with a beads dir", func(t *testing.T) {
		beadsDir := filepath.Join(".", ".beads")
		if err := os.MkdirAll(beadsDir, 0o755); err != nil {
			t.Fatal(err)
		}
		config.Set("sync.remote", "")
		config.Set("sync.git-remote", "")
		config.Set("dolt.local-only", false)
		stderr := captureHookStderr(t, func() { syncBeadsOnPush() })
		if stderr != "" {
			t.Fatalf("expected silent no-op without a sync remote, got stderr: %q", stderr)
		}
	})
}
