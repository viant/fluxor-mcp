package tool_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	coretool "github.com/viant/fluxor-mcp/mcp/tool"

	"github.com/viant/jsonrpc"
	transport "github.com/viant/jsonrpc/transport"
	mcp "github.com/viant/mcp"
	protocolclient "github.com/viant/mcp-protocol/client"
	mcpLogger "github.com/viant/mcp-protocol/logger"
	mcpschema "github.com/viant/mcp-protocol/schema"
	protoserver "github.com/viant/mcp-protocol/server"
	mcpclient "github.com/viant/mcp/client"
)

// echoHandler is a minimal Server tool that echos back the provided message.
func echoHandler(_ context.Context, req *mcpschema.CallToolRequest) (*mcpschema.CallToolResult, *jsonrpc.Error) {
	msg, _ := req.Params.Arguments["message"].(string)
	return &mcpschema.CallToolResult{Content: []mcpschema.CallToolResultContentElem{{
		Type: "text",
		Text: msg,
	}}}, nil
}

// emptyHandler returns an empty content array to simulate tools that
// respond with no data.
func emptyHandler(_ context.Context, _ *mcpschema.CallToolRequest) (*mcpschema.CallToolResult, *jsonrpc.Error) {
	return &mcpschema.CallToolResult{Content: nil}, nil
}

// structuredHandler returns only structuredContent.
func structuredHandler(_ context.Context, _ *mcpschema.CallToolRequest) (*mcpschema.CallToolResult, *jsonrpc.Error) {
	return &mcpschema.CallToolResult{StructuredContent: map[string]interface{}{"message": "hello-structured"}}, nil
}

// newTestServer spins up an in-process Server server exposing the echo tool and
// returns a client connected to it.
func newTestServer(t *testing.T) mcpclient.Interface {
	t.Helper()

	// Build an implementer with the echo tool registered.
	newImpl := func(ctx context.Context, _ transport.Notifier, _ mcpLogger.Logger, _ protocolclient.Operations) (protoserver.Handler, error) {
		impl := protoserver.NewDefaultHandler(nil, nil, nil)

		// Define input schema for the echo tool.
		inputSchema := mcpschema.ToolInputSchema{
			Type: "object",
			Properties: map[string]map[string]interface{}{
				"message": {"type": "string"},
			},
			Required: []string{"message"},
		}
		outputSchema := &mcpschema.ToolOutputSchema{
			Type: "object",
			Properties: map[string]map[string]interface{}{
				"message": {"type": "string"},
			},
			Required: []string{"message"},
		}

		impl.RegisterToolWithSchema("echo", "echo message back", inputSchema, outputSchema, echoHandler)

		// Register a tool that returns no content.
		emptyInput := mcpschema.ToolInputSchema{Type: "object"}
		emptyOutput := &mcpschema.ToolOutputSchema{Type: "object"}
		impl.RegisterToolWithSchema("empty", "returns empty content", emptyInput, emptyOutput, emptyHandler)

		// Register a tool that returns only structured content.
		impl.RegisterToolWithSchema("structured", "returns structured content only", emptyInput, emptyOutput, structuredHandler)
		return impl, nil
	}

	srv, err := mcp.NewServer(newImpl, nil)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	return srv.AsClient(context.Background())
}

func TestRemoteToolService_Echo(t *testing.T) {
	ctx := context.Background()
	cli := newTestServer(t)

	svc, err := coretool.NewProxy(ctx, "test", cli)
	if err != nil {
		t.Fatalf("failed to create remote service: %v", err)
	}

	// ------------------------------------------------------------------
	// Signature assertions
	// ------------------------------------------------------------------
	sig := svc.Methods().Lookup("echo")
	if sig == nil {
		t.Fatalf("expected signature for echo tool")
	}

	assert.EqualValues(t, reflect.Struct, sig.Input.Kind())
	field, ok := sig.Input.FieldByName("Message")
	if assert.True(t, ok, "expected Message field in generated struct") {
		assert.EqualValues(t, reflect.String, field.Type.Kind())
	}

	// ------------------------------------------------------------------
	// Execution assertions
	// ------------------------------------------------------------------
	exec, err := svc.Method("echo")
	if err != nil {
		t.Fatalf("Method lookup failed: %v", err)
	}

	var response string
	err = exec(ctx, map[string]interface{}{"message": "hello"}, &response)
	assert.NoError(t, err)
	assert.EqualValues(t, "hello", response)
}

func TestRemoteToolService_EmptyContent_NoPanic(t *testing.T) {
	ctx := context.Background()
	cli := newTestServer(t)

	svc, err := coretool.NewProxy(ctx, "test", cli)
	if err != nil {
		t.Fatalf("failed to create remote service: %v", err)
	}

	exec, err := svc.Method("empty")
	if err != nil {
		t.Fatalf("Method lookup failed: %v", err)
	}

	// Ask for a structured output; with empty content this should return
	// a descriptive error rather than panic.
	var dst struct{ Message string }
	err = exec(ctx, map[string]interface{}{}, &dst)
	if err == nil {
		t.Fatalf("expected error for empty content, got nil")
	}
}

func TestRemoteToolService_StructuredContent(t *testing.T) {
	ctx := context.Background()
	cli := newTestServer(t)

	svc, err := coretool.NewProxy(ctx, "test", cli)
	if err != nil {
		t.Fatalf("failed to create remote service: %v", err)
	}

	exec, err := svc.Method("structured")
	if err != nil {
		t.Fatalf("Method lookup failed: %v", err)
	}

	// Decode into a typed struct
	var dst struct{ Message string }
	err = exec(ctx, map[string]interface{}{}, &dst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dst.Message != "hello-structured" {
		t.Fatalf("unexpected message: %s", dst.Message)
	}
}
