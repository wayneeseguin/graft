package operators

// MultiplyOperator implements the * operator with type awareness
type MultiplyOperator struct {
	*ArithmeticOperatorBase
}

// NewMultiplyOperator creates a new multiply operator
func NewMultiplyOperator() MultiplyOperator {
	return MultiplyOperator{
		ArithmeticOperatorBase: NewArithmeticOperatorBase("*"),
	}
}

func init() {
	RegisterOp("*", NewMultiplyOperator())
}