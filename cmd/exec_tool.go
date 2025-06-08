package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// ExecCmd executes a registered tool (service-method pair) or generic Fluxor
// action from the CLI.  Arguments can be supplied either inline via -i/--input
// or loaded from a JSON file via -f/--file.
type ExecCmd struct {
	Name       string `short:"n" long:"name" positional-arg-name:"tool" description:"Tool name (service_method)" required:"yes"`
	Inline     string `short:"i" long:"input" description:"Inline JSON arguments (object)"`
	File       string `short:"f" long:"file" description:"Path to JSON file with arguments (use - for stdin)"`
	TimeoutSec int    `long:"timeout" description:"Seconds to wait for completion" default:"120"`
	JSON       bool   `long:"json" description:"Print result as JSON"`
}

func (c *ExecCmd) Execute(_ []string) error {
	if c.Inline != "" && c.File != "" {
		return fmt.Errorf("-i/--input and -f/--file are mutually exclusive")
	}

	svc, err := serviceSingleton()
	if err != nil {
		return err
	}

	// ------------------------------------------------------------------
	// Build argument map
	// ------------------------------------------------------------------
	var args map[string]interface{}

	switch {
	case c.Inline != "":
		if err := json.Unmarshal([]byte(c.Inline), &args); err != nil {
			return fmt.Errorf("invalid inline JSON: %w", err)
		}
	case c.File != "":
		var rdr io.Reader
		if c.File == "-" {
			rdr = os.Stdin
		} else {
			f, err := os.Open(c.File)
			if err != nil {
				return fmt.Errorf("open input file: %w", err)
			}
			defer f.Close()
			rdr = f
		}
		data, err := io.ReadAll(rdr)
		if err != nil {
			return fmt.Errorf("read input: %w", err)
		}
		if err := json.Unmarshal(data, &args); err != nil {
			return fmt.Errorf("decode JSON: %w", err)
		}
	default:
		// no arguments supplied – args remains nil / empty map
	}

	ctx := context.Background()
	timeout := time.Duration(c.TimeoutSec) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	out, err := svc.ExecuteTool(ctx, c.Name, args, timeout)
	if err != nil {
		return err
	}

	if c.JSON {
		bytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(bytes))
	} else {
		// When output is already a string/byte slice just print it.
		switch v := out.(type) {
		case string:
			fmt.Println(v)
		case []byte:
			fmt.Println(string(v))
		default:
			bytes, _ := json.MarshalIndent(v, "", "  ")
			fmt.Println(string(bytes))
		}
	}
	return nil
}
