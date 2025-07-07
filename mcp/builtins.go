package mcp

import (
	"github.com/viant/fluxor/model/types"
	"github.com/viant/fluxor/service/action/system/patch"

	"github.com/viant/fluxor-mcp/mcp/matcher"

	// Built-in action packages – only those with parameter-less New()
	nop "github.com/viant/fluxor/service/action/nop"
	printer "github.com/viant/fluxor/service/action/printer"
	exec "github.com/viant/fluxor/service/action/system/exec"
)

// builtinFactories lists all Fluxor action services that can be instantiated
// without external dependencies.  The key must match the service name exposed
// by its implementation so that pattern matching is intuitive.
var builtinFactories = map[string]func() types.Service{
	"nop":          func() types.Service { return nop.New() },
	"printer":      func() types.Service { return printer.New() },
	"system/exec":  func() types.Service { return exec.New() },
	"system/patch": func() types.Service { return patch.New() },
}

// resolveBuiltinServices converts pattern(s) – "*" for all, prefix or exact –
// into concrete service instances.  Duplicate patterns are ignored.
func resolveBuiltinServices(patterns []string) []types.Service {
	return ResolveServices(patterns, builtinFactories)
}

// ResolveServices resolves services
func ResolveServices(patterns []string, factories map[string]func() types.Service) []types.Service {
	selected := make(map[string]struct{})

	add := func(name string) { selected[name] = struct{}{} }

	for _, p := range patterns {
		for n := range factories {
			if matcher.Match(p, n) {
				add(n)
			}
		}
	}

	// Instantiate.
	out := make([]types.Service, 0, len(selected))
	for name := range selected {
		if factory := factories[name]; factory != nil {
			out = append(out, factory())
		}
	}
	return out
}
