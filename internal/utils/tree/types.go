package tree

import (
	"fmt"
	"strings"

	"github.com/wayneeseguin/graft/internal/utils/ansi"
)

// NameFields is a slice of common field names used for object identification
var NameFields = []string{"name", "key", "id"}

// Cursor represents a path through YAML/JSON data structure
type Cursor struct {
	Nodes []string
}

// SyntaxError represents a syntax error in path parsing
type SyntaxError struct {
	Problem  string
	Position int
}

// Error returns the error message for SyntaxError
func (e SyntaxError) Error() string {
	return fmt.Sprintf("syntax error: %s at position %d", e.Problem, e.Position)
}

// TypeMismatchError represents a type mismatch during path resolution
type TypeMismatchError struct {
	Path   []string
	Wanted string
	Got    string
	Value  interface{}
}

// Error returns the error message for TypeMismatchError with ANSI coloring
func (e TypeMismatchError) Error() string {
	if e.Got == "" {
		return ansi.Sprintf("@c{%s} @R{is not} @m{%s}", strings.Join(e.Path, "."), e.Wanted)
	}
	if e.Value != nil {
		return ansi.Sprintf("@c{$.%s} @R{[=%v] is %s (not} @m{%s}@R{)}", strings.Join(e.Path, "."), e.Value, e.Got, e.Wanted)
	}
	return ansi.Sprintf("@C{$.%s} @R{is %s (not} @m{%s}@R{)}", strings.Join(e.Path, "."), e.Got, e.Wanted)
}

// NotFoundError represents when a path cannot be found in the data structure
type NotFoundError struct {
	Path []string
}

// Error returns the error message for NotFoundError with ANSI coloring
func (e NotFoundError) Error() string {
	return ansi.Sprintf("@R{`}@c{$.%s}@R{` could not be found in the datastructure}", strings.Join(e.Path, "."))
}