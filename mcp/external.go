package mcp

import (
	"context"
	"fmt"

	"github.com/viant/afs"
	"github.com/viant/fluxor-mcp/mcp/tool"
	"github.com/viant/mcp"
	protocolclient "github.com/viant/mcp-protocol/client"
	"gopkg.in/yaml.v3"
)

// registerExternalActions loads external Server endpoints specified in the
// configuration, introspects the available tools and turns each of them into a
// Fluxor service whose methods proxy the remote calls.
func (s *Service) registerExternalActions(ctx context.Context) error {

	mcpConfigs, err := s.loadMCPClientConfig(ctx)
	if err != nil {
		return err
	}
	if len(mcpConfigs) == 0 {
		return nil // nothing to do – no externals configured
	}
	for _, mcpConfig := range mcpConfigs {

		// Ensure required defaults are applied so that name/version are never empty.
		if err = s.RegisterMcpClientTools(ctx, mcpConfig); err != nil {
			if err = s.mcpErrorHandler(mcpConfig, err); err != nil {
				return err
			}
		}
	}
	return nil
}

// RegisterMcpClientTools register Server clientHandler
func (s *Service) RegisterMcpClientTools(ctx context.Context, mcpConfig *mcp.ClientOptions) error {
	actions := s.Workflow.Service.Actions()
	mcpConfig.Init()

	impl := s.ClientHandler()

	cli, err := mcp.NewClient(impl, mcpConfig)
	if err != nil {
		return fmt.Errorf("create mcp clientHandler %q: %w", mcpConfig.Name, err)
	}

	mcpToolService, err := tool.NewProxy(ctx, mcpConfig.Name, cli)
	if err != nil {
		return fmt.Errorf("load tools for %q: %w", mcpConfig.Name, err)
	}

	if err := actions.Register(mcpToolService); err != nil {
		return err
	}
	return nil
}

func (s *Service) ClientHandler() protocolclient.Handler {
	impl := s.clientHandler
	if impl == nil {
		impl = newMcpClient()
	}
	return impl
}

// loadMCPClientConfig resolves Server clientHandler options either embedded directly in
// the config or referenced via URL.
func (s *Service) loadMCPClientConfig(ctx context.Context) ([]*mcp.ClientOptions, error) {
	if s.config == nil || s.config.MCP == nil {
		return nil, nil
	}

	// Inline options take precedence.
	if len(s.config.MCP.Items) > 0 {
		return s.config.MCP.Items, nil
	}

	if s.config.MCP.URL == "" {
		return nil, nil
	}

	fs := afs.New()
	data, err := fs.DownloadWithURL(ctx, s.config.MCP.URL)
	if err != nil {
		return nil, fmt.Errorf("download externals config %q: %w", s.config.MCP.URL, err)
	}

	var out []*mcp.ClientOptions
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse externals config %q: %w", s.config.MCP.URL, err)
	}
	return out, nil
}
