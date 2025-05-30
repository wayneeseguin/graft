package operators

import (
	"fmt"
	"strings"
)

// StringTypeHandler handles operations for string types
type StringTypeHandler struct {
	*BaseTypeHandler
}

// NewStringTypeHandler creates a new handler for string operations
func NewStringTypeHandler() *StringTypeHandler {
	handler := &StringTypeHandler{
		BaseTypeHandler: NewBaseTypeHandler(90), // High priority for string operations
	}
	
	// Support string-string operations and string-int for multiplication
	handler.AddSupportedTypes(
		TypePair{A: TypeString, B: TypeString},
		TypePair{A: TypeString, B: TypeInt},    // For repetition
		TypePair{A: TypeInt, B: TypeString},    // For repetition (reversed)
	)
	
	return handler
}


// Add performs string concatenation
func (h *StringTypeHandler) Add(a, b interface{}) (interface{}, error) {
	// For string + string, concatenate
	if aStr, aOk := a.(string); aOk {
		if bStr, bOk := b.(string); bOk {
			return aStr + bStr, nil
		}
		// String + other type: convert to string and concatenate
		return aStr + fmt.Sprintf("%v", b), nil
	}
	
	return nil, NotImplementedError("add", a, b)
}

// Subtract is not supported for strings
func (h *StringTypeHandler) Subtract(a, b interface{}) (interface{}, error) {
	return nil, fmt.Errorf("subtract operation not supported for string type")
}

// Multiply performs string repetition when multiplying by an integer
func (h *StringTypeHandler) Multiply(a, b interface{}) (interface{}, error) {
	// String * int = repeated string
	if aStr, aOk := a.(string); aOk {
		if bInt, err := toInt(b); err == nil {
			if bInt < 0 {
				return nil, fmt.Errorf("cannot repeat string negative times: %d", bInt)
			}
			if bInt == 0 {
				return "", nil
			}
			if bInt > 10000 {
				return nil, fmt.Errorf("string repetition count too large: %d", bInt)
			}
			return strings.Repeat(aStr, int(bInt)), nil
		}
	}
	
	// Int * string = repeated string (commutative)
	if aInt, err := toInt(a); err == nil {
		if bStr, bOk := b.(string); bOk {
			if aInt < 0 {
				return nil, fmt.Errorf("cannot repeat string negative times: %d", aInt)
			}
			if aInt == 0 {
				return "", nil
			}
			if aInt > 10000 {
				return nil, fmt.Errorf("string repetition count too large: %d", aInt)
			}
			return strings.Repeat(bStr, int(aInt)), nil
		}
	}
	
	return nil, NotImplementedError("multiply", a, b)
}

// Divide is not supported for strings
func (h *StringTypeHandler) Divide(a, b interface{}) (interface{}, error) {
	return nil, fmt.Errorf("divide operation not supported for string type")
}

// Modulo is not supported for strings
func (h *StringTypeHandler) Modulo(a, b interface{}) (interface{}, error) {
	return nil, fmt.Errorf("modulo operation not supported for string type")
}

// Equal performs string equality comparison
func (h *StringTypeHandler) Equal(a, b interface{}) (bool, error) {
	aStr, aOk := a.(string)
	bStr, bOk := b.(string)
	
	if aOk && bOk {
		return aStr == bStr, nil
	}
	
	// If one is string and other isn't, they're not equal
	if aOk || bOk {
		return false, nil
	}
	
	return false, NotImplementedError("equal", a, b)
}

// NotEqual performs string inequality comparison
func (h *StringTypeHandler) NotEqual(a, b interface{}) (bool, error) {
	equal, err := h.Equal(a, b)
	return !equal, err
}

// Less performs lexicographic comparison
func (h *StringTypeHandler) Less(a, b interface{}) (bool, error) {
	aStr, aOk := a.(string)
	bStr, bOk := b.(string)
	
	if aOk && bOk {
		return aStr < bStr, nil
	}
	
	return false, NotImplementedError("less", a, b)
}

// Greater performs lexicographic comparison
func (h *StringTypeHandler) Greater(a, b interface{}) (bool, error) {
	aStr, aOk := a.(string)
	bStr, bOk := b.(string)
	
	if aOk && bOk {
		return aStr > bStr, nil
	}
	
	return false, NotImplementedError("greater", a, b)
}

// LessOrEqual performs lexicographic comparison
func (h *StringTypeHandler) LessOrEqual(a, b interface{}) (bool, error) {
	greater, err := h.Greater(a, b)
	return !greater, err
}

// GreaterOrEqual performs lexicographic comparison
func (h *StringTypeHandler) GreaterOrEqual(a, b interface{}) (bool, error) {
	less, err := h.Less(a, b)
	return !less, err
}

// CanHandle checks if this handler can handle the given type combination
func (h *StringTypeHandler) CanHandle(aType, bType OperandType) bool {
	// Handle string with any type for concatenation
	if aType == TypeString || bType == TypeString {
		return true
	}
	return h.BaseTypeHandler.CanHandle(aType, bType)
}

// toInt converts a value to int64 if possible
func toInt(val interface{}) (int64, error) {
	switch v := val.(type) {
	case int:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int", val)
	}
}