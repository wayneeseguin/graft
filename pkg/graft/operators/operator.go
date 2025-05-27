package operators

import (
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
