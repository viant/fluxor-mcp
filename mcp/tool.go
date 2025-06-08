package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/viant/fluxor-mcp/internal/conv"
	"github.com/viant/fluxor-mcp/mcp/tool"
	"github.com/viant/fluxor-mcp/mcp/tool/conversion"
	"github.com/viant/fluxor/model/types"
	"github.com/viant/fluxor/runtime/execution"
	"github.com/viant/jsonrpc"
	"github.com/viant/mcp-protocol/schema"
	mcpschema "github.com/viant/mcp-protocol/schema"
	serverproto "github.com/viant/mcp-protocol/server"
	"time"
)

// Tools returns tool names
func (s *Service) Tools() serverproto.Tools {
	var result = make(serverproto.Tools, 0)

	actions := s.Workflow.Service.Actions()
	for _, name := range actions.Services() {
		service := actions.Lookup(name)
		for _, method := range service.Methods() {
			toolName := tool.NewName(name, method.Name)
			aTool, err := s.LookupTool(toolName.String())
			if err != nil {
				continue
			}
			result = append(result, aTool)
		}
	}
	return result
}

// LookupTool returns a pointer to the internal entry with the given name
// and a bool indicating presence. Internal helper for CLI inspection.
func (s *Service) LookupTool(name string) (*serverproto.ToolEntry, error) {
	toolName := tool.Name(name)
	actions := s.Workflow.Service.Actions()
	service := actions.Lookup(toolName.Service())
	toolMethod := toolName.Method()
	var err error
	for _, method := range service.Methods() {
		if method.Name == toolMethod {
			sig := &types.Signature{
				Name:   name,
				Input:  method.Input,
				Output: method.Output,
			}
			toolEntry := serverproto.ToolEntry{}
			if toolEntry.Metadata, err = conversion.BuildSchema(sig); err != nil {
				return nil, err
			}
			toolEntry.Handler = func(ctx context.Context, request *mcpschema.CallToolRequest) (*mcpschema.CallToolResult, *jsonrpc.Error) {
				output, err := s.ExecuteTool(ctx, request.Params.Name, request.Params.Arguments, 15*time.Minute)
				res := &mcpschema.CallToolResult{}
				if err != nil {
					res.IsError = conv.Pointer[bool](true)
					res.Content = append(res.Content, schema.CallToolResultContentElem{
						Text: err.Error(),
					})
					return res, nil
				}

				var data []byte
				switch actual := output.(type) {
				case string:
					data = []byte(actual)
				case []byte:
					data = actual
				default:
					data, _ = json.Marshal(output)
				}
				res.Content = append(res.Content, schema.CallToolResultContentElem{
					Text: string(data),
				})
				return res, nil
			}
			return &toolEntry, nil
		}
	}
	return nil, fmt.Errorf("unknown tool: %v", toolName)
}

// ExecuteTool invokes a registered fluxor action with the supplied arguments.
func (s *Service) ExecuteTool(ctx context.Context, name string, args map[string]interface{}, timeout time.Duration) (interface{}, error) {
	toolName := tool.Name(name)

	exec, err := execution.NewAtHocExecution(toolName.Service(), toolName.Method(), args)
	if err != nil {
		return "", err
	}

	waitFn, err := s.Runtime.ScheduleExecution(ctx, exec)
	if err != nil {
		return "", err
	}

	// expected until the background processor persists the execution.
	anExec, err := waitFn(timeout)
	if err != nil {
		return "", err
	}

	if anExec.Error != "" {
		var errorMap = map[string]interface{}{"error": anExec.Error}
		errorResponse, _ := json.Marshal(errorMap)
		return string(errorResponse), nil
	}
	return anExec.Output, nil
}
