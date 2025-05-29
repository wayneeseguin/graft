package operators

// AddOperator implements the + operator with type awareness
type AddOperator struct {
	*ArithmeticOperatorBase
}

// NewAddOperator creates a new add operator (for backward compatibility)
func NewAddOperator() AddOperator {
	return AddOperator{
		ArithmeticOperatorBase: NewArithmeticOperatorBase("+"),
	}
}

// Legacy performAddition function for backward compatibility
func performAddition(left, right interface{}) (interface{}, error) {
	// First try type-aware addition
	leftType := GetOperandType(left)
	rightType := GetOperandType(right)
	
	handler := GetGlobalTypeRegistry().FindHandler(leftType, rightType)
	if handler != nil {
		return handler.Add(left, right)
	}
	
	// Fall back to legacy arithmetic
	return performArithmetic(left, right, "+")
}

func init() {
	RegisterOp("+", NewAddOperator())
}
