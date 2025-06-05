package mcp

import (
	"context"
	"encoding/json"
	"reflect"

	iconv "github.com/viant/fluxor-mcp/internal/conv"
	conv "github.com/viant/fluxor-mcp/mcp/tool/conversion"
	"github.com/viant/fluxor/model/types"
	"github.com/viant/jsonrpc"
	mcpschema "github.com/viant/mcp-protocol/schema"
)

// toolEntry holds metadata and execution handler for one Server tool derived from
// a Fluxor action method.
type toolEntry struct {
	name        string
	description string
	inputSchema mcpschema.ToolInputSchema
	handler     func(context.Context, *mcpschema.CallToolRequest) (*mcpschema.CallToolResult, *jsonrpc.Error)
}

// addToolEntries appends tool entries to the shared registry, skipping
// duplicates so that every registration path behaves consistently.
func (s *Service) addToolEntries(entries []toolEntry) {
	if len(entries) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Build a set of already known tool names for O(1) look-ups.
	existing := make(map[string]struct{}, len(s.mcpTools))
	for _, e := range s.mcpTools {
		existing[e.name] = struct{}{}
	}

	for _, e := range entries {
		if _, dup := existing[e.name]; dup {
			continue // skip duplicates – keep first definition encountered
		}

		// Cache for later introspection.
		s.mcpTools = append(s.mcpTools, e)
		existing[e.name] = struct{}{}

		// Also register with the shared registry when available.
		if s.toolRegistry != nil {
			s.toolRegistry.RegisterToolWithSchema(e.name, e.description, e.inputSchema, e.handler)
		}
	}
}

// buildMcpToolRegistry creates the unified tool registry once during service
// bootstrap.
func (s *Service) buildMcpToolRegistry() {
	if s.Workflow.Service == nil {
		return
	}
	actions := s.Workflow.Service.Actions()
	for _, key := range actions.Services() {
		activeService := actions.Lookup(key)
		newEntries := serviceToToolEntries(activeService)
		s.addToolEntries(newEntries)
	}
}

// updateToolRegistryForService converts methods of a newly added service into
// tool entries and appends them to the global registry, avoiding duplicates.
func (s *Service) updateToolRegistryForService(svc types.Service) {
	newEntries := serviceToToolEntries(svc)
	s.addToolEntries(newEntries)
}

// serviceToToolEntries converts a single Fluxor service to tool entries.
func serviceToToolEntries(svc types.Service) []toolEntry {
	entries := make([]toolEntry, 0, len(svc.Methods()))
	for _, sig := range svc.Methods() {
		methodName := svc.Name() + "-" + sig.Name
		var toolMeta mcpschema.Tool
		var buildErr error
		// mcpClient request/response types are complex (contain recursive
		// definitions) – BuildSchema may overflow the stack.  For those services we
		// fall back to the minimal input-only schema.
		if svc.Name() != "mcpClient" {
			toolMeta, buildErr = conv.BuildSchema(&sig)
		}

		if buildErr != nil || svc.Name() == "mcpClient" {
			// Fallback: derive only input schema via reflection (previous logic).
			var inputSchema mcpschema.ToolInputSchema
			if sig.Input != nil {
				var sample interface{}
				if sig.Input.Kind() == reflect.Pointer {
					sample = reflect.New(sig.Input.Elem()).Interface()
				} else {
					sample = reflect.New(sig.Input).Interface()
				}
				_ = inputSchema.Load(sample)
			}
			if inputSchema.Type == "" {
				inputSchema.Type = "object"
			}
			toolMeta = mcpschema.Tool{
				Name:        methodName,
				Description: &sig.Description,
				InputSchema: inputSchema,
			}
		} else {
			toolMeta.Name = methodName
			if toolMeta.Description == nil {
				toolMeta.Description = &sig.Description
			}
		}
		svcCopy := svc
		sigCopy := sig
		handler := func(ctx context.Context, req *mcpschema.CallToolRequest) (*mcpschema.CallToolResult, *jsonrpc.Error) {
			var inVal interface{}
			if sigCopy.Input != nil {
				if sigCopy.Input.Kind() == reflect.Pointer {
					inVal = reflect.New(sigCopy.Input.Elem()).Interface()
				} else {
					inVal = reflect.New(sigCopy.Input).Interface()
				}
				if len(req.Params.Arguments) > 0 {
					if data, err := json.Marshal(req.Params.Arguments); err == nil {
						_ = json.Unmarshal(data, inVal)
					}
				}
			}

			var outVal interface{}
			if sigCopy.Output != nil {
				if sigCopy.Output.Kind() == reflect.Pointer {
					outVal = reflect.New(sigCopy.Output.Elem()).Interface()
				} else {
					outVal = reflect.New(sigCopy.Output).Interface()
				}
			}

			exec, err := svcCopy.Method(sigCopy.Name)
			if err != nil {
				return nil, jsonrpc.NewError(jsonrpc.InternalError, err.Error(), nil)
			}

			if err := exec(ctx, inVal, outVal); err != nil {
				return nil, jsonrpc.NewError(jsonrpc.InternalError, err.Error(), nil)
			}

			var text string
			if outVal != nil {
				if data, err := json.Marshal(outVal); err == nil {
					text = string(data)
				}
			}

			return &mcpschema.CallToolResult{Content: []mcpschema.CallToolResultContentElem{{
				Type: "text",
				Text: text,
			}}}, nil
		}

		entries = append(entries, toolEntry{
			name:        toolMeta.Name,
			description: iconv.Dereference[string](toolMeta.Description),
			inputSchema: toolMeta.InputSchema,
			handler:     handler,
		})
	}
	return entries
}

// toolEntries returns the read-only slice holding all converted tools.
func (s *Service) toolEntries() []toolEntry { return s.mcpTools }
