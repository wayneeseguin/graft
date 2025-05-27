package operators

import (
	"fmt"

	"github.com/starkandwayne/goutils/tree"
)

// NullOperatorEnhanced is an enhanced version that supports nested expressions
type NullOperatorEnhanced struct{}

// Setup ...
func (NullOperatorEnhanced) Setup() error {
	return nil
}

// Phase ...
func (NullOperatorEnhanced) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (NullOperatorEnhanced) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (NullOperatorEnhanced) Run(ev *Evaluator, args []*Expr) (*Response, error) {
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

// EnableEnhancedNull enables the enhanced null operator
func EnableEnhancedNull() {
	RegisterOp("null", NullOperatorEnhanced{})
}

func init() {
	// Don't register in init - let EnableEnhancedNull handle it
}
