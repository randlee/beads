package main

// Regression test for randlee/beads#13: `bd dolt status` reported "not running"
// for a launchd/systemd-managed local Dolt server in a dolt_mode=server
// workspace, because the PID-file path (doltserver.IsRunning) looks for a
// workspace-local PID file that an externally-managed server never writes —
// while CRUD worked fine. The fix falls back to a reachability probe and, if
// something is serving, describes it via the external/SQL-probe path.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/steveyegge/beads/internal/doltserver"
)

// writePortFile plants the runtime port file so DefaultConfig(dir).Port
// resolves to a known value without touching shared-server state.
func writePortFile(t *testing.T, dir string, port string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, doltserver.PortFileName), []byte(port+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLocalStatusNeedsExternalFallback(t *testing.T) {
	// Keep shared-server mode off so DefaultConfig resolves from our planted
	// port file rather than the fleet's shared dir.
	t.Setenv("BEADS_DOLT_SHARED_SERVER", "")
	t.Setenv("BEADS_DOLT_SERVER_PORT", "")
	t.Setenv("BEADS_DOLT_PORT", "")

	srvDir := t.TempDir()
	writePortFile(t, srvDir, "3308")

	reachable := func(host string, port int) bool { return port == 3308 }
	unreachable := func(host string, port int) bool { return false }

	cases := []struct {
		name  string
		state *doltserver.State
		probe serverProbe
		want  bool
	}{
		{
			name:  "bd-managed running server: no fallback (PID path is authoritative)",
			state: &doltserver.State{Running: true, PID: 1234, Port: 3308},
			probe: reachable,
			want:  false,
		},
		{
			name:  "no PID file but endpoint reachable: fall back to external (the #13 case)",
			state: &doltserver.State{Running: false},
			probe: reachable,
			want:  true,
		},
		{
			name:  "nil state but endpoint reachable: fall back",
			state: nil,
			probe: reachable,
			want:  true,
		},
		{
			name:  "no PID file and endpoint unreachable: genuinely down, no fallback",
			state: &doltserver.State{Running: false},
			probe: unreachable,
			want:  false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := localStatusNeedsExternalFallback(tc.state, srvDir, "127.0.0.1", tc.probe)
			if got != tc.want {
				t.Errorf("localStatusNeedsExternalFallback() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestLocalStatusNeedsExternalFallback_NoPort ensures a workspace with no
// resolvable port never triggers a probe (avoids dialing port 0).
func TestLocalStatusNeedsExternalFallback_NoPort(t *testing.T) {
	t.Setenv("BEADS_DOLT_SHARED_SERVER", "")
	t.Setenv("BEADS_DOLT_SERVER_PORT", "")
	t.Setenv("BEADS_DOLT_PORT", "")

	emptyDir := t.TempDir() // no port file planted
	probeCalled := false
	probe := func(host string, port int) bool { probeCalled = true; return true }

	if got := localStatusNeedsExternalFallback(&doltserver.State{Running: false}, emptyDir, "127.0.0.1", probe); got {
		t.Error("expected no fallback when the port cannot be resolved")
	}
	if probeCalled {
		t.Error("probe must not be called when port <= 0")
	}
}
