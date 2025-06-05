package cmd

import (
	"github.com/jessevdk/go-flags"
	"log"
	"strings"
)

// Run is the entry point for the CLI.  The function is intentionally separated
// from the main package to keep the command usable from tests as well.
func Run(args []string) {
	cfgPath := extractConfigPath(args)

	// Make config path discoverable by sub-commands via the global singleton.
	setConfigPath(cfgPath)

	opts := &Options{}
	var first string
	if len(args) > 0 {
		first = args[0]
	}
	opts.Init(first)

	parser := flags.NewParser(opts, flags.HelpFlag|flags.PassDoubleDash)
	if _, err := parser.ParseArgs(args); err != nil {
		// flags already prints user-friendly message – just set exit code.
		log.Fatalf("%v", err)
	}
}

// extractConfigPath searches the raw argument list for the -f/--config option
// before the full flags parsing is performed so that sub-commands can load the
// config early from a deterministic location.
func extractConfigPath(args []string) string {
	for i, a := range args {
		switch a {
		case "-f", "--config":
			if i+1 < len(args) {
				return args[i+1]
			}
		default:
			if strings.HasPrefix(a, "--config=") {
				return strings.TrimPrefix(a, "--config=")
			}
		}
	}
	return ""
}

// RunWithCommands is kept for symmetry with other Viant CLIs that provide the
// same helper (e.g. scy, agently).
func RunWithCommands(args []string) {
	Run(args)
}
