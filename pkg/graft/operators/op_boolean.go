package operators

import (
	"fmt"
	"reflect"

	"github.com/wayneeseguin/graft/internal/utils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// BooleanAndOperator implements logical AND (&&)
type BooleanAndOperator struct{}

// Setup initializes the operator
func (BooleanAndOperator) Setup() error {
	return nil
}

// Phase returns the operator phase
func (BooleanAndOperator) Phase() graft.OperatorPhase {
	return graft.EvalPhase
}

// Dependencies returns operator dependencies
func (BooleanAndOperator) Dependencies(_ *graft.Evaluator, args []*graft.Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	deps := make([]*tree.Cursor, 0)
	for _, arg := range args {
		if arg.Type == graft.Reference && arg.Reference != nil {
			deps = append(deps, arg.Reference)
		}
	}
	return append(auto, deps...)
}

// Run executes the logical AND
func (BooleanAndOperator) Run(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("&& operator requires exactly 2 arguments, got %d", len(args))
	}

	// Short-circuit evaluation: evaluate left first
	leftResp, err := graft.EvaluateExpr(args[0], ev)
	if err != nil {
		return nil, err
	}

	if !isTruthy(leftResp.Value) {
		// Left is falsy, return false without evaluating right
		return &graft.Response{
			Type:  graft.Replace,
			Value: false,
		}, nil
	}

	// Left is truthy, evaluate right
	rightResp, err := graft.EvaluateExpr(args[1], ev)
	if err != nil {
		return nil, err
	}

	return &graft.Response{
		Type:  graft.Replace,
		Value: isTruthy(rightResp.Value),
	}, nil
}

// OrElseOperator implements or-else behavior (||) - returns first non-nil value
type OrElseOperator struct{}

// Setup initializes the operator
func (OrElseOperator) Setup() error {
	return nil
}

// Phase returns the operator phase
func (OrElseOperator) Phase() graft.OperatorPhase {
	return graft.EvalPhase
}

// Dependencies returns operator dependencies
func (OrElseOperator) Dependencies(ev *graft.Evaluator, args []*graft.Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	deps := make([]*tree.Cursor, 0, len(auto))
	deps = append(deps, auto...)

	// Always collect all dependencies for cycle detection
	// The optimization logic (skipping dependencies after literals) should be
	// handled at a higher level, not here.
	for _, arg := range args {
		if arg != nil {
			deps = append(deps, arg.Dependencies(ev, locs)...)
		}
	}

	return deps
}

// Run executes the or-else operator
func (OrElseOperator) Run(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("|| operator requires exactly 2 arguments, got %d", len(args))
	}

	// Evaluate left first, but allow for missing values
	leftResp, err := graft.EvaluateExpr(args[0], ev)
	if err != nil {
		// If left evaluation fails (e.g., missing key), try right
		return graft.EvaluateExpr(args[1], ev)
	}

	// If left evaluation succeeded but the value is nil, try right
	if leftResp.Value == nil {
		return graft.EvaluateExpr(args[1], ev)
	}

	// Left evaluation succeeded and value is not nil, return it
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
	switch num := v.(type) {
	case int:
		return num != 0
	case int8:
		return num != 0
	case int16:
		return num != 0
	case int32:
		return num != 0
	case int64:
		return num != 0
	case uint:
		return num != 0
	case uint8:
		return num != 0
	case uint16:
		return num != 0
	case uint32:
		return num != 0
	case uint64:
		return num != 0
	case float32:
		return num != 0
	case float64:
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
	RegisterOp("||", &OrElseOperator{}) // Use or-else operator, not boolean OR
	RegisterOp("!", NewTypeAwareNotOperator())
}
