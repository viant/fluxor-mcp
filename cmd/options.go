package cmd

// Options is the root for the CLI.  Struct tags are interpreted by
// github.com/jessevdk/go-flags which is the same library used by other Viant
// CLIs (e.g. Agently).
type Options struct {
	Config string `short:"f" long:"config" description:"MCP/Fluxor service configuration YAML/JSON path"`

	Run         *RunCmd         `command:"run"          description:"Run a workflow"`
	AddClient   *AddClientCmd   `command:"add-client"   description:"Register external MCP endpoint and import its tools"`
	ListTools   *ListToolsCmd   `command:"list-tools"   description:"List all registered tools"`
	ListActions *ListActionsCmd `command:"list-actions" description:"List Fluxor services and their actions"`
	Action      *ActionCmd      `command:"action"       description:"Show detailed info about one Fluxor action"`
	Tool        *ToolCmd        `command:"tool"         description:"Show detailed info about one MCP tool"`
	Serve       *ServeCmd       `command:"serve"        description:"Start MCP server exposing the registered tools"`
}

// Init instantiates the sub-command referenced by the first positional argument
// so that go-flags can populate its fields.
func (o *Options) Init(firstArg string) {
	switch firstArg {
	case "run":
		o.Run = &RunCmd{}
	case "add-client":
		o.AddClient = &AddClientCmd{}
	case "list-tools":
		o.ListTools = &ListToolsCmd{}
	case "list-actions":
		o.ListActions = &ListActionsCmd{}
	case "action":
		o.Action = &ActionCmd{}
	case "tool":
		o.Tool = &ToolCmd{}
	case "serve":
		o.Serve = &ServeCmd{}
	}
}
