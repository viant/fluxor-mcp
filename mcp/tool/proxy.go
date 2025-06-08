package tool

import (
	"context"
	"encoding/json"
	"github.com/viant/fluxor-mcp/internal/conv"
	"reflect"

	"github.com/viant/fluxor-mcp/mcp/tool/conversion"
	"github.com/viant/fluxor/model/types"
	mcpschema "github.com/viant/mcp-protocol/schema"
	mcpclient "github.com/viant/mcp/client"
)

// --------------------- Remote tool service --------------------- //

// Proxy implements types.Proxy by delegating each method to a
// corresponding Server remote tool. The service is generated at runtime based on
// the server’s listTools response.
type Proxy struct {
	name    string
	client  mcpclient.Interface
	methods map[string]*mcpschema.Tool
	sigs    types.Signatures
}

func NewProxy(ctx context.Context, name string, cli mcpclient.Interface) (types.Service, error) {
	// Fetch all available tools (paging supported).
	tools := make([]mcpschema.Tool, 0)
	var cursor *string
	for {
		res, err := cli.ListTools(ctx, cursor)
		if err != nil {
			return nil, err
		}
		tools = append(tools, res.Tools...)
		if res.NextCursor == nil || *res.NextCursor == "" {
			break
		}
		cursor = res.NextCursor
	}

	m := make(map[string]*mcpschema.Tool, len(tools))
	sigs := make(types.Signatures, 0, len(tools))
	for i, t := range tools {
		tool := t // capture
		m[tool.Name] = &tools[i]

		// ------------------------------------------------------------------
		// Convert tool input/output JSON Schemas to Go types so that Fluxor
		// can expose accurate method signatures. The generated types are
		// later re-converted to JSON Schema when building the Server tool
		// registry, therefore the round-trip must preserve property
		// information.
		// ------------------------------------------------------------------

		var (
			inType  reflect.Type
			outType reflect.Type
			errConv error
		)

		// Input schema → reflect.Type. Fallback to generic map when the
		// conversion fails (should not happen unless the schema is invalid).
		if tool.InputSchema.Type != "" || len(tool.InputSchema.Properties) > 0 {
			if inType, errConv = conversion.TypeFromInputSchema(tool.InputSchema); errConv != nil {
				inType = reflect.TypeOf(map[string]interface{}{})
			}
		} else {
			inType = reflect.TypeOf(map[string]interface{}{})
		}

		// Output schema → reflect.Type. If the server does not provide one
		// create an empty object so callers receive *some* structured value.
		if tool.OutputSchema != nil {
			if outType, errConv = conversion.TypeFromOutputSchema(*tool.OutputSchema); errConv != nil {
				outType = reflect.TypeOf(map[string]interface{}{})
			}
		} else {
			// Per requirement: use object with properties (empty struct).
			outType = reflect.StructOf([]reflect.StructField{})
		}

		sigs = append(sigs, types.Signature{
			Name:        tool.Name,
			Description: conv.Dereference[string](tool.Description),
			Input:       inType,
			Output:      outType,
		})
	}

	return &Proxy{name: name, client: cli, methods: m, sigs: sigs}, nil
}

func (r *Proxy) Name() string {
	return r.name
}

func (r *Proxy) Methods() types.Signatures {
	return r.sigs
}

func (r *Proxy) Method(name string) (types.Executable, error) {
	tool, ok := r.methods[name]
	if !ok {
		return nil, types.NewMethodNotFoundError(name)
	}

	exec := func(ctx context.Context, input, output interface{}) error {
		// Coerce input into map[string]interface{} expected by MCP.
		args, _ := conv.ToMap(input)

		params := &mcpschema.CallToolRequestParams{
			Name:      tool.Name,
			Arguments: mcpschema.CallToolRequestParamsArguments(args),
		}

		res, err := r.client.CallTool(ctx, params)
		if err != nil {
			return err
		}

		// Propagate raw response into output when caller provided one.
		if output != nil {
			// Handle common cases: *string or *mcpschema.CallToolResult.
			switch v := output.(type) {
			case *string:
				if len(res.Content) == 1 && res.Content[0].Type == "text" {
					*v = res.Content[0].Text
				} else {
					data, _ := json.Marshal(res.Content)
					*v = string(data)
				}
			case **mcpschema.CallToolResult:
				if v != nil {
					*v = res
				}
			default:
				// Best effort: JSON encode then decode into provided pointer.
				data, err := json.Marshal(res)
				if err == nil {
					_ = json.Unmarshal(data, v)
				}
			}
		}
		return nil
	}

	return exec, nil
}
