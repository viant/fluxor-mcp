package clientaction

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/fluxor-mcp/internal/conv"
	"github.com/viant/fluxor/model/types"
	"github.com/viant/jsonrpc"
	protocolclient "github.com/viant/mcp-protocol/client"
	mcpschema "github.com/viant/mcp-protocol/schema"
)

// Service exposes MCP client-side operations (roots/list, elicit, …) as
// Fluxor actions so that they can be called from workflows or via the MCP →
// Fluxor tool bridge.
//
// Only operations that the underlying client `Implements` are exported.
type Service struct {
	cli       protocolclient.Operations
	sigs      types.Signatures
	executors map[string]types.Executable
}

const serviceName = "mcpClient"

// New builds the action service by introspecting the client capabilities.
func New(cli protocolclient.Operations) *Service {
	s := &Service{
		cli:       cli,
		executors: map[string]types.Executable{},
	}

	// Descriptor for every potential operation.
	type op struct {
		methodConst string // for Implements()
		name        string // Fluxor method name – camelCase for consistency
		in          reflect.Type
		out         reflect.Type
		call        func(ctx context.Context, c protocolclient.Operations, in interface{}) (interface{}, *jsonrpc.Error)
		desc        string
	}

	ops := []op{
		{
			methodConst: mcpschema.MethodElicitationCreate,
			name:        "elicit",
			in:          reflect.TypeOf(&mcpschema.ElicitRequestParams{}),
			out:         reflect.TypeOf(&mcpschema.ElicitResult{}),
			call: func(ctx context.Context, c protocolclient.Operations, in interface{}) (interface{}, *jsonrpc.Error) {
				p, _ := in.(*mcpschema.ElicitRequestParams)
				return c.Elicit(ctx, &jsonrpc.TypedRequest[*mcpschema.ElicitRequest]{Request: &mcpschema.ElicitRequest{Params: *p}})
			},
			desc: "Elicit content using the serverʼs elicitation endpoint",
		},
		{
			methodConst: mcpschema.MethodRootsList,
			name:        "listRoots",
			in:          reflect.TypeOf(&mcpschema.ListRootsRequestParams{}),
			out:         reflect.TypeOf(&mcpschema.ListRootsResult{}),
			call: func(ctx context.Context, c protocolclient.Operations, in interface{}) (interface{}, *jsonrpc.Error) {
				p, _ := in.(*mcpschema.ListRootsRequestParams)
				return c.ListRoots(ctx, &jsonrpc.TypedRequest[*mcpschema.ListRootsRequest]{Request: &mcpschema.ListRootsRequest{Params: p}})
			},
			desc: "List server roots",
		},
		{
			methodConst: mcpschema.MethodSamplingCreateMessage,
			name:        "createMessage",
			in:          reflect.TypeOf(&mcpschema.CreateMessageRequestParams{}),
			out:         reflect.TypeOf(&mcpschema.CreateMessageResult{}),
			call: func(ctx context.Context, c protocolclient.Operations, in interface{}) (interface{}, *jsonrpc.Error) {
				p, _ := in.(*mcpschema.CreateMessageRequestParams)
				return c.CreateMessage(ctx, &jsonrpc.TypedRequest[*mcpschema.CreateMessageRequest]{Request: &mcpschema.CreateMessageRequest{Params: *p}})
			},
			desc: "Create a message via sampling endpoint",
		},
	}

	// Build signatures & executors for supported operations.
	for _, o := range ops {
		if !cli.Implements(o.methodConst) {
			continue
		}

		// Capture for closure.
		opCopy := o
		exec := func(ctx context.Context, input, output interface{}) error {
			// Accept either typed *struct or generic map; perform best-effort conv.
			var param interface{}
			if input == nil {
				// leave param nil – ops allow nil
			} else if reflect.TypeOf(input) == opCopy.in {
				param = input
			} else {
				// Convert using generic helper.
				paramVal := reflect.New(opCopy.in.Elem()).Interface()
				if err := conv.Convert(input, paramVal); err != nil {
					return err
				}
				param = paramVal
			}

			res, jerr := opCopy.call(ctx, s.cli, param)
			if jerr != nil {
				return fmt.Errorf(jerr.Message)
			}

			if output != nil {
				switch outPtr := output.(type) {
				case *interface{}:
					*outPtr = res
				default:
					_ = conv.Convert(res, outPtr)
				}
			}
			return nil
		}

		s.executors[opCopy.name] = exec

		sig := types.Signature{
			Name:        opCopy.name,
			Description: opCopy.desc,
			Input:       opCopy.in,
			Output:      opCopy.out,
		}
		// Mark MCP client operations as internal if the underlying fluxor
		// Signature struct supports an exported bool field `Internal`.
		setInternalFlag(&sig)
		s.sigs = append(s.sigs, sig)
	}

	return s
}

// ------------------------------------------------------------------
// types.Service implementation
// ------------------------------------------------------------------

func (s *Service) Name() string { return serviceName }

func (s *Service) Methods() types.Signatures { return s.sigs }

func (s *Service) Method(name string) (types.Executable, error) {
	if exec, ok := s.executors[name]; ok {
		return exec, nil
	}
	return nil, types.NewMethodNotFoundError(name)
}

// setInternalFlag sets Signature.Internal = true if the field exists.
func setInternalFlag(sig *types.Signature) {
	v := reflect.ValueOf(sig).Elem()
	f := v.FieldByName("Internal")
	if f.IsValid() && f.CanSet() && f.Kind() == reflect.Bool {
		f.SetBool(true)
	}
}
