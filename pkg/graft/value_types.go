package graft

import (
	"fmt"
	"math"
)

// Value represents a type-safe value that can hold different types
// This provides better type safety than interface{} while maintaining flexibility
type Value interface {
	// Type returns the underlying type of the value
	Type() ValueType
	
	// Raw returns the raw interface{} value for backward compatibility
	Raw() interface{}
	
	// String returns the string representation
	String() string
	
	// IsNil returns true if the value is nil
	IsNil() bool
	
	// AsString attempts to convert the value to a string
	AsString() (string, error)
	
	// AsInt attempts to convert the value to an int
	AsInt() (int, error)
	
	// AsInt64 attempts to convert the value to an int64
	AsInt64() (int64, error)
	
	// AsFloat64 attempts to convert the value to a float64
	AsFloat64() (float64, error)
	
	// AsBool attempts to convert the value to a bool
	AsBool() (bool, error)
	
	// AsSlice attempts to convert the value to a slice
	AsSlice() ([]interface{}, error)
	
	// AsMap attempts to convert the value to a map
	AsMap() (map[string]interface{}, error)
}

// ValueType represents the type of a Value
type ValueType int

const (
	NilValue ValueType = iota
	StringValue
	IntValue
	Int64Value
	Float64Value
	BoolValue
	SliceValue
	MapValue
	UnknownValue
)

// String returns the string representation of the ValueType
func (vt ValueType) String() string {
	switch vt {
	case NilValue:
		return "nil"
	case StringValue:
		return "string"
	case IntValue:
		return "int"
	case Int64Value:
		return "int64"
	case Float64Value:
		return "float64"
	case BoolValue:
		return "bool"
	case SliceValue:
		return "slice"
	case MapValue:
		return "map"
	default:
		return "unknown"
	}
}

// valueImpl is the concrete implementation of Value
type valueImpl struct {
	value interface{}
	vtype ValueType
}

// NewValue creates a new Value from an interface{}
func NewValue(v interface{}) Value {
	if v == nil {
		return &valueImpl{value: nil, vtype: NilValue}
	}
	
	switch val := v.(type) {
	case string:
		return &valueImpl{value: val, vtype: StringValue}
	case int:
		return &valueImpl{value: val, vtype: IntValue}
	case int64:
		return &valueImpl{value: val, vtype: Int64Value}
	case float64:
		return &valueImpl{value: val, vtype: Float64Value}
	case bool:
		return &valueImpl{value: val, vtype: BoolValue}
	case []interface{}:
		return &valueImpl{value: val, vtype: SliceValue}
	case map[string]interface{}:
		return &valueImpl{value: val, vtype: MapValue}
	case map[interface{}]interface{}:
		// Convert to string-keyed map for consistency
		converted := make(map[string]interface{})
		for k, v := range val {
			if key, ok := k.(string); ok {
				converted[key] = v
			} else {
				// Keep as original type if keys aren't strings
				return &valueImpl{value: val, vtype: UnknownValue}
			}
		}
		return &valueImpl{value: converted, vtype: MapValue}
	default:
		return &valueImpl{value: val, vtype: UnknownValue}
	}
}

// Type returns the ValueType
func (v *valueImpl) Type() ValueType {
	return v.vtype
}

// Raw returns the raw value
func (v *valueImpl) Raw() interface{} {
	return v.value
}

// String returns the string representation
func (v *valueImpl) String() string {
	if v.value == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", v.value)
}

// IsNil returns true if the value is nil
func (v *valueImpl) IsNil() bool {
	return v.value == nil || v.vtype == NilValue
}

// AsString converts to string
func (v *valueImpl) AsString() (string, error) {
	if v.IsNil() {
		return "", fmt.Errorf("cannot convert nil to string")
	}
	
	switch v.vtype {
	case StringValue:
		return v.value.(string), nil
	case IntValue:
		return fmt.Sprintf("%d", v.value.(int)), nil
	case Int64Value:
		return fmt.Sprintf("%d", v.value.(int64)), nil
	case Float64Value:
		return fmt.Sprintf("%g", v.value.(float64)), nil
	case BoolValue:
		return fmt.Sprintf("%t", v.value.(bool)), nil
	default:
		return fmt.Sprintf("%v", v.value), nil
	}
}

// AsInt converts to int
func (v *valueImpl) AsInt() (int, error) {
	if v.IsNil() {
		return 0, fmt.Errorf("cannot convert nil to int")
	}
	
	switch v.vtype {
	case IntValue:
		return v.value.(int), nil
	case Int64Value:
		val := v.value.(int64)
		if val > math.MaxInt32 || val < math.MinInt32 {
			return 0, fmt.Errorf("int64 value %d overflows int", val)
		}
		return int(val), nil
	case Float64Value:
		val := v.value.(float64)
		if val != float64(int(val)) {
			return 0, fmt.Errorf("float64 value %g is not an integer", val)
		}
		return int(val), nil
	default:
		return 0, fmt.Errorf("cannot convert %s to int", v.vtype)
	}
}

// AsInt64 converts to int64
func (v *valueImpl) AsInt64() (int64, error) {
	if v.IsNil() {
		return 0, fmt.Errorf("cannot convert nil to int64")
	}
	
	switch v.vtype {
	case IntValue:
		return int64(v.value.(int)), nil
	case Int64Value:
		return v.value.(int64), nil
	case Float64Value:
		val := v.value.(float64)
		if val != float64(int64(val)) {
			return 0, fmt.Errorf("float64 value %g is not an integer", val)
		}
		return int64(val), nil
	default:
		return 0, fmt.Errorf("cannot convert %s to int64", v.vtype)
	}
}

// AsFloat64 converts to float64
func (v *valueImpl) AsFloat64() (float64, error) {
	if v.IsNil() {
		return 0, fmt.Errorf("cannot convert nil to float64")
	}
	
	switch v.vtype {
	case IntValue:
		return float64(v.value.(int)), nil
	case Int64Value:
		return float64(v.value.(int64)), nil
	case Float64Value:
		return v.value.(float64), nil
	default:
		return 0, fmt.Errorf("cannot convert %s to float64", v.vtype)
	}
}

// AsBool converts to bool
func (v *valueImpl) AsBool() (bool, error) {
	if v.IsNil() {
		return false, fmt.Errorf("cannot convert nil to bool")
	}
	
	switch v.vtype {
	case BoolValue:
		return v.value.(bool), nil
	default:
		return false, fmt.Errorf("cannot convert %s to bool", v.vtype)
	}
}

// AsSlice converts to slice
func (v *valueImpl) AsSlice() ([]interface{}, error) {
	if v.IsNil() {
		return nil, fmt.Errorf("cannot convert nil to slice")
	}
	
	switch v.vtype {
	case SliceValue:
		return v.value.([]interface{}), nil
	default:
		return nil, fmt.Errorf("cannot convert %s to slice", v.vtype)
	}
}

// AsMap converts to map
func (v *valueImpl) AsMap() (map[string]interface{}, error) {
	if v.IsNil() {
		return nil, fmt.Errorf("cannot convert nil to map")
	}
	
	switch v.vtype {
	case MapValue:
		return v.value.(map[string]interface{}), nil
	default:
		return nil, fmt.Errorf("cannot convert %s to map", v.vtype)
	}
}

// TypedResponse is a more type-safe version of Response
type TypedResponse struct {
	Type  Action
	Value Value
}

// NewTypedResponse creates a new TypedResponse
func NewTypedResponse(action Action, value interface{}) *TypedResponse {
	return &TypedResponse{
		Type:  action,
		Value: NewValue(value),
	}
}

// ToLegacyResponse converts to the legacy Response format for backward compatibility
func (tr *TypedResponse) ToLegacyResponse() *Response {
	return &Response{
		Type:  tr.Type,
		Value: tr.Value.Raw(),
	}
}

// NewResponseFromLegacy creates a TypedResponse from a legacy Response
func NewResponseFromLegacy(r *Response) *TypedResponse {
	return &TypedResponse{
		Type:  r.Type,
		Value: NewValue(r.Value),
	}
}