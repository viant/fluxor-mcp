package mcp

import (
	"context"

	"github.com/viant/jsonrpc"
	protoclient "github.com/viant/mcp-protocol/client"
	mcpschema "github.com/viant/mcp-protocol/schema"
)

// defaultClient provides no-op implementations for all clientHandler-side RPC
// operations so that outgoing Server clients can be instantiated without the
// caller having to care about server-initiated callbacks.
type defaultClient struct {
	implements map[string]bool
}

func (d *defaultClient) Init(ctx context.Context, capabilities *mcpschema.ClientCapabilities) {
	if len(d.implements) == 0 {
		d.implements = make(map[string]bool)
	}
	if capabilities.Elicitation != nil {
		d.implements[mcpschema.MethodElicitationCreate] = true
	}
	if capabilities.Roots != nil {
		d.implements[mcpschema.MethodRootsList] = true
	}
	if capabilities.Sampling != nil {
		d.implements[mcpschema.MethodSamplingCreateMessage] = true
	}
}
func (*defaultClient) LastRequestID() jsonrpc.RequestId {
	return 0
}

func (*defaultClient) NextRequestID() jsonrpc.RequestId {
	return 0
}

func (*defaultClient) OnNotification(context.Context, *jsonrpc.Notification) {}
func (d *defaultClient) Implements(method string) bool {
	if len(d.implements) == 0 {
		d.implements = make(map[string]bool)
	}
	return d.implements[method]
}

func (*defaultClient) ListRoots(context.Context, *jsonrpc.TypedRequest[*mcpschema.ListRootsRequest]) (*mcpschema.ListRootsResult, *jsonrpc.Error) {
	return nil, jsonrpc.NewError(jsonrpc.MethodNotFound, "not implemented", nil)
}
func (*defaultClient) CreateMessage(context.Context, *jsonrpc.TypedRequest[*mcpschema.CreateMessageRequest]) (*mcpschema.CreateMessageResult, *jsonrpc.Error) {
	return nil, jsonrpc.NewError(jsonrpc.MethodNotFound, "not implemented", nil)
}
func (*defaultClient) Elicit(context.Context, *jsonrpc.TypedRequest[*mcpschema.ElicitRequest]) (*mcpschema.ElicitResult, *jsonrpc.Error) {
	return nil, jsonrpc.NewError(jsonrpc.MethodNotFound, "not implemented", nil)
}

func (*defaultClient) Notify(ctx context.Context, notification *jsonrpc.Notification) error {
	return jsonrpc.NewError(jsonrpc.MethodNotFound, "not implemented", nil)
}

func newMcpClient() protoclient.Handler { return &defaultClient{} }
