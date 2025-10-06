package conversion

import (
	"reflect"
	"testing"

	schema "github.com/viant/mcp-protocol/schema"

	"github.com/stretchr/testify/assert"
)

func TestTypeFromInputSchema(t *testing.T) {
	testCases := []struct {
		name         string
		schema       schema.ToolInputSchema
		expFieldInfo map[string]reflect.Kind
	}{
		{
			name: "required string field",
			schema: schema.ToolInputSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"id": {"type": "string"},
				},
				Required: []string{"id"},
			},
			expFieldInfo: map[string]reflect.Kind{"Id": reflect.String},
		},
		{
			name: "mixed required/optional fields",
			schema: schema.ToolInputSchema{
				Type: "object",
				Properties: map[string]map[string]interface{}{
					"name":   {"type": "string"},
					"active": {"type": "boolean"},
				},
				Required: []string{"name"},
			},
			expFieldInfo: map[string]reflect.Kind{"Name": reflect.String, "Active": reflect.Bool},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rType, err := TypeFromInputSchema(tc.schema)
			assert.NoError(t, err)
			assert.EqualValues(t, reflect.Struct, rType.Kind())

			for fieldName, kind := range tc.expFieldInfo {
				field, ok := rType.FieldByName(fieldName)
				if assert.True(t, ok, "expected field %s", fieldName) {
					assert.EqualValues(t, kind, field.Type.Kind())
				}
			}
		})
	}
}
