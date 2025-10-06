package cmd

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// ActionCmd shows detailed information about one Fluxor action method.
type ActionCmd struct {
	Name string `short:"n" long:"name" description:"identifier in form service/method" positional-arg-name:"name" required:"yes"`
	JSON bool   `long:"json" description:"print result as JSON"`
}

func (c *ActionCmd) Execute(_ []string) error {
	if c.Name == "" {
		return fmt.Errorf("--name is required")
	}
	parts := strings.SplitN(c.Name, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("name must be service/method")
	}
	svcName, method := parts[0], parts[1]

	svc, err := serviceSingleton()
	if err != nil {
		return err
	}

	actions := svc.WorkflowService().Actions()
	s := actions.Lookup(svcName)
	if s == nil {
		return fmt.Errorf("service %q not found", svcName)
	}
	sig := s.Methods().Lookup(method)
	if sig == nil {
		return fmt.Errorf("method %q not found in service %q", method, svcName)
	}

	info := struct {
		Service     string `json:"service"`
		Method      string `json:"method"`
		Description string `json:"description"`
		InputType   string `json:"inputType"`
		OutputType  string `json:"outputType"`
		InputDef    string `json:"inputDefinition,omitempty"`
		OutputDef   string `json:"outputDefinition,omitempty"`
	}{
		Service:     svcName,
		Method:      method,
		Description: sig.Description,
		InputType:   typeString(sig.Input),
		OutputType:  typeString(sig.Output),
		InputDef:    typeDefinition(sig.Input, ""),
		OutputDef:   typeDefinition(sig.Output, ""),
	}

	if c.JSON {
		data, _ := json.MarshalIndent(info, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("Service : %s\n", info.Service)
		fmt.Printf("Method  : %s\n", info.Method)
		fmt.Printf("Desc    : %s\n", info.Description)
		fmt.Printf("Input   : %s\n", info.InputType)
		fmt.Printf("Output  : %s\n", info.OutputType)
		if info.InputDef != "" {
			fmt.Printf("\nInput Definition:\n%s\n", info.InputDef)
		}
		if info.OutputDef != "" {
			fmt.Printf("\nOutput Definition:\n%s\n", info.OutputDef)
		}
	}
	return nil
}

func typeString(t reflect.Type) string {
	if t == nil {
		return "<none>"
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
		return "*" + t.String()
	}
	return t.String()
}

// typeDefinition returns a Go-like struct definition for anonymous types or
// an empty string for named/builtin ones.
func typeDefinition(t reflect.Type, indent string) string {
	if t == nil {
		return ""
	}
	if t.Kind() == reflect.Pointer {
		return typeDefinition(t.Elem(), indent)
	}
	if t.Name() != "" { // named type
		return ""
	}

	switch t.Kind() {
	case reflect.Struct:
		var b strings.Builder
		b.WriteString("struct {\n")
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			b.WriteString(indent)
			b.WriteString("    ")
			b.WriteString(f.Name)
			b.WriteString(" ")
			b.WriteString(simpleTypeExpr(f.Type))
			if tag := strings.TrimSpace(string(f.Tag)); tag != "" {
				b.WriteString(" `")
				b.WriteString(tag)
				b.WriteString("`")
			}
			b.WriteString("\n")
		}
		b.WriteString(indent)
		b.WriteString("}")
		return b.String()
	default:
		return "" // other anonymous kinds not elaborated
	}
}

func simpleTypeExpr(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		return "*" + simpleTypeExpr(t.Elem())
	}
	if t.Name() != "" {
		return t.String()
	}
	switch t.Kind() {
	case reflect.Slice:
		return "[]" + simpleTypeExpr(t.Elem())
	case reflect.Array:
		return fmt.Sprintf("[%d]%s", t.Len(), simpleTypeExpr(t.Elem()))
	case reflect.Map:
		return fmt.Sprintf("map[%s]%s", simpleTypeExpr(t.Key()), simpleTypeExpr(t.Elem()))
	case reflect.Struct:
		return "struct{…}"
	default:
		return t.String()
	}
}
