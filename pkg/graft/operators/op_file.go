package operators

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"
)

// FileOperator ...
type FileOperator struct{}

// Setup ...
func (FileOperator) Setup() error {
	return nil
}

// Phase ...
func (FileOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (FileOperator) Dependencies(ev *Evaluator, args []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
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
func (FileOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( file ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( file ... )) operation at $%s\n", ev.Here)

	if len(args) != 1 {
		return nil, fmt.Errorf("file operator requires exactly one string or reference argument")
	}

	fbasepath := os.Getenv("GRAFT_FILE_BASE_PATH")

	// Use ResolveOperatorArgument to support nested expressions
	val, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		DEBUG("  arg[0]: failed to resolve expression to a concrete value")
		DEBUG("     [0]: error was: %s", err)
		return nil, err
	}

	// Check if it's a non-scalar type
	switch val.(type) {
	case map[interface{}]interface{}, map[string]interface{}:
		DEBUG("  arg[0]: %v is not a string scalar", val)
		return nil, ansi.Errorf("@R{tried to read file} @c{%v}@R{, which is not a string scalar}", val)
	case []interface{}:
		DEBUG("  arg[0]: %v is not a string scalar", val)
		return nil, ansi.Errorf("@R{tried to read file} @c{%v}@R{, which is not a string scalar}", val)
	}

	// Convert to string
	fname, err := AsString(val)
	if err != nil {
		DEBUG("  arg[0]: %v cannot be converted to string", val)
		return nil, ansi.Errorf("@R{tried to read file} @c{%v}@R{, which cannot be converted to string}", val)
	}

	DEBUG("  resolved argument to filename: %s", fname)

	if !filepath.IsAbs(fname) {
		fname = filepath.Join(fbasepath, fname)
	}

	contents, err := os.ReadFile(fname)
	if err != nil {
		DEBUG("  File %s cannot be read: %s", fname, err)
		return nil, ansi.Errorf("@R{tried to read file} @c{%s}@R{: could not be read - %s}", fname, err)
	}

	DEBUG("  resolved (( file ... )) operation to the string:\n    \"%s\"", string(contents))

	return &Response{
		Type:  Replace,
		Value: string(contents),
	}, nil
}

func init() {
	RegisterOp("file", FileOperator{})
}
