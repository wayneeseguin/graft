package operators

import (
	"fmt"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"
)

// GrabOperator handles nested operator calls
type GrabOperator struct{}

// Setup ...
func (GrabOperator) Setup() error {
	return nil
}

// Phase ...
func (GrabOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (GrabOperator) Dependencies(ev *Evaluator, args []*Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	deps := make([]*tree.Cursor, 0, len(auto))
	deps = append(deps, auto...)

	for _, arg := range args {
		if arg != nil {
			argDeps := arg.Dependencies(ev, locs)
			deps = append(deps, argDeps...)
		}
	}

	return deps
}

// Run ...
func (GrabOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( grab ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( grab ... )) operation at $%s\n", ev.Here)

	var vals []interface{}

	for i, arg := range args {
		// Use ResolveOperatorArgument to handle nested expressions
		val, err := ResolveOperatorArgument(ev, arg)
		if err != nil {
			DEBUG("     [%d]: resolution failed\n    error: %s", i, err)
			return nil, err
		}

		// Allow nil values to pass through - they are valid values
		if val == nil {
			DEBUG("     [%d]: resolved to nil (allowed)", i)
		}

		// For LogicalOr expressions, the resolved value is already what we want
		// Don't try to double-resolve it
		if arg.Type == LogicalOr {
			DEBUG("  arg[%d]: LogicalOr expression resolved to value (type: %T)", i, val)
			vals = append(vals, val)
		} else if arg.Type == Reference {
			// Direct reference, resolve it normally
			DEBUG("  arg[%d]: direct reference $.%s", i, arg.Reference)
			resolved, err := arg.Reference.Resolve(ev.Tree)
			if err != nil {
				DEBUG("     [%d]: resolution failed\n    error: %s", i, err)
				return nil, fmt.Errorf("Unable to resolve `%s`: %s", arg.Reference, err)
			}
			vals = append(vals, resolved)
		} else if pathStr, ok := val.(string); ok && arg.Type != Literal && arg.Type != EnvVar {
			// If the resolved value is a string from an expression (not a literal or env var),
			// it might be a reference path
			cursor, err := tree.ParseCursor(pathStr)
			if err == nil {
				// It's a valid path, try to resolve it
				DEBUG("  arg[%d]: expression resolved to path '%s', attempting to grab from tree", i, pathStr)
				resolved, err := cursor.Resolve(ev.Tree)
				if err != nil {
					DEBUG("     [%d]: resolution of path failed\n    error: %s", i, err)
					return nil, fmt.Errorf("Unable to resolve `%s`: %s", pathStr, err)
				}
				vals = append(vals, resolved)
			} else {
				// Not a valid path, use the string value as-is
				DEBUG("  arg[%d]: resolved to string '%s' (not a valid path)", i, pathStr)
				vals = append(vals, pathStr)
			}
		} else {
			// For literals and other non-string values, use them directly
			DEBUG("  arg[%d]: resolved to value (type: %T)", i, val)
			vals = append(vals, val)
		}
		DEBUG("")
	}

	switch len(args) {
	case 0:
		DEBUG("  no arguments supplied to (( grab ... )) operation.  oops.")
		return nil, ansi.Errorf("no arguments specified to @c{(( grab ... ))}")

	case 1:
		DEBUG("  called with only one argument; returning value as-is")
		return &Response{
			Type:  Replace,
			Value: vals[0],
		}, nil

	default:
		DEBUG("  called with more than one arguments; flattening top-level lists into a single list")
		flat := []interface{}{}
		for i, lst := range vals {
			switch lst.(type) {
			case []interface{}:
				DEBUG("    [%d]: value is a list; flattening it out", i)
				flat = append(flat, lst.([]interface{})...)
			default:
				DEBUG("    [%d]: value is not a list; appending it as-is", i)
				flat = append(flat, lst)
			}
		}
		DEBUG("")

		return &Response{
			Type:  Replace,
			Value: flat,
		}, nil
	}
}

func init() {
	RegisterOp("grab", GrabOperator{})
}