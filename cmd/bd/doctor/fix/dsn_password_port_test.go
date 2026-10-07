package fix

// Regression test for randlee/beads#11: 'bd doctor --fix' failed with
// "Access denied for user ..." on the shared dolt server while CRUD and
// read-only doctor checks connected fine. Root cause: openFixDB resolved the
// credentials-file password with cfg.GetDoltServerPassword() — keyed to the
// CONFIG port (metadata.json / env / default 3307) — while dialing the
// RUNTIME port (doltserver.DefaultConfig: env > port file > config.yaml).
// The credentials file is sectioned [host:port], so any divergence between
// the two ports (the shared server's default 3308 vs config default 3307)
// misses the section and silently yields an EMPTY password. The store path
// (internal/storage/dolt/open.go) and doctor read path (cmd/bd/doctor/dolt.go)
// both use GetDoltServerPasswordForPort(runtimePort) since bd-h5k7; the fix
// path was missed. fixDBDSN now pairs password lookup with the dialed port.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/steveyegge/beads/internal/configfile"
)

func TestFixDBDSN_PasswordMatchesRuntimePort(t *testing.T) {
	beadsDir := t.TempDir()

	// Runtime port source: the local port file (gitignored, authoritative at
	// runtime) says 3308 — the shared-server default.
	if err := os.WriteFile(filepath.Join(beadsDir, "dolt-server.port"), []byte("3308\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Credentials file is keyed to the RUNTIME port only — exactly the fleet
	// shape that broke: there is NO [127.0.0.1:3307] section.
	credFile := filepath.Join(t.TempDir(), "credentials")
	cred := "[127.0.0.1:3308]\npassword = SECRET-runtime\n"
	if err := os.WriteFile(credFile, []byte(cred), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BEADS_CREDENTIALS_FILE", credFile)

	// Keep env overrides out of the picture: BEADS_DOLT_PASSWORD would mask
	// the credentials lookup, BEADS_DOLT_SERVER_PORT would collapse the
	// config/runtime port distinction.
	t.Setenv("BEADS_DOLT_PASSWORD", "")
	t.Setenv("BEADS_DOLT_SERVER_PORT", "")
	t.Setenv("BEADS_DOLT_PORT", "")
	t.Setenv("BEADS_DOLT_SHARED_SERVER", "")

	cfg := &configfile.Config{
		Database:       "dolt",
		DoltMode:       configfile.DoltModeServer,
		DoltServerHost: "127.0.0.1",
		DoltServerUser: "skillrx",
		DoltServerPort: 3307, // config port DIFFERS from runtime port 3308
		DoltDatabase:   "hendrix",
	}

	dsn := fixDBDSN(beadsDir, cfg)
	parsed, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("fixDBDSN produced an unparseable DSN %q: %v", dsn, err)
	}

	// The DSN must dial the RUNTIME port...
	if !strings.HasSuffix(parsed.Addr, ":3308") {
		t.Errorf("DSN addr = %q, want runtime port :3308", parsed.Addr)
	}
	// ...and carry the password from the [host:3308] credentials section.
	// Pre-fix behavior: lookup keyed [127.0.0.1:3307] → section miss → "".
	if parsed.Passwd != "SECRET-runtime" {
		t.Errorf("DSN password = %q, want SECRET-runtime (credentials lookup must use the runtime port)", parsed.Passwd)
	}
	if parsed.User != "skillrx" || parsed.DBName != "hendrix" {
		t.Errorf("DSN user/db = %q/%q, want skillrx/hendrix", parsed.User, parsed.DBName)
	}
}

func TestFixDBDSN_EnvPasswordStillWins(t *testing.T) {
	// BEADS_DOLT_PASSWORD takes precedence in GetDoltServerPasswordForPort —
	// the fix must not change that contract.
	beadsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(beadsDir, "dolt-server.port"), []byte("3308\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	credFile := filepath.Join(t.TempDir(), "credentials")
	if err := os.WriteFile(credFile, []byte("[127.0.0.1:3308]\npassword = from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BEADS_CREDENTIALS_FILE", credFile)
	t.Setenv("BEADS_DOLT_PASSWORD", "from-env")
	t.Setenv("BEADS_DOLT_SERVER_PORT", "")
	t.Setenv("BEADS_DOLT_PORT", "")
	t.Setenv("BEADS_DOLT_SHARED_SERVER", "")

	cfg := &configfile.Config{
		Database:       "dolt",
		DoltMode:       configfile.DoltModeServer,
		DoltServerHost: "127.0.0.1",
		DoltServerUser: "u",
		DoltServerPort: 3307,
		DoltDatabase:   "db",
	}
	parsed, err := mysql.ParseDSN(fixDBDSN(beadsDir, cfg))
	if err != nil {
		t.Fatalf("parse DSN: %v", err)
	}
	if parsed.Passwd != "from-env" {
		t.Errorf("DSN password = %q, want from-env (env precedence preserved)", parsed.Passwd)
	}
}
