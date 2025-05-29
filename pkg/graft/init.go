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
	// Return a NullOperator for unknown operators
	return NullOperator{Missing: name}
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
	// For unknown operators, return the original operator call string unchanged
	// This allows the template to be processed again later or remain as-is
	
	// Reconstruct the original operator call
	var argStrings []string
	for _, arg := range args {
		if arg.Type == Literal {
			if s, ok := arg.Literal.(string); ok {
				argStrings = append(argStrings, fmt.Sprintf(`"%s"`, s))
			} else {
				argStrings = append(argStrings, fmt.Sprintf("%v", arg.Literal))
			}
		} else {
			// For non-literal args, use a placeholder
			argStrings = append(argStrings, "...")
		}
	}
	
	var argsStr string
	if len(argStrings) > 0 {
		argsStr = " " + fmt.Sprintf("%v", argStrings[0])
		for _, arg := range argStrings[1:] {
			argsStr += " " + arg
		}
	}
	
	originalCall := fmt.Sprintf("(( %s%s ))", n.Missing, argsStr)
	
	return &Response{
		Type:  Replace,
		Value: originalCall,
	}, nil
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
	// Deep merge implementation for maps
	dstMap, dstOk := dst.(map[interface{}]interface{})
	srcMap, srcOk := src.(map[interface{}]interface{})
	
	if !dstOk || !srcOk {
		return fmt.Errorf("Merge: both arguments must be maps")
	}
	
	// Deep merge all keys from src to dst
	for k, srcVal := range srcMap {
		if dstVal, exists := dstMap[k]; exists {
			// If both are maps, merge recursively
			if dstSubMap, dstIsMap := dstVal.(map[interface{}]interface{}); dstIsMap {
				if srcSubMap, srcIsMap := srcVal.(map[interface{}]interface{}); srcIsMap {
					err := Merge(dstSubMap, srcSubMap)
					if err != nil {
						return err
					}
					continue
				}
			}
		}
		// Otherwise just copy the value
		dstMap[k] = srcVal
	}
	
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