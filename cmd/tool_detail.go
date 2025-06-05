package cmd

import (
    "encoding/json"
    "fmt"
)

// ToolCmd prints metadata & input schema for a single tool.
type ToolCmd struct {
    Name string `short:"n" long:"name" description:"tool name (service/method)" positional-arg-name:"name" required:"yes"`
    JSON bool   `long:"json" description:"print result as JSON"`
}

func (c *ToolCmd) Execute(_ []string) error {
    svc, err := serviceSingleton()
    if err != nil {
        return err
    }

    var found *struct {
        Name        string      `json:"name"`
        Description string      `json:"description"`
        InputSchema interface{} `json:"inputSchema"`
    }

    description, schema, ok := svc.ToolMetadata(c.Name)
    if ok {
        found = &struct {
            Name        string      `json:"name"`
            Description string      `json:"description"`
            InputSchema interface{} `json:"inputSchema"`
        }{c.Name, description, schema}
    }

    if found == nil {
        return fmt.Errorf("tool %q not found", c.Name)
    }

    if c.JSON {
        data, _ := json.MarshalIndent(found, "", "  ")
        fmt.Println(string(data))
    } else {
        fmt.Printf("Name : %s\n", found.Name)
        fmt.Printf("Desc : %s\n", found.Description)
        js, _ := json.MarshalIndent(found.InputSchema, "", "  ")
        fmt.Printf("InputSchema:\n%s\n", string(js))
    }
    return nil
}
