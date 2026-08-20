package main

import "strings"

// normalizePrefixFilter canonicalizes a user-supplied or store-sourced prefix
// into the trailing-dash form the IDPrefix filter expects. The stored
// issue_prefix is normalized without a trailing dash (bd init strips it), but
// the SQL filter matches `id LIKE '<prefix>%'` where the separator is
// significant: `skillrx-` must not also match `skillrx2-`. This appends the
// dash when absent so a boundary between `skillrx-` and `skillrx2-` holds.
// Empty input stays empty.
func normalizePrefixFilter(prefix string) string {
	prefix = strings.TrimRight(prefix, "-")
	if prefix == "" {
		return ""
	}
	return prefix + "-"
}
