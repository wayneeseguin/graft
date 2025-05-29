package operators

// ModuloOperator implements the % operator with type awareness
type ModuloOperator struct {
	*ArithmeticOperatorBase
}

// NewModuloOperator creates a new modulo operator
func NewModuloOperator() ModuloOperator {
	return ModuloOperator{
		ArithmeticOperatorBase: NewArithmeticOperatorBase("%"),
	}
}

func init() {
	RegisterOp("%", NewModuloOperator())
}