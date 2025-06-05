package mcp

import (
    "strings"

    "github.com/viant/fluxor/model/types"

    // Built-in action packages – only those with parameter-less New()
    nop "github.com/viant/fluxor/service/action/nop"
    printer "github.com/viant/fluxor/service/action/printer"
    exec "github.com/viant/fluxor/service/action/system/exec"
    storage "github.com/viant/fluxor/service/action/system/storage"
    secret "github.com/viant/fluxor/service/action/system/secret"
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

    add := func(name string) {
        if _, ok := selected[name]; !ok {
            selected[name] = struct{}{}
        }
    }

    for _, p := range patterns {
        switch p {
        case "*":
            for n := range builtinFactories {
                add(n)
            }
        default:
            // prefix match if ends with "/" otherwise exact.
            isPrefix := strings.HasSuffix(p, "/")
            for n := range builtinFactories {
                if (isPrefix && strings.HasPrefix(n, p)) || (!isPrefix && n == p) {
                    add(n)
                }
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
