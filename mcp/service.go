package mcp

import (
	"context"
	"github.com/viant/fluxor"
	"github.com/viant/fluxor-mcp/mcp/config"
	"github.com/viant/fluxor/model/types"
	"github.com/viant/x"
	"sync"
	"sync/atomic"

	protocolclient "github.com/viant/mcp-protocol/client"
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
