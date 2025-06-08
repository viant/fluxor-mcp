package conversion

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToStructAndToJSON_RoundTrip(t *testing.T) {
	testCases := []struct {
		name        string
		schemaJSON  string
		payloadJSON string
	}{
		{
			name:        "simple object",
			schemaJSON:  `{ "properties": { "foo": { "type": "string" } }, "required": ["foo"], "type": "object" }`,
			payloadJSON: `{"foo":"bar"}`,
		},
		{
			name: "nested object",
			schemaJSON: `{
               "properties": {
                   "user": {
                       "type": "object",
                       "properties": {
                           "id": { "type": "integer" },
                           "name": { "type": "string" }
                       },
                       "required": ["id","name"]
                   }
               },
               "type": "object"
           }`,
			payloadJSON: `{"user":{"id":123,"name":"alice"}}`,
		},
		{
			name:        "array of numbers",
			schemaJSON:  `{ "properties": { "values": { "type": "array", "items": { "type": "number" } } }, "type": "object" }`,
			payloadJSON: `{"values":[1,2,3.5]}`,
		},
		{
			name:        "boolean field",
			schemaJSON:  `{ "properties": { "active": { "type": "boolean" } }, "type": "object" }`,
			payloadJSON: `{"active":true}`,
		},
		{
			name:        "array of objects",
			schemaJSON:  `{ "properties": { "items": { "type": "array", "items": { "type": "object", "properties": { "id": { "type": "integer" }, "name": { "type": "string" } }, "required": ["id","name"] } } }, "type": "object" }`,
			payloadJSON: `{"items":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]}`,
		},
		{
			name:        "array of arrays of integers",
			schemaJSON:  `{ "properties": { "matrix": { "type": "array", "items": { "type": "array", "items": { "type": "integer" } } } }, "type": "object" }`,
			payloadJSON: `{"matrix":[[1,2],[3,4]]}`,
		},
		{
			name:        "date-time string",
			schemaJSON:  `{ "properties": { "timestamp": { "type": "string", "format": "date-time" } }, "type": "object" }`,
			payloadJSON: `{"timestamp":"2023-06-01T12:00:00Z"}`,
		},
		{
			name: "mixed nested",
			schemaJSON: `{
               "properties": {
                   "meta": {
                       "type": "object",
                       "properties": {
                           "created": { "type": "string", "format": "date-time" },
                           "tags": { "type": "array", "items": { "type": "string" } }
                       },
                       "required": ["created","tags"]
                   },
                   "count": { "type": "integer" }
               },
               "required": ["meta"],
               "type": "object"
           }`,
			payloadJSON: `{"meta":{"created":"2023-06-01T12:00:00Z","tags":["go","test"]},"count":42}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inst, err := ToStruct([]byte(tc.schemaJSON), []byte(tc.payloadJSON))
			require.NoError(t, err)

			jsonOut, err := ToJSON(inst)
			require.NoError(t, err)

			var actual interface{}
			require.NoError(t, json.Unmarshal(jsonOut, &actual))

			var expected interface{}
			require.NoError(t, json.Unmarshal([]byte(tc.payloadJSON), &expected))

			assert.EqualValues(t, expected, actual)
		})
	}
}

// TestToStruct_OmitEmptyTags verifies that non-required fields get `,omitempty` in their JSON tags.
func TestToStruct_OmitEmptyTags(t *testing.T) {
	schemaJSON := `{
       "properties": {
           "foo": { "type": "string" },
           "bar": { "type": "integer" }
       },
       "required": ["foo"],
       "type": "object"
   }`
	inst, err := ToStruct([]byte(schemaJSON), []byte(`{"foo":"value"}`))
	require.NoError(t, err)

	typ := reflect.TypeOf(inst).Elem()
	fooField, found := typ.FieldByName("Foo")
	require.True(t, found, "expected field Foo to be present")
	fooTag := fooField.Tag.Get("json")
	barField, found := typ.FieldByName("Bar")
	require.True(t, found, "expected field Bar to be present")
	barTag := barField.Tag.Get("json")
	assert.Equal(t, "foo", fooTag)
	assert.Equal(t, "bar,omitempty", barTag)
}

// TestDescriptionAndEnumTags verifies that description and enum values are
// injected into struct tags when converting JSON Schema to Go struct type.
func TestDescriptionAndEnumTags(t *testing.T) {
	schemaJSON := `{
        "properties": {
            "status": {
                "type": "string",
                "description": "current status",
                "enum": ["open","closed"]
            }
        },
        "type": "object"
    }`

	inst, err := ToStruct([]byte(schemaJSON), []byte(`{"status":"open"}`))
	require.NoError(t, err)

	typ := reflect.TypeOf(inst).Elem()
	field, ok := typ.FieldByName("Status")
	require.True(t, ok)

	tag := string(field.Tag)
	assert.Contains(t, tag, `description:"current status"`)
	assert.Contains(t, tag, `choice:"open"`)
	assert.Contains(t, tag, `choice:"closed"`)
}
