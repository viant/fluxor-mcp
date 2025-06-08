package matcher

import "testing"

func TestMatch(t *testing.T) {
	var testCases = []struct {
		pattern   string
		candidate string
		matched   bool
	}{
		{"*", "anything", true},
		{"", "anything", false},

		// Exact matches
		{"system/exec", "system/exec", true},
		{"system_exec", "system_exec", true},
		{"system/exec", "system/exec2", true},

		// Prefix matches with "/"
		{"system/", "system/exec", true},
		{"sys/", "system/exec", false},

		// Prefix matches with "_"
		{"system_", "system_exec", true},
		{"sys_", "system_exec", false},
	}

	for i, tc := range testCases {
		if got := Match(tc.pattern, tc.candidate); got != tc.matched {
			t.Fatalf("[%d] Match(%q, %q) = %v; expected %v", i, tc.pattern, tc.candidate, got, tc.matched)
		}
	}
}
