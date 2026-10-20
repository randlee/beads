package main

// Regression tests for randlee/beads#20: `--var` substitution left padded
// placeholders `{{ key }}` literal while substituting `{{key}}`, because
// variablePattern required the braces to hug the identifier and
// substituteVariables sliced the name by fixed offsets. Both now accept and
// strip optional inner whitespace.

import (
	"reflect"
	"testing"
)

func TestSubstituteVariables_WhitespaceForms(t *testing.T) {
	vars := map[string]string{"slug": "ZZZ", "gate_repo": "acme/x"}

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"no-space", "t {{slug}}", "t ZZZ"},
		{"single-space", "t {{ slug }}", "t ZZZ"},
		{"tabs-and-multi-space", "t {{\t slug \t}}", "t ZZZ"},
		{"underscore-padded", "repo={{ gate_repo }}!", "repo=acme/x!"},
		{"mixed forms", "{{slug}}/{{ slug }}/{{  slug  }}", "ZZZ/ZZZ/ZZZ"},
		{"unknown var left literal (padded)", "x {{ nope }} y", "x {{ nope }} y"},
		{"unknown var left literal (tight)", "x {{nope}} y", "x {{nope}} y"},
		{"empty braces untouched", "x {{}} y", "x {{}} y"},
		{"invalid identifier untouched", "x {{ 1bad }} y", "x {{ 1bad }} y"},
		{"no vars in text", "plain text", "plain text"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := substituteVariables(tc.in, vars); got != tc.want {
				t.Errorf("substituteVariables(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestExtractVariables_WhitespaceForms(t *testing.T) {
	// extractVariables feeds the missing-var hints; padded placeholders must
	// be recognized as the same variable as tight ones (deduped).
	got := extractVariables("{{slug}} then {{ slug }} then {{gate_repo}}")
	want := []string{"slug", "gate_repo"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("extractVariables = %v, want %v", got, want)
	}
}
