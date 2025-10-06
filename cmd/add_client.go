package cmd

import (
	"context"
	"fmt"

	mcpconfig "github.com/viant/fluxor-mcp/mcp/config"
	mcp "github.com/viant/mcp"
)

// AddClientCmd dynamically imports tools exposed by a remote MCP service and
// re-registers them locally so that they become regular Fluxor actions.
type AddClientCmd struct {
	Name    string `short:"n" long:"name"    description:"Identifier for the external endpoint"`
	Address string `short:"a" long:"address" description:"HTTP or WS address of the external MCP server"`
	Version string `short:"v" long:"version" description:"Expected protocol version (optional)"`
}

func (c *AddClientCmd) Execute(_ []string) error {
	if c.Name == "" || c.Address == "" {
		return fmt.Errorf("both --name and --address are required")
	}

	svc, err := serviceSingleton()
	if err != nil {
		return err
	}

	opts := &mcpconfig.MCPClient{ClientOptions: &mcp.ClientOptions{
		Name:    c.Name,
		Version: c.Version,
		Transport: mcp.ClientTransport{
			Type:                "sse",
			ClientTransportHTTP: mcp.ClientTransportHTTP{URL: c.Address},
		},
	}}

	if err := svc.RegisterMcpClientTools(context.Background(), opts); err != nil {
		return err
	}
	fmt.Printf("imported tools from %s (%s)\n", c.Name, c.Address)
	return nil
}
