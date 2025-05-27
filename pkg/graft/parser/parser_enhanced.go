package parser

import (
	"github.com/wayneeseguin/graft/pkg/graft"
)
import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"github.com/wayneeseguin/graft/log"
	"github.com/starkandwayne/goutils/tree"
)

// EnhancedParser implements a precedence-climbing parser for Graft expressions
type EnhancedParser struct {
	tokens   []Token
	current  int
	registry *OperatorRegistry
	source   string
	errors   *ErrorRecoveryContext
}

// NewEnhancedParser creates a new parser with the given tokens
func NewEnhancedParser(tokens []Token, registry *OperatorRegistry) *EnhancedParser {
	// Check if error collection is enabled
	collectErrors := os.Getenv("GRAFT_COLLECT_ERRORS") == "1"
	maxErrors := 10
	if !collectErrors {
		maxErrors = 1
	}
	
	return &EnhancedParser{
		tokens:   tokens,
		current:  0,
		registry: registry,
		errors:   NewErrorRecoveryContext(maxErrors),
	}
}

// WithSource sets the source code for better error messages
func (p *EnhancedParser) WithSource(source string) *EnhancedParser {
	p.source = source
	return p
}

// WithErrorRecovery configures error recovery behavior
func (p *EnhancedParser) WithErrorRecovery(stopOnFirst bool, maxErrors int) *EnhancedParser {
	p.errors.StopOnFirst = stopOnFirst
	p.errors.MaxErrors = maxErrors
	return p
}

// EnableErrorCollection enables collecting multiple errors
func (p *EnhancedParser) EnableErrorCollection() *EnhancedParser {
	p.errors.StopOnFirst = false
	return p
}

// Parse parses the tokens into an expression tree
func (p *EnhancedParser) Parse() (*Expr, error) {
	if len(p.tokens) == 0 {
		return nil, p.syntaxError("no tokens to parse", Position{Line: 1, Column: 1})
	}
	
	expr, err := p.parseExpression(PrecedenceLowest)
	if err != nil {
		return nil, err
	}
	
	if !p.isAtEnd() {
		err := p.syntaxError(fmt.Sprintf("unexpected token '%s' after expression", p.currentToken().Value), p.tokenPosition(p.currentToken()))
		if !p.errors.RecordError(err) {
			return nil, p.errors.GetError()
		}
		// Try to recover by skipping tokens
		p.synchronize()
	}
	
	if p.errors.HasErrors() {
		return nil, p.errors.GetError()
	}
	
	return expr, nil
}

// ParseExpression is a convenience function that uses memoization for common expressions
func ParseExpression(input string, registry *OperatorRegistry) (*Expr, error) {
	// Record pattern for analytics
	GlobalPatternTracker.RecordPattern(input)
	
	// Use memoized parser
	parser := NewMemoizedEnhancedParser(input, registry)
	return parser.Parse()
}

// ParseMultiple parses multiple space-separated expressions
// This is used for parsing operator arguments
func (p *EnhancedParser) ParseMultiple() ([]*Expr, error) {
	if len(p.tokens) == 0 {
		return []*Expr{}, nil
	}
	
	expressions := make([]*Expr, 0, 4) // Pre-allocate for typical arg counts
	
	for !p.isAtEnd() {
		// Skip commas between arguments
		if p.currentToken().Type == TokenComma {
			p.advance()
			continue
		}
		
		// For operator arguments, we parse at the primary level
		// This prevents operators from consuming multiple arguments
		expr, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, expr)
	}
	
	return expressions, nil
}

// parseExpression parses an expression with precedence climbing
func (p *EnhancedParser) parseExpression(minPrecedence Precedence) (*Expr, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	
	for !p.isAtEnd() {
		// Check if current token is an infix operator
		token := p.currentToken()
		
		// Handle ternary operator specially
		if token.Type == TokenQuestion {
			if minPrecedence > PrecedenceTernary {
				break
			}
			
			p.advance() // consume ?
			
			// Parse the true expression
			trueExpr, err := p.parseExpression(PrecedenceTernary + 1)
			if err != nil {
				return nil, err
			}
			
			// Expect :
			if !p.consume(TokenColon) {
				return nil, p.syntaxError("expected ':' in ternary expression", p.tokenPosition(p.currentToken())).
					WithContext("ternary operator requires '?' followed by ':'")
			}
			
			// Parse the false expression
			// Ternary is right-associative
			falseExpr, err := p.parseExpression(PrecedenceTernary)
			if err != nil {
				return nil, err
			}
			
			// Create a ternary operator call
			left = NewOperatorCallWithPos("?:", []*Expr{left, trueExpr, falseExpr}, left.Pos)
			continue
		}
		
		// Note: TokenLogicalOr is now handled in the switch below as a binary operator
		
		// Handle comma (for function arguments)
		if token.Type == TokenComma {
			break // Let parent handle collecting arguments
		}
		
		// Handle operators and arithmetic tokens
		var opName string
		isOperatorToken := false
		
		switch token.Type {
		case TokenOperator:
			opName = token.Value
			isOperatorToken = true
		case TokenLogicalOr:
			opName = "||"
			isOperatorToken = true
		case TokenLogicalAnd:
			opName = "&&"
			isOperatorToken = true
		case TokenEquals:
			opName = "=="
			isOperatorToken = true
		case TokenNotEquals:
			opName = "!="
			isOperatorToken = true
		case TokenLessThan:
			opName = "<"
			isOperatorToken = true
		case TokenGreaterThan:
			opName = ">"
			isOperatorToken = true
		case TokenLessEqual:
			opName = "<="
			isOperatorToken = true
		case TokenGreaterEqual:
			opName = ">="
			isOperatorToken = true
		case TokenPlus:
			opName = "+"
			isOperatorToken = true
		case TokenMinus:
			opName = "-"
			isOperatorToken = true
		case TokenMultiply:
			opName = "*"
			isOperatorToken = true
		case TokenDivide:
			opName = "/"
			isOperatorToken = true
		case TokenModulo:
			opName = "%"
			isOperatorToken = true
		}
		
		if isOperatorToken {
			// Special handling for || which is not a registered operator
			// but a special expression type (LogicalOr)
			if opName == "||" {
				// || has very low precedence (PrecedenceOr)
				if PrecedenceOr < minPrecedence {
					break
				}
				
				p.advance() // consume ||
				
				// || is right-associative
				right, err := p.parseExpression(PrecedenceOr)
				if err != nil {
					return nil, err
				}
				
				left = &Expr{
					Type:  LogicalOr,
					Left:  left,
					Right: right,
					Pos:   left.Pos,
				}
				continue
			}
			
			opInfo := p.registry.Get(opName)
			if opInfo == nil {
				// Not a known operator, might be part of next expression
				break
			}
			
			if opInfo.Precedence < minPrecedence {
				break
			}
			
			// Check if this could be a binary operator
			if p.canBeBinaryOperator(opInfo, left) {
				nextPrecedence := opInfo.Precedence
				if opInfo.Associativity == AssociativityLeft {
					nextPrecedence++
				}
				
				p.advance() // consume operator
				right, err := p.parseExpression(nextPrecedence)
				if err != nil {
					return nil, err
				}
				
				// Create binary operator expression
				left = NewOperatorCallWithPos(opInfo.Name, []*Expr{left, right}, left.Pos)
				continue
			}
		}
		
		// If we get here, we can't continue parsing
		break
	}
	
	return left, nil
}

// parsePrimary parses a primary expression (literal, reference, parenthesized, or operator call)
func (p *EnhancedParser) parsePrimary() (*Expr, error) {
	if p.isAtEnd() {
		return nil, p.syntaxError("unexpected end of expression", Position{Line: 1, Column: 1})
	}
	
	token := p.currentToken()
	
	switch token.Type {
	case TokenLiteral:
		p.advance()
		return p.parseLiteral(token.Value, p.tokenPosition(token))
		
	case TokenReference:
		p.advance()
		cursor, err := tree.ParseCursor(token.Value)
		if err != nil {
			return nil, p.syntaxError(fmt.Sprintf("invalid reference '%s': %v", token.Value, err), p.tokenPosition(token))
		}
		DEBUG("parser: creating reference expression for '%s'", token.Value)
		return &Expr{
			Type:      Reference,
			Reference: cursor,
			Pos:       p.tokenPosition(token),
		}, nil
		
	case TokenEnvVar:
		p.advance()
		// Remove the $ prefix from environment variable
		name := token.Value
		if strings.HasPrefix(name, "$") {
			name = name[1:]
		}
		return &Expr{
			Type: EnvVar,
			Name: name,
			Pos:  p.tokenPosition(token),
		}, nil
		
	case TokenOpenParen:
		p.advance() // consume (
		expr, err := p.parseExpression(PrecedenceLowest)
		if err != nil {
			return nil, err
		}
		if !p.consume(TokenCloseParen) {
			return nil, p.expectToken(TokenCloseParen, "to match opening parenthesis")
		}
		return expr, nil
		
	case TokenOperator:
		// This could be a prefix operator, a function call, or a reference
		// Check if it's a unary operator
		if token.Value == "!" {
			p.advance() // consume !
			arg, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}
			return NewOperatorCallWithPos("!", []*Expr{arg}, p.tokenPosition(token)), nil
		}
		
		// Check if this looks like an operator call (has arguments)
		// If the next token suggests this is not an operator call, treat as reference
		nextToken := p.peek()
		if nextToken.Type == TokenCloseParen || nextToken.Type == TokenComma || 
		   nextToken.Type == TokenEOF || p.isBinaryOperatorToken(nextToken.Type) {
			// No arguments follow, so this is likely a reference
			p.advance()
			cursor, err := tree.ParseCursor(token.Value)
			if err != nil {
				return nil, p.syntaxError(fmt.Sprintf("invalid reference '%s': %v", token.Value, err), p.tokenPosition(token))
			}
			return &Expr{
				Type:      Reference,
				Reference: cursor,
				Pos:       p.tokenPosition(token),
			}, nil
		}
		
		return p.parseOperatorCall()
		
	case TokenMinus:
		// This could be a negative number or a minus operator
		// For now, treat it as negative number if followed by a literal
		nextToken := p.peek()
		if nextToken.Type == TokenLiteral {
			p.advance() // consume -
			numToken := p.currentToken()
			p.advance()
			return p.parseLiteral("-" + numToken.Value, p.tokenPosition(token))
		}
		// Otherwise treat as operator
		return p.parseOperatorCall()
		
	case TokenPlus, TokenMultiply, TokenDivide, TokenModulo:
		// These should only appear as binary operators, not in primary position
		return nil, p.syntaxError(fmt.Sprintf("unexpected operator '%s'", token.Value), p.tokenPosition(token)).
			WithContext("operators must appear between operands")
		
	default:
		return nil, p.syntaxError(fmt.Sprintf("unexpected token '%s'", token.Value), p.tokenPosition(token))
	}
}

// parseOperatorCall parses an operator call like (( grab foo.bar ))
func (p *EnhancedParser) parseOperatorCall() (*Expr, error) {
	if p.isAtEnd() {
		return nil, p.syntaxError("expected operator", Position{Line: 1, Column: 1})
	}
	
	token := p.currentToken()
	var opName string
	
	// Map token types to operator names
	switch token.Type {
	case TokenOperator:
		opName = token.Value
	case TokenPlus:
		opName = "+"
	case TokenMinus:
		opName = "-"
	case TokenMultiply:
		opName = "*"
	case TokenDivide:
		opName = "/"
	case TokenModulo:
		opName = "%"
	default:
		return nil, p.syntaxError("expected operator name", p.tokenPosition(token))
	}
	
	// Extract operator name and modifiers
	opNameParts := strings.Split(opName, ":")
	baseOpName := opNameParts[0]
	
	opInfo := p.registry.Get(baseOpName)
	if opInfo == nil {
		return nil, p.syntaxError(fmt.Sprintf("unknown operator '%s'", baseOpName), p.tokenPosition(token))
	}
	
	p.advance() // consume operator
	
	// Parse arguments
	args := make([]*Expr, 0, 4) // Pre-allocate for typical arg counts
	
	// Determine if this is a function-style call
	// Function style is when there's no space between operator and opening paren
	// Since our tokenizer doesn't track whitespace, we'll use a heuristic:
	// If the operator was created from a TokenOperator (not arithmetic), and
	// the next token is an open paren, check the positions
	isFunctionStyle := false
	// Disabled function-style detection because the tokenizer doesn't preserve whitespace
	// if !p.isAtEnd() && p.currentToken().Type == TokenOpenParen && token.Type == TokenOperator {
	// 	openParenPos := p.currentToken().Pos
	// 	operatorEndPos := token.Pos + len(opName)
	// 	isFunctionStyle = (openParenPos == operatorEndPos)
	// }
	
	if isFunctionStyle {
		// Parse parenthesized arguments
		p.advance() // consume (
		
		if !p.isAtEnd() && p.currentToken().Type != TokenCloseParen {
			for {
				arg, err := p.parseExpression(PrecedenceLowest)
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
				
				if !p.consume(TokenComma) {
					break
				}
			}
		}
		
		if err := p.expectToken(TokenCloseParen, "after operator arguments"); err != nil {
			return nil, err
		}
	} else {
		// Parse space-separated arguments until we hit something that ends the operator
		argIndex := 0
		for !p.isAtEnd() {
			// Check if we've hit something that ends the operator call
			token := p.currentToken()
			if token.Type == TokenLogicalOr || token.Type == TokenCloseParen || token.Type == TokenComma {
				break
			}
			
			// Special handling for operators that expect references
			var arg *Expr
			var err error
			
			if argIndex == 0 && p.isReferenceExpectingOperator(baseOpName) && token.Type == TokenOperator {
				// First argument of grab, param, etc. should be treated as a reference
				// even if it matches an operator name
				p.advance()
				cursor, err := tree.ParseCursor(token.Value)
				if err != nil {
					return nil, p.syntaxError(fmt.Sprintf("invalid reference '%s': %v", token.Value, err), p.tokenPosition(token))
				}
				arg = &Expr{
					Type:      Reference,
					Reference: cursor,
					Pos:       p.tokenPosition(token),
				}
			} else {
				arg, err = p.parsePrimary()
				if err != nil {
					return nil, err
				}
			}
			
			args = append(args, arg)
			argIndex++
			
			// Continue parsing space-separated arguments
		}
	}
	
	// Validate argument count
	if opInfo.MinArgs >= 0 && len(args) < opInfo.MinArgs {
		return nil, p.syntaxError(fmt.Sprintf("operator '%s' requires at least %d arguments, got %d", baseOpName, opInfo.MinArgs, len(args)), 
			p.tokenPosition(p.currentToken())).WithContext("not enough arguments")
	}
	if opInfo.MaxArgs >= 0 && len(args) > opInfo.MaxArgs {
		return nil, p.syntaxError(fmt.Sprintf("operator '%s' accepts at most %d arguments, got %d", baseOpName, opInfo.MaxArgs, len(args)),
			p.tokenPosition(p.currentToken())).WithContext("too many arguments")
	}
	
	expr := NewOperatorCallWithPos(baseOpName, args, p.tokenPosition(token))
	
	// Parse operator modifiers (if any)
	modifiers := p.parseOperatorModifiers(opName)
	if len(modifiers) > 0 {
		for modifier, value := range modifiers {
			expr.SetModifier(modifier, value)
		}
	}
	
	return expr, nil
}

// parseLiteral parses a literal value (string, number, boolean, null)
func (p *EnhancedParser) parseLiteral(value string, pos Position) (*Expr, error) {
	// Check if it's a quoted string
	isQuoted := len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"'
	if isQuoted {
		// Remove quotes and return as string
		return &Expr{
			Type:    Literal,
			Literal: value[1 : len(value)-1],
			Pos:     pos,
		}, nil
	}
	
	// Try to parse as number (only for unquoted values)
	if strings.Contains(value, ".") {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return &Expr{
				Type:    Literal,
				Literal: f,
				Pos:     pos,
			}, nil
		}
	} else if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return &Expr{
			Type:    Literal,
			Literal: i,
			Pos:     pos,
		}, nil
	}
	
	// Check for special literals
	switch strings.ToLower(value) {
	case "null", "nil", "~":
		return &Expr{
			Type:    Literal,
			Literal: nil,
			Pos:     pos,
		}, nil
	case "true", "yes", "on":
		return &Expr{
			Type:    Literal,
			Literal: true,
			Pos:     pos,
		}, nil
	case "false", "no", "off":
		return &Expr{
			Type:    Literal,
			Literal: false,
			Pos:     pos,
		}, nil
	}
	
	// Default to string
	return &Expr{
		Type:    Literal,
		Literal: value,
		Pos:     pos,
	}, nil
}

// isBinaryOperatorToken checks if a token type is a binary operator
func (p *EnhancedParser) isBinaryOperatorToken(tokType TokenType) bool {
	switch tokType {
	case TokenLogicalOr, TokenLogicalAnd, TokenEquals, TokenNotEquals,
		TokenLessThan, TokenGreaterThan, TokenLessEqual, TokenGreaterEqual,
		TokenPlus, TokenMinus, TokenMultiply, TokenDivide, TokenModulo,
		TokenQuestion:
		return true
	}
	return false
}

// isReferenceExpectingOperator checks if an operator expects references as arguments
func (p *EnhancedParser) isReferenceExpectingOperator(opName string) bool {
	switch opName {
	case "grab", "param", "prune", "static_ips", "ips":
		return true
	}
	return false
}

// canBeBinaryOperator checks if an operator can be used as a binary operator in the current context
func (p *EnhancedParser) canBeBinaryOperator(opInfo *OperatorInfo, left *Expr) bool {
	// Check if this is a binary operator
	switch opInfo.Name {
	case "||", "&&", "+", "-", "*", "/", "%", "==", "!=", "<", ">", "<=", ">=":
		return true
	}
	return false
}

// Helper methods

func (p *EnhancedParser) currentToken() Token {
	if p.isAtEnd() {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.current]
}

// Position returns the position of a token (compatibility helper)
func (t Token) Position() int {
	return t.Pos
}

func (p *EnhancedParser) advance() {
	if !p.isAtEnd() {
		p.current++
	}
}

func (p *EnhancedParser) isAtEnd() bool {
	return p.current >= len(p.tokens)
}

func (p *EnhancedParser) consume(tokenType TokenType) bool {
	if p.isAtEnd() || p.currentToken().Type != tokenType {
		return false
	}
	p.advance()
	return true
}

func (p *EnhancedParser) peek() Token {
	if p.current+1 >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.current+1]
}

// Error handling helpers

// syntaxError creates a syntax error with position information
func (p *EnhancedParser) syntaxError(msg string, pos Position) *ExprError {
	err := NewSyntaxError(msg, pos)
	if p.source != "" {
		err.WithSource(p.source)
	}
	return err
}

// tokenPosition converts token position to Position
func (p *EnhancedParser) tokenPosition(token Token) Position {
	return Position{
		Offset: token.Pos,
		Line:   token.Line,
		Column: token.Col,
	}
}

// synchronize attempts to recover from a parse error by finding a safe synchronization point
func (p *EnhancedParser) synchronize() {
	p.advance()
	
	// Skip tokens until we find a safe stopping point
	for !p.isAtEnd() {
		token := p.currentToken()
		
		// Stop at these tokens which typically start new expressions
		switch token.Type {
		case TokenCloseParen, TokenComma, TokenLogicalOr:
			return
		}
		
		// Also stop if we see a new operator call
		if token.Type == TokenOperator && p.peek().Type == TokenOpenParen {
			return
		}
		
		p.advance()
	}
}

// expectToken checks for an expected token and generates an error if not found
func (p *EnhancedParser) expectToken(tokenType TokenType, context string) error {
	if p.isAtEnd() {
		return p.syntaxError(fmt.Sprintf("unexpected end of expression, expected %s %s", tokenTypeString(tokenType), context), 
			Position{Line: 1, Column: 1})
	}
	
	token := p.currentToken()
	if token.Type != tokenType {
		return p.syntaxError(fmt.Sprintf("expected %s %s, got '%s'", tokenTypeString(tokenType), context, token.Value),
			p.tokenPosition(token))
	}
	
	p.advance()
	return nil
}

// parseOperatorModifiers parses operator modifiers like :nocache
func (p *EnhancedParser) parseOperatorModifiers(opName string) map[string]bool {
	modifiers := make(map[string]bool)
	
	// Check if the operator name contains modifiers (format: "operator:modifier1:modifier2")
	parts := strings.Split(opName, ":")
	if len(parts) > 1 {
		// The first part is the actual operator name
		// The rest are modifiers
		for i := 1; i < len(parts); i++ {
			modifier := strings.TrimSpace(parts[i])
			if modifier != "" {
				modifiers[modifier] = true
			}
		}
	}
	
	return modifiers
}

// tokenTypeString returns a human-readable name for a token type
func tokenTypeString(tt TokenType) string {
	switch tt {
	case TokenOpenParen:
		return "("
	case TokenCloseParen:
		return ")"
	case TokenComma:
		return ","
	case TokenQuestion:
		return "?"
	case TokenColon:
		return ":"
	case TokenPlus:
		return "+"
	case TokenMinus:
		return "-"
	case TokenMultiply:
		return "*"
	case TokenDivide:
		return "/"
	case TokenModulo:
		return "%"
	case TokenEquals:
		return "=="
	case TokenNotEquals:
		return "!="
	case TokenLessThan:
		return "<"
	case TokenGreaterThan:
		return ">"
	case TokenLessEqual:
		return "<="
	case TokenGreaterEqual:
		return ">="
	case TokenLogicalAnd:
		return "&&"
	case TokenLogicalOr:
		return "||"
	// Note: There's no TokenNot defined, but "!" is mapped to negate operator
	case TokenOperator:
		return "operator"
	case TokenLiteral:
		return "literal"
	case TokenReference:
		return "reference"
	default:
		return fmt.Sprintf("token(%d)", tt)
	}
}