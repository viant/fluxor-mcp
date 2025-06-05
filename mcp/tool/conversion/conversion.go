package conversion

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/viant/fluxor/model/types"
	schema "github.com/viant/mcp-protocol/schema"
	"github.com/viant/x"
)

func BuildSchema(sig *types.Signature) (schema.Tool, error) {
	var inputSchema schema.ToolInputSchema
	var sample any
	if sig.Input.Kind() == reflect.Pointer {
		sample = reflect.New(sig.Input.Elem()).Interface()
	} else {
		sample = reflect.New(sig.Input).Interface()
	}
	if err := inputSchema.Load(sample); err != nil {
		return schema.Tool{}, fmt.Errorf("failed to build input schema for %s: %w", sig.Name, err)
	}
	var props map[string]map[string]interface{}
	var required []string
	if sig.Output.Kind() == reflect.Pointer {
		props, required = schema.StructToProperties(sig.Output.Elem())
	} else {
		props, required = schema.StructToProperties(sig.Output)
	}
	outputSchema := &schema.ToolOutputSchema{Properties: props, Required: required, Type: "object"}
	desc := sig.Description
	return schema.Tool{Name: sig.Name, Description: &desc, InputSchema: inputSchema, OutputSchema: outputSchema}, nil
}

// typeRegistry holds dynamic Go types generated from JSON Schemas.
var typeRegistry = x.NewRegistry()

// Registry returns the registry of dynamic types.
func Registry() *x.Registry {
	return typeRegistry
}

// RegisterType registers a Go type for schema-based conversion.
func RegisterType(t reflect.Type, options ...x.Option) {
	typeRegistry.Register(x.NewType(t, options...))
}

// ------------------------------------------------------------------
//  JSON Schema → Go reflect.Type helpers
// ------------------------------------------------------------------

// TypeFromInputSchema converts an Server ToolInputSchema into a dynamically
// generated Go struct type. The resulting type is suitable for
// types.Signature.Input so that subsequent calls to StructToProperties (for
// example in service/tool registry building) can introspect the structure and
// recreate the original JSON schema.
//
// When the schema does not define any properties an empty struct type is
// returned. Using a struct – even if empty – is crucial because
// StructToProperties panics for non-struct kinds (e.g. map[string]interface{}).
func TypeFromInputSchema(inputSchema schema.ToolInputSchema) (reflect.Type, error) {
	if len(inputSchema.Properties) == 0 {
		return reflect.StructOf([]reflect.StructField{}), nil
	}

	fields, err := buildFields(inputSchema.Properties, inputSchema.Required)
	if err != nil {
		return nil, err
	}

	t := reflect.StructOf(fields)
	// Keep the dynamically generated type in the registry so it can be reused
	// elsewhere (for example during JSON → struct conversions).
	RegisterType(t)
	return t, nil
}

// TypeFromOutputSchema behaves like TypeFromInputSchema but accepts a
// ToolOutputSchema instance. Having a dedicated helper improves readability at
// the call-site while sharing the implementation logic.
func TypeFromOutputSchema(outputSchema schema.ToolOutputSchema) (reflect.Type, error) {
	if len(outputSchema.Properties) == 0 {
		return reflect.StructOf([]reflect.StructField{}), nil
	}

	fields, err := buildFields(outputSchema.Properties, outputSchema.Required)
	if err != nil {
		return nil, err
	}

	t := reflect.StructOf(fields)
	RegisterType(t)
	return t, nil
}

func buildFields(props map[string]map[string]interface{}, required []string) ([]reflect.StructField, error) {
	keys := make([]string, 0, len(props))
	for name := range props {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	var fields []reflect.StructField
	requiredSet := make(map[string]struct{}, len(required))
	for _, n := range required {
		requiredSet[n] = struct{}{}
	}
	for _, name := range keys {
		def := props[name]
		fieldType, err := goTypeFromDef(def)
		if err != nil {
			return nil, fmt.Errorf("failed to determine type for field %q: %w", name, err)
		}
		tagName := name
		if _, ok := requiredSet[name]; !ok {
			tagName += ",omitempty"
		}
		fields = append(fields, reflect.StructField{
			Name: strings.Title(name),
			Type: fieldType,
			Tag:  reflect.StructTag(fmt.Sprintf("json:%q", tagName)),
		})
	}
	return fields, nil
}

func goTypeFromDef(def map[string]interface{}) (reflect.Type, error) {
	rawType, _ := def["type"]
	var typeStr string
	switch v := rawType.(type) {
	case string:
		typeStr = v
	case []interface{}:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				typeStr = s
			}
		}
	}
	switch typeStr {
	case "string":
		if format, ok := def["format"].(string); ok && (format == "date-time" || format == "date") {
			return reflect.TypeOf(time.Time{}), nil
		}
		return reflect.TypeOf(""), nil
	case "integer":
		return reflect.TypeOf(int64(0)), nil
	case "number":
		return reflect.TypeOf(float64(0)), nil
	case "boolean":
		return reflect.TypeOf(true), nil
	case "object":
		nested := map[string]map[string]interface{}{}
		var nestedRequired []string
		if rawReq, ok := def["required"].([]interface{}); ok {
			for _, raw := range rawReq {
				if s, ok := raw.(string); ok {
					nestedRequired = append(nestedRequired, s)
				}
			}
		}
		if raw, ok := def["properties"].(map[string]interface{}); ok {
			for k, v := range raw {
				if m, ok := v.(map[string]interface{}); ok {
					nested[k] = m
				}
			}
		}
		fields, err := buildFields(nested, nestedRequired)
		if err != nil {
			return nil, err
		}
		nestedType := reflect.StructOf(fields)
		RegisterType(nestedType)
		return nestedType, nil
	case "array":
		if raw, ok := def["items"].(map[string]interface{}); ok {
			itemType, err := goTypeFromDef(raw)
			if err != nil {
				return nil, err
			}
			return reflect.SliceOf(itemType), nil
		}
		return reflect.SliceOf(reflect.TypeOf(new(interface{})).Elem()), nil
	default:
		return reflect.TypeOf(new(interface{})).Elem(), nil
	}
}

// ToStruct builds a Go struct type from JSON Schema and unmarshals the raw payload.
// ToStruct builds a Go struct type from JSON Schema and unmarshals the raw payload.
// It returns the populated Go value as interface{} (a pointer to the generated struct).
// Returning the concrete value instead of reflect.Value makes the helper easier to use
// by callers that do not need to work with reflection directly.
func ToStruct(schemaJSON, payloadJSON []byte) (any, error) {
	var schemaInput schema.ToolInputSchema
	if err := json.Unmarshal(schemaJSON, &schemaInput); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}
	// build Go struct fields from JSON schema properties
	fields, err := buildFields(schemaInput.Properties, schemaInput.Required)
	if err != nil {
		return nil, err
	}
	structType := reflect.StructOf(fields)
	RegisterType(structType)
	instPtr := reflect.New(structType)
	if err := json.Unmarshal(payloadJSON, instPtr.Interface()); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	return instPtr.Interface(), nil
}

// ToJSON marshals a Go value (typically returned by ToStruct) back to JSON.
func ToJSON(val any) ([]byte, error) {
	if val == nil {
		return nil, fmt.Errorf("invalid value: nil")
	}
	return json.Marshal(val)
}
