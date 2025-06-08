package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestServiceMatchTools verifies that the MatchTools helper applies the same
// pattern-matching semantics as resolveBuiltinServices (see builtins.go).
func TestServiceMatchTools(t *testing.T) {
	ctx := context.Background()

	svc, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	if err = svc.Start(ctx); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}

	// 1. '*' should return the full registry.
	all := svc.Tools()
	star := svc.MatchTools("*")
	assert.EqualValues(t, len(all), len(star))

	if len(all) == 0 {
		t.Skip("no tools registered – nothing left to test")
	}

	// Take an arbitrary tool to derive service prefix and exact name.
	any := all[0].Metadata.Name // e.g. "system_storage-clean"

	// Build prefix pattern in slash notation (service/).
	servicePart := any
	if idx := strings.LastIndex(any, "-"); idx != -1 {
		servicePart = any[:idx]
	}
	prefixPattern := strings.ReplaceAll(servicePart, "_", "/") + "/"

	pref := svc.MatchTools(prefixPattern)
	assert.GreaterOrEqual(t, len(pref), 1)

	// The selected tool must be part of the prefix result set.
	var found bool
	for _, te := range pref {
		if te.Metadata.Name == any {
			found = true
			break
		}
	}
	assert.True(t, found, "expected tool %s to match prefix %s", any, prefixPattern)

	// 3. Exact match should return a single entry.
	exact := svc.MatchTools(any)
	assert.EqualValues(t, 1, len(exact))
	assert.EqualValues(t, any, exact[0].Metadata.Name)
}
