package operators

import (
	"fmt"
	"reflect"

	"github.com/starkandwayne/goutils/tree"
)

// BooleanAndOperator implements logical AND (&&)
type BooleanAndOperator struct{}

// Setup initializes the operator
func (BooleanAndOperator) Setup() error {
	return nil
}

// Phase returns the operator phase
func (BooleanAndOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies returns operator dependencies
func (BooleanAndOperator) Dependencies(_ *Evaluator, args []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	deps := make([]*tree.Cursor, 0)
	for _, arg := range args {
		if arg.Type == Reference && arg.Reference != nil {
			deps = append(deps, arg.Reference)
		}
	}
	return append(auto, deps...)
}

// Run executes the logical AND
func (BooleanAndOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("&& operator requires exactly 2 arguments, got %d", len(args))
	}

	// Short-circuit evaluation: evaluate left first
	leftResp, err := EvaluateExpr(args[0], ev)
	if err != nil {
		return nil, err
	}

	if !isTruthy(leftResp.Value) {
		// Left is falsy, return false without evaluating right
		return &Response{
			Type:  Replace,
			Value: false,
		}, nil
	}

	// Left is truthy, evaluate right
	rightResp, err := EvaluateExpr(args[1], ev)
	if err != nil {
		return nil, err
	}

	return &Response{
		Type:  Replace,
		Value: isTruthy(rightResp.Value),
	}, nil
}

// FallbackOperator implements fallback behavior (||) - returns first non-nil value
type FallbackOperator struct{}

// Setup initializes the operator
func (FallbackOperator) Setup() error {
	return nil
}

// Phase returns the operator phase
func (FallbackOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies returns operator dependencies
func (FallbackOperator) Dependencies(_ *Evaluator, args []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	deps := make([]*tree.Cursor, 0)
	for _, arg := range args {
		if arg.Type == Reference && arg.Reference != nil {
			deps = append(deps, arg.Reference)
		}
	}
	return append(auto, deps...)
}

// Run executes the fallback operator
func (FallbackOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("|| operator requires exactly 2 arguments, got %d", len(args))
	}

	// Evaluate left first, but allow for missing values
	leftResp, err := EvaluateExpr(args[0], ev)
	if err != nil {
		// If left evaluation fails (e.g., missing key), try right
		return EvaluateExpr(args[1], ev)
	}

	// If evaluation succeeded, return the result even if it's nil
	// This implements: "|| operator treats nil as a found value"
	// A successful evaluation with nil value is different from a failed evaluation
	return leftResp, nil
}

// isTruthy determines if a value is truthy
// false, nil, 0, "", [], {} are falsy
// Everything else is truthy
func isTruthy(v interface{}) bool {
	if v == nil {
		return false
	}

	// Check for boolean
	if b, ok := v.(bool); ok {
		return b
	}

	// Check for numeric zero
	if num, ok := toFloat64(v); ok {
		return num != 0
	}

	// Check for empty string
	if s, ok := v.(string); ok {
		return s != ""
	}

	// Check for empty slice/array
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return rv.Len() > 0
	case reflect.Map:
		return rv.Len() > 0
	}

	// Everything else is truthy
	return true
}

// Register boolean operators
func init() {
	// Use type-aware boolean operators
	RegisterOp("&&", NewTypeAwareAndOperator())
	RegisterOp("||", FallbackOperator{}) // Keep fallback operator for now - TODO: decide on || semantics
}
