package operators

import (
	"fmt"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"
	
	"github.com/wayneeseguin/graft/log"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// GrabOperator ...
type GrabOperator struct{}

// Setup ...
func (GrabOperator) Setup() error {
	return nil
}

// Phase ...
func (GrabOperator) Phase() graft.OperatorPhase {
	return graft.EvalPhase
}

// Dependencies ...
func (GrabOperator) Dependencies(_ *graft.Evaluator, args []*graft.Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	for _, arg := range args {
		if arg.Type == graft.Reference {
			auto = append(auto, arg.Reference)
		}
	}
	return auto
}

// Run ...
func (GrabOperator) Run(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
	log.DEBUG("running (( grab ... )) operation at $.%s", ev.Here)
	defer log.DEBUG("done with (( grab ... )) operation at $.%s\n", ev.Here)

	if len(args) < 1 {
		return nil, fmt.Errorf("grab operator requires at least one argument")
	}

	// Implementation placeholder
	return nil, fmt.Errorf("grab not yet implemented")
}

func init() {
	RegisterOp("grab", GrabOperator{})
}
