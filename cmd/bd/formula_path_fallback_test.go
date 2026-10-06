package main

// Regression tests for the formula-resolver path/name inconsistency
// (sc-compose#615): `bd cook <arg>` accepted a registry name OR a file path
// (bd-hp8g), but the pour/wisp/bond/seed resolver
// (resolveAndCookFormulaWithVars) accepted only a registry name — the two
// near-duplicate resolvers had drifted. Both now share
// loadFormulaByNameOrPath: registry lookup first, file-path fallback second.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steveyegge/beads/internal/formula"
)

const pathFallbackFormulaTOML = `formula = "path-fallback-resolver-test"
version = 1
type = "workflow"

[[steps]]
id = "only-step"
title = "Step reachable by path only"
`

// writePathFallbackFormula writes the fixture OUTSIDE any registry dir
// (no .beads/formulas, no GT_ROOT) so name resolution cannot find it and
// only the ParseFile fallback can.
func writePathFallbackFormula(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "path-fallback-resolver-test.formula.toml")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(pathFallbackFormulaTOML), 0o600); err != nil {
		t.Fatalf("write formula fixture: %v", err)
	}
	return p
}

func TestLoadFormulaByNameOrPath_PathFallback(t *testing.T) {
	path := writePathFallbackFormula(t)
	parser := formula.NewParser(t.TempDir()) // empty search paths: no registry hit possible

	// A registry-style name must still fail (no silent behavior change).
	if _, err := loadFormulaByNameOrPath(parser, "path-fallback-resolver-test"); err == nil {
		t.Fatal("expected name lookup to fail with empty search paths")
	}

	// The file path resolves via the fallback.
	f, err := loadFormulaByNameOrPath(parser, path)
	if err != nil {
		t.Fatalf("path fallback failed: %v", err)
	}
	if f.Formula != "path-fallback-resolver-test" {
		t.Errorf("formula name = %q, want path-fallback-resolver-test", f.Formula)
	}
	if f.Source != path && !strings.HasSuffix(f.Source, "path-fallback-resolver-test.formula.toml") {
		t.Errorf("formula Source = %q, want the fixture path", f.Source)
	}
}

func TestLoadFormulaByNameOrPath_RegistryWins(t *testing.T) {
	// Same-name formula in BOTH registry and a stray file: the registry
	// copy must win (name lookup is tried first, preserving cook's
	// bd-hp8g precedence).
	gtRoot := t.TempDir()
	registryDir := filepath.Join(gtRoot, ".beads", "formulas")
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	registryTOML := strings.Replace(pathFallbackFormulaTOML,
		`title = "Step reachable by path only"`,
		`title = "Registry copy wins"`, 1)
	if err := os.WriteFile(filepath.Join(registryDir, "path-fallback-resolver-test.formula.toml"),
		[]byte(registryTOML), 0o600); err != nil {
		t.Fatal(err)
	}

	parser := formula.NewParser(registryDir)
	f, err := loadFormulaByNameOrPath(parser, "path-fallback-resolver-test")
	if err != nil {
		t.Fatalf("registry lookup failed: %v", err)
	}
	if len(f.Steps) != 1 || !strings.Contains(f.Steps[0].Title, "Registry copy wins") {
		t.Errorf("expected the registry copy to win; got steps %+v", f.Steps)
	}
}

// TestResolveAndCookFormulaWithVars_AcceptsPath is the end-to-end contract
// pour/wisp/bond/seed rely on: a bare file path outside any registry cooks.
func TestResolveAndCookFormulaWithVars_AcceptsPath(t *testing.T) {
	path := writePathFallbackFormula(t)

	sg, err := resolveAndCookFormulaWithVars(path, []string{t.TempDir()}, nil)
	if err != nil {
		t.Fatalf("resolveAndCookFormulaWithVars(%q) failed: %v", path, err)
	}
	if sg == nil || sg.Root == nil {
		t.Fatal("expected a cooked subgraph")
	}

	// And a non-formula, non-name argument still fails cleanly (pour.go then
	// falls back to proto-ID resolution — unchanged).
	if _, err := resolveAndCookFormulaWithVars(filepath.Join(t.TempDir(), "missing.formula.toml"), nil, nil); err == nil {
		t.Fatal("expected failure for a nonexistent path")
	}
}
