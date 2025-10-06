package discovery

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/viant/fluxor-mcp/internal/conv"
	"github.com/viant/fluxor-mcp/mcp/config"
	"github.com/viant/fluxor/model/types"
	mcpschema "github.com/viant/mcp-protocol/schema"
	mcpclient "github.com/viant/mcp/client"
)

// Service is a lightweight Fluxor service exposing MCP discovery endpoints
// (resources and prompts) as callable actions. One Service instance maps to
// a single logical namespace (e.g. "<server>/resources").
type Service struct {
	name      string
	sigs      types.Signatures
	executors map[string]types.Executable
}

func (s *Service) Name() string              { return s.name }
func (s *Service) Methods() types.Signatures { return s.sigs }
func (s *Service) Method(name string) (types.Executable, error) {
	if e, ok := s.executors[name]; ok {
		return e, nil
	}
	return nil, types.NewMethodNotFoundError(name)
}

// New builds discovery services for the provided MCP client. It returns
// multiple services to achieve intuitive tool names like:
//
//	<prefix>/resources-list
//	<prefix>/resources-read
//	<prefix>/resources_templates-list
//	<prefix>/prompts-list
//	<prefix>/prompts-get
//
// where <prefix> is derived from the MCP client name (underscores → slashes).
//
// The cfg parameter may carry optional description overrides.
func New(ctx context.Context, cfg *config.MCPClient, cli mcpclient.Interface) ([]types.Service, error) { //nolint:revive // ctx used for capability probe
	if cfg == nil || cfg.ClientOptions == nil {
		return nil, fmt.Errorf("nil client options")
	}
	prefix := strings.ReplaceAll(cfg.Name, "_", "/")

	// Helper to pull an override or use default.
	desc := func(key, def string) string {
		if cfg.Descriptions != nil {
			if v, ok := cfg.Descriptions[key]; ok && v != "" {
				return v
			}
		}
		return def
	}

	var out []types.Service

	// -------- Prefer Initialize() capability check; fallback to probes --------
	var hasResources, hasTemplates, hasPrompts bool
	if initRes, err := func() (*mcpschema.InitializeResult, error) {
		pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return cli.Initialize(pctx)
	}(); err == nil && initRes != nil {
		if initRes.Capabilities.Resources != nil {
			hasResources = true
			hasTemplates = true // templates are part of resources surface
		}
		if initRes.Capabilities.Prompts != nil {
			hasPrompts = true
		}
	} else {
		// Fallback to lightweight probes when Initialize is not available/allowed.
		probe := func(call func(ctx context.Context) error) bool {
			pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			return call(pctx) == nil
		}
		hasResources = probe(func(cctx context.Context) error {
			_, err := cli.ListResources(cctx, nil)
			return err
		})
		hasTemplates = probe(func(cctx context.Context) error {
			_, err := cli.ListResourceTemplates(cctx, nil)
			return err
		})
		hasPrompts = probe(func(cctx context.Context) error {
			_, err := cli.ListPrompts(cctx, nil)
			return err
		})
	}

	// -------- <prefix>/resources --------
	if hasResources {
		s := &Service{name: prefix + "/resources", executors: map[string]types.Executable{}}

		// resources/list
		{
			inT := reflect.TypeOf(&mcpschema.ListResourcesRequestParams{})
			outT := reflect.TypeOf(&mcpschema.ListResourcesResult{})
			name := "list"
			s.sigs = append(s.sigs, types.Signature{
				Name:        name,
				Description: desc("resources/list", "List available resources on the server"),
				Input:       inT,
				Output:      outT,
			})
			s.executors[name] = func(ctx context.Context, input, output interface{}) error {
				var p *mcpschema.ListResourcesRequestParams
				switch v := input.(type) {
				case nil:
					// leave p nil; cursor is optional
				case *mcpschema.ListResourcesRequestParams:
					p = v
				default:
					tmp := &mcpschema.ListResourcesRequestParams{}
					if err := conv.Convert(input, tmp); err != nil {
						return err
					}
					p = tmp
				}
				var cursor *string
				if p != nil {
					cursor = p.Cursor
				}
				res, err := cli.ListResources(ctx, cursor)
				if err != nil {
					return err
				}
				// Augment metadata with server identity and indexing hint.
				if res != nil {
					if res.Meta == nil {
						res.Meta = map[string]interface{}{}
					}
					if len(cfg.Metadata) > 0 {
						// Preserve any server-provided metadata["metadata"].
						if existing, ok := res.Meta["metadata"].(map[string]interface{}); ok {
							for k, v := range cfg.Metadata {
								existing[k] = v
							}
							res.Meta["metadata"] = existing
						} else {
							res.Meta["metadata"] = cfg.Metadata
						}
					}
				}
				if output != nil {
					_ = conv.Convert(res, output)
				}
				return nil
			}
		}

		// resources/read
		{
			inT := reflect.TypeOf(&mcpschema.ReadResourceRequestParams{})
			outT := reflect.TypeOf(&mcpschema.ReadResourceResult{})
			name := "read"
			s.sigs = append(s.sigs, types.Signature{
				Name:        name,
				Description: desc("resources/read", "Read the content of a specific resource"),
				Input:       inT,
				Output:      outT,
			})
			s.executors[name] = func(ctx context.Context, input, output interface{}) error {
				var p *mcpschema.ReadResourceRequestParams
				switch v := input.(type) {
				case *mcpschema.ReadResourceRequestParams:
					p = v
				default:
					tmp := &mcpschema.ReadResourceRequestParams{}
					if err := conv.Convert(input, tmp); err != nil {
						return err
					}
					p = tmp
				}
				res, err := cli.ReadResource(ctx, p)
				if err != nil {
					return err
				}
				if res != nil {
					if res.Meta == nil {
						res.Meta = map[string]interface{}{}
					}
					if len(cfg.Metadata) > 0 {
						if existing, ok := res.Meta["metadata"].(map[string]interface{}); ok {
							for k, v := range cfg.Metadata {
								existing[k] = v
							}
							res.Meta["metadata"] = existing
						} else {
							res.Meta["metadata"] = cfg.Metadata
						}
					}
				}
				if output != nil {
					_ = conv.Convert(res, output)
				}
				return nil
			}
		}

		out = append(out, s)
	}

	// -------- <prefix>/resources/templates --------
	if hasTemplates {
		s := &Service{name: prefix + "/resources/templates", executors: map[string]types.Executable{}}
		inT := reflect.TypeOf(&mcpschema.ListResourceTemplatesRequestParams{})
		outT := reflect.TypeOf(&mcpschema.ListResourceTemplatesResult{})
		name := "list"
		s.sigs = append(s.sigs, types.Signature{
			Name:        name,
			Description: desc("resources/templates/list", "List resource templates offered by the server"),
			Input:       inT,
			Output:      outT,
		})
		s.executors[name] = func(ctx context.Context, input, output interface{}) error {
			var p *mcpschema.ListResourceTemplatesRequestParams
			switch v := input.(type) {
			case nil:
				// leave p nil; cursor optional
			case *mcpschema.ListResourceTemplatesRequestParams:
				p = v
			default:
				tmp := &mcpschema.ListResourceTemplatesRequestParams{}
				if err := conv.Convert(input, tmp); err != nil {
					return err
				}
				p = tmp
			}
			var cursor *string
			if p != nil {
				cursor = p.Cursor
			}
			res, err := cli.ListResourceTemplates(ctx, cursor)
			if err != nil {
				return err
			}
			if output != nil {
				_ = conv.Convert(res, output)
			}
			return nil
		}
		out = append(out, s)
	}

	// -------- <prefix>/prompts --------
	if hasPrompts {
		s := &Service{name: prefix + "/prompts", executors: map[string]types.Executable{}}

		// prompts/list
		{
			inT := reflect.TypeOf(&mcpschema.ListPromptsRequestParams{})
			outT := reflect.TypeOf(&mcpschema.ListPromptsResult{})
			name := "list"
			s.sigs = append(s.sigs, types.Signature{
				Name:        name,
				Description: desc("prompts/list", "List available prompts exposed by the server"),
				Input:       inT,
				Output:      outT,
			})
			s.executors[name] = func(ctx context.Context, input, output interface{}) error {
				var p *mcpschema.ListPromptsRequestParams
				switch v := input.(type) {
				case nil:
					// cursor optional
				case *mcpschema.ListPromptsRequestParams:
					p = v
				default:
					tmp := &mcpschema.ListPromptsRequestParams{}
					if err := conv.Convert(input, tmp); err != nil {
						return err
					}
					p = tmp
				}
				var cursor *string
				if p != nil {
					cursor = p.Cursor
				}
				res, err := cli.ListPrompts(ctx, cursor)
				if err != nil {
					return err
				}
				if output != nil {
					_ = conv.Convert(res, output)
				}
				return nil
			}
		}

		// prompts/get
		{
			inT := reflect.TypeOf(&mcpschema.GetPromptRequestParams{})
			outT := reflect.TypeOf(&mcpschema.GetPromptResult{})
			name := "get"
			s.sigs = append(s.sigs, types.Signature{
				Name:        name,
				Description: desc("prompts/get", "Retrieve a specific prompt definition by name"),
				Input:       inT,
				Output:      outT,
			})
			s.executors[name] = func(ctx context.Context, input, output interface{}) error {
				var p *mcpschema.GetPromptRequestParams
				switch v := input.(type) {
				case *mcpschema.GetPromptRequestParams:
					p = v
				default:
					tmp := &mcpschema.GetPromptRequestParams{}
					if err := conv.Convert(input, tmp); err != nil {
						return err
					}
					p = tmp
				}
				res, err := cli.GetPrompt(ctx, p)
				if err != nil {
					return err
				}
				if output != nil {
					_ = conv.Convert(res, output)
				}
				return nil
			}
		}

		out = append(out, s)
	}

	return out, nil
}
