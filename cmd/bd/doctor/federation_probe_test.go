package doctor

// Regression tests for randlee/beads#26: CheckFederationRemotesAPI gated on
// doltserver.IsRunning (workspace PID file), which a launchd/systemd-managed
// server never writes — so the remotes check skipped with "N/A (server not
// running)" (or falsely warned "Server not running (N peers configured)")
// while the server was up and CRUD worked. Same probe-gap family as
// randlee/beads#13 (bd dolt status). The fix falls back to an endpoint probe.
//
// Not cgo-gated (unlike federation_test.go): these exercise pure decision
// helpers with injected state, so they run in the thin/pure-Go build too.

import (
	"testing"

	"github.com/steveyegge/beads/internal/doltserver"
)

func TestServerRunningOrProbeable(t *testing.T) {
	cases := []struct {
		name      string
		state     *doltserver.State
		probeSays bool
		want      bool
	}{
		{
			name:  "PID-tracked running server: probe not consulted",
			state: &doltserver.State{Running: true, PID: 4242, Port: 3308},
			want:  true,
		},
		{
			name:      "no PID file, endpoint serving (launchd shape, #26): probe decides",
			state:     &doltserver.State{Running: false},
			probeSays: true,
			want:      true,
		},
		{
			name:      "nil state (PID read error), endpoint serving: probe decides",
			state:     nil,
			probeSays: true,
			want:      true,
		},
		{
			name:      "no PID file and endpoint down: genuinely not running",
			state:     &doltserver.State{Running: false},
			probeSays: false,
			want:      false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			probe := func() bool { calls++; return tc.probeSays }
			got := serverRunningOrProbeable(tc.state, probe)
			if got != tc.want {
				t.Errorf("serverRunningOrProbeable() = %v, want %v", got, tc.want)
			}
			// The PID-tracked case must short-circuit without dialing.
			if tc.state != nil && tc.state.Running && calls != 0 {
				t.Errorf("probe called %d times for a PID-tracked running server; want 0 (no needless dial)", calls)
			}
		})
	}
}
