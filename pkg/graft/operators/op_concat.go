package operators

import (
	"fmt"
	"strings"

	"github.com/wayneeseguin/graft/internal/utils/ansi"
	"github.com/wayneeseguin/graft/internal/utils/tree"
)

// ConcatOperator handles nested operator calls
type ConcatOperator struct{}

// Setup ...
func (ConcatOperator) Setup() error {
	return nil
}

// Phase ...
func (ConcatOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (ConcatOperator) Dependencies(ev *Evaluator, args []*Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	// Include dependencies from nested operator calls
	deps := auto

	for _, arg := range args {
		if arg.Type == OperatorCall {
			// Get dependencies from nested operator
			nestedOp := OperatorFor(arg.Op())
			if _, ok := nestedOp.(NullOperator); !ok {
				nestedDeps := nestedOp.Dependencies(ev, arg.Args(), locs, auto)
				deps = append(deps, nestedDeps...)
			}
		}
	}

	return deps
}

// Run ...
func (ConcatOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( concat ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( concat ... )) operation at $%s\n", ev.Here)

	l := GetStringSlice()
	defer PutStringSlice(l)

	if len(args) < 2 {
		return nil, fmt.Errorf("concat operator requires at least two arguments")
	}

	for i, arg := range args {
		// Use the helper to resolve arguments, including nested operators
		v, err := ResolveOperatorArgument(ev, arg)
		if err != nil {
			DEBUG("  arg[%d]: failed to resolve expression to a concrete value", i)
			DEBUG("     [%d]: error was: %s", i, err)
			return nil, err
		}

		// Convert to string
		var stringVal string
		switch val := v.(type) {
		case string:
			stringVal = val
		case []interface{}:
			stringSlice := make([]string, len(val))
			for j, elem := range val {
				stringSlice[j] = fmt.Sprintf("%v", elem)
			}
			stringVal = strings.Join(stringSlice, "")
		default:
			stringVal = fmt.Sprintf("%v", v)
		}

		DEBUG("  arg[%d]: using '%s'", i, stringVal)
		*l = append(*l, stringVal)
	}

	DEBUG("  result: %s", ansi.Sprintf("@c{%s}", strings.Join(*l, "")))
	return &Response{
		Type:  Replace,
		Value: strings.Join(*l, ""),
	}, nil
}

func init() {
	RegisterOp("concat", ConcatOperator{})
}
