package cmd

import (
	"fmt"
	"sort"
)

// ListToolsCmd prints every registered tool in `service/method` form.
type ListToolsCmd struct{}

func (c *ListToolsCmd) Execute(_ []string) error {
	svc, err := serviceSingleton()
	if err != nil {
		return err
	}

	tools := svc.Tools()
	// Sorting for deterministic output (helpful for tests & scripting).
	sort.Slice(tools, func(i, j int) bool { return tools[i].Metadata.Name < tools[j].Metadata.Name })
	for _, t := range tools {
		desc := ""
		if t.Metadata.Description != nil {
			desc = *t.Metadata.Description
		}
		fmt.Printf("%s\t%s\n", t.Metadata.Name, desc)
	}
	return nil
}
