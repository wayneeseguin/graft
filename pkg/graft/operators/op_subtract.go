package operators

// SubtractOperator implements the - operator with type awareness
type SubtractOperator struct {
	*ArithmeticOperatorBase
}

// NewSubtractOperator creates a new subtract operator
func NewSubtractOperator() SubtractOperator {
	return SubtractOperator{
		ArithmeticOperatorBase: NewArithmeticOperatorBase("-"),
	}
}

func init() {
	RegisterOp("-", NewSubtractOperator())
}
