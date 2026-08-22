//go:build cgo

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setWorkspaceIssuePrefix writes an active issue-prefix into .beads/config.yaml,
// simulating a workspace that declares its own prefix on a shared DB (the exact
// scenario behind feat/create-workspace-prefix and the pour/wisp fix). It
// replaces any commented/active issue-prefix line, or appends one if absent.
func setWorkspaceIssuePrefix(t *testing.T, dir, prefix string) {
	t.Helper()
	path := filepath.Join(dir, ".beads", "config.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config.yaml: %v", err)
	}
	lines := strings.Split(string(b), "\n")
	target := `issue-prefix: "` + prefix + `"`
	found := false
	for i, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		if strings.HasPrefix(trimmed, "# issue-prefix:") || strings.HasPrefix(trimmed, "issue-prefix:") {
			lines[i] = target
			found = true
		}
	}
	if !found {
		lines = append(lines, target)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
}

// TestEmbeddedMolPrefixOverride verifies end-to-end (embedded Dolt, real CLI
// subprocess) that pour/wisp honor the workspace issue-prefix over the DB
// scalar, and that the no-override fallback is unchanged.
func TestEmbeddedMolPrefixOverride(t *testing.T) {
	if os.Getenv("BEADS_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set BEADS_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()
	bd := buildEmbeddedBD(t)

	t.Run("pour_mints_workspace_mol_prefix", func(t *testing.T) {
		dir, _, _ := bdInit(t, bd, "--prefix", "skillrx")
		setWorkspaceIssuePrefix(t, dir, "ap")
		proto := bdCreate(t, bd, dir, "Feature proto", "--type", "epic", "--label", "template")
		bdCreate(t, bd, dir, "Implement", "--type", "task", "--parent", proto.ID)

		out, err := bdRunWithFlockRetry(t, bd, dir, "mol", "pour", proto.ID, "--json")
		if err != nil {
			t.Fatalf("bd mol pour --json: %v\n%s", err, out)
		}
		var got struct {
			NewEpicID string `json:"new_epic_id"`
		}
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, out)
		}
		if !strings.HasPrefix(got.NewEpicID, "ap-mol-") {
			t.Errorf("poured root = %q, want prefix ap-mol-", got.NewEpicID)
		}
	})

	t.Run("wisp_mints_workspace_wisp_prefix", func(t *testing.T) {
		dir, _, _ := bdInit(t, bd, "--prefix", "skillrx")
		setWorkspaceIssuePrefix(t, dir, "ap")
		proto := bdCreate(t, bd, dir, "Wisp proto", "--type", "epic", "--label", "template")

		out, err := bdRunWithFlockRetry(t, bd, dir, "mol", "wisp", proto.ID, "--json")
		if err != nil {
			t.Fatalf("bd mol wisp --json: %v\n%s", err, out)
		}
		var got struct {
			NewEpicID string `json:"new_epic_id"`
		}
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, out)
		}
		if !strings.HasPrefix(got.NewEpicID, "ap-wisp-") {
			t.Errorf("wisp root = %q, want prefix ap-wisp-", got.NewEpicID)
		}
	})

	t.Run("no_workspace_prefix_falls_back_to_db_scalar", func(t *testing.T) {
		dir, _, _ := bdInit(t, bd, "--prefix", "skillrx")
		// No setWorkspaceIssuePrefix: config.yaml keeps the commented default.
		proto := bdCreate(t, bd, dir, "Fallback proto", "--type", "epic", "--label", "template")

		out, err := bdRunWithFlockRetry(t, bd, dir, "mol", "pour", proto.ID, "--json")
		if err != nil {
			t.Fatalf("bd mol pour --json: %v\n%s", err, out)
		}
		var got struct {
			NewEpicID string `json:"new_epic_id"`
		}
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, out)
		}
		if !strings.HasPrefix(got.NewEpicID, "skillrx-mol-") {
			t.Errorf("poured root = %q, want prefix skillrx-mol-", got.NewEpicID)
		}
	})
}
