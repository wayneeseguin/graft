package parser

import (
	"testing"
	"github.com/starkandwayne/goutils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// mockOperator is a simple mock for testing
type mockOperator struct{}

func (m *mockOperator) Setup() error { return nil }
func (m *mockOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	return &Response{Type: 0, Value: "mock"}, nil // Replace action
}
func (m *mockOperator) Dependencies(ev *Evaluator, args []*Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return nil
}
func (m *mockOperator) Phase() OperatorPhase { return EvalPhase }

func TestTokenizeTargetSyntax(t *testing.T) {
	// Register test operators in global registry for tokenization
	graft.OpRegistry["vault"] = &mockOperator{}
	graft.OpRegistry["nats"] = &mockOperator{}
	graft.OpRegistry["awsparam"] = &mockOperator{}
	graft.OpRegistry["awssecret"] = &mockOperator{}
	defer func() {
		// Clean up
		delete(graft.OpRegistry, "vault")
		delete(graft.OpRegistry, "nats") 
		delete(graft.OpRegistry, "awsparam")
		delete(graft.OpRegistry, "awssecret")
	}()
	
	// Test tokenization of @ symbol
	tests := []struct {
		input    string
		expected []TokenType
	}{
		{
			input:    "vault@production",
			expected: []TokenType{TokenOperator}, // Should be tokenized as single operator token
		},
		{
			input:    "vault @ production",
			expected: []TokenType{TokenOperator, TokenAt, TokenReference},
		},
		{
			input:    "vault@staging:nocache",
			expected: []TokenType{TokenOperator}, // Should handle both @ and : in operator names
		},
	}

	for _, test := range tests {
		tokenizer := NewTokenizer(test.input)
		tokens := tokenizer.Tokenize()
		
		if len(tokens) != len(test.expected) {
			t.Errorf("Input '%s': expected %d tokens, got %d", test.input, len(test.expected), len(tokens))
			continue
		}
		
		for i, expectedType := range test.expected {
			if tokens[i].Type != expectedType {
				t.Errorf("Input '%s': token %d expected type %v, got %v (value: %s)", 
					test.input, i, expectedType, tokens[i].Type, tokens[i].Value)
			}
		}
	}
}

func TestParseOperatorWithTarget(t *testing.T) {
	// Register test operators in global registry for parsing
	graft.OpRegistry["vault"] = &mockOperator{}
	graft.OpRegistry["nats"] = &mockOperator{}
	graft.OpRegistry["awsparam"] = &mockOperator{}
	graft.OpRegistry["awssecret"] = &mockOperator{}
	defer func() {
		// Clean up
		delete(graft.OpRegistry, "vault")
		delete(graft.OpRegistry, "nats") 
		delete(graft.OpRegistry, "awsparam")
		delete(graft.OpRegistry, "awssecret")
	}()
	
	// Test parsing operator@target expressions  
	tests := []struct {
		input        string
		shouldSucceed bool
	}{
		{
			input:        "(( vault@production \"secret/path:key\" ))",
			shouldSucceed: true,
		},
		{
			input:        "(( nats@staging \"kv.bucket.key\" ))",
			shouldSucceed: true,
		},
		{
			input:        "(( vault \"secret/path:key\" ))",
			shouldSucceed: true,
		},
		{
			input:        "(( awsparam@prod \"parameter-name\" ))",
			shouldSucceed: true,
		},
	}

	for _, test := range tests {
		// Test ParseOpcall function directly
		opcall, err := ParseOpcall(EvalPhase, test.input)
		
		if test.shouldSucceed {
			if err != nil {
				t.Errorf("Input '%s': expected success but got error: %v", test.input, err)
				continue
			}
			
			if opcall == nil {
				t.Errorf("Input '%s': expected opcall but got nil", test.input)
				continue
			}
			
			// Just verify that we can parse the expression successfully
			// The actual target extraction can be tested separately
			t.Logf("Successfully parsed '%s'", test.input)
		} else {
			if err == nil && opcall != nil {
				t.Errorf("Input '%s': expected failure but parsing succeeded", test.input)
			}
		}
	}
}

func TestParseOpcallWithTarget(t *testing.T) {
	// Test the ParseOpcall function directly
	tests := []struct {
		input          string
		expectedOp     string
		shouldSucceed  bool
	}{
		{
			input:         "(( vault@production \"secret/path:key\" ))",
			expectedOp:    "vault",
			shouldSucceed: true,
		},
		{
			input:         "(( nats@staging \"kv.bucket\" ))",
			expectedOp:    "nats",
			shouldSucceed: true,
		},
	}

	for _, test := range tests {
		opcall, err := ParseOpcall(EvalPhase, test.input)
		
		if test.shouldSucceed {
			if err != nil {
				t.Errorf("Input '%s': expected success but got error: %v", test.input, err)
				continue
			}
			
			if opcall == nil {
				t.Errorf("Input '%s': expected opcall but got nil", test.input)
				continue
			}
		} else {
			if err == nil && opcall != nil {
				t.Errorf("Input '%s': expected failure but parsing succeeded", test.input)
			}
		}
	}
}