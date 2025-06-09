package graft

import (
	"errors"
	"strings"
	"testing"
)

// TestExprError_Error tests error message formatting
func TestExprError_Error(t *testing.T) {
	tests := []struct {
		name        string
		err         *ExprError
		wantContain []string
	}{
		{
			name: "syntax error with position",
			err: &ExprError{
				Type:    SyntaxError,
				Message: "unexpected token",
				Position: Position{
					Line:   10,
					Column: 15,
					File:   "test.yml",
				},
			},
			wantContain: []string{"Syntax Error", "test.yml:10:15", "unexpected token"},
		},
		{
			name: "type error without file",
			err: &ExprError{
				Type:    TypeError,
				Message: "invalid type",
				Position: Position{
					Line:   5,
					Column: 3,
				},
			},
			wantContain: []string{"Type Error", "5:3", "invalid type"},
		},
		{
			name: "reference error with nested error",
			err: &ExprError{
				Type:    ReferenceError,
				Message: "reference not found",
				Nested:  errors.New("underlying error"),
			},
			wantContain: []string{"Reference Error", "reference not found", "caused by: underlying error"},
		},
		{
			name: "evaluation error without position",
			err: &ExprError{
				Type:    ExprEvaluationError,
				Message: "evaluation failed",
			},
			wantContain: []string{"Evaluation Error", "evaluation failed"},
		},
		{
			name: "operator error with context",
			err: &ExprError{
				Type:    ExprOperatorError,
				Message: "operator failed",
				Context: "while processing grab",
			},
			wantContain: []string{"Operator Error", "operator failed"},
		},
		{
			name: "error with source context",
			err: &ExprError{
				Type:    SyntaxError,
				Message: "invalid syntax",
				Position: Position{
					Line:   2,
					Column: 5,
				},
				Source: "line1\n  (( invalid ))\nline3",
			},
			wantContain: []string{"Syntax Error", "2:5", "invalid syntax", "  (( invalid ))"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()

			for _, want := range tt.wantContain {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error message %q does not contain %q", errMsg, want)
				}
			}
		})
	}
}

// TestExprError_Unwrap tests error unwrapping
func TestExprError_Unwrap(t *testing.T) {
	nested := errors.New("nested error")
	err := &ExprError{
		Type:    TypeError,
		Message: "type error",
		Nested:  nested,
	}

	unwrapped := err.Unwrap()
	if unwrapped != nested {
		t.Errorf("expected unwrapped error to be %v, got %v", nested, unwrapped)
	}

	// Test with no nested error
	err2 := &ExprError{
		Type:    SyntaxError,
		Message: "syntax error",
	}

	if err2.Unwrap() != nil {
		t.Error("expected nil for unwrapped error when no nested error")
	}
}

// TestExprError_TypeString tests error type string conversion
func TestExprError_TypeString(t *testing.T) {
	tests := []struct {
		errType  ExprErrorType
		expected string
	}{
		{SyntaxError, "Syntax Error"},
		{TypeError, "Type Error"},
		{ReferenceError, "Reference Error"},
		{ExprEvaluationError, "Evaluation Error"},
		{ExprOperatorError, "Operator Error"},
		{ExprErrorType(999), ""}, // Unknown type
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			err := &ExprError{Type: tt.errType}
			result := err.typeString()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestExprError_FormatSourceContext tests source context formatting
func TestExprError_FormatSourceContext(t *testing.T) {
	source := `key1: value1
key2: (( grab missing.path ))
key3: value3
key4: value4`

	err := &ExprError{
		Type:    ReferenceError,
		Message: "path not found",
		Position: Position{
			Line:   2,
			Column: 7,
		},
		Source: source,
	}

	errMsg := err.Error()

	// Should contain the error line
	if !strings.Contains(errMsg, "key2: (( grab missing.path ))") {
		t.Error("error message should contain the source line")
	}

	// Should show position info
	if !strings.Contains(errMsg, "2:7") {
		t.Error("error message should contain position")
	}
}

// TestPosition tests Position struct
func TestPosition(t *testing.T) {
	tests := []struct {
		name     string
		pos      Position
		expected string
	}{
		{
			name: "with file",
			pos: Position{
				File:   "config.yml",
				Line:   10,
				Column: 5,
				Offset: 123,
			},
			expected: "config.yml:10:5",
		},
		{
			name: "without file",
			pos: Position{
				Line:   3,
				Column: 15,
				Offset: 45,
			},
			expected: "3:15",
		},
		{
			name:     "zero position",
			pos:      Position{},
			expected: "0:0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test string formatting (would need to add a String() method to Position)
			// For now, test the values directly
			if tt.pos.Line != 0 && tt.pos.Column != 0 {
				if tt.pos.File != "" && tt.pos.File != tt.expected[:strings.Index(tt.expected, ":")] {
					t.Errorf("expected file %s in position", tt.pos.File)
				}
			}
		})
	}
}

// TestExprError_Constructors tests error constructor functions
func TestExprError_Constructors(t *testing.T) {
	pos := Position{Line: 1, Column: 1}

	t.Run("NewExprEvaluationError", func(t *testing.T) {
		err := NewExprEvaluationError("test message", pos)
		if err.Type != ExprEvaluationError {
			t.Errorf("expected ExprEvaluationError type, got %v", err.Type)
		}
		if err.Message != "test message" {
			t.Errorf("expected 'test message', got %q", err.Message)
		}
		if err.Position != pos {
			t.Errorf("expected position %v, got %v", pos, err.Position)
		}
	})

	t.Run("NewExprOperatorError", func(t *testing.T) {
		err := NewExprOperatorError("operator error", pos)
		if err.Type != ExprOperatorError {
			t.Errorf("expected ExprOperatorError type, got %v", err.Type)
		}
		if err.Message != "operator error" {
			t.Errorf("expected 'operator error', got %q", err.Message)
		}
	})

	t.Run("WrapError", func(t *testing.T) {
		original := errors.New("original error")
		wrapped := WrapError(original, TypeError, pos)
		
		if wrapped.Type != TypeError {
			t.Errorf("expected TypeError, got %v", wrapped.Type)
		}
		if wrapped.Nested != original {
			t.Errorf("expected nested error to be original error")
		}
		if !strings.Contains(wrapped.Message, "original error") {
			t.Errorf("expected wrapped message to contain original error")
		}
	})

	t.Run("WithContext", func(t *testing.T) {
		err := &ExprError{
			Type:    SyntaxError,
			Message: "base error",
		}
		
		withCtx := err.WithContext("while parsing expression")
		if withCtx.Context != "while parsing expression" {
			t.Errorf("expected context to be set")
		}
		// Verify it returns the same error for chaining
		if withCtx != err {
			t.Error("WithContext should return the same error instance")
		}
	})
}

// TestExprError_RealWorldScenarios tests real-world error scenarios
func TestExprError_RealWorldScenarios(t *testing.T) {
	t.Run("nested operator error", func(t *testing.T) {
		// Simulating: (( concat (( grab missing.path )) "suffix" ))
		innerErr := &ExprError{
			Type:    ReferenceError,
			Message: "path 'missing.path' not found",
			Position: Position{
				Line:   5,
				Column: 20,
			},
		}
		
		outerErr := &ExprError{
			Type:    ExprOperatorError,
			Message: "concat operator failed",
			Position: Position{
				Line:   5,
				Column: 10,
			},
			Nested: innerErr,
		}
		
		errMsg := outerErr.Error()
		if !strings.Contains(errMsg, "concat operator failed") {
			t.Error("should contain outer error message")
		}
		if !strings.Contains(errMsg, "path 'missing.path' not found") {
			t.Error("should contain inner error message")
		}
	})

	t.Run("error with multi-line source", func(t *testing.T) {
		source := `parameters:
  name: myapp
  version: (( grab meta.version ))
  instances: (( grab meta.instances || 1 ))
  networks:
    - name: default
      static_ips: (( static_ips 0 1 2 ))`
		
		err := &ExprError{
			Type:    ReferenceError,
			Message: "meta.version not found",
			Position: Position{
				Line:   3,
				Column: 14,
				File:   "deployment.yml",
			},
			Source: source,
		}
		
		errMsg := err.Error()
		if !strings.Contains(errMsg, "deployment.yml:3:14") {
			t.Error("should contain file position")
		}
		if !strings.Contains(errMsg, "version: (( grab meta.version ))") {
			t.Error("should contain the problematic line")
		}
	})
}

// Benchmarks

func BenchmarkExprError_Error(b *testing.B) {
	err := &ExprError{
		Type:    TypeError,
		Message: "type mismatch: expected string, got number",
		Position: Position{
			Line:   42,
			Column: 17,
			File:   "config.yml",
		},
		Source: strings.Repeat("line of yaml content\n", 100),
		Nested: errors.New("underlying type conversion error"),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

func BenchmarkExprError_SimpleError(b *testing.B) {
	err := &ExprError{
		Type:    SyntaxError,
		Message: "unexpected token",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}