package operators

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/starkandwayne/goutils/tree"
)

// FileOperator handles nested operator calls
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
func (FileOperator) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (FileOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( file ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( file ... )) operation at $%s\n", ev.Here)

	var filename string

	// Debug the incoming arguments
	DEBUG("file operator received %d arguments", len(args))
	for i, arg := range args {
		if arg != nil {
			DEBUG("  arg[%d]: type=%v, operator=%s", i, arg.Type, arg.Operator)
		} else {
			DEBUG("  arg[%d]: nil", i)
		}
	}

	// Argument validation and processing
	if len(args) == 1 {
		// Use ResolveOperatorArgument to handle nested expressions
		val, err := ResolveOperatorArgument(ev, args[0])
		if err != nil {
			DEBUG("failed to resolve expression to a concrete value")
			DEBUG("error was: %s", err)
			return nil, err
		}

		if val == nil {
			return nil, fmt.Errorf("file operator argument resolved to nil")
		}

		filename = fmt.Sprintf("%v", val)
		DEBUG("using filename '%s'", filename)

	} else if len(args) == 2 {
		// Handle base path + filename
		basePath, err := ResolveOperatorArgument(ev, args[0])
		if err != nil {
			DEBUG("failed to resolve base path expression")
			DEBUG("error was: %s", err)
			return nil, err
		}

		fileName, err := ResolveOperatorArgument(ev, args[1])
		if err != nil {
			DEBUG("failed to resolve filename expression")
			DEBUG("error was: %s", err)
			return nil, err
		}

		if basePath == nil || fileName == nil {
			return nil, fmt.Errorf("file operator arguments cannot be nil")
		}

		filename = filepath.Join(fmt.Sprintf("%v", basePath), fmt.Sprintf("%v", fileName))
		DEBUG("using combined path '%s'", filename)

	} else {
		DEBUG("file operator error: expected 1 or 2 args, got %d", len(args))
		for i, arg := range args {
			if arg != nil {
				DEBUG("  arg[%d] details: Type=%v, IsOperator=%v", i, arg.Type, arg.IsOperator())
			}
		}
		return nil, fmt.Errorf("file operator requires one or two string arguments")
	}

	// Prepend the optional Graft base path override for relative paths
	if !filepath.IsAbs(filename) && os.Getenv("GRAFT_FILE_BASE_PATH") != "" {
		filename = filepath.Join(os.Getenv("GRAFT_FILE_BASE_PATH"), filename)
		DEBUG("using GRAFT_FILE_BASE_PATH, final path: %s", filename)
	}

	// Read the file
	file, err := os.ReadFile(filename) // #nosec G304 - file operator needs to read user-specified files
	if err != nil {
		DEBUG("failed to read file")
		DEBUG("error was: %s", err)
		return nil, err
	}

	contents := string(file)
	DEBUG("file read successfully")

	return &Response{
		Type:  Replace,
		Value: contents,
	}, nil
}

func init() {
	RegisterOp("file", FileOperator{})
}