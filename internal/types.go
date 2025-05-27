package internal

import (
	"github.com/wayneeseguin/graft/pkg/graft"
	"github.com/wayneeseguin/graft/pkg/graft/parser"
)

// Type aliases for convenience
type Evaluator = graft.Evaluator
type Opcall = graft.Opcall
type Operator = graft.Operator
type Expr = graft.Expr
type Response = graft.Response
type OperatorPhase = graft.OperatorPhase
type Token = parser.Token
type TokenType = parser.TokenType

// Function aliases
var ParseExpression = parser.ParseExpression
var NewOperatorRegistry = parser.NewOperatorRegistry
