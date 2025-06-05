package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// RunCmd starts a workflow.  A minimal subset of options is supported for now –
// the command will evolve once additional use-cases emerge.
type RunCmd struct {
	Location   string `short:"l" long:"location" description:"Workflow definition path (YAML)"`
	InputFile  string `short:"i" long:"input"    description:"JSON file with initial state (stdin if empty)"`
	State      string `short:"s" long:"state" description:"JSON Object with initial state (stdin if empty)"`
	TimeoutSec int    `long:"timeout" description:"Seconds to wait for completion" default:"30"`
}

func (c *RunCmd) Execute(_ []string) error {
	svc, err := serviceSingleton()
	if err != nil {
		return err
	}

	rt := svc.WorkflowRuntime()

	if c.Location == "" {
		return fmt.Errorf("workflow location must be provided via -l/--location")
	}

	ctx := context.Background()
	wf, err := rt.LoadWorkflow(ctx, c.Location)
	if err != nil {
		return fmt.Errorf("load workflow: %w", err)
	}

	initState := make(map[string]interface{})

	if c.State != "" {
		data := strings.TrimSpace(c.State)
		if err := json.Unmarshal([]byte(data), &initState); err != nil {
			return fmt.Errorf("decode initial state: %w", err)
		}
	} else {

		// ------------------------------------------------------------------
		// Decode initial state
		// ------------------------------------------------------------------
		var reader io.Reader = os.Stdin
		if c.InputFile != "" {
			f, err := os.Open(c.InputFile)
			if err != nil {
				return fmt.Errorf("open input file: %w", err)
			}
			defer f.Close()
			reader = f
		}

		// Ignore EOF when no input is provided
		if data, err := io.ReadAll(reader); err == nil && len(data) > 0 {
			if err := json.Unmarshal(data, &initState); err != nil {
				return fmt.Errorf("decode initial state: %w", err)
			}
		}

	}
	// ------------------------------------------------------------------
	// Run workflow (all tasks).  When the workflow defines multiple entry
	// points users can still select specific tasks via --task in the future.
	// ------------------------------------------------------------------
	process, wait, err := rt.StartProcess(ctx, wf, initState)
	if err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	timeout := time.Duration(c.TimeoutSec) * time.Second
	output, err := wait(ctx, timeout)
	if err != nil {
		return fmt.Errorf("wait for process: %w", err)
	}

	// Print final output in JSON for easy consumption.
	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Printf("Workflow output:\n")
	fmt.Println(string(data))
	fmt.Fprintf(os.Stderr, "process %s completed\n", process.ID)
	return nil
}
