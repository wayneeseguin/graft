package graft

import (
	"github.com/wayneeseguin/graft/log"
)

func init() {
	// Initialize operator registry if not already done
	if OpRegistry == nil {
		OpRegistry = make(map[string]Operator)
	}
	
	// TODO: Implement and register arithmetic operators
	// RegisterOp("+", AddOperator{})
	// RegisterOp("-", SubtractOperator{})
	// RegisterOp("*", MultiplyOperator{})
	// RegisterOp("/", DivideOperator{})
	// RegisterOp("%", ModuloOperator{})
	
	// TODO: Implement and register comparison operators
	// RegisterOp("==", ComparisonOperator{op: "=="})
	// RegisterOp("!=", ComparisonOperator{op: "!="})
	// RegisterOp("<", ComparisonOperator{op: "<"})
	// RegisterOp(">", ComparisonOperator{op: ">"})
	// RegisterOp("<=", ComparisonOperator{op: "<="})
	// RegisterOp(">=", ComparisonOperator{op: ">="})
	
	// TODO: Implement and register boolean operators
	// RegisterOp("&&", BooleanAndOperator{})
	// NOTE: || is not registered as an operator - it's handled specially as alternation in expressions
	// RegisterOp("!", NegationOperator{})
	
	// TODO: Implement and register ternary operator
	// RegisterOp("?:", TernaryOperator{})
	
	log.DEBUG("Operators initialized")
}

// RegisterOp is a helper function to register operators
func RegisterOp(name string, op Operator) {
	OpRegistry[name] = op
}

// DEBUG is a helper function for debug logging
func DEBUG(format string, args ...interface{}) {
	log.DEBUG(format, args...)
}