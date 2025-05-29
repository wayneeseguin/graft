package operators

import (
	"fmt"
	"strings"

	"github.com/wayneeseguin/graft/pkg/graft"
	"github.com/wayneeseguin/graft/pkg/graft/parser"
)

// VaultExpressionParser handles parsing of vault expressions with sub-operators
type VaultExpressionParser struct {
	tokens []parser.Token
	pos    int
	src    string
}

// NewVaultExpressionParser creates a new vault expression parser
func NewVaultExpressionParser(input string) *VaultExpressionParser {
	return &VaultExpressionParser{
		tokens: parser.TokenizeExpression(input),
		pos:    0,
		src:    input,
	}
}

// ParseVaultExpression parses a vault expression with sub-operators
// Returns the parsed expression tree
func (p *VaultExpressionParser) ParseVaultExpression() (*graft.Expr, error) {
	if len(p.tokens) == 0 {
		return nil, fmt.Errorf("empty vault expression")
	}

	// Parse the main expression with precedence handling
	expr, err := p.parseExpression(parser.PrecedenceLowest)
	if err != nil {
		return nil, err
	}

	// Ensure we consumed all tokens
	if !p.isAtEnd() {
		return nil, fmt.Errorf("unexpected token after expression: %s", p.currentToken().Value)
	}

	return expr, nil
}

// parseExpression parses an expression with operator precedence
func (p *VaultExpressionParser) parseExpression(precedence parser.Precedence) (*graft.Expr, error) {
	// Parse the left side
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	// Handle binary operators with precedence
	for !p.isAtEnd() && p.shouldContinue(precedence) {
		token := p.currentToken()

		switch token.Type {
		case parser.TokenPipe:
			// Handle | choice operator
			p.advance() // consume |
			right, err := p.parseExpression(parser.GetTokenPrecedence(parser.TokenPipe) + 1)
			if err != nil {
				return nil, err
			}
			left = &graft.Expr{
				Type:  graft.VaultChoice,
				Left:  left,
				Right: right,
			}

		case parser.TokenLogicalOr:
			// Handle || logical or - this ends vault sub-operator parsing
			// Don't consume it, let the caller handle it
			return left, nil

		default:
			// Check if we should handle space concatenation
			if p.shouldHandleSpaceConcatenation(token) {
				// Handle implicit space concatenation
				right, err := p.parseUnary()
				if err != nil {
					return nil, err
				}
				// Create a space concatenation expression (we'll use a special type)
				left = &graft.Expr{
					Type:  graft.List, // Reuse List type for space concatenation
					Left:  left,
					Right: right,
				}
			} else {
				// Unknown binary operator in this context
				return left, nil
			}
		}
	}

	return left, nil
}

// parseUnary parses unary expressions and primary expressions
func (p *VaultExpressionParser) parseUnary() (*graft.Expr, error) {
	if p.isAtEnd() {
		return nil, fmt.Errorf("unexpected end of expression")
	}

	token := p.currentToken()

	switch token.Type {
	case parser.TokenOpenParen:
		return p.parseGroup()

	case parser.TokenLiteral:
		return p.parseLiteral()

	case parser.TokenReference:
		return p.parseReference()

	case parser.TokenEnvVar:
		return p.parseEnvVar()

	case parser.TokenOperator:
		return p.parseOperatorCall()

	default:
		return nil, fmt.Errorf("unexpected token in vault expression: %s", token.Value)
	}
}

// parseGroup parses a parenthesized group expression
func (p *VaultExpressionParser) parseGroup() (*graft.Expr, error) {
	if !p.expectToken(parser.TokenOpenParen) {
		return nil, fmt.Errorf("expected '('")
	}
	p.advance() // consume (

	// Parse the inner expression
	inner, err := p.parseExpression(parser.PrecedenceLowest)
	if err != nil {
		return nil, err
	}

	if !p.expectToken(parser.TokenCloseParen) {
		return nil, fmt.Errorf("expected ')'")
	}
	p.advance() // consume )

	// Wrap in a VaultGroup expression
	return &graft.Expr{
		Type: graft.VaultGroup,
		Left: inner,
	}, nil
}

// parseLiteral parses a literal value
func (p *VaultExpressionParser) parseLiteral() (*graft.Expr, error) {
	token := p.currentToken()
	p.advance()

	// Parse the literal value
	var value interface{}
	if strings.HasPrefix(token.Value, `"`) && strings.HasSuffix(token.Value, `"`) && len(token.Value) >= 2 {
		// Quoted string - remove quotes
		value = token.Value[1 : len(token.Value)-1]
	} else {
		// Use the raw value
		value = token.Value
	}

	return &graft.Expr{
		Type:    graft.Literal,
		Literal: value,
	}, nil
}

// parseReference parses a reference to another part of the document
func (p *VaultExpressionParser) parseReference() (*graft.Expr, error) {
	token := p.currentToken()
	p.advance()

	// For now, store the reference as a string
	// In a full implementation, this would parse the cursor path
	return &graft.Expr{
		Type:      graft.Reference,
		Reference: nil, // TODO: Parse actual cursor
		Name:      token.Value,
	}, nil
}

// parseEnvVar parses an environment variable reference
func (p *VaultExpressionParser) parseEnvVar() (*graft.Expr, error) {
	token := p.currentToken()
	p.advance()

	// Remove the $ prefix
	varName := token.Value
	if strings.HasPrefix(varName, "$") {
		varName = varName[1:]
	}

	return &graft.Expr{
		Type: graft.EnvVar,
		Name: varName,
	}, nil
}

// parseOperatorCall parses a nested operator call
func (p *VaultExpressionParser) parseOperatorCall() (*graft.Expr, error) {
	token := p.currentToken()
	p.advance()

	// For now, create a simplified operator call
	// In a full implementation, this would parse the full operator syntax
	return &graft.Expr{
		Type:     graft.OperatorCall,
		Operator: token.Value,
		Name:     token.Value,
	}, nil
}

// Helper methods

// currentToken returns the current token
func (p *VaultExpressionParser) currentToken() parser.Token {
	if p.pos >= len(p.tokens) {
		return parser.Token{Type: parser.TokenEOF}
	}
	return p.tokens[p.pos]
}

// advance moves to the next token
func (p *VaultExpressionParser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

// isAtEnd checks if we're at the end of tokens
func (p *VaultExpressionParser) isAtEnd() bool {
	return p.pos >= len(p.tokens) || p.currentToken().Type == parser.TokenEOF
}

// expectToken checks if the current token matches the expected type
func (p *VaultExpressionParser) expectToken(expected parser.TokenType) bool {
	return !p.isAtEnd() && p.currentToken().Type == expected
}

// shouldContinue determines if we should continue parsing at the current precedence level
func (p *VaultExpressionParser) shouldContinue(minPrecedence parser.Precedence) bool {
	if p.isAtEnd() {
		return false
	}

	token := p.currentToken()
	
	// Check for explicit operators first
	tokenPrecedence := parser.GetTokenPrecedence(token.Type)
	if tokenPrecedence > parser.PrecedenceLowest {
		return tokenPrecedence > minPrecedence
	}
	
	// Check for space concatenation (implicit operator)
	if p.shouldHandleSpaceConcatenation(token) {
		// Space concatenation has lower precedence than most operators
		spacePrecedence := parser.PrecedenceLowest + 1
		return spacePrecedence > minPrecedence
	}

	return false
}

// shouldHandleSpaceConcatenation determines if we should treat the current token as part of space concatenation
func (p *VaultExpressionParser) shouldHandleSpaceConcatenation(token parser.Token) bool {
	// Space concatenation applies to literals, references, env vars, and operator calls
	// but not to operators themselves or structural tokens
	switch token.Type {
	case parser.TokenLiteral, parser.TokenReference, parser.TokenEnvVar, parser.TokenOperator:
		return true
	case parser.TokenOpenParen:
		// Parentheses can start a new grouped expression for concatenation
		return true
	case parser.TokenCloseParen, parser.TokenPipe, parser.TokenLogicalOr:
		// These end the current expression level
		return false
	default:
		return false
	}
}

// ContainsSubOperators checks if an expression string contains vault sub-operators
func ContainsSubOperators(input string) bool {
	if input == "" {
		return false
	}

	// Quick check for sub-operator characters
	hasParens := strings.Contains(input, "(") || strings.Contains(input, ")")
	hasPipe := strings.Contains(input, "|") && !strings.Contains(input, "||")

	return hasParens || hasPipe
}

// ParseVaultArgs parses vault operator arguments, detecting sub-operators
func ParseVaultArgs(args []*graft.Expr) ([]*graft.Expr, bool, error) {
	hasSubOps := false
	parsedArgs := make([]*graft.Expr, len(args))

	for i, arg := range args {
		if arg == nil {
			parsedArgs[i] = arg
			continue
		}

		// Check if this argument needs sub-operator parsing
		if needsSubOperatorParsing(arg) {
			// Convert the argument to a string for parsing
			argStr := exprToString(arg)
			if ContainsSubOperators(argStr) {
				parser := NewVaultExpressionParser(argStr)
				parsed, err := parser.ParseVaultExpression()
				if err != nil {
					return nil, false, fmt.Errorf("failed to parse vault sub-operators in argument %d: %s", i, err)
				}
				parsedArgs[i] = parsed
				hasSubOps = true
			} else {
				parsedArgs[i] = arg
			}
		} else {
			parsedArgs[i] = arg
		}
	}

	return parsedArgs, hasSubOps, nil
}

// needsSubOperatorParsing checks if an expression needs sub-operator parsing
func needsSubOperatorParsing(expr *graft.Expr) bool {
	if expr == nil {
		return false
	}

	switch expr.Type {
	case graft.Literal:
		// Check if the literal string contains sub-operators
		if str, ok := expr.Literal.(string); ok {
			return ContainsSubOperators(str)
		}
		return false
	case graft.Reference:
		// References don't need sub-operator parsing themselves
		return false
	case graft.OperatorCall:
		// Nested operator calls are handled separately
		return false
	default:
		return false
	}
}

// exprToString converts an expression to a string for parsing
func exprToString(expr *graft.Expr) string {
	if expr == nil {
		return ""
	}

	switch expr.Type {
	case graft.Literal:
		if str, ok := expr.Literal.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", expr.Literal)
	case graft.Reference:
		if expr.Name != "" {
			return expr.Name
		}
		return ""
	case graft.OperatorCall:
		return expr.Name
	default:
		return ""
	}
}