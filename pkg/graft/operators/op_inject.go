package operators

import (
	"fmt"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"
)

// InjectOperator ...
type InjectOperator struct{}

// Setup ...
func (InjectOperator) Setup() error {
	return nil
}

// Phase ...
func (InjectOperator) Phase() OperatorPhase {
	return MergePhase
}

// Dependencies ...
func (InjectOperator) Dependencies(ev *Evaluator, args []*Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	l := []*tree.Cursor{}

	for _, arg := range args {
		if arg.Type == Reference {
			for _, other := range locs {
				canon, err := arg.Reference.Canonical(ev.Tree)
				if err != nil {
					return []*tree.Cursor{}
				}
				if other.Under(canon) {
					l = append(l, other)
				}
			}
		} else if arg.Type == OperatorCall {
			// Get dependencies from nested operator
			nestedOp := OperatorFor(arg.Op())
			if _, ok := nestedOp.(NullOperator); !ok {
				nestedDeps := nestedOp.Dependencies(ev, arg.Args(), locs, auto)
				l = append(l, nestedDeps...)
			}
		}
	}

	for _, dep := range auto {
		l = append(l, dep)
	}

	return l
}

// Run ...
func (InjectOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( inject ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( inject ... )) operation at $%s\n", ev.Here)

	var vals []map[interface{}]interface{}

	for i, arg := range args {
		// Special handling for references vs expressions
		if arg.Type == Reference {
			// Direct reference - resolve it directly
			DEBUG("  arg[%d]: trying to resolve reference $.%s", i, arg.Reference)
			s, err := arg.Reference.Resolve(ev.Tree)
			if err != nil {
				DEBUG("     [%d]: resolution failed\n    error: %s", i, err)
				return nil, err
			}

			m, ok := s.(map[interface{}]interface{})
			if !ok {
				DEBUG("     [%d]: resolved to something that is not a map.  that is unacceptable.", i)
				return nil, ansi.Errorf("@c{%s} @R{is not a map}", arg.Reference)
			}

			DEBUG("     [%d]: resolved to a map; appending to the list of maps to merge/inject", i)
			// Deep copy the map to avoid modifying the original
			vals = append(vals, DeepCopyMap(m))
		} else {
			// Use ResolveOperatorArgument for all other expressions (including nested operators)
			val, err := ResolveOperatorArgument(ev, arg)
			if err != nil {
				DEBUG("  arg[%d]: failed to resolve expression to a concrete value", i)
				DEBUG("     [%d]: error was: %s", i, err)
				return nil, err
			}

			if val == nil {
				DEBUG("  arg[%d]: resolved to nil", i)
				return nil, fmt.Errorf("inject operator argument cannot be nil")
			}

			// Check if the resolved value is a map
			m, ok := val.(map[interface{}]interface{})
			if !ok {
				// Also check for map[string]interface{}
				if sm, ok := val.(map[string]interface{}); ok {
					// Convert to map[interface{}]interface{}
					m = make(map[interface{}]interface{})
					for k, v := range sm {
						m[k] = v
					}
				} else {
					DEBUG("     [%d]: resolved to something that is not a map", i)
					return nil, ansi.Errorf("@R{inject operator argument must resolve to a map}")
				}
			}

			DEBUG("     [%d]: resolved to a map; appending to the list of maps to merge/inject", i)
			// Deep copy the map to avoid modifying the original
			vals = append(vals, DeepCopyMap(m))
		}
		DEBUG("")
	}

	switch len(vals) {
	case 0:
		DEBUG("  no arguments supplied to (( inject ... )) operation.  oops.")
		return nil, ansi.Errorf("no arguments specified to @c{(( inject ... ))}")

	default:
		DEBUG("  merging found maps into a single map to be injected")
		// Merge all maps together
		merged := make(map[interface{}]interface{})
		for _, val := range vals {
			err := Merge(merged, val)
			if err != nil {
				DEBUG("  failed: %s\n", err)
				return nil, err
			}
		}
		return &Response{
			Type:  Inject,
			Value: merged,
		}, nil
	}
}

func init() {
	RegisterOp("inject", InjectOperator{})
}
