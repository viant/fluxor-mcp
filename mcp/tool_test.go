package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestServiceTools ensures that the service exposes a tool entry for every
// action method that is available via the workflow service. Historically the
// lookup helper failed to return the built entry which caused list-tools to
// print an empty result. This regression test protects against similar issues
// in the future.
func TestServiceTools(t *testing.T) {
	ctx := context.Background()

	svc, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	if err = svc.Start(ctx); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}

	// Calculate the number of action methods exposed by the workflow service.
	var expected int
	actions := svc.WorkflowService().Actions()
	for _, svcName := range actions.Services() {
		expected += len(actions.Lookup(svcName).Methods())
	}

	// Fetch tool registry via the public helper and compare counts.
	tools := svc.Tools()

	assert.EqualValues(t, expected, len(tools))

	// Additionally, verify that each tool can be resolved individually using
	// LookupTool to guard the happy path.
	for _, te := range tools {
		entry, err := svc.LookupTool(te.Metadata.Name)
		if assert.NoError(t, err, "LookupTool(%q) returned error", te.Metadata.Name) {
			assert.EqualValues(t, te.Metadata.Name, entry.Metadata.Name)
		}
	}
}
