package graft

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/starkandwayne/goutils/tree"

	. "github.com/wayneeseguin/graft/log"
)

// FileOperatorEnhanced is an enhanced version that supports nested expressions
type FileOperatorEnhanced struct{}

// Setup ...
func (FileOperatorEnhanced) Setup() error {
	return nil
}

// Phase ...
func (FileOperatorEnhanced) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (FileOperatorEnhanced) Dependencies(_ *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run ...
func (FileOperatorEnhanced) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	DEBUG("running (( file ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( file ... )) operation at $%s\n", ev.Here)

	var filename string

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
		return nil, fmt.Errorf("file operator requires one or two string arguments")
	}

	// Read the file
	file, err := ioutil.ReadFile(filename)
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

// EnableEnhancedFile enables the enhanced file operator
func EnableEnhancedFile() {
	RegisterOp("file", FileOperatorEnhanced{})
}

func init() {
	// Don't register in init - let EnableEnhancedFile handle it
}