//go:build cgo

package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestProxiedServerMolPrefixOverride verifies the same workspace-prefix minting
// behavior over the proxied dolt sql-server (the team's production mode): the
// proxied-server pour/wisp front doors carry the same PrefixOverride.
func TestProxiedServerMolPrefixOverride(t *testing.T) {
	requireSharedProxiedServer(t)
	t.Parallel()
	bd := buildEmbeddedBD(t)

	t.Run("pour_mints_workspace_mol_prefix", func(t *testing.T) {
		t.Parallel()
		p := newSharedProxiedProject(t, bd, "mpou")
		setWorkspaceIssuePrefix(t, p.dir, "ap")
		proto := bdProxiedCreate(t, bd, p.dir, "Feature proto", "--type", "epic", "--label", "template")
		bdProxiedCreate(t, bd, p.dir, "Implement", "--type", "task", "--parent", proto.ID)

		out, err := bdProxiedRun(t, bd, p.dir, "mol", "pour", proto.ID, "--json")
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
		t.Parallel()
		p := newSharedProxiedProject(t, bd, "mpw")
		setWorkspaceIssuePrefix(t, p.dir, "ap")
		proto := bdProxiedCreate(t, bd, p.dir, "Wisp proto", "--type", "epic", "--label", "template")

		out, err := bdProxiedRun(t, bd, p.dir, "mol", "wisp", proto.ID, "--json")
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
}
