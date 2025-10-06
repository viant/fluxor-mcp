package main

import (
	"github.com/viant/fluxor-mcp/cmd"
	"os"
)

func main() {

	cmd.Run(os.Args[1:])
}
