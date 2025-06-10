package mcp

import (
	"context"

	"github.com/viant/jsonrpc/transport"
	protocolclient "github.com/viant/mcp-protocol/client"
	"github.com/viant/mcp-protocol/logger"
	serverproto "github.com/viant/mcp-protocol/server"
)

// NewHandler returns an Server implementer that exposes the already-built
// shared tool registry. Every incoming connection therefore reuses the same
// Registry instance – tools are registered once during Service bootstrap
// rather than on each connection.
func (s *Service) NewHandler(ctx context.Context, notifier transport.Notifier, l logger.Logger, cli protocolclient.Operations) (serverproto.Handler, error) {
	impl := serverproto.NewDefaultHandler(notifier, l, cli)
	for _, tool := range s.Tools() {
		impl.Registry.ToolRegistry.Put(tool.Metadata.Name, tool)
	}
	return impl, nil
}
