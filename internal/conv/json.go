package conv

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Convert performs a best-effort conversion of the input value into the type
// pointed to by outPtr.
//
// Fast-path: when input is already assignable to the destination element type
// it is copied directly. Otherwise Convert falls back to JSON marshal/
// unmarshal round-trip which handles the majority of simple struct/map cases
// without requiring reflection heavy field walking at the call-site.
//
// A nil input leaves outPtrʼs value untouched (zero value).
func Convert(in any, outPtr any) error {
	if outPtr == nil {
		return fmt.Errorf("conv.Convert: outPtr cannot be nil")
	}
	v := reflect.ValueOf(outPtr)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("conv.Convert: outPtr must be a non-nil pointer")
	}

	if in == nil {
		return nil // leave zero value
	}

	// Fast-path when types match or are assignable.
	inVal := reflect.ValueOf(in)
	if inVal.Type().AssignableTo(v.Elem().Type()) {
		v.Elem().Set(inVal)
		return nil
	}

	// Fallback: JSON round-trip.
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, outPtr)
}

// ToMap converts an arbitrary input value into a map[string]interface{} using
// the same strategy as Convert. It is a convenience wrapper frequently used by
// tool proxies to coerce action arguments.
func ToMap(in any) (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := Convert(in, &m); err != nil {
		return nil, err
	}
	return m, nil
}
