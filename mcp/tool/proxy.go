package tool

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/viant/fluxor-mcp/internal/conv"
	"github.com/viant/fluxor-mcp/mcp/tool/conversion"
	"github.com/viant/fluxor/model/types"
	mcpschema "github.com/viant/mcp-protocol/schema"
	mcpclient "github.com/viant/mcp/client"
)

// Proxy implements types.Service by delegating each method to a remote MCP tool.
// It discovers the server tool registry at runtime and exposes strongly‑typed
// Go signatures to Fluxor by converting JSON Schema ⇄ reflect.Type in both
// directions.
//
// Key points of this implementation:
//   1. **Zero boilerplate** – any new server‑side tool is auto‑discovered via
//      `refresh()` and exposed without codegen.
//   2. **Spec‑accurate decoding** – handles MCP‑defined content types (`text`,
//      `data`, `resource`) plus legacy aliases (e.g. `jsondata`) so older
//      servers still work.
//   3. **Rich MIME routing** – JSON, XML, CSV, images (base64 inline) and a
//      catch‑all bucket for anything else.
//   4. **Minimal duplication** – a single `decode()` routine coerces results
//      into the caller‑supplied `output` value.

// -----------------------------------------------------------------------------
// Proxy definition & registry refresh
// -----------------------------------------------------------------------------

type Proxy struct {
	name    string
	client  mcpclient.Interface
	methods map[string]*mcpschema.Tool
	sigs    types.Signatures
}

// NewProxy creates a new tool proxy and immediately discovers the server's
// tool registry. Call `proxy.refresh(ctx)` later if you need to pick up tools
// added after start‑up.
func NewProxy(ctx context.Context, name string, cli mcpclient.Interface) (types.Service, error) {
	p := &Proxy{name: name, client: cli}
	if err := p.refresh(ctx); err != nil {
		return nil, err
	}
	return p, nil
}

// refresh (re)hydrates the local tool registry.
func (p *Proxy) refresh(ctx context.Context) error {
	var (
		tools  []mcpschema.Tool
		cursor *string
	)
	for {
		res, err := p.client.ListTools(ctx, cursor)
		if err != nil {
			return fmt.Errorf("list tools: %w", err)
		}
		tools = append(tools, res.Tools...)
		if res.NextCursor == nil || *res.NextCursor == "" {
			break
		}
		cursor = res.NextCursor
	}

	p.methods = make(map[string]*mcpschema.Tool, len(tools))
	sigs := make(types.Signatures, 0, len(tools))

	for i, t := range tools {
		tool := t // capture
		p.methods[tool.Name] = &tools[i]

		// ---------------- schema → reflect types ---------------- //
		inT, _ := conversion.TypeFromInputSchema(tool.InputSchema)
		if inT == nil {
			inT = reflect.TypeOf(map[string]any{})
		}

		var outT reflect.Type
		if tool.OutputSchema != nil {
			outT, _ = conversion.TypeFromOutputSchema(*tool.OutputSchema)
		}
		if outT == nil {
			outT = reflect.StructOf([]reflect.StructField{}) // empty object
		}

		sigs = append(sigs, types.Signature{
			Name:        tool.Name,
			Description: conv.Dereference[string](tool.Description),
			Input:       inT,
			Output:      outT,
		})
	}
	p.sigs = sigs
	return nil
}

// Name returns the service name.
func (p *Proxy) Name() string { return p.name }

// Methods returns all discovered tool signatures.
func (p *Proxy) Methods() types.Signatures { return p.sigs }

// Method returns an executable for the requested tool.
func (p *Proxy) Method(name string) (types.Executable, error) {
	tool, ok := p.methods[name]
	if !ok {
		return nil, types.NewMethodNotFoundError(name)
	}

	exec := func(ctx context.Context, input, output interface{}) error {
		// ---------- invoke remote tool ---------- //
		args, _ := conv.ToMap(input)
		res, err := p.client.CallTool(ctx, &mcpschema.CallToolRequestParams{
			Name:      tool.Name,
			Arguments: mcpschema.CallToolRequestParamsArguments(args),
		})
		if err != nil {
			return fmt.Errorf("call tool %q: %w", tool.Name, err)
		}
		if res.IsError != nil && *res.IsError {
			return toolError(res)
		}

		// ---------- map response → caller out ---------- //
		if output == nil {
			return nil
		}
		return decode(output, classify(res.Content))
	}

	return exec, nil
}

// -----------------------------------------------------------------------------
// Content bucketing & decoding helpers
// -----------------------------------------------------------------------------

type resultBuckets struct {
	json         []string                              // application/json
	xml          []string                              // application/xml, text/xml
	csv          []string                              // text/csv
	text         []string                              // human‑readable
	imagesBase64 []string                              // inline base64 images (image/*)
	others       []mcpschema.CallToolResultContentElem // anything else
}

// classify groups `CallToolResultContentElem` items by (type, MIME). The MCP
// 2025‑03‑26 spec defines three primary `type` values:
//   - "text"      – plain text payloads
//   - "data"      – structured content (JSON, XML, CSV, etc.)
//   - "resource"  – external assets referenced by URI / ID
//
// Older servers may still send "jsondata"; we treat it as an alias of "data".
func classify(items []mcpschema.CallToolResultContentElem) resultBuckets {
	var b resultBuckets
	for _, it := range items {
		mime := strings.ToLower(it.MimeType)
		itemType := strings.ToLower(it.Type) // normalise for comparison

		// Handle inlined images regardless of declared `type`.
		if strings.HasPrefix(mime, "image/") {
			if it.Data != "" {
				b.imagesBase64 = append(b.imagesBase64, it.Data)
			} else {
				b.others = append(b.others, it)
			}
			continue
		}

		switch itemType {
		case "":
			if it.Data != "" {
				it.Type = "data"
				b.json = append(b.json, it.Data)
			} else if it.Text != "" {
				it.Type = "text"
				b.text = append(b.text, it.Text)
			}
		case "data":
			switch mime {
			case "", "application/json":
				b.json = append(b.json, it.Data)
			case "application/xml", "text/xml":
				b.xml = append(b.xml, it.Data)
			case "text/csv":
				b.csv = append(b.csv, it.Data)
			default:
				b.others = append(b.others, it)
			}
		case "text":
			b.text = append(b.text, it.Text)
		default: // "resource" and future/unknown types
			b.others = append(b.others, it)
		}
	}
	return b
}

// decode coerces buckets into the caller‑provided `out` value.
func decode(out interface{}, b resultBuckets) error {
	switch v := out.(type) {
	case *interface{}:
		// Flexible catch‑all – pick the richest content available.
		switch {
		case len(b.json) > 0:
			return json.Unmarshal([]byte(b.json[0]), v)
		case len(b.imagesBase64) > 0:
			*v = b.imagesBase64[0]
		case len(b.text) > 0:
			*v = strings.Join(b.text, "")
		default:
			*v = b // expose raw buckets
		}
		return nil

	case *string:
		switch {
		case len(b.text) > 0:
			*v = strings.Join(b.text, "")
		case len(b.json) > 0:
			*v = strings.Join(b.json, "")
		case len(b.imagesBase64) > 0:
			*v = b.imagesBase64[0]
		case len(b.xml) > 0:
			*v = strings.Join(b.xml, "")
		case len(b.csv) > 0:
			*v = strings.Join(b.csv, "")
		default:
			*v = ""
		}
		return nil

	case *[]byte:
		// Prefer decoded image bytes when available.
		if len(b.imagesBase64) > 0 {
			img, err := base64.StdEncoding.DecodeString(b.imagesBase64[0])
			if err != nil {
				return fmt.Errorf("decode base64 image: %w", err)
			}
			*v = img
			return nil
		}
		switch {
		case len(b.json) > 0:
			*v = []byte(b.json[0])
		case len(b.xml) > 0:
			*v = []byte(b.xml[0])
		case len(b.csv) > 0:
			*v = []byte(b.csv[0])
		default:
			return errors.New("no binary‑compatible payload found")
		}
	}
	if len(b.json) == 0 && len(b.text) > 0 {
		return json.Unmarshal([]byte(b.text[0]), out)
	}
	return json.Unmarshal([]byte(b.json[0]), out)
}

// toolError converts an error‑flagged MCP result into Go error.
func toolError(res *mcpschema.CallToolResult) error {
	if len(res.Content) == 0 {
		return errors.New("tool returned error without content")
	}
	if msg := res.Content[0].Text; msg != "" {
		return errors.New(msg)
	}
	raw, _ := json.Marshal(res.Content[0])
	return errors.New(string(raw))
}
