package fix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steveyegge/beads/internal/configfile"
)

// writeCredsFile writes an INI-style credentials file keyed [host:port],
// the format LookupCredentialsPassword parses.
func writeCredsFile(t *testing.T, sections map[string]string) string {
	t.Helper()
	var b strings.Builder
	for key, pass := range sections {
		b.WriteString("[" + key + "]\npassword = " + pass + "\n")
	}
	path := filepath.Join(t.TempDir(), "credentials")
	if err := os.WriteFile(path, []byte(b.String()), 0600); err != nil {
		t.Fatalf("write credentials file: %v", err)
	}
	return path
}

// TestFixDBDSN_PasswordResolvedAgainstDialedPort covers the shared-server
// credentials miss (randlee/beads#11, upstream #5573/#5517): metadata omits
// dolt_server_port, so the deprecated GetDoltServerPassword() keys the
// credentials lookup on DefaultDoltServerPort (3307) while openFixDB dials
// the DefaultConfig-resolved port (here 3308 via the port file). The DSN
// must carry the password stored under the DIALED port, not empty.
func TestFixDBDSN_PasswordResolvedAgainstDialedPort(t *testing.T) {
	beadsDir := t.TempDir()
	// DefaultConfig resolves the port from .beads/dolt-server.port.
	if err := os.WriteFile(filepath.Join(beadsDir, "dolt-server.port"), []byte("3308\n"), 0644); err != nil {
		t.Fatalf("write port file: %v", err)
	}

	// Credentials exist ONLY for the dialed port 3308 — the fleet shape that
	// reproduced the auth-fail storm (no [127.0.0.1:3307] entry).
	creds := writeCredsFile(t, map[string]string{"127.0.0.1:3308": "s3cret"})
	t.Setenv("BEADS_CREDENTIALS_FILE", creds)
	t.Setenv("BEADS_DOLT_PASSWORD", "") // never take the env shortcut

	// metadata.json shape: dolt_server_port OMITTED (zero value), so the
	// deprecated getter falls back to 3307.
	cfg := &configfile.Config{
		DoltServerHost: "127.0.0.1",
		DoltServerUser: "skillrx",
		DoltDatabase:   "beads",
	}

	// Sanity: the deprecated getter DOES miss (this is the bug being fixed).
	if got := cfg.GetDoltServerPassword(); got != "" {
		t.Fatalf("precondition broken: deprecated getter returned %q, want empty (3307 miss)", got)
	}

	dsn := fixDBDSN(beadsDir, cfg)

	if !strings.Contains(dsn, "s3cret") {
		t.Errorf("DSN missing the credential for the dialed port 3308:\n%s", dsn)
	}
	if !strings.Contains(dsn, "tcp(127.0.0.1:3308)") {
		t.Errorf("DSN does not dial the resolved port 3308:\n%s", dsn)
	}
	if !strings.Contains(dsn, "skillrx") {
		t.Errorf("DSN missing user:\n%s", dsn)
	}
}

// TestFixDBDSN_FallsBackToConfigPort locks the no-regression side: when
// metadata DOES carry dolt_server_port and credentials are keyed on it, the
// resolved port equals the config port and the password still lands.
func TestFixDBDSN_FallsBackToConfigPort(t *testing.T) {
	beadsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(beadsDir, "dolt-server.port"), []byte("3309\n"), 0644); err != nil {
		t.Fatalf("write port file: %v", err)
	}
	creds := writeCredsFile(t, map[string]string{"127.0.0.1:3309": "p@ss"})
	t.Setenv("BEADS_CREDENTIALS_FILE", creds)
	t.Setenv("BEADS_DOLT_PASSWORD", "")

	cfg := &configfile.Config{
		DoltServerHost: "127.0.0.1",
		DoltServerPort: 3309,
		DoltServerUser: "root",
		DoltDatabase:   "beads",
	}

	dsn := fixDBDSN(beadsDir, cfg)
	if !strings.Contains(dsn, "p@ss") {
		t.Errorf("DSN missing password when config port matches credentials:\n%s", dsn)
	}
}
