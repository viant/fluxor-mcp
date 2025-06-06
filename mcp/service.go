package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/viant/fluxor"
	"github.com/viant/fluxor-mcp/mcp/config"
	"github.com/viant/fluxor/model/types"
	"github.com/viant/x"

	protocolclient "github.com/viant/mcp-protocol/client"
	mcpschema "github.com/viant/mcp-protocol/schema"
	serverproto "github.com/viant/mcp-protocol/server"
)

// Service bundles configuration, a Fluxor Workflow engine and auxiliary state
// required by the Server adapter. All heavy lifting during instantiation lives in
// bootstrap.go to keep this file focused on the public surface and to avoid
// large, monolithic functions.
type Service struct {
	Workflow
	started int32
	client  protocolclient.Operations
	config  *config.Config

	// guard concurrent modifications.
	mu sync.RWMutex
	// Cached Server tool definitions built from Fluxor actions.
	mcpTools []toolEntry

	// Shared registry instance passed to every Server implementer so that tools
	// are registered once system-wide instead of per connection.
	toolRegistry *serverproto.Registry
}

type Workflow struct {
	Options        []fluxor.Option
	Runtime        *fluxor.Runtime
	Service        *fluxor.Service
	Extensions     []types.Service
	ExtensionTypes []*x.Type `json:"-"`
}

// WorkflowRuntime returns the underlying Fluxor runtime. Prefer this accessor
// over the deprecated Runtime field.
func (s *Service) WorkflowRuntime() *fluxor.Runtime { return s.Workflow.Runtime }

// WorkflowService returns the generated Fluxor service instance that exposes
// all actions. Prefer this accessor over the deprecated Service field.
func (s *Service) WorkflowService() *fluxor.Service { return s.Workflow.Service }

// Config returns the effective configuration instance passed to the service at
// construction time.  Callers must treat the returned object as read-only.
func (s *Service) Config() *config.Config { return s.config }

// ToolNames returns all unique MCP tool names registered on the service.  The
// slice is a copy and therefore safe for callers to modify.
func (s *Service) ToolNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, len(s.mcpTools))
	for i, e := range s.mcpTools {
		names[i] = e.name
	}
	return names
}

// ToolDescriptors returns basic metadata for every tool (name & description).
// The returned slice is detached from internal state and therefore read-only
// for callers.
func (s *Service) ToolDescriptors() []struct{ Name, Description string } {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]struct{ Name, Description string }, len(s.mcpTools))
	for i, e := range s.mcpTools {
		out[i] = struct{ Name, Description string }{e.name, e.description}
	}
	return out
}

// toolEntryByName returns a pointer to the internal entry with the given name
// and a bool indicating presence. Internal helper for CLI inspection.
func (s *Service) toolEntryByName(name string) (*toolEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i, e := range s.mcpTools {
		if e.name == name {
			return &s.mcpTools[i], true
		}
	}
	return nil, false
}

// ToolMetadata returns description and input schema for a named tool when
// present. The second return value is false when the tool does not exist.
func (s *Service) ToolMetadata(name string) (string, interface{}, bool) {
	e, ok := s.toolEntryByName(name)
	if !ok {
		return "", nil, false
	}
	return e.description, e.inputSchema, true
}

// ExecuteTool invokes a registered tool by name with the supplied arguments.
// It converts the generic arguments map into an MCP CallToolRequest and passes
// it to the internally cached handler. The function returns the textual result
// or an error if the tool does not exist or the handler failed.
//
// The helper is intended for in-process callers that want to use the exact same
// execution path as an external MCP client without going through JSON-RPC or
// network transport.
func (s *Service) ExecuteTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	e, ok := s.toolEntryByName(name)
	if !ok {
		return "", fmt.Errorf("tool %q not found", name)
	}

	// Build minimal CallToolRequest expected by the handler.
	req := &mcpschema.CallToolRequest{
		Method: mcpschema.MethodToolsCall,
		Params: mcpschema.CallToolRequestParams{
			Name:      name,
			Arguments: mcpschema.CallToolRequestParamsArguments(args),
		},
	}

	res, jerr := e.handler(ctx, req)
	if jerr != nil {
		return "", fmt.Errorf(jerr.Message)
	}

	if res == nil || len(res.Content) == 0 {
		return "", nil
	}
	// For now, return first textual element or JSON-encode full content when
	// multiple parts are present.
	if len(res.Content) == 1 && res.Content[0].Type == "text" {
		return res.Content[0].Text, nil
	}
	data, err := json.Marshal(res.Content)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Option modifies a service instance before it is initialised. Users can pass
// an arbitrary number of options to New.
type Option func(*Service)

// WithConfig sets a custom configuration instance. When omitted a zero value
// config is assumed.
func WithConfig(cfg *config.Config) Option {
	return func(s *Service) {
		s.config = cfg
	}
}

// WithWorkflowOptions appends additional Fluxor options that will be used when
// the Workflow engine gets instantiated.
func WithWorkflowOptions(opts ...fluxor.Option) Option {
	return func(s *Service) {
		s.Workflow.Options = append(s.Workflow.Options, opts...)
	}
}

// WithExtensions registers custom Fluxor services that should be available in
// addition to those coming from the configuration file.
func WithExtensions(ext ...types.Service) Option {
	return func(s *Service) {
		s.config.Extensions = append(s.config.Extensions, ext...)
	}
}

// WithClient overrides the default stub implementer used for
// outgoing Server client connections to externals.
func WithClient(impl protocolclient.Operations) Option {
	return func(s *Service) {
		s.client = impl
	}
}

// New constructs a new service instance. The actual bootstrap is handled by
// init() in bootstrap.go so that callers do not need to care about the
// internal initialisation sequence.
func New(ctx context.Context, opts ...Option) (*Service, error) {
	svc := &Service{}
	for _, opt := range opts {
		opt(svc)
	}
	if err := svc.init(ctx); err != nil {
		return nil, err
	}
	return svc, nil
}

// NewWithConfig preserves the old constructor signature to avoid breaking
// existing callers. Additional options may be supplied after the configuration
// instance.
func NewWithConfig(ctx context.Context, cfg *config.Config, opts ...Option) (*Service, error) {
	return New(ctx, append([]Option{WithConfig(cfg)}, opts...)...)
}

// Start launches the underlying Fluxor runtime. Multiple invocations are safe
// – subsequent calls will be ignored.
func (s *Service) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		return nil
	}
	return s.Workflow.Runtime.Start(ctx)
}

// Shutdown terminates the Fluxor runtime. Additional invocations after the
// first successful call have no effect.
func (s *Service) Shutdown(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&s.started, 1, 2) {
		return nil
	}
	return s.Workflow.Runtime.Shutdown(ctx)
}
