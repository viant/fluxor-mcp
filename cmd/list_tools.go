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

	tools := svc.ToolDescriptors()
	// Sorting for deterministic output (helpful for tests & scripting).
	sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })
	for _, t := range tools {
		fmt.Printf("%s\t%s\n", t.Name, t.Description)
	}
	return nil
}
