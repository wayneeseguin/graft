package operators

import (
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/geofffranks/yaml"
	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"
	
	"github.com/wayneeseguin/graft/log"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// RegisterOp registers an operator in the main graft package registry
func RegisterOp(name string, op graft.Operator) {
	graft.OpRegistry[name] = op
}

// SetupOperators initializes all operators for a given phase
func SetupOperators(phase graft.OperatorPhase) error {
	errors := graft.MultiError{Errors: []error{}}
	for _, op := range graft.OpRegistry {
		if op.Phase() == phase {
			if err := op.Setup(); err != nil {
				errors.Append(err)
			}
		}
	}
	if len(errors.Errors) > 0 {
		return errors
	}
	return nil
}

// NullOperator is returned for unknown operators
type NullOperator struct {
	Missing string
}

func (n NullOperator) Setup() error {
	return nil
}

func (n NullOperator) Run(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
	return nil, fmt.Errorf("unknown operator: %s", n.Missing)
}

func (n NullOperator) Dependencies(ev *graft.Evaluator, args []*graft.Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

func (n NullOperator) Phase() graft.OperatorPhase {
	return graft.EvalPhase
}

// ResolveOperatorArgument resolves an expression argument for use by operators
func ResolveOperatorArgument(ev *graft.Evaluator, arg *graft.Expr) (interface{}, error) {
	// This is a helper function that operators can use to resolve arguments
	// Implementation should be moved from the original code
	if arg == nil {
		return nil, nil
	}
	return arg.Evaluate(ev.Tree)
}
