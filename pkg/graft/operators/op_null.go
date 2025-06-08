package operators

import (
	"fmt"

	"github.com/wayneeseguin/graft/internal/utils/tree"
)

// NullOperator handles nested operator calls
type NullOperator struct {
	Missing string
}

// Setup ...
func (NullOperator) Setup() error {
	return nil
}

// Phase ...
func (NullOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (NullOperator) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (n NullOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( null ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( null ... )) operation at $.%s\n", ev.Here)

	DEBUG("null operator received %d arguments", len(args))
	for i, arg := range args {
		DEBUG("  arg %d: %s (type: %v)", i, arg.String(), arg.Type)
	}

	if len(args) > 1 {
		return nil, fmt.Errorf("null operator takes at most one argument")
	}

	// If no arguments, just return nil
	if len(args) == 0 {
		DEBUG("no arguments, returning nil")
		return &Response{
			Type:  Replace,
			Value: nil,
		}, nil
	}

	// With one argument, check if it's null/nil
	val, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		DEBUG("failed to resolve expression: %s", err)
		return nil, err
	}

	isNull := val == nil
	DEBUG("checking if value is null: %v", isNull)

	// If used as a check, return boolean
	return &Response{
		Type:  Replace,
		Value: isNull,
	}, nil
}

func init() {
	RegisterOp("null", NullOperator{})
}