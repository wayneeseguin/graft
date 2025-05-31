package graft

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/starkandwayne/goutils/tree"
	
	"github.com/wayneeseguin/graft/log"
)

// ParseOpcallCompat provides backward compatibility while allowing enhanced parser usage
func ParseOpcallCompat(phase OperatorPhase, src string) (*Opcall, error) {
	// Only parse strings that look like operator expressions
	if !strings.HasPrefix(strings.TrimSpace(src), "((") || !strings.HasSuffix(strings.TrimSpace(src), "))") {
		log.DEBUG("ParseOpcallCompat: '%s' does not look like an operator expression", src)
		return nil, nil
	}
	
	if strings.Contains(src, "base + addend") || strings.Contains(src, "base * multiplier") {
		log.DEBUG("ParseOpcallCompat: found expression with base: '%s'", src)
	}
	
	log.DEBUG("ParseOpcallCompat: checking '%s' in phase %v", src, phase)
	log.DEBUG("ParseOpcallCompat: OpRegistry has %d operators", len(OpRegistry))
	
	// Try original parser first
	if opcall, err := ParseOpcall(phase, src); err == nil && opcall != nil {
		// If the original parser returns a NullOperator, it might be an infix expression
		// that the original parser couldn't handle
		if _, isNull := opcall.Operator().(NullOperator); !isNull {
			log.DEBUG("ParseOpcallCompat: original parser succeeded with operator type %T", opcall.Operator())
			return opcall, nil
		}
		log.DEBUG("ParseOpcallCompat: original parser returned NullOperator, trying infix parser")
	} else {
		log.DEBUG("ParseOpcallCompat: original parser failed or returned nil, trying infix parser")
	}
	
	// If original parser fails or returns NullOperator, try enhanced parser for simple infix expressions
	return ParseOpcallInfix(phase, src)
}

// ParseOpcallInfix parses expressions with infix operators support
func ParseOpcallInfix(phase OperatorPhase, src string) (*Opcall, error) {
	log.DEBUG("ParseOpcallInfix: parsing '%s'", src)
	
	// Remove (( and )) wrapper
	content := strings.TrimSpace(src)
	if strings.HasPrefix(content, "((") && strings.HasSuffix(content, "))") {
		content = strings.TrimSpace(content[2 : len(content)-2])
	}
	
	log.DEBUG("ParseOpcallInfix: content = '%s'", content)
	
	// Try to parse as an expression with precedence
	opcall, err := parseExpressionWithPrecedence(phase, content)
	if err != nil {
		return nil, err
	}
	if opcall != nil {
		return opcall, nil
	}
	
	// If we can't parse it as an infix expression, return nil
	// Don't fall back to ParseOpcall as it would create a loop
	log.DEBUG("ParseOpcallInfix: failed to parse as infix expression")
	return nil, nil
}

// isSimpleBinaryExpression checks if this looks like a simple binary expression
// to avoid interfering with complex expressions that the original parser should handle
func isSimpleBinaryExpression(content string) bool {
	// Avoid expressions with parentheses (complex expressions)
	if strings.Contains(content, "(") || strings.Contains(content, ")") {
		return false
	}
	
	// Avoid expressions with multiple operators that might need precedence handling
	opCount := 0
	operators := []string{"||", "&&", "==", "!=", "<=", ">=", "<", ">", "+", "-", "*", "/", "%"}
	for _, op := range operators {
		if strings.Contains(content, op) {
			opCount++
			if opCount > 1 {
				return false // Complex expression with multiple operators
			}
		}
	}
	
	return opCount == 1 // Exactly one operator
}

// tryParseInfixOperator attempts to parse an infix operator expression
func tryParseInfixOperator(phase OperatorPhase, content, opName string) *Opcall {
	op := OpRegistry[opName]
	if op == nil || op.Phase() != phase {
		return nil
	}
	
	log.DEBUG("tryParseInfixOperator: trying operator '%s'", opName)
	
	// Handle ternary operator specially (a ? b : c)
	if opName == "?:" {
		return tryParseTernary(phase, content)
	}
	
	// Find the operator in the string (outside of quotes)
	opIndex := findOperatorIndex(content, opName)
	if opIndex == -1 {
		return nil
	}
	
	left := strings.TrimSpace(content[:opIndex])
	right := strings.TrimSpace(content[opIndex+len(opName):])
	
	log.DEBUG("tryParseInfixOperator: found '%s' at index %d, left='%s', right='%s'", opName, opIndex, left, right)
	
	if left == "" || right == "" {
		return nil
	}
	
	// Parse left and right expressions
	leftExpr, err := parseExpression(left)
	if err != nil {
		log.DEBUG("tryParseInfixOperator: failed to parse left expression '%s': %v", left, err)
		return nil
	}
	
	rightExpr, err := parseExpression(right)
	if err != nil {
		log.DEBUG("tryParseInfixOperator: failed to parse right expression '%s': %v", right, err)
		return nil
	}
	
	return &Opcall{
		src:  content,
		op:   op,
		args: []*Expr{leftExpr, rightExpr},
	}
}

// tryParseTernary handles ternary operator (a ? b : c)
func tryParseTernary(phase OperatorPhase, content string) *Opcall {
	op := OpRegistry["?:"]
	if op == nil || op.Phase() != phase {
		return nil
	}
	
	log.DEBUG("tryParseTernary: parsing '%s'", content)
	
	// Find ? and : operators
	qIndex := findOperatorIndex(content, "?")
	if qIndex == -1 {
		return nil
	}
	
	remaining := content[qIndex+1:]
	cIndex := findOperatorIndex(remaining, ":")
	if cIndex == -1 {
		// If we found a '?' but no ':', this is a malformed ternary expression
		// We need to return an error, but since this function returns *Opcall,
		// we'll create a special error opcall that can be detected later
		log.DEBUG("tryParseTernary: found '?' but missing ':' in '%s'", content)
		return nil
	}
	cIndex += qIndex + 1 // Adjust for full string
	
	condition := strings.TrimSpace(content[:qIndex])
	trueValue := strings.TrimSpace(content[qIndex+1 : cIndex])
	falseValue := strings.TrimSpace(content[cIndex+1:])
	
	log.DEBUG("tryParseTernary: condition='%s', true='%s', false='%s'", condition, trueValue, falseValue)
	
	if condition == "" || trueValue == "" || falseValue == "" {
		return nil
	}
	
	condExpr, err := parseExpression(condition)
	if err != nil {
		log.DEBUG("tryParseTernary: failed to parse condition '%s': %v", condition, err)
		return nil
	}
	
	trueExpr, err := parseExpression(trueValue)
	if err != nil {
		log.DEBUG("tryParseTernary: failed to parse true value '%s': %v", trueValue, err)
		return nil
	}
	
	falseExpr, err := parseExpression(falseValue)
	if err != nil {
		log.DEBUG("tryParseTernary: failed to parse false value '%s': %v", falseValue, err)
		return nil
	}
	
	return &Opcall{
		src:  content,
		op:   op,
		args: []*Expr{condExpr, trueExpr, falseExpr},
	}
}

// findOperatorIndex finds the index of an operator outside of quotes
func findOperatorIndex(content, op string) int {
	inQuotes := false
	quoteChar := byte(0)
	
	for i := 0; i <= len(content)-len(op); i++ {
		char := content[i]
		
		// Handle quotes
		if !inQuotes && (char == '"' || char == '\'') {
			inQuotes = true
			quoteChar = char
			continue
		}
		if inQuotes && char == quoteChar {
			inQuotes = false
			quoteChar = 0
			continue
		}
		
		// Skip if we're inside quotes
		if inQuotes {
			continue
		}
		
		// Check for operator
		if content[i:i+len(op)] == op {
			return i
		}
	}
	
	return -1
}

// parseExpression parses a single expression (literal, reference, or nested operator)
func parseExpression(expr string) (*Expr, error) {
	expr = strings.TrimSpace(expr)
	
	log.DEBUG("parseExpression: parsing '%s'", expr)
	
	// Check if it's a nested operator expression
	if strings.HasPrefix(expr, "((") && strings.HasSuffix(expr, "))") {
		// Parse the nested operator expression
		nestedOpcall, err := ParseOpcallCompat(EvalPhase, expr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse nested operator: %v", err)
		}
		if nestedOpcall == nil {
			// Not a valid operator expression, treat as literal
			return &Expr{
				Type:    Literal,
				Literal: expr,
			}, nil
		}
		
		// Extract operator name from the opcall
		opName := ""
		if nestedOpcall.op != nil {
			// Get the operator name via reflection or type assertion
			// For now, extract from the source string
			src := strings.TrimSpace(expr)
			src = strings.TrimPrefix(src, "((")
			src = strings.TrimSuffix(src, "))")
			src = strings.TrimSpace(src)
			parts := strings.Fields(src)
			if len(parts) > 0 {
				opName = parts[0]
			}
		}
		
		return &Expr{
			Type:     OperatorCall,
			Operator: opName,
			Call:     nestedOpcall,
		}, nil
	}
	
	// Check if it's a function-style operator call: op(args) or (op args)
	funcStyleRegex := regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9_-]*)\s*\((.*)\)$`)
	parenStyleRegex := regexp.MustCompile(`^\(([a-zA-Z][a-zA-Z0-9_-]*)\s+(.*)\)$`)
	
	if matches := funcStyleRegex.FindStringSubmatch(expr); len(matches) > 0 {
		opName := matches[1]
		argsStr := matches[2]
		
		log.DEBUG("parseExpression: found function-style operator call: %s(%s)", opName, argsStr)
		
		// Parse the nested operator call
		nestedOpcall, err := ParseOpcallCompat(EvalPhase, fmt.Sprintf("(( %s %s ))", opName, argsStr))
		if err != nil {
			return nil, err
		}
		
		if nestedOpcall != nil {
			return &Expr{
				Type:     OperatorCall,
				Operator: opName,
				Call:     nestedOpcall,
			}, nil
		}
	}
	
	// Check for paren-style: (op args)
	if matches := parenStyleRegex.FindStringSubmatch(expr); len(matches) > 0 {
		opName := matches[1]
		argsStr := matches[2]
		
		log.DEBUG("parseExpression: found paren-style operator call: (%s %s)", opName, argsStr)
		
		// Parse the nested operator call
		nestedOpcall, err := ParseOpcallCompat(EvalPhase, fmt.Sprintf("(( %s %s ))", opName, argsStr))
		if err != nil {
			log.DEBUG("parseExpression: ParseOpcallCompat failed: %v", err)
			return nil, err
		}
		
		if nestedOpcall != nil {
			return &Expr{
				Type:     OperatorCall,
				Operator: opName,
				Call:     nestedOpcall,
			}, nil
		}
	}
	
	// Check if it's a parenthesized expression like (3 * 4)
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") && len(expr) > 2 {
		inner := strings.TrimSpace(expr[1:len(expr)-1])
		
		// Check if the inner content looks like an infix expression
		// Look for operators with spaces around them
		infixOps := []string{"+", "-", "*", "/", "%", "==", "!=", "<=", ">=", "<", ">", "||", "&&"}
		hasInfixOp := false
		for _, op := range infixOps {
			if strings.Contains(inner, " "+op+" ") {
				hasInfixOp = true
				break
			}
		}
		
		if hasInfixOp {
			// This is a parenthesized infix expression, parse it as an operator expression
			log.DEBUG("parseExpression: found parenthesized infix expression: %s", inner)
			nestedOpcall, err := ParseOpcallCompat(EvalPhase, "(( " + inner + " ))")
			if err != nil {
				return nil, fmt.Errorf("failed to parse parenthesized expression: %v", err)
			}
			if nestedOpcall == nil {
				// If parsing failed, treat as literal
				return &Expr{
					Type:    Literal,
					Literal: expr,
				}, nil
			}
			
			// Extract operator name from the opcall (this is a bit hacky but works for now)
			opName := "expr" // default name
			if nestedOpcall.op != nil {
				// Try to get operator name via type assertion
				switch op := nestedOpcall.op.(type) {
				case interface{ String() string }:
					opName = op.String()
				default:
					// Use type name
					typeName := fmt.Sprintf("%T", op)
					if idx := strings.LastIndex(typeName, "."); idx >= 0 {
						typeName = typeName[idx+1:]
					}
					opName = strings.TrimSuffix(typeName, "Operator")
				}
			}
			
			return &Expr{
				Type:     OperatorCall,
				Operator: opName,
				Call:     nestedOpcall,
			}, nil
		}
	}
	
	// Try to parse as literal first (nil, booleans, numbers, strings)
	if literalExpr, err := parseLiteral(expr); err == nil {
		// If it parsed as a literal, check if it's a special keyword
		if expr == "nil" || expr == "true" || expr == "false" {
			return literalExpr, nil
		}
		// If it looks like a number or quoted string, it's definitely a literal
		if len(expr) > 0 && (expr[0] == '"' || expr[0] == '\'' || (expr[0] >= '0' && expr[0] <= '9')) {
			return literalExpr, nil
		}
		// Otherwise, continue to check if it's a reference
	}
	
	// Check if it's a reference (path.to.value)
	if isReference(expr) {
		cursor, err := tree.ParseCursor(expr)
		if err != nil {
			return nil, fmt.Errorf("invalid reference '%s': %v", expr, err)
		}
		return &Expr{
			Type:      Reference,
			Reference: cursor,
		}, nil
	}
	
	// If all else fails, try to parse as literal
	return parseLiteral(expr)
}

// isReference checks if a string looks like a reference path
func isReference(s string) bool {
	// Simple heuristic: contains dots or is a valid identifier
	if strings.Contains(s, ".") {
		return true
	}
	// Check if it's a simple identifier (not a quoted string or number)
	if len(s) > 0 && (s[0] == '"' || s[0] == '\'' || (s[0] >= '0' && s[0] <= '9')) {
		return false
	}
	// Check for simple identifier pattern
	return regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(s)
}

// unescapeString processes escape sequences in a string
func unescapeString(s string) string {
	// Handle common escape sequences
	result := strings.Builder{}
	escaped := false
	
	for i := 0; i < len(s); i++ {
		ch := s[i]
		
		if escaped {
			switch ch {
			case 'n':
				result.WriteByte('\n')
			case 'r':
				result.WriteByte('\r')
			case 't':
				result.WriteByte('\t')
			case '\\':
				result.WriteByte('\\')
			case '"':
				result.WriteByte('"')
			case '\'':
				result.WriteByte('\'')
			default:
				// If it's not a recognized escape sequence, include the backslash
				result.WriteByte('\\')
				result.WriteByte(ch)
			}
			escaped = false
			continue
		}
		
		if ch == '\\' {
			escaped = true
			continue
		}
		
		result.WriteByte(ch)
	}
	
	// If we ended with a backslash, include it
	if escaped {
		result.WriteByte('\\')
	}
	
	return result.String()
}

// parseLiteral parses a literal value (string, number, boolean, nil)
func parseLiteral(s string) (*Expr, error) {
	log.DEBUG("parseLiteral: parsing '%s'", s)
	
	// Handle nil
	if s == "nil" || s == "~" {
		return &Expr{
			Type:    Literal,
			Literal: nil,
		}, nil
	}
	
	// Handle booleans
	if s == "true" {
		return &Expr{
			Type:    Literal,
			Literal: true,
		}, nil
	}
	if s == "false" {
		return &Expr{
			Type:    Literal,
			Literal: false,
		}, nil
	}
	
	// Handle quoted strings
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		// Remove quotes and process escape sequences
		unquoted := s[1 : len(s)-1]
		unescaped := unescapeString(unquoted)
		return &Expr{
			Type:    Literal,
			Literal: unescaped,
		}, nil
	}
	
	// Try to parse as number
	if val, err := strconv.ParseInt(s, 10, 64); err == nil {
		return &Expr{
			Type:    Literal,
			Literal: val,
		}, nil
	}
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return &Expr{
			Type:    Literal,
			Literal: val,
		}, nil
	}
	
	// If all else fails, treat as string literal
	return &Expr{
		Type:    Literal,
		Literal: s,
	}, nil
}

// ParseOpcall parses an operator call expression  
func ParseOpcall(phase OperatorPhase, src string) (*Opcall, error) {
	// Basic implementation - this will be enhanced later
	log.DEBUG("ParseOpcall: parsing '%s' for phase %v", src, phase)
	if strings.Contains(src, "base + addend") {
		log.DEBUG("ParseOpcall: found 'base + addend' expression")
	}
	
	// Removed the early check for infix operators - let the regular parsing try first
	
	// Check if it's an operator expression
	// Note: The first pattern's optional group (?:\s*\((.*)\))? will NOT match when there's a space
	// between operator and parentheses, so "file (concat...)" will fall through to second pattern
	for _, pattern := range []string{
		`^\(\(\s*([a-zA-Z][a-zA-Z0-9_-]*)\((.*)\)\s*\)\)$`, // (( op(x,y,z) )) - no space between op and (
		`^\(\(\s*([a-zA-Z][a-zA-Z0-9_-]*)\s+(\(.*\))\s*\)\)$`, // (( op (x,y,z) )) - space between op and (
		`^\(\(\s*([a-zA-Z][a-zA-Z0-9_-]*)(?:\s+(.*))?\s*\)\)$`,     // (( op x y z ))
	} {
		re := regexp.MustCompile(pattern)
		if !re.MatchString(src) {
			continue
		}
		log.DEBUG("ParseOpcall: matched pattern %s", pattern)

		m := re.FindStringSubmatch(src)
		log.DEBUG("parsing `%s': looks like a (( %s ... )) operator", src, m[1])

		// Special case: BOSH variable syntax ((var-name)) should be ignored
		// m[2] contains the arguments, if it's empty and name contains dash, it's likely BOSH syntax
		// But first check if it's actually a registered operator
		if strings.Contains(m[1], "-") && (len(m) < 3 || m[2] == "") {
			// Only ignore if it's not a registered operator
			if OpRegistry[m[1]] == nil {
				log.DEBUG("  - ignoring BOSH variable syntax: %s", m[1])
				return nil, nil
			}
		}

		op := OpRegistry[m[1]]
		if op == nil {
			log.DEBUG("  - unknown operator: %s, creating NullOperator", m[1])
			// Create a NullOperator for unknown operators
			op = NullOperator{Missing: m[1]}
		} else if op.Phase() != phase {
			log.DEBUG("  - skipping (( %s ... )) operation; it belongs to a different phase", m[1])
			return nil, nil
		}

		// Parse arguments - pass the phase to parseArgs
		log.DEBUG("ParseOpcall: calling parseArgs with: '%s'", m[2])
		args, err := parseArgs(phase, m[2])
		if err != nil {
			return nil, err
		}
		log.DEBUG("ParseOpcall: parseArgs returned %d arguments", len(args))

		return &Opcall{
			src:  src,
			op:   op,
			args: args,
		}, nil
	}

	// If we couldn't parse it as a standard operator but it looks like an operator expression,
	// check if it might be an infix expression
	trimmed := strings.TrimSpace(src)
	if strings.HasPrefix(trimmed, "((") && strings.HasSuffix(trimmed, "))") {
		inner := strings.TrimSpace(trimmed[2:len(trimmed)-2])
		
		// Check if it contains infix operators
		// Look for operators with spaces around them
		infixOps := []string{"+", "-", "*", "/", "%", "==", "!=", "<=", ">=", "<", ">", "||", "&&", "?"}
		
		for _, op := range infixOps {
			// Check for operator with spaces
			if strings.Contains(inner, " "+op+" ") {
				log.DEBUG("ParseOpcall: detected potential infix expression with operator '%s' in '%s', returning NullOperator", op, inner)
				// Return a NullOperator to signal ParseOpcallCompat to try infix parsing
				return &Opcall{
					src: src,
					op: NullOperator{Missing: "__infix__"},
					args: []*Expr{},
				}, nil
			}
		}
	}

	return nil, nil
}

// parseArgs parses operator arguments with support for || operator
func parseArgs(phase OperatorPhase, src string) ([]*Expr, error) {
	if src == "" {
		return []*Expr{}, nil
	}

	log.DEBUG("parseArgs: parsing arguments: '%s'", src)

	// Handle arguments with different separators
	args := []*Expr{}
	
	// Check if the entire argument list is wrapped in parentheses
	src = strings.TrimSpace(src)
	// Don't strip parentheses if they might be part of a nested expression
	// We'll let tokenizeRespectingQuotes handle parentheses properly
	
	// Handle comma-separated arguments for operators like concat
	parts := splitRespectingQuotes(src, ",")
	if len(parts) > 1 {
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			
			// Check if this part has || operator
			if strings.Contains(part, "||") {
				// Check for invalid syntax: || at the end with nothing after
				if strings.HasSuffix(strings.TrimSpace(part), "||") {
					// Extract what comes before ||
					beforeOr := strings.TrimSpace(part[:strings.LastIndex(part, "||")])
					if beforeOr != "" {
						return nil, fmt.Errorf("unexpected token '%s' after expression", beforeOr)
					}
					return nil, fmt.Errorf("unexpected end of expression")
				}
				
				subParts := splitRespectingQuotes(part, "||")
				if len(subParts) > 1 {
					// Build a LogicalOr expression for this argument
					var expr *Expr
					
					for i := len(subParts) - 1; i >= 0; i-- {
						subPart := strings.TrimSpace(subParts[i])
						if subPart == "" {
							return nil, fmt.Errorf("unexpected end of expression")
						}
						parsedExpr, err := parseSingleExpression(subPart)
						if err != nil {
							return nil, err
						}
						
						if expr == nil {
							expr = parsedExpr
						} else {
							expr = &Expr{
								Type:  LogicalOr,
								Left:  parsedExpr,
								Right: expr,
							}
						}
					}
					
					args = append(args, expr)
					continue
				}
			}
			
			expr, err := parseSingleExpression(part)
			if err != nil {
				return nil, err
			}
			args = append(args, expr)
		}
		return args, nil
	}
	
	// Otherwise parse space-separated arguments (for operators like concat without commas)
	// Use quote-aware tokenization instead of strings.Fields
	parts = tokenizeRespectingQuotes(src)
	log.DEBUG("parseArgs: tokenized '%s' into %d parts: %v", src, len(parts), parts)
	i := 0
	for i < len(parts) {
		part := parts[i]
		
		// Check for standalone || operator (invalid syntax)
		if part == "||" {
			// Check if this is at the beginning (nothing before ||)
			if i == 0 {
				return nil, fmt.Errorf("unexpected end of expression")
			}
			// Check if this is at the end (nothing after ||)
			if i == len(parts)-1 {
				// Get what came before ||
				beforeOr := strings.TrimSpace(parts[i-1])
				return nil, fmt.Errorf("unexpected token '%s' after expression", beforeOr)
			}
			// Check for || || (double ||)
			if i+1 < len(parts) && parts[i+1] == "||" {
				// Get what came before the first ||
				if i > 0 {
					beforeOr := strings.TrimSpace(parts[i-1])
					return nil, fmt.Errorf("unexpected token '%s' after expression", beforeOr)
				}
				return nil, fmt.Errorf("unexpected end of expression")
			}
		}
		
		// Check if this is the start of a || expression
		if i+2 < len(parts) && parts[i+1] == "||" {
			// Check if the right side is another || (invalid syntax)
			if parts[i+2] == "||" {
				return nil, fmt.Errorf("unexpected token '%s' after expression", part)
			}
			
			// Build LogicalOr expression
			left, err := parseSingleExpression(part)
			if err != nil {
				return nil, err
			}
			
			right, err := parseSingleExpression(parts[i+2])
			if err != nil {
				return nil, err
			}
			
			expr := &Expr{
				Type:  LogicalOr,
				Left:  left,
				Right: right,
			}
			
			// Check if there are more || operators
			i += 3
			for i+1 < len(parts) && parts[i] == "||" {
				nextExpr, err := parseSingleExpression(parts[i+1])
				if err != nil {
					return nil, err
				}
				expr = &Expr{
					Type:  LogicalOr,
					Left:  expr,
					Right: nextExpr,
				}
				i += 2
			}
			
			args = append(args, expr)
		} else {
			// Regular argument
			expr, err := parseSingleExpression(part)
			if err != nil {
				return nil, err
			}
			args = append(args, expr)
			i++
		}
	}
	
	return args, nil
}

// splitRespectingQuotes splits a string by delimiter but respects quoted strings
func splitRespectingQuotes(s string, delimiter string) []string {
	var result []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)
	
	runes := []rune(s)
	delimRunes := []rune(delimiter)
	
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		
		// Check for quote characters
		if !inQuotes && (r == '"' || r == '\'') {
			inQuotes = true
			quoteChar = r
			current.WriteRune(r)
			continue
		} else if inQuotes && r == quoteChar {
			inQuotes = false
			quoteChar = 0
			current.WriteRune(r)
			continue
		}
		
		// Check for delimiter
		if !inQuotes && i+len(delimRunes) <= len(runes) {
			match := true
			for j, dr := range delimRunes {
				if runes[i+j] != dr {
					match = false
					break
				}
			}
			if match {
				result = append(result, current.String())
				current.Reset()
				i += len(delimRunes) - 1
				continue
			}
		}
		
		current.WriteRune(r)
	}
	
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	
	return result
}

// parseSingleExpression parses a single expression (no operators)
func parseSingleExpression(s string) (*Expr, error) {
	s = strings.TrimSpace(s)
	
	// Check for special literals
	switch s {
	case "nil", "null", "~":
		return &Expr{Type: Literal, Literal: nil}, nil
	case "true", "TRUE", "True":
		return &Expr{Type: Literal, Literal: true}, nil
	case "false", "FALSE", "False":
		return &Expr{Type: Literal, Literal: false}, nil
	}
	
	// Check for environment variable
	if strings.HasPrefix(s, "$") {
		return &Expr{Type: EnvVar, Name: s[1:]}, nil
	}
	
	// Check for quoted string
	if (strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") && len(s) >= 2) ||
	   (strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") && len(s) >= 2) {
		// Remove quotes and process escape sequences
		unquoted := s[1:len(s)-1]
		unescaped := unescapeString(unquoted)
		return &Expr{Type: Literal, Literal: unescaped}, nil
	}
	
	// Try to parse as number
	if strings.HasPrefix(s, ".") || strings.Contains(s, ".") {
		// Try float
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return &Expr{Type: Literal, Literal: f}, nil
		}
	} else {
		// Try int first
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return &Expr{Type: Literal, Literal: i}, nil
		}
		// Try float
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return &Expr{Type: Literal, Literal: f}, nil
		}
	}
	
	// Try to parse as negative number
	if strings.HasPrefix(s, "-") && len(s) > 1 {
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return &Expr{Type: Literal, Literal: i}, nil
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return &Expr{Type: Literal, Literal: f}, nil
		}
	}
	
	// Check for additional nil keywords that should be treated as nil literals
	switch s {
	case "Nil", "NIL", "Null", "NULL":
		return &Expr{Type: Literal, Literal: nil}, nil
	}
	
	// Check if it's a function-style expression: (op args)
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") && len(s) > 2 {
		// Check if this looks like an operator expression
		inner := strings.TrimSpace(s[1:len(s)-1])
		parts := strings.Fields(inner)
		if len(parts) > 0 {
			// Check if the first part is a known operator
			if _, isOp := OpRegistry[parts[0]]; isOp {
				// This is a nested operator call without (( ))
				// We need to wrap it in (( )) for ParseOpcallCompat
				wrappedExpr := fmt.Sprintf("(( %s ))", inner)
				opcall, err := ParseOpcallCompat(EvalPhase, wrappedExpr)
				if err != nil {
					return nil, err
				}
				if opcall != nil {
					return &Expr{
						Type:     OperatorCall,
						Operator: parts[0],
						Call:     opcall,
					}, nil
				}
			}
		}
		
		// Otherwise try to parse it as a general expression
		return parseExpression(s)
	}
	
	// Otherwise it's a reference
	// Handle special cases for references that might not be valid cursors
	if s == "TrUe" || s == "FaLSe" || s == "NuLL" {
		// These are references to actual keys in the YAML, not literals
		cursor, err := tree.ParseCursor(s)
		if err != nil {
			return nil, fmt.Errorf("invalid reference: %s", s)
		}
		return &Expr{Type: Reference, Reference: cursor}, nil
	}
	
	cursor, err := tree.ParseCursor(s)
	if err != nil {
		return nil, fmt.Errorf("invalid reference: %s", s)
	}
	return &Expr{Type: Reference, Reference: cursor}, nil
}

// tokenizeRespectingQuotes splits a string on whitespace but respects quoted strings and parentheses
func tokenizeRespectingQuotes(s string) []string {
	var result []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)
	parenDepth := 0
	
	runes := []rune(s)
	
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		
		// Check for quote characters
		if !inQuotes && (r == '"' || r == '\'') {
			inQuotes = true
			quoteChar = r
			current.WriteRune(r)
			continue
		} else if inQuotes && r == quoteChar {
			inQuotes = false
			quoteChar = 0
			current.WriteRune(r)
			continue
		}
		
		// If we're in quotes, add everything including spaces
		if inQuotes {
			current.WriteRune(r)
			continue
		}
		
		// Handle parentheses
		if r == '(' {
			parenDepth++
			current.WriteRune(r)
			continue
		} else if r == ')' {
			parenDepth--
			current.WriteRune(r)
			continue
		}
		
		// If we're inside parentheses, keep everything together
		if parenDepth > 0 {
			current.WriteRune(r)
			continue
		}
		
		// If we're not in quotes or parentheses and hit whitespace, finish current token
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
			continue
		}
		
		// Regular character outside quotes and parentheses
		current.WriteRune(r)
	}
	
	// Add the last token if any
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	
	return result
}

// Operator precedence levels (higher number = higher precedence)
var operatorPrecedence = map[string]int{
	"?:": 1,  // ternary (lowest precedence)
	"?":  1,  // ternary question mark (for parsing)
	// Note: ":" is not in precedence table because it's handled specially in ternary context
	"||": 2,  // logical or / fallback
	"&&": 3,  // logical and
	"==": 4,  // equality
	"!=": 4,  // inequality
	"<":  5,  // less than
	">":  5,  // greater than
	"<=": 5,  // less than or equal
	">=": 5,  // greater than or equal
	"+":  6,  // addition
	"-":  6,  // subtraction
	"*":  7,  // multiplication
	"/":  7,  // division
	"%":  7,  // modulo
	"!":  8,  // logical not (highest precedence, prefix operator)
}

// parseExpressionWithPrecedence parses expressions with proper operator precedence
func parseExpressionWithPrecedence(phase OperatorPhase, content string) (*Opcall, error) {
	log.DEBUG("parseExpressionWithPrecedence: parsing content '%s'", content)
	
	// Parse expression using precedence climbing algorithm
	expr, err := parseExpressionPrecedence(content, 0)
	if err != nil {
		log.DEBUG("parseExpressionWithPrecedence: failed to parse: %v", err)
		return nil, err
	}
	
	if expr == nil {
		log.DEBUG("parseExpressionWithPrecedence: parseExpressionPrecedence returned nil")
		return nil, nil
	}
	
	log.DEBUG("parseExpressionWithPrecedence: parsed expression tree with root operator '%s'", expr.Value)
	
	// Convert expression to Opcall
	return exprToOpcall(phase, expr), nil
}

// Token represents a token in the expression
type Token struct {
	Type  string      // "operator", "operand", "lparen", "rparen"
	Value string      // the actual token string
	Pos   int         // position in original string
}

// TokenizeExpressionForTest is a wrapper for testing
func TokenizeExpressionForTest(content string) ([]Token, error) {
	return tokenizeExpression(content)
}

// tokenizeExpression tokenizes an expression for parsing
func tokenizeExpression(content string) ([]Token, error) {
	log.DEBUG("tokenizeExpression: tokenizing '%s'", content)
	var tokens []Token
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)
	pos := 0
	
	runes := []rune(content)
	
	flushToken := func() {
		if current.Len() > 0 {
			str := strings.TrimSpace(current.String())
			if str != "" {
				tokens = append(tokens, Token{Type: "operand", Value: str, Pos: pos - len(str)})
			}
			current.Reset()
		}
	}
	
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		pos = i
		
		// Handle quotes
		if !inQuotes && (r == '"' || r == '\'') {
			inQuotes = true
			quoteChar = r
			current.WriteRune(r)
			continue
		} else if inQuotes && r == quoteChar {
			inQuotes = false
			quoteChar = 0
			current.WriteRune(r)
			continue
		}
		
		if inQuotes {
			current.WriteRune(r)
			continue
		}
		
		// Handle parentheses - check for parenthesized operator calls first
		if r == '(' {
			flushToken()
			
			// Look ahead to see if this is a parenthesized operator call like (grab base)
			// Find the matching closing parenthesis
			parenCount := 1
			j := i + 1
			for j < len(runes) && parenCount > 0 {
				if runes[j] == '(' {
					parenCount++
				} else if runes[j] == ')' {
					parenCount--
				}
				j++
			}
			
			if parenCount == 0 {
				// We found the matching closing paren
				parenContent := string(runes[i+1 : j-1]) // content between parens
				
				// Check if this looks like an operator call: "operator args"
				parenContent = strings.TrimSpace(parenContent)
				if parenContent != "" {
					// Check if it starts with an identifier (potential operator name)
					parts := strings.Fields(parenContent)
					if len(parts) > 0 {
						// Check if the first part looks like an operator name
						firstPart := parts[0]
						isOperatorName := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`).MatchString(firstPart)
						
						// Also check if this contains arithmetic operators, which means it's an expression, not an operator call
						hasArithmeticOp := false
						for _, op := range []string{"+", "-", "*", "/", "%", "==", "!=", "<=", ">=", "<", ">", "||", "&&"} {
							if strings.Contains(parenContent, " "+op+" ") {
								hasArithmeticOp = true
								break
							}
						}
						
						if isOperatorName && !hasArithmeticOp {
							// This looks like a parenthesized operator call, treat as single operand
							fullExpr := string(runes[i:j])
							tokens = append(tokens, Token{Type: "operand", Value: fullExpr, Pos: i})
							i = j - 1 // Skip past the entire parenthesized expression
							continue
						}
					}
				}
			}
			
			// Not a parenthesized operator call, treat as regular parenthesis
			tokens = append(tokens, Token{Type: "lparen", Value: "(", Pos: i})
			continue
		}
		if r == ')' {
			flushToken()
			tokens = append(tokens, Token{Type: "rparen", Value: ")", Pos: i})
			continue
		}
		
		// Check for multi-character operators
		if i+1 < len(runes) {
			twoChar := string(runes[i:i+2])
			if _, exists := operatorPrecedence[twoChar]; exists {
				flushToken()
				tokens = append(tokens, Token{Type: "operator", Value: twoChar, Pos: i})
				i++ // skip next character
				continue
			}
		}
		
		// Check for single-character operators
		oneChar := string(r)
		// Include : for tokenization even though it's not in precedence table
		if _, exists := operatorPrecedence[oneChar]; exists || oneChar == ":" {
			flushToken()
			tokens = append(tokens, Token{Type: "operator", Value: oneChar, Pos: i})
			continue
		}
		
		// Handle whitespace
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			flushToken()
			continue
		}
		
		// Regular character
		current.WriteRune(r)
	}
	
	flushToken()
	
	// Debug log the tokens
	log.DEBUG("tokenizeExpression: produced %d tokens", len(tokens))
	for i, tok := range tokens {
		log.DEBUG("  token[%d]: type=%s, value='%s', pos=%d", i, tok.Type, tok.Value, tok.Pos)
	}
	
	return tokens, nil
}

// ExprNode represents a node in the expression tree
type ExprNode struct {
	Type     string    // "operator", "operand"
	Value    string    // operator name or operand value
	Left     *ExprNode // left child
	Right    *ExprNode // right child
	Children []*ExprNode // for ternary operator (condition, true_val, false_val)
}

// parseExpressionPrecedence implements precedence climbing algorithm
func parseExpressionPrecedence(content string, minPrec int) (*ExprNode, error) {
	tokens, err := tokenizeExpression(content)
	if err != nil {
		return nil, err
	}
	
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty expression")
	}
	
	pos := 0
	return parseExpressionTokens(&tokens, &pos, minPrec)
}

// parseExpressionTokens parses expression with precedence climbing
func parseExpressionTokens(tokens *[]Token, pos *int, minPrec int) (*ExprNode, error) {
	// Parse left operand
	left, err := parsePrimary(tokens, pos)
	if err != nil {
		return nil, err
	}
	
	for *pos < len(*tokens) {
		token := (*tokens)[*pos]
		
		if token.Type != "operator" {
			break
		}
		
		prec, exists := operatorPrecedence[token.Value]
		if !exists || prec < minPrec {
			log.DEBUG("parseExpressionTokens: stopping at operator '%s' (prec %d < minPrec %d)", token.Value, prec, minPrec)
			break
		}
		
		// Special case for ternary operator
		if token.Value == "?" {
			log.DEBUG("parseExpressionTokens: found ternary operator at pos %d", *pos)
			return parseTernaryFromLeft(tokens, pos, left)
		}
		
		op := token.Value
		*pos++ // consume operator
		
		log.DEBUG("parseExpressionTokens: parsing operator '%s' at pos %d", op, *pos-1)
		
		// Parse right operand with higher precedence
		right, err := parseExpressionTokens(tokens, pos, prec+1)
		if err != nil {
			return nil, err
		}
		
		left = &ExprNode{
			Type:  "operator",
			Value: op,
			Left:  left,
			Right: right,
		}
	}
	
	return left, nil
}

// parsePrimary parses primary expressions (operands, parenthesized expressions, prefix operators)
func parsePrimary(tokens *[]Token, pos *int) (*ExprNode, error) {
	if *pos >= len(*tokens) {
		return nil, fmt.Errorf("unexpected end of expression")
	}
	
	token := (*tokens)[*pos]
	
	// Check for prefix operators (like !)
	if token.Type == "operator" && token.Value == "!" {
		*pos++ // consume '!'
		
		// Parse the operand after the prefix operator
		operand, err := parsePrimary(tokens, pos)
		if err != nil {
			return nil, err
		}
		
		return &ExprNode{
			Type:  "operator",
			Value: "!",
			Left:  operand,
		}, nil
	}
	
	if token.Type == "operand" {
		*pos++
		return &ExprNode{
			Type:  "operand",
			Value: token.Value,
		}, nil
	}
	
	if token.Type == "lparen" {
		*pos++ // consume '('
		
		expr, err := parseExpressionTokens(tokens, pos, 0)
		if err != nil {
			return nil, err
		}
		
		if *pos >= len(*tokens) || (*tokens)[*pos].Type != "rparen" {
			return nil, fmt.Errorf("missing closing parenthesis")
		}
		*pos++ // consume ')'
		
		return expr, nil
	}
	
	return nil, fmt.Errorf("unexpected token: %s", token.Value)
}

// parseTernaryFromLeft handles ternary operator parsing
func parseTernaryFromLeft(tokens *[]Token, pos *int, condition *ExprNode) (*ExprNode, error) {
	log.DEBUG("parseTernaryFromLeft: parsing ternary with condition %s", condition.Value)
	
	if *pos >= len(*tokens) || (*tokens)[*pos].Value != "?" {
		return nil, fmt.Errorf("expected '?' for ternary operator")
	}
	*pos++ // consume '?'
	
	// Parse true value - parse with precedence higher than ternary to avoid consuming ':'
	trueVal, err := parseExpressionTokens(tokens, pos, 2) // precedence 2 is higher than ternary (1)
	if err != nil {
		return nil, err
	}
	
	log.DEBUG("parseTernaryFromLeft: parsed true value")
	
	// Expect ':'
	if *pos >= len(*tokens) || (*tokens)[*pos].Value != ":" {
		return nil, fmt.Errorf("expected ':' for ternary operator")
	}
	*pos++ // consume ':'
	
	// Parse false value - parse with ternary precedence to allow nested ternaries
	falseVal, err := parseExpressionTokens(tokens, pos, 1) // precedence 1 allows nested ternaries
	if err != nil {
		return nil, err
	}
	
	log.DEBUG("parseTernaryFromLeft: parsed false value, creating ternary node")
	
	return &ExprNode{
		Type:     "operator",
		Value:    "?:",
		Children: []*ExprNode{condition, trueVal, falseVal},
	}, nil
}

// exprToOpcall converts an ExprNode to an Opcall
func exprToOpcall(phase OperatorPhase, node *ExprNode) *Opcall {
	if node == nil {
		return nil
	}
	
	log.DEBUG("exprToOpcall: converting node type=%s, value=%s", node.Type, node.Value)
	
	if node.Type == "operand" {
		// This is a simple operand, not an operation
		return nil
	}
	
	if node.Type == "operator" {
		op := OpRegistry[node.Value]
		if op == nil {
			log.DEBUG("exprToOpcall: operator '%s' not found in registry", node.Value)
			return nil
		}
		if op.Phase() != phase {
			log.DEBUG("exprToOpcall: operator '%s' wrong phase (got %v, expected %v)", node.Value, op.Phase(), phase)
			return nil
		}
		
		var args []*Expr
		
		if node.Value == "?:" && len(node.Children) == 3 {
			// Ternary operator
			for _, child := range node.Children {
				expr := nodeToExpr(child)
				if expr != nil {
					args = append(args, expr)
				}
			}
		} else {
			// Binary operator
			if node.Left != nil {
				expr := nodeToExpr(node.Left)
				if expr != nil {
					args = append(args, expr)
				}
			}
			if node.Right != nil {
				expr := nodeToExpr(node.Right)
				if expr != nil {
					args = append(args, expr)
				}
			}
		}
		
		return NewOpcall(op, args, fmt.Sprintf("(( %s ))", node.Value))
	}
	
	return nil
}

// nodeToExpr converts an ExprNode to an Expr
func nodeToExpr(node *ExprNode) *Expr {
	if node == nil {
		return nil
	}
	
	if node.Type == "operand" {
		// Parse the operand value
		log.DEBUG("nodeToExpr: parsing operand '%s'", node.Value)
		expr, err := parseExpression(node.Value)
		if err != nil {
			log.DEBUG("nodeToExpr: parseExpression failed: %v", err)
			// If parsing fails, treat as string literal
			return &Expr{
				Type:    Literal,
				Literal: node.Value,
			}
		}
		log.DEBUG("nodeToExpr: parseExpression returned expr type=%v", expr.Type)
		return expr
	}
	
	if node.Type == "operator" {
		// Create a nested OperatorCall expression
		return &Expr{
			Type:     OperatorCall,
			Operator: node.Value, // Set the operator name
			Call:     exprToOpcall(EvalPhase, node), // Always use EvalPhase for nested calls
		}
	}
	
	return nil
}