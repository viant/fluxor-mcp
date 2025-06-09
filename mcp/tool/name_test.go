package tool

import "testing"

func TestCanonical(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"system_exec-execute", "system_exec-execute"},
		{"system/exec.execute", "system_exec-execute"},
		{"system/exec-execute", "system_exec-execute"},
		{"system/exec/execute", "system_exec-execute"},
	}

	for i, tc := range cases {
		if got := Canonical(tc.in); got != tc.out {
			t.Fatalf("case %d: Canonical(%q) = %q, want %q", i, tc.in, got, tc.out)
		}
	}
}
