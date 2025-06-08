package mcp

import (
	"github.com/viant/fluxor/model/types"

	"github.com/viant/fluxor-mcp/mcp/matcher"

	// Built-in action packages – only those with parameter-less New()
	nop "github.com/viant/fluxor/service/action/nop"
	printer "github.com/viant/fluxor/service/action/printer"
	exec "github.com/viant/fluxor/service/action/system/exec"
	secret "github.com/viant/fluxor/service/action/system/secret"
	storage "github.com/viant/fluxor/service/action/system/storage"
)

// builtinFactories lists all Fluxor action services that can be instantiated
// without external dependencies.  The key must match the service name exposed
// by its implementation so that pattern matching is intuitive.
var builtinFactories = map[string]func() types.Service{
	"nop":            func() types.Service { return nop.New() },
	"printer":        func() types.Service { return printer.New() },
	"system/exec":    func() types.Service { return exec.New() },
	"system/storage": func() types.Service { return storage.New() },
	"system/secret":  func() types.Service { return secret.New() },
}

// resolveBuiltinServices converts pattern(s) – "*" for all, prefix or exact –
// into concrete service instances.  Duplicate patterns are ignored.
func resolveBuiltinServices(patterns []string) []types.Service {
	selected := make(map[string]struct{})

	add := func(name string) { selected[name] = struct{}{} }

	for _, p := range patterns {
		for n := range builtinFactories {
			if matcher.Match(p, n) {
				add(n)
			}
		}
	}

	// Instantiate.
	out := make([]types.Service, 0, len(selected))
	for name := range selected {
		if factory := builtinFactories[name]; factory != nil {
			out = append(out, factory())
		}
	}
	return out
}
