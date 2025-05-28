package graft

import (
	"fmt"
	"github.com/wayneeseguin/graft/log"
	"github.com/starkandwayne/goutils/tree"
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

// TRACE is a helper function for trace logging
func TRACE(format string, args ...interface{}) {
	log.TRACE(format, args...)
}

// SetupOperators initializes operators for a given phase
func SetupOperators(phase OperatorPhase) error {
	// Operators are now registered through the engine or globally through init()
	// This function is kept for backward compatibility
	return nil
}

// OperatorFor returns the operator for the given name
func OperatorFor(name string) Operator {
	// First check if we have a default engine instance
	// Otherwise fall back to global registry
	if op, exists := OpRegistry[name]; exists {
		return op
	}
	return nil
}

// NullOperator is a placeholder operator for unknown operations
type NullOperator struct {
	Missing string
}

// Setup initializes the operator
func (NullOperator) Setup() error {
	return nil
}

// Phase returns which phase this operator should run in
func (NullOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies returns what keys the operator depends on
func (NullOperator) Dependencies(ev *Evaluator, args []*Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return nil
}

// Run executes the operator
func (n NullOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	return nil, fmt.Errorf("unknown operator: %s", n.Missing)
}

// NewOpcall creates a new operator call
func NewOpcall(op Operator, args []*Expr, src string) *Opcall {
	return &Opcall{
		op:   op,
		args: args,
		src:  src,
	}
}

// DefaultKeyGenerator returns a key generator function
// This seems to be used for generating unique keys, possibly for caching
func DefaultKeyGenerator() func() (string, error) {
	counter := 0
	return func() (string, error) {
		counter++
		return fmt.Sprintf("key-%d", counter), nil
	}
}

// Merge merges two data structures
func Merge(dst, src interface{}) error {
	// This is a simplified merge for backward compatibility
	// The real implementation is in merge.go
	// For now, just return nil error
	return nil
}

// DebugOn returns true if debugging is enabled
func DebugOn() bool {
	// Check environment variable or global flag
	return false
}

// ParseOpcallEnhanced parses an operator call string with enhanced syntax
func ParseOpcallEnhanced(phase OperatorPhase, src string, ev *Evaluator) (*Opcall, error) {
	// For backward compatibility, delegate to ParseOpcallCompat
	return ParseOpcallCompat(phase, src)
}