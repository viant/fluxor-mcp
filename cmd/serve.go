package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/viant/mcp"
)

// ServeCmd launches an MCP server that exposes the locally registered tools.
// The server configuration (port, transport, auth, …) is taken from the same
// config file that the service uses.  If no transport/port is configured a
// sensible default (stdio + :5000) is assumed by the underlying library.
type ServeCmd struct{}

func (c *ServeCmd) Execute(_ []string) error {
	svc, err := serviceSingleton()
	if err != nil {
		return err
	}

	cfg := svc.Config() // we will add accessor for config in service
	var srvOpts *mcp.ServerOptions
	if cfg != nil {
		srvOpts = cfg.Server
	}

	mcpServer, err := mcp.NewServer(svc.NewHandler, srvOpts)
	if err != nil {
		return err
	}

	httpSrv := mcpServer.HTTP(context.Background(), "")
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err.Error() != "http: Server closed" {
			log.Fatalf("http server: %v", err)
		}
	}()

	fmt.Printf("MCP server listening on %s\n", httpSrv.Addr)

	// Wait for SIGINT/SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	fmt.Println("shutting down…")
	return httpSrv.Close()
}
