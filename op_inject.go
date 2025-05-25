package spruce

import (
	"fmt"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"

	. "github.com/geofffranks/spruce/log"
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
		// For inject, we need to handle this specially since it expects references
		if arg.Type == OperatorCall {
			// Use ResolveOperatorArgument for nested expressions
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
			vals = append(vals, m)

		} else {
			// Handle regular references
			v, err := arg.Resolve(ev.Tree)
			if err != nil {
				DEBUG("  arg[%d]: failed to resolve expression to a concrete value", i)
				DEBUG("     [%d]: error was: %s", i, err)
				return nil, err
			}

			switch v.Type {
			case Literal:
				DEBUG("  arg[%d]: found string literal '%s'", i, v.Literal)
				DEBUG("           (inject operator only handles references to other parts of the YAML tree)")
				return nil, fmt.Errorf("inject operator only accepts key reference arguments")

			case Reference:
				DEBUG("  arg[%d]: trying to resolve reference $.%s", i, v.Reference)
				s, err := v.Reference.Resolve(ev.Tree)
				if err != nil {
					DEBUG("     [%d]: resolution failed\n    error: %s", i, err)
					return nil, err
				}

				m, ok := s.(map[interface{}]interface{})
				if !ok {
					DEBUG("     [%d]: resolved to something that is not a map.  that is unacceptable.", i)
					return nil, ansi.Errorf("@c{%s} @R{is not a map}", v.Reference)
				}

				DEBUG("     [%d]: resolved to a map; appending to the list of maps to merge/inject", i)
				vals = append(vals, m)

			default:
				DEBUG("  arg[%d]: I don't know what to do with '%v'", i, arg)
				return nil, fmt.Errorf("inject operator only accepts key reference arguments")
			}
		}
		DEBUG("")
	}

	switch len(vals) {
	case 0:
		DEBUG("  no arguments supplied to (( inject ... )) operation.  oops.")
		return nil, ansi.Errorf("no arguments specified to @c{(( inject ... ))}")

	default:
		DEBUG("  merging found maps into a single map to be injected")
		merged, err := Merge(vals...)
		if err != nil {
			DEBUG("  failed: %s\n", err)
			return nil, err
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
