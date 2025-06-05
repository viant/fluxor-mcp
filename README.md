# Fluxor-MCP: Bridging Fluxor Workflows with the MCP Protocol

Fluxor-MCP is an adapter that turns every [Fluxor](https://github.com/viant/fluxor) action into an
[MCP](https://github.com/viant/mcp-protocol) **tool** and vice-versa.  It lets you

* run Fluxor workflows from the command line,
* serve those workflows through an MCP server so that they can be consumed by
  other processes (e.g. LLM agents), and
* import remote MCP endpoints and use their tools as if they were native
  Fluxor actions.

Fluxor-MCP therefore provides a *single, unified execution surface* whether the
logic lives inside your Go binary, in another process on the local host, on a
remote machine – or a mix of all three.

---

## Table of Contents

1. [Key Features](#key-features)
2. [Architecture Overview](#architecture-overview)
3. [Getting Started](#getting-started)
4. [Command-line Interface](#command-line-interface)
5. [Configuration](#configuration)
6. [Examples](#examples)
7. [Contributing](#contributing)
8. [License](#license)

---

## Key Features

* **Unified registry** – Fluxor actions and MCP tools share the same
  namespace so that every component can be called through both interfaces.
* **Automatic schema generation** – Action input/output types are converted
  to JSON-schema so that MCP clients get proper validation & tooltips.
* **Built-in action discovery** – Common Fluxor services (`printer`,
  `system/exec`, …) are loaded automatically; wildcard or prefix selection is
  supported.
* **Dynamic client import** – Point the CLI at a remote MCP server and all of
  its tools are available instantly inside your workflow.
* **Ready-to-use CLI** – Run workflows, start a server, inspect actions/tools
  or add remote clients without writing any Go code.


## Architecture Overview

```
┌────────────┐           import             ┌────────────┐
│  Remote    │ ───────────────────────────▶ │  Fluxor    │
│  MCP       │  (MCP protocol)              │  Runtime   │
│  Server    │                              └─────┬──────┘
└────────────┘                                   │ actions ↔ tools
      ▲                                          ▼
      │                               ┌──────────────────────┐
      │ serve MCP tools               │   Fluxor-MCP         │
      └──────────────────────────────▶│   Service & CLI      │
                                      │  – tool registry     │
                                      │  – server adapter    │
                                      └─────────┬────────────┘
                                                │
                                                ▼
                                      ┌──────────────────────┐
                                      │   Built-in Actions   │
                                      │  (printer, exec, …)  │
                                      └──────────────────────┘
```

1. **Fluxor-MCP Service** – wraps a standard Fluxor runtime and keeps a global
   tool registry that is shared by every incoming/outgoing MCP connection.
2. **Server Adapter** – exposes the registry via HTTP/SSE.
3. **Client Importer** – connects to foreign MCP servers, introspects their
   tools and registers thin proxy actions on the local Fluxor instance.


## Getting Started

### Installation

```bash
go get github.com/viant/fluxor-mcp@latest

# optional – install the standalone CLI
go install github.com/viant/fluxor-mcp/cmd/fluxor-mcp@latest
```

### Minimal Example (library usage)

```go
package main

import (
    "context"
    "fmt"

    mcp "github.com/viant/fluxor-mcp/mcp" // import path for the service
)

func main() {
    ctx := context.Background()

    // Create a new service with default configuration. All built-in actions
    // (printer, exec, ...) are loaded automatically.
    svc, err := mcp.New(ctx)
    if err != nil {
        panic(err)
    }

    // Use the embedded Fluxor runtime just like you would without MCP.
    rt := svc.WorkflowRuntime()

    workflow, _ := rt.LoadWorkflow(ctx, "parent.yaml")
    _, wait, _ := rt.StartProcess(ctx, workflow, nil)
    output, _ := wait(ctx, 0)
    fmt.Printf("workflow finished: %+v\n", output)
}
```


## Command-line Interface

The `fluxor-mcp` binary is a thin wrapper around the service above and covers
the most common tasks. Run `fluxor-mcp --help` to see the full list of
options.

```
Usage: fluxor-mcp [global options] <command> [command options]

Global Options:
  -f, --config <file>   Service configuration YAML/JSON path (optional)

Commands:
  run            Run a workflow                 (see `run --help`)
  serve          Start an MCP server            (see `serve --help`)
  add-client     Register remote MCP endpoint   (see `add-client --help`)
  list-tools     List all registered tools      (service/method)
  list-actions   List Fluxor services & actions
  tool           Show detailed info about one tool
  action         Show detailed info about one action
```

### Selected Sub-commands

1. **Run a workflow locally**

   ```bash
   fluxor-mcp run -l examples/hello.yaml -s '{"name":"World"}'
   ```

2. **Expose tools over HTTP** (defaults to `:5000`)

   ```bash
   fluxor-mcp serve -f config.yaml
   ```

3. **Import tools from a remote server**

   ```bash
   fluxor-mcp add-client \
     --name prod \
     --address https://mcp.example.com/tools \
     --version v1
   ```


## Configuration

All features can be used without a configuration file.  When present, the YAML
shown below illustrates the available knobs.

```yaml
# config.yaml

# 1) Built-in actions – choose any subset by exact match, prefix or wildcard
builtins:
  - "*"            # ← load every built-in service (default when omitted)
  # - "printer"   # single service
  # - "system/"   # prefix – everything underneath system/

# 2) MCP Server options used by the `serve` command (all fields optional)
server:
  transport: "http"     # http | ws | stdio | … (see github.com/viant/mcp)
  port: 6000            # default 5000

# 3) Remote MCP endpoints to import on startup
mcp:
  items:
    - name: analytics
      version: v1
      transport:
        type: sse
        url: https://analytics.example.com/tools

#   Alternatively load the list from another file/URL:
# mcp:
#   url: file://externals.yaml

```


## Examples

Clone the repository to get a couple of ready-to-run examples:

```bash
git clone https://github.com/viant/fluxor-mcp.git
cd fluxor-mcp

# Run the hello-world workflow
go run ./cmd/fluxor-mcp run -l examples/hello.yaml -s '{"name":"Bob"}'

# Start a local MCP server and play with the tool registry
go run ./cmd/fluxor-mcp serve

# Inspect available tools / actions
go run ./cmd/fluxor-mcp list-tools
go run ./cmd/fluxor-mcp list-actions
```


## Contributing

Issues and pull requests are welcome!  Please open an issue first to discuss
the intended change when you plan to work on larger features so that we can
avoid duplicate effort.


## License

Fluxor-MCP is licensed under the Apache 2.0 license.  See the [LICENSE](LICENSE)
file for details.

---

© 2012-2023 Viant, Inc. All rights reserved.
