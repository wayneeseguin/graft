package operators

import (
	"fmt"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"
)

// GrabOperatorEnhanced is an enhanced version that supports nested expressions
type GrabOperatorEnhanced struct{}

// Setup ...
func (GrabOperatorEnhanced) Setup() error {
	return nil
}

// Phase ...
func (GrabOperatorEnhanced) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (GrabOperatorEnhanced) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (GrabOperatorEnhanced) Run(ev *Evaluator, args []*Expr) (*Response, error) {
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

		if val == nil {
			DEBUG("     [%d]: resolved to nil", i)
			return nil, fmt.Errorf("grab operator argument resolved to nil")
		}

		// Check if this is a direct reference that should be dereferenced
		if arg.Type == Reference {
			// Direct reference, resolve it normally
			DEBUG("  arg[%d]: direct reference $.%s", i, arg.Reference)
			resolved, err := arg.Reference.Resolve(ev.Tree)
			if err != nil {
				DEBUG("     [%d]: resolution failed\n    error: %s", i, err)
				return nil, fmt.Errorf("Unable to resolve `%s`: %s", arg.Reference, err)
			}
			vals = append(vals, resolved)
		} else if pathStr, ok := val.(string); ok && arg.Type != Literal {
			// If the resolved value is a string from an expression (not a literal),
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
	// Don't register in init - let EnableEnhancedGrab handle it
}
