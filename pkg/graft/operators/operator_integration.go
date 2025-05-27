package operators

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/starkandwayne/goutils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
	"github.com/wayneeseguin/graft/pkg/graft/parser"
)

// Type aliases for this file
type Opcall = graft.Opcall
type OperatorRegistry = parser.OperatorRegistry

// ParseOpcall delegates to parser package
func ParseOpcall(phase OperatorPhase, src string) (*Opcall, error) {
	return parser.ParseOpcall(phase, src)
}

// ParseOpcallOriginal is the original ParseOpcall implementation
// We keep this for reference and fallback
var ParseOpcallOriginal = ParseOpcall

// IntegrateEnhancedParserCore replaces the core ParseOpcall with enhanced parser support
func IntegrateEnhancedParserCore() {
	// This would require ParseOpcall to be a function variable, which is a breaking change
	// Instead, we'll provide a new implementation that can be swapped in
	DEBUG("Enhanced parser core integration initialized")
}

// ParseOpcallIntegrated is the new integrated version that uses enhanced parser when appropriate
func ParseOpcallIntegrated(phase OperatorPhase, src string) (*Opcall, error) {
	// Check if we should use enhanced parser
	if UseEnhancedParser() || shouldUseEnhancedParserForExpression(src) {
		DEBUG("Using enhanced parser for: %s", src)
		result, err := parseOpcallWithEnhancedParser(phase, src)
		if err != nil {
			// Check if this is a parse error that should be reported
			// Don't fall back for syntax errors in expressions
			if strings.Contains(err.Error(), "expected") || strings.Contains(err.Error(), "ternary") ||
				strings.Contains(err.Error(), "argument") || strings.Contains(err.Error(), "position") {
				return nil, err
			}
			// Fall back to original parser for other errors
			DEBUG("Enhanced parser failed, falling back to original: %v", err)
		} else if result != nil {
			return result, nil
		}
		// Enhanced parser returned nil without error, fall back to original
		DEBUG("Enhanced parser returned nil, falling back to original")
	}

	// Use original implementation
	return parseOpcallWithOriginalParser(phase, src)
}

// shouldUseEnhancedParserForExpression determines if an expression benefits from enhanced parser
func shouldUseEnhancedParserForExpression(src string) bool {
	// Check for nested operator patterns
	if strings.Contains(src, " (") && strings.Contains(src, ")") {
		return true
	}

	// Check for arithmetic operators
	if strings.ContainsAny(src, "+-*/") {
		// But not for negative numbers or dates
		if !regexp.MustCompile(`^\(\(\s*-?\d+`).MatchString(src) &&
			!strings.Contains(src, "T") { // ISO dates
			return true
		}
	}

	// Check for complex logical expressions
	if strings.Count(src, "||") > 1 {
		return true
	}

	return false
}

// parseOpcallWithEnhancedParser uses the new parser infrastructure
func parseOpcallWithEnhancedParser(phase OperatorPhase, src string) (*Opcall, error) {
	// Use ParseOpcallEnhanced which has the special pattern handling
	return ParseOpcallEnhanced(phase, src, nil)
}

// parseOpcallWithOriginalParser is the original implementation
func parseOpcallWithOriginalParser(phase OperatorPhase, src string) (*Opcall, error) {
	// This is a copy of the original ParseOpcall implementation
	// We duplicate it here to avoid circular dependencies

	split := func(src string) []string {
		list := make([]string, 0, 0)

		buf := ""
		escaped := false
		quoted := false

		for _, c := range src {
			if escaped {
				switch c {
				case 'n':
					buf += "\n"
				case 'r':
					buf += "\r"
				case 't':
					buf += "\t"
				default:
					buf += string(c)
				}
				escaped = false
				continue
			}

			if c == '\\' {
				escaped = true
				continue
			}

			if c == ' ' || c == '\t' || c == ',' {
				if quoted {
					buf += string(c)
					continue
				} else {
					if buf != "" {
						list = append(list, buf)
						buf = ""
					}
					if c == ',' {
						list = append(list, ",")
					}
				}
				continue
			}

			if c == '"' {
				buf += string(c)
				quoted = !quoted
				continue
			}

			buf += string(c)
		}

		if buf != "" {
			list = append(list, buf)
		}

		return list
	}

	argify := func(src string) (args []*Expr, err error) {
		qstring := regexp.MustCompile(`(?s)^"(.*)"$`)
		integer := regexp.MustCompile(`^[+-]?\d+(\.\d+)?$`)
		float := regexp.MustCompile(`^[+-]?\d*\.\d+$`)
		envvar := regexp.MustCompile(`^\$[a-zA-Z_][a-zA-Z0-9_.]*$`)

		var final []*Expr
		var left, op *Expr

		pop := func() {
			if left != nil {
				final = append(final, left)
				left = nil
			}
		}

		push := func(e *Expr) {
			TRACE("expr: pushing data expression `%s' onto stack", e)
			TRACE("expr:   start: left=`%s', op=`%s'", left, op)
			defer func() { TRACE("expr:     end: left=`%s', op=`%s'\n", left, op) }()

			if left == nil {
				left = e
				return
			}
			if op == nil {
				pop()
				left = e
				return
			}
			op.Left = left
			op.Right = e
			left = op
			op = nil
		}

		TRACE("expr: parsing `%s'", src)
		for i, arg := range split(src) {
			switch {
			case arg == ",":
				DEBUG("  #%d: literal comma found; treating what we've seen so far as a complete expression", i)
				pop()

			case envvar.MatchString(arg):
				DEBUG("  #%d: parsed as unquoted environment variable reference '%s'", i, arg)
				push(&Expr{Type: EnvVar, Name: arg[1:]})

			case qstring.MatchString(arg):
				m := qstring.FindStringSubmatch(arg)
				DEBUG("  #%d: parsed as quoted string literal '%s'", i, m[1])
				push(&Expr{Type: Literal, Literal: m[1]})

			case float.MatchString(arg):
				DEBUG("  #%d: parsed as unquoted floating point literal '%s'", i, arg)
				v, err := strconv.ParseFloat(arg, 64)
				if err != nil {
					DEBUG("  #%d: %s is not parsable as a floating point number: %s", i, arg, err)
					return args, err
				}
				push(&Expr{Type: Literal, Literal: v})

			case integer.MatchString(arg):
				DEBUG("  #%d: parsed as unquoted integer literal '%s'", i, arg)
				v, err := strconv.ParseInt(arg, 10, 64)
				if err == nil {
					push(&Expr{Type: Literal, Literal: v})
					break
				}
				DEBUG("  #%d: %s is not parsable as an integer, falling back to parsing as float: %s", i, arg, err)
				f, err := strconv.ParseFloat(arg, 64)
				push(&Expr{Type: Literal, Literal: f})
				if err != nil {
					panic("Could not actually parse as an int or a float. Need to fix regexp?")
				}

			case arg == "||":
				DEBUG("  #%d: parsed logical-or operator, '||'", i)

				if left == nil || op != nil {
					return args, fmt.Errorf(`syntax error near: %s`, src)
				}
				TRACE("expr: pushing || expr-op onto the stack")
				op = &Expr{Type: LogicalOr}

			case arg == "nil" || arg == "null" || arg == "~" || arg == "Nil" || arg == "Null" || arg == "NIL" || arg == "NULL":
				DEBUG("  #%d: parsed the nil value token '%s'", i, arg)
				push(&Expr{Type: Literal, Literal: nil})

			case arg == "false" || arg == "False" || arg == "FALSE":
				DEBUG("  #%d: parsed the false value token '%s'", i, arg)
				push(&Expr{Type: Literal, Literal: false})

			case arg == "true" || arg == "True" || arg == "TRUE":
				DEBUG("  #%d: parsed the true value token '%s'", i, arg)
				push(&Expr{Type: Literal, Literal: true})

			default:
				c, err := tree.ParseCursor(arg)
				if err != nil {
					DEBUG("  #%d: %s is a malformed reference: %s", i, arg, err)
					return args, err
				}
				DEBUG("  #%d: parsed as a reference to $.%s", i, c)
				push(&Expr{Type: Reference, Reference: c})
			}
		}
		pop()
		if left != nil || op != nil {
			return nil, fmt.Errorf(`syntax error near: %s`, src)
		}
		DEBUG("")

		for _, e := range final {
			TRACE("expr: pushing expression `%v' onto the operand list", e)
			reduced, err := parser.ReduceExpr(e)
			if err != nil {
				if warning, isWarning := err.(WarningError); isWarning {
					warning.Warn()
				} else {
					fmt.Fprintf(os.Stdout, "warning: %s\n", err)
				}
			}
			// Special case: preserve LogicalOr expressions for operators that can handle them
			if e.Type == LogicalOr && reduced.Type != LogicalOr {
				args = append(args, e) // Use original instead of reduced
			} else {
				args = append(args, reduced)
			}
		}
		DEBUG("")
		return args, nil
	}

	// Strip outer whitespace
	src = strings.TrimSpace(src)

	// Match operator patterns
	patterns := []string{
		`^\Q((\E\s*(@\w+)\s*\Q))\E$`,
		`^\Q((\E\s*(\w+(?:\.\w+)*)\s*\Q))\E$`,
		`^\Q((\E\s*([a-zA-Z][a-zA-Z0-9_-]*)\s*\((.*)\)\s*\Q))\E$`,
		`^\Q((\E\s*([a-zA-Z][a-zA-Z0-9_-]*)\s+(.*)\s*\Q))\E$`,
		`^\Q((\E\s*([a-zA-Z][a-zA-Z0-9_-]*)\s*\Q))\E$`,
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if !re.MatchString(src) {
			continue
		}

		m := re.FindStringSubmatch(src)
		DEBUG("parsing `%s': looks like a (( %s ... )) operator call to me", src, m[1])
		opname := m[1]
		op := OperatorFor(opname)

		if _, ok := op.(NullOperator); ok {
			DEBUG("  - no operator registered for `%s`", opname)
			continue
		}

		DEBUG("  - found an operator for `%s'", opname)
		if op.Phase() != phase {
			DEBUG("  - operator `%s' is not for the %v phase", opname, phase)
			continue
		}

		args := make([]*Expr, 0, 0)
		if len(m) > 2 {
			arglist, err := argify(m[2])
			if err != nil {
				DEBUG("  - failed to parse arguments: %s", err)
				return nil, err
			}
			args = arglist
		}

		DEBUG("  - successfully parsed (( %s ... )) operation with %d arguments:", opname, len(args))
		for i, arg := range args {
			DEBUG("    arg[%d]: %s", i, arg)
		}
		return graft.NewOpcall(op, args, src), nil
	}
	DEBUG("parsing `%s': not an operator (no match)", src)
	return nil, nil
}

// processArgumentExpressions processes parsed expressions similar to original argify
func processArgumentExpressions(args []*Expr) ([]*Expr, error) {
	var final []*Expr

	for _, e := range args {
		// Handle nested operator calls
		if e.Type == OperatorCallCall {
			// For now, keep nested operator calls as-is
			// They will be evaluated during operator execution
			final = append(final, e)
			continue
		}

		// Process other expressions as before
		reduced, err := parser.ReduceExpr(e)
		if err != nil {
			if warning, isWarning := err.(WarningError); isWarning {
				warning.Warn()
			} else {
				fmt.Fprintf(os.Stdout, "warning: %s\n", err)
			}
		}

		// Special case: preserve LogicalOr expressions
		if e.Type == LogicalOr && reduced.Type != LogicalOr {
			final = append(final, e)
		} else {
			final = append(final, reduced)
		}
	}

	return final, nil
}

// createFullOperatorRegistry creates a complete registry of all operators
func createFullOperatorRegistry() *OperatorRegistry {
	registry := NewOperatorRegistry()

	// Register all operators from OpRegistry
	for name, op := range OpRegistry {
		// Determine min/max args based on operator type
		// This is a simplified version - ideally each operator would declare its arg count
		minArgs := 1
		maxArgs := -1

		// Special cases based on known operators
		switch name {
		case "calc", "empty", "grab", "param", "defer", "stringify":
			maxArgs = 1
		case "file":
			maxArgs = 2
		case "join", "static_ips":
			minArgs = 1
		case "ips":
			minArgs = 3
			maxArgs = 6
		}

		registry.Register(&OperatorInfo{
			Name:       name,
			Precedence: PrecedencePostfix,
			MinArgs:    minArgs,
			MaxArgs:    maxArgs,
			Phase:      op.Phase(),
		})
	}

	// Register binary operators
	binaryOps := []struct {
		name          string
		precedence    Precedence
		associativity Associativity
	}{
		// Logical operators
		{"||", PrecedenceOr, RightAssociative},
		{"&&", PrecedenceAnd, LeftAssociative},
		// Comparison operators
		{"==", PrecedenceEquality, LeftAssociative},
		{"!=", PrecedenceEquality, LeftAssociative},
		{"<", PrecedenceComparison, LeftAssociative},
		{">", PrecedenceComparison, LeftAssociative},
		{"<=", PrecedenceComparison, LeftAssociative},
		{">=", PrecedenceComparison, LeftAssociative},
		// Arithmetic operators
		{"+", PrecedenceAdditive, LeftAssociative},
		{"-", PrecedenceAdditive, LeftAssociative},
		{"*", PrecedenceMultiplicative, LeftAssociative},
		{"/", PrecedenceMultiplicative, LeftAssociative},
		{"%", PrecedenceMultiplicative, LeftAssociative},
	}

	for _, binOp := range binaryOps {
		registry.Register(&OperatorInfo{
			Name:          binOp.name,
			Precedence:    binOp.precedence,
			Associativity: binOp.associativity,
			MinArgs:       2,
			MaxArgs:       2,
			Phase:         EvalPhase,
		})
	}

	// Register ternary operator
	registry.Register(&OperatorInfo{
		Name:          "?:",
		Precedence:    PrecedenceTernary,
		Associativity: RightAssociative,
		MinArgs:       3,
		MaxArgs:       3,
		Phase:         EvalPhase,
	})

	// Register unary operators
	registry.Register(&OperatorInfo{
		Name:          "!",
		Precedence:    PrecedenceUnary,
		Associativity: RightAssociative,
		MinArgs:       1,
		MaxArgs:       1,
		Phase:         EvalPhase,
	})

	return registry
}

// ExpressionEvaluatorOperator is a synthetic operator that evaluates parsed expressions
type ExpressionEvaluatorOperator struct {
	expr *Expr
}

// Setup ...
func (e *ExpressionEvaluatorOperator) Setup() error {
	return nil
}

// Phase ...
func (e *ExpressionEvaluatorOperator) Phase() OperatorPhase {
	return EvalPhase
}

// Dependencies ...
func (e *ExpressionEvaluatorOperator) Dependencies(ev *Evaluator, _ []*Expr, _ []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	// Extract dependencies from the expression
	deps := extractExpressionDependencies(e.expr, ev)
	return append(auto, deps...)
}

// Run ...
func (e *ExpressionEvaluatorOperator) Run(ev *Evaluator, _ []*Expr) (*Response, error) {
	DEBUG("evaluating expression: %s", e.expr.String())

	// Evaluate the expression
	result, err := EvaluateExpr(e.expr, ev)
	if err != nil {
		return nil, err
	}

	// EvaluateExpr returns a Response, so we just return it directly
	return result, nil
}

// extractExpressionDependencies recursively extracts dependencies from an expression
func extractExpressionDependencies(expr *Expr, ev *Evaluator) []*tree.Cursor {
	if expr == nil {
		return nil
	}

	var deps []*tree.Cursor

	switch expr.Type {
	case Reference:
		if expr.Reference != nil {
			deps = append(deps, expr.Reference)
		}

	case OperatorCall:
		// Get dependencies from operator arguments
		for _, arg := range expr.Args() {
			deps = append(deps, extractExpressionDependencies(arg, ev)...)
		}

	case LogicalOr:
		deps = append(deps, extractExpressionDependencies(expr.Left, ev)...)
		deps = append(deps, extractExpressionDependencies(expr.Right, ev)...)
	}

	return deps
}
