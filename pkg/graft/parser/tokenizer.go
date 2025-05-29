package parser

import (
	"strings"
)

// TokenType represents the type of a parsed token
type TokenType int

const (
	TokenOperator TokenType = iota
	TokenLiteral
	TokenReference
	TokenLogicalOr
	TokenLogicalAnd // Future
	TokenComma
	TokenEnvVar
	TokenOpenParen
	TokenCloseParen
	TokenPlus       // Future
	TokenMinus      // Future
	TokenMultiply   // Future
	TokenDivide     // Future
	TokenModulo     // Future
	TokenEquals     // Future
	TokenNotEquals  // Future
	TokenLessThan   // Future
	TokenGreaterThan // Future
	TokenLessEqual   // Future: <=
	TokenGreaterEqual // Future: >=
	TokenQuestion    // Future: ? (for ternary)
	TokenColon       // Future: : (for ternary)
	TokenPipe        // | for vault choice sub-operator
	TokenEOF
	TokenUnknown
)

// Token represents a parsed token with its type and position
type Token struct {
	Value string
	Type  TokenType
	Pos   int
	Line  int
	Col   int
}

// String returns string representation of token for debugging
func (t Token) String() string {
	switch t.Type {
	case TokenEOF:
		return "EOF"
	case TokenLogicalOr:
		return "||"
	case TokenLogicalAnd:
		return "&&"
	case TokenOpenParen:
		return "("
	case TokenCloseParen:
		return ")"
	case TokenComma:
		return ","
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
	case TokenQuestion:
		return "?"
	case TokenColon:
		return ":"
	case TokenPipe:
		return "|"
	default:
		return t.Value
	}
}

// Tokenizer provides context-aware tokenization
type Tokenizer struct {
	input    string
	pos      int
	line     int
	col      int
	tokens   []Token
	inQuotes bool
	escaped  bool
}

// NewTokenizer creates a new tokenizer
func NewTokenizer(input string) *Tokenizer {
	return &Tokenizer{
		input:  input,
		pos:    0,
		line:   1,
		col:    1,
		tokens: make([]Token, 0, 8), // Pre-allocate for typical expressions
	}
}

// Tokenize performs the tokenization
func (t *Tokenizer) Tokenize() []Token {
	t.tokens = make([]Token, 0, cap(t.tokens)) // Reuse existing capacity
	var current strings.Builder
	
	for t.pos < len(t.input) {
		ch := t.input[t.pos]
		
		if t.escaped {
			t.handleEscaped(&current, ch)
			t.advance()
			continue
		}
		
		if ch == '\\' {
			t.escaped = true
			t.advance()
			continue
		}
		
		if ch == '"' {
			if t.inQuotes {
				// End of quoted string
				current.WriteByte(ch)
				t.addToken(current.String(), TokenLiteral)
				current.Reset()
				t.inQuotes = false
			} else {
				// Start of quoted string
				t.flushCurrent(&current)
				current.WriteByte(ch)
				t.inQuotes = true
			}
			t.advance()
			continue
		}
		
		if t.inQuotes {
			current.WriteByte(ch)
			t.advance()
			continue
		}
		
		// Not in quotes - check for special characters and operators
		switch ch {
		case ' ', '\t', '\n', '\r':
			t.flushCurrent(&current)
			if ch == '\n' {
				t.line++
				t.col = 0
			}
			t.advance()
			
		case ',':
			t.flushCurrent(&current)
			t.addToken(",", TokenComma)
			t.advance()
			
		case '(':
			t.flushCurrent(&current)
			t.addToken("(", TokenOpenParen)
			t.advance()
			
		case ')':
			t.flushCurrent(&current)
			t.addToken(")", TokenCloseParen)
			t.advance()
			
		case '|':
			// Check for ||
			if t.peek() == '|' {
				t.flushCurrent(&current)
				startPos := t.pos
				startCol := t.col
				t.advance() // consume first |
				t.advance() // consume second |
				t.addTokenAt("||", TokenLogicalOr, startPos, startCol)
			} else {
				// Single pipe | for vault choice sub-operator
				t.flushCurrent(&current)
				t.addToken("|", TokenPipe)
				t.advance()
			}
			
		case '&':
			// Check for && (future)
			if t.peek() == '&' {
				t.flushCurrent(&current)
				t.addToken("&&", TokenLogicalAnd)
				t.advance()
				t.advance()
			} else {
				current.WriteByte(ch)
				t.advance()
			}
			
		case '=':
			// Check for ==
			if t.peek() == '=' {
				t.flushCurrent(&current)
				t.addToken("==", TokenEquals)
				t.advance()
				t.advance()
			} else {
				current.WriteByte(ch)
				t.advance()
			}
			
		case '!':
			// Check for !=
			if t.peek() == '=' {
				t.flushCurrent(&current)
				t.addToken("!=", TokenNotEquals)
				t.advance()
				t.advance()
			} else {
				// Standalone ! is a negation operator
				t.flushCurrent(&current)
				t.addToken("!", TokenOperator)
				t.advance()
			}
			
		case '<':
			// Check for <=
			if t.peek() == '=' {
				t.flushCurrent(&current)
				t.addToken("<=", TokenLessEqual)
				t.advance()
				t.advance()
			} else {
				t.flushCurrent(&current)
				t.addToken("<", TokenLessThan)
				t.advance()
			}
			
		case '>':
			// Check for >=
			if t.peek() == '=' {
				t.flushCurrent(&current)
				t.addToken(">=", TokenGreaterEqual)
				t.advance()
				t.advance()
			} else {
				t.flushCurrent(&current)
				t.addToken(">", TokenGreaterThan)
				t.advance()
			}
			
		case '+':
			t.flushCurrent(&current)
			t.addToken("+", TokenPlus)
			t.advance()
			
		case '-':
			// Check if this is part of an operator name or a standalone minus
			if current.Len() > 0 && t.isOperatorChar(t.peek()) {
				// Part of an operator name like "vault-try"
				current.WriteByte(ch)
				t.advance()
			} else {
				t.flushCurrent(&current)
				t.addToken("-", TokenMinus)
				t.advance()
			}
			
		case '*':
			t.flushCurrent(&current)
			t.addToken("*", TokenMultiply)
			t.advance()
			
		case '/':
			t.flushCurrent(&current)
			t.addToken("/", TokenDivide)
			t.advance()
			
		case '%':
			t.flushCurrent(&current)
			t.addToken("%", TokenModulo)
			t.advance()
			
		case '?':
			t.flushCurrent(&current)
			t.addToken("?", TokenQuestion)
			t.advance()
			
		case ':':
			// Special handling for colons within operator names (e.g., vault:nocache)
			if current.Len() > 0 && t.isOperatorChar(t.peek()) {
				// Part of an operator name with modifiers
				current.WriteByte(ch)
				t.advance()
			} else {
				t.flushCurrent(&current)
				t.addToken(":", TokenColon)
				t.advance()
			}
			
		default:
			current.WriteByte(ch)
			t.advance()
		}
	}
	
	t.flushCurrent(&current)
	return t.tokens
}

// advance moves to the next character
func (t *Tokenizer) advance() {
	t.pos++
	t.col++
}

// peek looks at the next character without consuming it
func (t *Tokenizer) peek() byte {
	if t.pos+1 < len(t.input) {
		return t.input[t.pos+1]
	}
	return 0
}

// isOperatorChar checks if a character could be part of an operator name
func (t *Tokenizer) isOperatorChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || 
		   (ch >= '0' && ch <= '9') || ch == '-' || ch == '_'
}

// handleEscaped handles escaped characters
func (t *Tokenizer) handleEscaped(current *strings.Builder, ch byte) {
	switch ch {
	case 'n':
		current.WriteByte('\n')
	case 'r':
		current.WriteByte('\r')
	case 't':
		current.WriteByte('\t')
	default:
		current.WriteByte(ch)
	}
	t.escaped = false
}

// flushCurrent adds the current token if it's not empty
func (t *Tokenizer) flushCurrent(current *strings.Builder) {
	if current.Len() > 0 {
		value := current.String()
		tokType := t.classifyToken(value)
		// Calculate position of the start of this token
		pos := t.pos - current.Len()
		col := t.col - current.Len()
		t.addTokenAt(value, tokType, pos, col)
		current.Reset()
	}
}

// classifyToken determines the type of a token
func (t *Tokenizer) classifyToken(value string) TokenType {
	// Check if it's a quoted string
	if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return TokenLiteral
	}
	
	// Check if it's an environment variable
	// Environment variables start with $ followed by a letter or underscore
	// This excludes $.something which is a reference to root
	if strings.HasPrefix(value, "$") && len(value) > 1 && value[1] != '.' {
		return TokenEnvVar
	}
	
	// Check for special literals first (before operator check)
	// For booleans, we need exact case matching
	switch value {
	case "true", "false":
		return TokenLiteral
	case "TRUE", "FALSE":
		return TokenLiteral
	case "True", "False":
		return TokenLiteral
	}
	
	// For nil values, check specific cases
	switch value {
	case "nil", "null", "~":
		return TokenLiteral
	case "Nil", "Null":
		return TokenLiteral
	case "NIL", "NULL":
		return TokenLiteral
	}
	
	// Check if it's a registered operator
	if IsRegisteredOperator(value) {
		return TokenOperator
	}
	
	// Check if it's an operator with modifiers (e.g., "vault:nocache")
	if strings.Contains(value, ":") {
		parts := strings.Split(value, ":")
		if len(parts) > 1 && IsRegisteredOperator(parts[0]) {
			return TokenOperator
		}
	}
	
	// Check if it's a number
	if isNumericString(value) {
		return TokenLiteral
	}
	
	// Default to reference
	return TokenReference
}

// isNumericString checks if a string represents a number
func isNumericString(s string) bool {
	if s == "" {
		return false
	}
	
	// Handle negative numbers
	if s[0] == '-' || s[0] == '+' {
		s = s[1:]
	}
	
	hasDecimal := false
	for i, ch := range s {
		if ch == '.' {
			if hasDecimal || i == len(s)-1 {
				return false
			}
			hasDecimal = true
		} else if ch < '0' || ch > '9' {
			return false
		}
	}
	
	// Must have at least one digit
	return len(s) > 0 && s != "."
}

// addToken adds a token to the list at the current position
func (t *Tokenizer) addToken(value string, tokType TokenType) {
	// For operators added directly, calculate position based on value length
	pos := t.pos - len(value)
	col := t.col - len(value)
	
	t.addTokenAt(value, tokType, pos, col)
}

// addTokenAt adds a token at a specific position
func (t *Tokenizer) addTokenAt(value string, tokType TokenType, pos, col int) {
	// Intern operator names and common literals
	if tokType == TokenOperator || tokType == TokenLiteral {
		if tokType == TokenOperator || value == "true" || value == "false" || value == "null" || value == "nil" {
			value = InternString(value)
		}
	}
	
	t.tokens = append(t.tokens, Token{
		Value: value,
		Type:  tokType,
		Pos:   pos,
		Line:  t.line,
		Col:   col,
	})
}

// TokenizeExpression is a convenience function for tokenizing expressions
func TokenizeExpression(input string) []Token {
	tokenizer := NewTokenizer(input)
	return tokenizer.Tokenize()
}

// Binary operator precedence mapping
var binaryOperatorPrecedence = map[TokenType]Precedence{
	TokenLogicalOr:    PrecedenceOr,
	TokenLogicalAnd:   PrecedenceAnd,
	TokenPipe:         PrecedenceOr + 1, // Higher than || but lower than most operators
	TokenEquals:       PrecedenceEquality,
	TokenNotEquals:    PrecedenceEquality,
	TokenLessThan:     PrecedenceComparison,
	TokenGreaterThan:  PrecedenceComparison,
	TokenLessEqual:    PrecedenceComparison,
	TokenGreaterEqual: PrecedenceComparison,
	TokenPlus:         PrecedenceAdditive,
	TokenMinus:        PrecedenceAdditive,
	TokenMultiply:     PrecedenceMultiplicative,
	TokenDivide:       PrecedenceMultiplicative,
	TokenModulo:       PrecedenceMultiplicative,
}

// Operator associativity mapping
var operatorAssociativity = map[TokenType]Associativity{
	TokenLogicalOr:    RightAssociative,
	TokenLogicalAnd:   LeftAssociative,
	TokenPipe:         LeftAssociative, // | associates left to right
	TokenEquals:       LeftAssociative,
	TokenNotEquals:    LeftAssociative,
	TokenLessThan:     LeftAssociative,
	TokenGreaterThan:  LeftAssociative,
	TokenLessEqual:    LeftAssociative,
	TokenGreaterEqual: LeftAssociative,
	TokenPlus:         LeftAssociative,
	TokenMinus:        LeftAssociative,
	TokenMultiply:     LeftAssociative,
	TokenDivide:       LeftAssociative,
	TokenModulo:       LeftAssociative,
}

// GetTokenPrecedence returns the precedence of a token type
func GetTokenPrecedence(tokType TokenType) Precedence {
	if prec, ok := binaryOperatorPrecedence[tokType]; ok {
		return prec
	}
	return PrecedenceLowest
}

// GetTokenAssociativity returns the associativity of a token type
func GetTokenAssociativity(tokType TokenType) Associativity {
	if assoc, ok := operatorAssociativity[tokType]; ok {
		return assoc
	}
	return LeftAssociative
}