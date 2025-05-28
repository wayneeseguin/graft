package parser

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"github.com/wayneeseguin/graft/pkg/graft"
	"github.com/starkandwayne/goutils/tree"
)

// UseParser is a feature flag to enable the new parser
// This allows gradual rollout and easy rollback
// Default is true to use parser - set to false to use legacy parser
var UseParser = true

// ParseOpcall is the enhanced version of ParseOpcall that uses the new parser
func ParseOpcall(phase OperatorPhase, src string) (*Opcall, error) {
	// Strip outer (( )) markers and validate format
	originalSrc := src
	src = strings.TrimSpace(src)
	if !strings.HasPrefix(src, "((") {
		return nil, nil
	}
	if !strings.HasSuffix(src, "))") {
		// Has opening (( but missing closing ))
		return nil, &ExprError{
			Type:     SyntaxError,
			Message:  "missing closing parenthesis",
			Position: Position{Line: 1, Column: len(src) + 1},
			Source:   originalSrc,
		}
	}
	
	inner := strings.TrimSpace(src[2:len(src)-2])
	if inner == "" {
		return nil, nil
	}
	
	DEBUG("ParseOpcall: parsing '%s' (inner: '%s') in phase %v", src, inner, phase)
	
	// Check for special patterns that should be ignored
	// ((!something)) - spiff-like bang notation
	if strings.HasPrefix(inner, "!") {
		return nil, nil
	}
	
	// Try traditional operator patterns first
	// But skip if the expression contains arithmetic/comparison operators
	// EXCEPT for defer which needs special handling to preserve || in arguments
	isDeferOp := strings.HasPrefix(inner, "defer ")
	if isDeferOp || !strings.ContainsAny(inner, "+-*/%<>=!&|?:") {
		patterns := []string{
			`^([a-zA-Z][a-zA-Z0-9_-]*)\s*$`,         // op (no args)
			`^([a-zA-Z][a-zA-Z0-9_-]*)\s+(.*)$`,     // op args
		}
		
		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			if m := re.FindStringSubmatch(inner); m != nil {
				opname := m[1]
				argStr := ""
				if len(m) > 2 {
					argStr = strings.TrimSpace(m[2])
				}
				
				// Check if it's a valid operator
				op := OperatorFor(opname)
				if _, ok := op.(*NullOperator); ok && argStr == "" {
					// Not a real operator with no arguments - might be BOSH variable
					return nil, nil
				}
				
				// Check phase
				if op.Phase() != phase {
					// Wrong phase - ignore this operator
					return nil, nil
				}
				
				// Special handling for defer operator - use legacy parsing
				if opname == "defer" && argStr != "" {
					// For defer, we want to use the legacy argument parsing to preserve the original behavior
					// This means we parse it like the old argify function would
					return parseOpcallLegacyArgs(op, argStr, src, phase)
				}
				
				// Parse arguments with the parser
				args, err := ParseArguments(phase, argStr)
				if err != nil {
					return nil, err
				}
				
				return graft.NewOpcall(op, args, src), nil
			}
		}
	}
	
	// If no traditional operator pattern matched, try parsing as an expression
	// This handles arithmetic expressions, comparisons, etc.
	registry := createFullOperatorRegistry()
	
	tokenizer := NewTokenizer(inner)
	tokens := tokenizer.Tokenize()
	
	DEBUG("ParseOpcall: tokenized '%s' into %d tokens", inner, len(tokens))
	for i, tok := range tokens {
		DEBUG("ParseOpcall:   token[%d]: '%s' (type=%v)", i, tok.Value, tok.Type)
	}
	
	if len(tokens) == 0 {
		return nil, nil
	}
	
	parser := NewParser(tokens, registry).WithSource(originalSrc)
	expr, err := parser.Parse()
	if err != nil {
		DEBUG("ParseOpcall: parser.Parse() error: %v", err)
		return nil, err
	}
	
	DEBUG("ParseOpcall: parsed expression type: %v", expr.Type)
	
	// Handle the parsed expression
	if expr.Type == OperatorCall {
		// It's an operator call (could be arithmetic, comparison, etc.)
		opname := expr.Op()
		args := expr.Args()
		DEBUG("ParseOpcall: operator '%s' with %d args", opname, len(args))
		for i, arg := range args {
			DEBUG("ParseOpcall:   arg[%d]: type=%v, value=%v", i, arg.Type, arg.String())
		}
		
		op := OperatorFor(opname)
		if _, ok := op.(*NullOperator); ok {
			// Unknown operator
			DEBUG("ParseOpcall: unknown operator '%s'", opname)
			return nil, nil
		}
		
		// Check phase
		if op.Phase() != phase {
			DEBUG("ParseOpcall: operator '%s' is in phase %v, wanted %v", opname, op.Phase(), phase)
			return nil, nil
		}
		
		DEBUG("ParseOpcall: returning operator '%s' with %d args", opname, len(args))
		return graft.NewOpcall(op, args, src), nil
	}
	
	// If it's just a simple reference and looks like a BOSH varname, ignore it
	if expr.Type == Reference && expr.Reference != nil {
		path := expr.Reference.String()
		if !strings.Contains(path, ".") && strings.Contains(path, "-") {
			// Looks like a BOSH varname (e.g., var-name)
			return nil, nil
		}
	}
	
	// For other expression types (that aren't simple operator calls),
	// we need to create a synthetic operator to evaluate them
	// This includes LogicalOr expressions and direct literals
	
	// Create an expression wrapper operator that just evaluates the expression
	return graft.NewOpcall(&ExpressionWrapperOperator{expr: expr}, []*graft.Expr{}, src), nil
}

// ParseArguments parses operator arguments using the parser
func ParseArguments(phase OperatorPhase, src string) ([]*graft.Expr, error) {
	if src == "" {
		return []*graft.Expr{}, nil
	}
	
	// Create operator registry with all known operators
	registry := createOperatorRegistry()
	
	// For operators like concat that take multiple space-separated arguments,
	// we can now use ParseMultiple to handle them properly
	
	// Use the parser to parse multiple arguments
	tokenizer := NewTokenizer(src)
	tokens := tokenizer.Tokenize()
	
	parser := NewParser(tokens, registry).WithSource(src)
	args, err := parser.ParseMultiple()
	if err != nil {
		return nil, err
	}
	
	// Process expressions similar to the original argify
	var final []*graft.Expr
	for _, e := range args {
		// Try to reduce the expression
		reduced, err := ReduceExpr(e)
		if err != nil {
			if warning, isWarning := err.(WarningError); isWarning {
				warning.Warn()
			} else {
				fmt.Fprintf(os.Stdout, "warning: %s\n", err)
			}
		}
		
		// Special case: preserve LogicalOr expressions for operators that can handle them
		if e.Type == LogicalOr && reduced.Type != LogicalOr {
			final = append(final, e) // Use original instead of reduced
		} else {
			final = append(final, reduced)
		}
	}
	
	return final, nil
}

// createOperatorRegistry creates a registry with all known operators
func createOperatorRegistry() *OperatorRegistry {
	registry := NewOperatorRegistry()
	
	// Register all operators from the actual OpRegistry
	for name, op := range OpRegistry {
		// Determine min/max args based on operator type
		// This is a heuristic approach since operators don't declare their arg counts
		minArgs := 1
		maxArgs := -1 // unlimited by default
		
		// Special cases based on known operators
		switch name {
		case "calc", "empty", "grab", "param", "defer", "stringify", "negate", "null":
			maxArgs = 1
		case "base64", "base64-decode":
			maxArgs = 1
		case "file":
			maxArgs = 2
		case "sort":
			minArgs = 1
			maxArgs = 2
		case "ips":
			minArgs = 3
			maxArgs = 6
		case "inject", "keys", "load":
			minArgs = 1
			maxArgs = 1
		case "vault-try":
			minArgs = 2
		case "awsparam", "awssecret":
			minArgs = 1
			maxArgs = 3
		case "?:": // ternary
			minArgs = 3
			maxArgs = 3
		case "+", "-", "*", "/", "%", "==", "!=", "<", ">", "<=", ">=", "&&", "||":
			minArgs = 2
			maxArgs = 2
		case "!":
			minArgs = 1
			maxArgs = 1
		}
		
		// Get the phase from the operator
		phase := op.Phase()
		
		// Register the operator
		registry.Register(&OperatorInfo{
			Name:       name,
			Precedence: PrecedencePostfix,
			MinArgs:    minArgs,
			MaxArgs:    maxArgs,
			Phase:      phase,
		})
	}
	
	return registry
}

// argify is the enhanced version of argify using the new parser
func argify(phase OperatorPhase, src string) ([]*graft.Expr, error) {
	return ParseArguments(phase, src)
}

// IntegrateParser updates the existing ParseOpcall to optionally use the parser
// This is called during initialization if the feature flag is set
func IntegrateParser() {
	if !UseParser {
		return
	}
	
	// Override the global ParseOpcall function
	// This would require making ParseOpcall a variable, which is a bigger change
	// For now, we'll need to update call sites individually
	fmt.Fprintf(os.Stderr, " parser integration enabled\n")
}

// ExpressionWrapperOperator wraps a parsed expression for evaluation
// This allows us to evaluate expressions that aren't traditional operator calls
type ExpressionWrapperOperator struct {
	expr *graft.Expr
}

func (op *ExpressionWrapperOperator) Setup() error {
	return nil
}

func (op *ExpressionWrapperOperator) Phase() graft.OperatorPhase {
	return graft.EvalPhase
}

func (op *ExpressionWrapperOperator) Dependencies(ev *graft.Evaluator, args []*graft.Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	// Extract dependencies from the wrapped expression
	deps := make([]*tree.Cursor, 0)
	var extractDeps func(e *graft.Expr)
	extractDeps = func(e *graft.Expr) {
		if e == nil {
			return
		}
		switch e.Type {
		case Reference:
			if e.Reference != nil {
				deps = append(deps, e.Reference)
			}
		case LogicalOr:
			extractDeps(e.Left)
			extractDeps(e.Right)
		case OperatorCall:
			for _, arg := range e.Args() {
				extractDeps(arg)
			}
		}
	}
	extractDeps(op.expr)
	return append(auto, deps...)
}

func (op *ExpressionWrapperOperator) Run(ev *Evaluator, args []*graft.Expr) (*Response, error) {
	// Simply evaluate the wrapped expression
	return EvaluateExpr(op.expr, ev)
}

// parseOpcallLegacyArgs parses arguments using legacy-style parsing for specific operators
func parseOpcallLegacyArgs(op Operator, argStr string, src string, phase OperatorPhase) (*Opcall, error) {
	// Use a simplified version of the legacy argify function
	// The legacy argify builds a LogicalOr expression when it sees ||
	
	var final []*graft.Expr
	var left, opExpr *graft.Expr
	
	// Split by whitespace, handling quoted strings
	tokens := splitLegacyArgs(argStr)
	
	pop := func() {
		if left != nil {
			final = append(final, left)
			left = nil
		}
	}
	
	push := func(e *graft.Expr) {
		if left == nil {
			left = e
			return
		}
		if opExpr == nil {
			pop()
			left = e
			return
		}
		opExpr.Left = left
		opExpr.Right = e
		left = opExpr
		opExpr = nil
	}
	
	for _, token := range tokens {
		if token == "||" {
			if left == nil || opExpr != nil {
				// Invalid syntax, but let the operator handle it
				final = append(final, &Expr{Type: Literal, Literal: token})
			} else {
				opExpr = &Expr{Type: LogicalOr}
			}
			continue
		}
		
		var expr *graft.Expr
		
		// Check for quoted string
		if strings.HasPrefix(token, `"`) && strings.HasSuffix(token, `"`) && len(token) >= 2 {
			// Remove quotes
			literal := token[1:len(token)-1]
			expr = &Expr{Type: Literal, Literal: literal}
		} else if strings.HasPrefix(token, "$") {
			// Environment variable
			expr = &Expr{Type: EnvVar, Name: token[1:]}
		} else if token == "nil" || token == "null" || token == "~" {
			// Nil literal
			expr = &Expr{Type: Literal, Literal: nil}
		} else if token == "true" || token == "false" {
			// Boolean literal
			expr = &Expr{Type: Literal, Literal: token == "true"}
		} else if num, err := strconv.ParseInt(token, 10, 64); err == nil {
			// Integer literal
			expr = &Expr{Type: Literal, Literal: num}
		} else if num, err := strconv.ParseFloat(token, 64); err == nil {
			// Float literal
			expr = &Expr{Type: Literal, Literal: num}
		} else {
			// Must be a reference or literal string
			if strings.Contains(token, ".") || !strings.Contains(token, "-") {
				// Looks like a reference
				cursor, err := tree.ParseCursor(token)
				if err == nil {
					expr = &Expr{Type: Reference, Reference: cursor}
				} else {
					// Treat as literal if can't parse as cursor
					expr = &Expr{Type: Literal, Literal: token}
				}
			} else {
				// Treat as literal (could be an operator name like "grab")
				expr = &Expr{Type: Literal, Literal: token}
			}
		}
		
		push(expr)
	}
	
	pop()
	if left != nil || opExpr != nil {
		// Incomplete expression
		return nil, fmt.Errorf("syntax error near: %s", argStr)
	}
	
	return graft.NewOpcall(op, final, src), nil
}

// splitLegacyArgs splits arguments respecting quoted strings
func splitLegacyArgs(src string) []string {
	tokens := make([]string, 0, 8) // Pre-allocate for typical argument counts
	var current strings.Builder
	inQuotes := false
	escaped := false
	
	for _, ch := range src {
		if escaped {
			current.WriteRune(ch)
			escaped = false
			continue
		}
		
		if ch == '\\' {
			current.WriteRune(ch)
			escaped = true
			continue
		}
		
		if ch == '"' {
			current.WriteRune(ch)
			if inQuotes {
				// End of quoted string
				tokens = append(tokens, current.String())
				current.Reset()
				inQuotes = false
			} else {
				// Start of quoted string
				inQuotes = true
			}
			continue
		}
		
		if inQuotes {
			current.WriteRune(ch)
			continue
		}
		
		// Outside quotes
		if ch == ' ' || ch == '\t' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(ch)
		}
	}
	
	// Add any remaining token
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	
	return tokens
}