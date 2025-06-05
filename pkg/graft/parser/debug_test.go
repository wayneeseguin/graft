package parser

import (
	"testing"
	"fmt"
)

func TestDebugTokenization(t *testing.T) {
	// Register test operators
	OpRegistry["vault"] = &mockOperator{}
	defer delete(OpRegistry, "vault")
	
	// Test without target first
	fmt.Printf("=== Testing without target ===\n")
	fullInputNoTarget := "(( vault \"secret/path:key\" ))"
	fmt.Printf("Testing ParseOpcall with: %s\n", fullInputNoTarget)
	opcall1, err1 := ParseOpcall(EvalPhase, fullInputNoTarget)
	fmt.Printf("ParseOpcall result: err=%v, opcall=%v\n", err1, opcall1)
	
	fmt.Printf("\n=== Testing with target ===\n")
	input := "vault@production \"secret/path:key\""
	tokenizer := NewTokenizer(input)
	tokens := tokenizer.Tokenize()
	
	fmt.Printf("Input: %s\n", input)
	fmt.Printf("Tokens (%d):\n", len(tokens))
	for i, token := range tokens {
		fmt.Printf("  [%d] Type: %v, Value: %q, Pos: %d\n", i, token.Type, token.Value, token.Pos)
	}
	
	// Try parsing with the main parser
	parser := NewParser(tokens, NewOperatorRegistry())
	expr, err := parser.Parse()
	
	fmt.Printf("Parser result: err=%v, expr=%v\n", err, expr)
	
	// Also test ParseOpcall function with target
	fullInput := "(( vault@production \"secret/path:key\" ))"
	fmt.Printf("\nTesting ParseOpcall with: %s\n", fullInput)
	opcall, err2 := ParseOpcall(EvalPhase, fullInput)
	fmt.Printf("ParseOpcall result: err=%v, opcall=%v\n", err2, opcall)
}