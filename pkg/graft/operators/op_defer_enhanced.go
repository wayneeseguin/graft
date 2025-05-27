package operators

import (
	"fmt"
	"strings"

	"github.com/starkandwayne/goutils/tree"
)

// DeferOperatorEnhanced is an enhanced version that supports nested expressions
type DeferOperatorEnhanced struct{}

// Setup ...
func (DeferOperatorEnhanced) Setup() error {
	return nil
}

// Phase ...
func (DeferOperatorEnhanced) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (DeferOperatorEnhanced) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (DeferOperatorEnhanced) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( defer ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( defer ... )) operation at $.%s\n", ev.Here)

	if len(args) == 0 {
		return nil, fmt.Errorf("Defer has no arguments - what are you deferring?")
	}

	// The defer operator's purpose is to preserve the expression for later evaluation
	// Build the deferred expression from all arguments

	components := []string{"(("} // Join these with spaces at the end

	for _, arg := range args {
		components = append(components, arg.String())
	}
	components = append(components, "))")

	deferred := strings.Join(components, " ")
	DEBUG("deferring expression: %s", deferred)

	return &Response{
		Type:  Replace,
		Value: deferred,
	}, nil
}

// EnableEnhancedDefer enables the enhanced defer operator
func EnableEnhancedDefer() {
	RegisterOp("defer", DeferOperatorEnhanced{})
}

func init() {
	// Don't register in init - let EnableEnhancedDefer handle it
}

// reconstructExpr reconstructs a string representation of an expression
func reconstructExpr(e *Expr) string {
	if e == nil {
		return ""
	}

	switch e.Type {
	case Literal:
		if e.Literal == nil {
			return "nil"
		}
		if s, ok := e.Literal.(string); ok {
			return fmt.Sprintf(`"%s"`, s)
		}
		return fmt.Sprintf("%v", e.Literal)

	case Reference:
		if e.Reference != nil {
			return e.Reference.String()
		}
		return ""

	case EnvVar:
		return fmt.Sprintf("$%s", e.Name)

	case LogicalOr:
		left := reconstructExpr(e.Left)
		right := reconstructExpr(e.Right)
		return fmt.Sprintf("%s || %s", left, right)

	case OperatorCall:
		op := e.Op()
		args := e.Args()
		if op == "" {
			return e.String() // fallback
		}
		argStrs := make([]string, len(args))
		for i, arg := range args {
			argStrs[i] = reconstructExpr(arg)
		}
		if len(argStrs) == 0 {
			return op
		}
		return fmt.Sprintf("%s %s", op, strings.Join(argStrs, " "))

	default:
		return e.String()
	}
}
