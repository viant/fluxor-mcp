// Package cmd implements all sub-commands that make up the fluxor-mcp
// command-line interface.  Each file in this directory registers a single
// sub-command (run, serve, exec, list-tools, …).  The plumbing that is shared
// between commands such as configuration loading or service initialisation is
// located in shared.go.
package cmd
