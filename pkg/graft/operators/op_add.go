package operators

import (
	"fmt"

	"github.com/starkandwayne/goutils/tree"
	
	"github.com/wayneeseguin/graft/log"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// AddOperator implements the + operator
type AddOperator struct{}

// Setup ...
func (AddOperator) Setup() error {
	return nil
}

// Phase ...
func (AddOperator) Phase() graft.OperatorPhase {
	return graft.EvalPhase
}

// Dependencies ...
func (AddOperator) Dependencies(_ *graft.Evaluator, _ []*graft.Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (AddOperator) Run(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
	log.DEBUG("running (( + ... )) operation at $.%s", ev.Here)
	defer log.DEBUG("done with (( + ... )) operation at $.%s\n", ev.Here)

	if len(args) != 2 {
		return nil, fmt.Errorf("add operator requires exactly two arguments")
	}

	left, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		return nil, err
	}

	right, err := ResolveOperatorArgument(ev, args[1])
	if err != nil {
		return nil, err
	}

	log.DEBUG("  [0] = %v", left)
	log.DEBUG("  [1] = %v", right)

	// Determine result type and perform addition
	result, err := performAddition(left, right)
	if err != nil {
		return nil, err
	}

	return &graft.Response{
		Type:  graft.Replace,
		Value: result,
	}, nil
}

func performAddition(left, right interface{}) (interface{}, error) {
	// Implementation of addition logic here
	// This is a placeholder - the actual logic should be moved from the original file
	return nil, fmt.Errorf("addition not yet implemented")
}

func init() {
	RegisterOp("+", AddOperator{})
}
