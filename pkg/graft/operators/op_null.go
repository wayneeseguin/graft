package operators

import (
	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// NullOperator ...
type NullOperator struct {
	Missing string
}

// Setup ...
func (NullOperator) Setup() error {
	return nil
}

// Phase ...
func (NullOperator) Phase() graft.OperatorPhase {
	return graft.EvalPhase
}

// Dependencies ...
func (NullOperator) Dependencies(_ *graft.Evaluator, _ []*graft.Expr, _ []*tree.Cursor, _ []*tree.Cursor) []*tree.Cursor {
	return nil
}

// Run ...
func (n NullOperator) Run(ev *graft.Evaluator, _ []*graft.Expr) (*graft.Response, error) {
	return nil, ansi.Errorf("@c{(( %s ))} @R{operator not defined}", n.Missing)
}
