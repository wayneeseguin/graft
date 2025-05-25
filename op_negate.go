package spruce

import (
	"fmt"

	"github.com/geofffranks/spruce/log"
	"github.com/starkandwayne/goutils/tree"
)

// NegateOperator ...
type NegateOperator struct{}

// Setup ...
func (NegateOperator) Setup() error {
	return nil
}

// Phase ...
func (NegateOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (NegateOperator) Dependencies(ev *Evaluator, args []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	deps := auto
	
	for _, arg := range args {
		if arg.Type == OperatorCall {
			// Get dependencies from nested operator
			nestedOp := OperatorFor(arg.Op())
			if _, ok := nestedOp.(NullOperator); !ok {
				nestedDeps := nestedOp.Dependencies(ev, arg.Args(), nil, nil)
				deps = append(deps, nestedDeps...)
			}
		} else if arg.Type == Reference {
			deps = append(deps, arg.Reference)
		}
	}
	
	return deps
}

// Run ...
func (NegateOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	log.DEBUG("running (( negate ... )) operation at $.%s", ev.Here)
	defer log.DEBUG("done with (( negate ... )) operation at $%s\n", ev.Here)

	if len(args) != 1 {
		return nil, fmt.Errorf("negate operator requires exactly one reference argument")
	}

	// Use ResolveOperatorArgument to support nested expressions
	resolved, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		log.DEBUG(" resolution failed\n error: %s", err)
		return nil, err
	}

	// Check if it's a boolean
	switch v := resolved.(type) {
	case bool:
		log.DEBUG("  resolved to boolean value: %v", v)
		return &Response{
			Type:  Replace,
			Value: !v,
		}, nil
	default:
		log.DEBUG("  resolved to non-boolean value: %T", resolved)
		return nil, fmt.Errorf("negate operator only operates on bools, got %T", resolved)
	}
}

func init() {
	RegisterOp("negate", NegateOperator{})
}
