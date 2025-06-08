package matcher

import "strings"

// Match reports whether name satisfies pattern using common CLI semantics
// adopted across the project.
func Match(pattern, name string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == "" {
		return false
	}
	return strings.HasPrefix(name, pattern)
}
