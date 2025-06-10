package operators

import (
	"fmt"
	"math"
)

// NumericTypeHandler handles arithmetic and comparison operations for numeric types (int and float)
type NumericTypeHandler struct {
	*BaseTypeHandler
}

// NewNumericTypeHandler creates a new handler for numeric operations
func NewNumericTypeHandler() *NumericTypeHandler {
	handler := &NumericTypeHandler{
		BaseTypeHandler: NewBaseTypeHandler(100), // High priority for numeric operations
	}

	// Support int-int, int-float, float-int, and float-float combinations
	handler.AddSupportedTypes(
		TypePair{A: TypeInt, B: TypeInt},
		TypePair{A: TypeInt, B: TypeFloat},
		TypePair{A: TypeFloat, B: TypeInt},
		TypePair{A: TypeFloat, B: TypeFloat},
	)

	return handler
}

// toFloat converts a single numeric value to float64
func toFloat(val interface{}) float64 {
	switch v := val.(type) {
	case int64:
		return float64(v)
	case float64:
		return v
	default:
		// This should not happen if toNumeric was called first
		return 0
	}
}

// toInteger converts a numeric value to int64, handling floats that represent whole numbers
func toInteger(val interface{}) (int64, error) {
	switch v := val.(type) {
	case int64:
		return v, nil
	case float64:
		// Check if the float represents a whole number
		if v == math.Trunc(v) && v >= math.MinInt64 && v <= math.MaxInt64 {
			return int64(v), nil
		}
		return 0, fmt.Errorf("float %v is not a whole number or out of int64 range", v)
	default:
		return 0, fmt.Errorf("cannot convert %T to integer", val)
	}
}

// Add performs addition on numeric types
func (h *NumericTypeHandler) Add(a, b interface{}) (interface{}, error) {
	aNum, err := toNumeric(a)
	if err != nil {
		return nil, fmt.Errorf("cannot convert %v to numeric: %v", a, err)
	}

	bNum, err := toNumeric(b)
	if err != nil {
		return nil, fmt.Errorf("cannot convert %v to numeric: %v", b, err)
	}

	// If either operand is a float, result is float
	if _, ok := aNum.(float64); ok {
		return aNum.(float64) + toFloat(bNum), nil
	}
	if _, ok := bNum.(float64); ok {
		return toFloat(aNum) + bNum.(float64), nil
	}

	// Both are integers
	aInt := aNum.(int64)
	bInt := bNum.(int64)

	// Check for overflow and convert to float if necessary
	if (bInt > 0 && aInt > math.MaxInt64-bInt) || (bInt < 0 && aInt < math.MinInt64-bInt) {
		// Convert to float to handle overflow
		return float64(aInt) + float64(bInt), nil
	}

	return aInt + bInt, nil
}

// Subtract performs subtraction on numeric types
func (h *NumericTypeHandler) Subtract(a, b interface{}) (interface{}, error) {
	aNum, err := toNumeric(a)
	if err != nil {
		return nil, fmt.Errorf("cannot convert %v to numeric: %v", a, err)
	}

	bNum, err := toNumeric(b)
	if err != nil {
		return nil, fmt.Errorf("cannot convert %v to numeric: %v", b, err)
	}

	// If either operand is a float, result is float
	if _, ok := aNum.(float64); ok {
		return aNum.(float64) - toFloat(bNum), nil
	}
	if _, ok := bNum.(float64); ok {
		return toFloat(aNum) - bNum.(float64), nil
	}

	// Both are integers
	aInt := aNum.(int64)
	bInt := bNum.(int64)

	// Check for overflow
	if (bInt < 0 && aInt > math.MaxInt64+bInt) || (bInt > 0 && aInt < math.MinInt64+bInt) {
		return nil, fmt.Errorf("integer overflow in subtraction: %d - %d", aInt, bInt)
	}

	return aInt - bInt, nil
}

// Multiply performs multiplication on numeric types
func (h *NumericTypeHandler) Multiply(a, b interface{}) (interface{}, error) {
	aNum, err := toNumeric(a)
	if err != nil {
		return nil, fmt.Errorf("cannot convert %v to numeric: %v", a, err)
	}

	bNum, err := toNumeric(b)
	if err != nil {
		return nil, fmt.Errorf("cannot convert %v to numeric: %v", b, err)
	}

	// If either operand is a float, result is float
	if _, ok := aNum.(float64); ok {
		return aNum.(float64) * toFloat(bNum), nil
	}
	if _, ok := bNum.(float64); ok {
		return toFloat(aNum) * bNum.(float64), nil
	}

	// Both are integers
	aInt := aNum.(int64)
	bInt := bNum.(int64)

	// Check for overflow
	if aInt != 0 && bInt != 0 {
		result := aInt * bInt
		if result/aInt != bInt {
			return nil, fmt.Errorf("integer overflow in multiplication: %d * %d", aInt, bInt)
		}
		return result, nil
	}

	return int64(0), nil
}

// Divide performs division on numeric types (always returns float64)
func (h *NumericTypeHandler) Divide(a, b interface{}) (interface{}, error) {
	aNum, err := toNumeric(a)
	if err != nil {
		return nil, fmt.Errorf("cannot convert %v to numeric: %v", a, err)
	}

	bNum, err := toNumeric(b)
	if err != nil {
		return nil, fmt.Errorf("cannot convert %v to numeric: %v", b, err)
	}

	bFloat := toFloat(bNum)
	if bFloat == 0 {
		return nil, fmt.Errorf("division by zero")
	}

	return toFloat(aNum) / bFloat, nil
}

// Modulo performs modulo operation on numeric types
func (h *NumericTypeHandler) Modulo(a, b interface{}) (interface{}, error) {
	aNum, err := toNumeric(a)
	if err != nil {
		return nil, fmt.Errorf("cannot convert %v to numeric: %v", a, err)
	}

	bNum, err := toNumeric(b)
	if err != nil {
		return nil, fmt.Errorf("cannot convert %v to numeric: %v", b, err)
	}

	// Convert operands to integers, handling floats that represent whole numbers
	aInt, err := toInteger(aNum)
	if err != nil {
		return nil, fmt.Errorf("not an integer")
	}

	bInt, err := toInteger(bNum)
	if err != nil {
		return nil, fmt.Errorf("not an integer")
	}

	if bInt == 0 {
		return nil, fmt.Errorf("modulo by zero")
	}

	return aInt % bInt, nil
}

// Equal performs equality comparison on numeric types
func (h *NumericTypeHandler) Equal(a, b interface{}) (bool, error) {
	aNum, err := toNumeric(a)
	if err != nil {
		return false, fmt.Errorf("cannot convert %v to numeric: %v", a, err)
	}

	bNum, err := toNumeric(b)
	if err != nil {
		return false, fmt.Errorf("cannot convert %v to numeric: %v", b, err)
	}

	// Convert both to float for comparison to handle int-float comparisons
	return toFloat(aNum) == toFloat(bNum), nil
}

// NotEqual performs inequality comparison on numeric types
func (h *NumericTypeHandler) NotEqual(a, b interface{}) (bool, error) {
	equal, err := h.Equal(a, b)
	return !equal, err
}

// Less performs less-than comparison on numeric types
func (h *NumericTypeHandler) Less(a, b interface{}) (bool, error) {
	aNum, err := toNumeric(a)
	if err != nil {
		return false, fmt.Errorf("cannot convert %v to numeric: %v", a, err)
	}

	bNum, err := toNumeric(b)
	if err != nil {
		return false, fmt.Errorf("cannot convert %v to numeric: %v", b, err)
	}

	// Convert both to float for comparison
	return toFloat(aNum) < toFloat(bNum), nil
}

// Greater performs greater-than comparison on numeric types
func (h *NumericTypeHandler) Greater(a, b interface{}) (bool, error) {
	aNum, err := toNumeric(a)
	if err != nil {
		return false, fmt.Errorf("cannot convert %v to numeric: %v", a, err)
	}

	bNum, err := toNumeric(b)
	if err != nil {
		return false, fmt.Errorf("cannot convert %v to numeric: %v", b, err)
	}

	// Convert both to float for comparison
	return toFloat(aNum) > toFloat(bNum), nil
}

// LessOrEqual performs less-than-or-equal comparison on numeric types
func (h *NumericTypeHandler) LessOrEqual(a, b interface{}) (bool, error) {
	greater, err := h.Greater(a, b)
	return !greater, err
}

// GreaterOrEqual performs greater-than-or-equal comparison on numeric types
func (h *NumericTypeHandler) GreaterOrEqual(a, b interface{}) (bool, error) {
	less, err := h.Less(a, b)
	return !less, err
}
