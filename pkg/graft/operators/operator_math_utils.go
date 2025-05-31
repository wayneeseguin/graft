package operators

import (
	"fmt"
	"math"
)

// toNumeric converts a value to a numeric type (int64 or float64)
// Returns the numeric value and nil error if successful
// Returns nil and error if the value cannot be converted to a number
func toNumeric(val interface{}) (interface{}, error) {
	if val == nil {
		return int64(0), nil
	}

	switch v := val.(type) {
	case int:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		// Don't convert strings to numbers - this is explicit
		return nil, fmt.Errorf("cannot use string '%s' in arithmetic operation", v)
	case bool:
		// Don't convert bools to numbers
		return nil, fmt.Errorf("cannot use boolean '%v' in arithmetic operation", v)
	default:
		return nil, fmt.Errorf("cannot convert %T to numeric value", val)
	}
}

// promoteToFloat converts two numeric values to float64
// Used when at least one operand is a float or when division is performed
func promoteToFloat(a, b interface{}) (float64, float64, error) {
	aNum, err := toNumeric(a)
	if err != nil {
		return 0, 0, fmt.Errorf("left operand: %v", err)
	}

	bNum, err := toNumeric(b)
	if err != nil {
		return 0, 0, fmt.Errorf("right operand: %v", err)
	}

	var aFloat, bFloat float64

	switch v := aNum.(type) {
	case int64:
		aFloat = float64(v)
	case float64:
		aFloat = v
	}

	switch v := bNum.(type) {
	case int64:
		bFloat = float64(v)
	case float64:
		bFloat = v
	}

	return aFloat, bFloat, nil
}

// promoteToInt converts two numeric values to int64 if both are integers
// Returns error if either value is a float
func promoteToInt(a, b interface{}) (int64, int64, error) {
	aNum, err := toNumeric(a)
	if err != nil {
		return 0, 0, fmt.Errorf("left operand: %v", err)
	}

	bNum, err := toNumeric(b)
	if err != nil {
		return 0, 0, fmt.Errorf("right operand: %v", err)
	}

	aInt, aIsInt := aNum.(int64)
	bInt, bIsInt := bNum.(int64)

	if !aIsInt {
		return 0, 0, fmt.Errorf("left operand is not an integer")
	}
	if !bIsInt {
		return 0, 0, fmt.Errorf("right operand is not an integer")
	}

	return aInt, bInt, nil
}

// isNumeric checks if a value can be converted to a number
func isNumeric(val interface{}) bool {
	_, err := toNumeric(val)
	return err == nil
}

// performArithmetic executes an arithmetic operation and returns the result
// It maintains type consistency: int op int = int, anything with float = float
func performArithmetic(a, b interface{}, op string) (interface{}, error) {
	aNum, err := toNumeric(a)
	if err != nil {
		return nil, fmt.Errorf("left operand: %v", err)
	}

	bNum, err := toNumeric(b)
	if err != nil {
		return nil, fmt.Errorf("right operand: %v", err)
	}

	// Check if either operand is a float
	_, aIsFloat := aNum.(float64)
	_, bIsFloat := bNum.(float64)

	if aIsFloat || bIsFloat || op == "/" {
		// Promote to float for float operations or division
		aFloat, bFloat, err := promoteToFloat(a, b)
		if err != nil {
			return nil, err
		}

		switch op {
		case "+":
			return aFloat + bFloat, nil
		case "-":
			return aFloat - bFloat, nil
		case "*":
			return aFloat * bFloat, nil
		case "/":
			if bFloat == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			return aFloat / bFloat, nil
		case "%":
			return nil, fmt.Errorf("not an integer")
		default:
			return nil, fmt.Errorf("unknown arithmetic operator: %s", op)
		}
	} else {
		// Both are integers, keep as integer except for division
		aInt := aNum.(int64)
		bInt := bNum.(int64)

		switch op {
		case "+":
			result := aInt + bInt
			// Check for overflow
			if (result > aInt) != (bInt > 0) {
				// Overflow occurred, promote to float
				return float64(aInt) + float64(bInt), nil
			}
			return result, nil
		case "-":
			result := aInt - bInt
			// Check for overflow
			if (result < aInt) != (bInt > 0) {
				// Overflow occurred, promote to float
				return float64(aInt) - float64(bInt), nil
			}
			return result, nil
		case "*":
			result := aInt * bInt
			// Check for overflow
			if aInt != 0 && result/aInt != bInt {
				// Overflow occurred, promote to float
				return float64(aInt) * float64(bInt), nil
			}
			return result, nil
		case "%":
			if bInt == 0 {
				return nil, fmt.Errorf("modulo by zero")
			}
			return aInt % bInt, nil
		default:
			return nil, fmt.Errorf("unknown arithmetic operator: %s", op)
		}
	}
}

// checkNumericOverflow checks if a float64 is too large to be represented accurately
func checkNumericOverflow(val float64) error {
	if math.IsInf(val, 0) {
		return fmt.Errorf("numeric overflow: result is infinite")
	}
	if math.IsNaN(val) {
		return fmt.Errorf("numeric error: result is not a number (NaN)")
	}
	return nil
}
