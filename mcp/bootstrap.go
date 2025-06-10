package mcp

import (
	"context"
	"fmt"

	"github.com/viant/fluxor"
	"github.com/viant/fluxor-mcp/mcp/clientaction"
	"github.com/viant/fluxor-mcp/mcp/config"
)

// init is the main bootstrap routine invoked by proxy.go once all options
// have been applied. Its sole responsibility is to orchestrate the individual
// preparation steps so that the logic stays easy to read and to maintain.
func (s *Service) init(ctx context.Context) error {
	s.initDefaults()

	// Validate configuration early to fail fast when possible.
	if err := s.config.Validate(); err != nil {
		return err
	}

	s.initWorkflowService()

	// Register external Server tools – turn them into dynamic Fluxor services so
	// they can be consumed like native actions.
	if err := s.registerExternalActions(ctx); err != nil {
		return fmt.Errorf("register externals: %w", err)
	}

	// Auto-start runtime so that callers get a ready-to-use instance without
	// requiring an additional Start() call.
	return s.Workflow.Runtime.Start(ctx)
}

// initDefaults applies fall-back values for optional dependencies that were
// not supplied through options.
func (s *Service) initDefaults() {
	if s.config == nil {
		s.config = &config.Config{}
	}

	if len(s.config.Builtins) == 0 { //add all buildin fluxor action
		s.config.Builtins = append(s.config.Builtins, "*")
	}
	// Further defaults can be added here later without modifying callers.
}

// initWorkflowService assembles the list of Fluxor options, instantiates the
// engine and stores convenience shortcuts for backwards compatibility.
func (s *Service) initWorkflowService() {
	// Start with options coming from the configuration.
	opts := append([]fluxor.Option{}, s.config.Options...)

	if len(s.config.ExtensionTypes) > 0 {
		opts = append(opts, fluxor.WithExtensionTypes(s.config.ExtensionTypes...))
	}

	if len(s.config.Extensions) > 0 {
		opts = append(opts, fluxor.WithExtensionServices(s.config.Extensions...))
	}

	// --------------------------------------------------------------
	// Built-in action auto-loading based on config patterns
	// --------------------------------------------------------------
	if len(s.config.Builtins) > 0 {
		for _, svc := range resolveBuiltinServices(s.config.Builtins) {
			s.Workflow.Extensions = append(s.Workflow.Extensions, svc)
		}
	}

	// Always expose MCP clientHandler operations that this process implements.
	mcpClientSvc := clientaction.New(s.ClientHandler())
	if len(mcpClientSvc.Methods()) > 0 {
		s.Workflow.Extensions = append(s.Workflow.Extensions, mcpClientSvc)
	}

	if len(s.Workflow.Extensions) > 0 {
		opts = append(opts, fluxor.WithExtensionServices(s.Workflow.Extensions...))
	}

	// Finally append any additional Workflow options passed through WithWorkflowOptions
	// to give callers the chance to override defaults where appropriate.
	opts = append(opts, s.Workflow.Options...)

	s.Workflow.Service = fluxor.New(opts...)
	s.Workflow.Runtime = s.Workflow.Service.Runtime()

}
