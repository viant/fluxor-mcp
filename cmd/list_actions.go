package cmd

import (
	"fmt"
	"sort"
)

// ListActionsCmd prints every Fluxor service and its action methods.
type ListActionsCmd struct{}

func (c *ListActionsCmd) Execute(_ []string) error {
	svc, err := serviceSingleton()
	if err != nil {
		return err
	}

	actions := svc.WorkflowService().Actions()
	names := actions.Services()
	sort.Strings(names)
	for _, name := range names {
		s := actions.Lookup(name)
		if s == nil {
			continue
		}
		fmt.Println(name)
		// Deterministic order – sort method signatures by name.
		sigs := s.Methods()
		sort.Slice(sigs, func(i, j int) bool { return sigs[i].Name < sigs[j].Name })
		for _, sig := range sigs {
			fmt.Printf("  %s\t%s\n", sig.Name, sig.Description)
		}
	}
	return nil
}
