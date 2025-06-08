package matcher

import "strings"

// Match reports whether name satisfies pattern using common CLI semantics
// adopted across the project.
//
// Rules:
//  1. "*" or empty pattern – always true.
//  2. Prefix           – pattern ending with "/" or "_" matches any name that
//     starts with the pattern prefix (separator removed). Example:
//     pattern "system/"  matches "system/exec"   (built-in actions)
//     pattern "system_"  matches "system_exec"   (tool names)
//  3. Exact            – every other pattern must equal the full name.
//
// The function is intentionally minimal – callers are responsible for any
// additional normalisation (like replacing "/" with "_" for tool names).
func Match(pattern, name string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}

	// Prefix mode when pattern ends with a known separator. Keep the suffix so
	// that the boundary is significant (e.g. "sys/" should NOT match
	// "system/exec").
	if strings.HasSuffix(pattern, "/") || strings.HasSuffix(pattern, "_") || strings.HasSuffix(pattern, "-") {
		return strings.HasPrefix(name, pattern)
	}

	// Exact match otherwise.
	return name == pattern
}
