package graft

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/wayneeseguin/graft/internal/utils/tree"
)

// TestExprOperator implements Operator for expression testing
type TestExprOperator struct {
	name    string
	runFunc func(ev *Evaluator, args []*Expr) (*Response, error)
}

func (m *TestExprOperator) Setup() error {
	return nil
}

func (m *TestExprOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	if m.runFunc != nil {
		return m.runFunc(ev, args)
	}
	return &Response{
		Type:  Replace,
		Value: fmt.Sprintf("test_result_%s", m.name),
	}, nil
}

func (m *TestExprOperator) Dependencies(ev *Evaluator, args []*Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

func (m *TestExprOperator) Phase() OperatorPhase {
	return EvalPhase
}

// TestEvaluateExpr_Literal tests literal expression evaluation
func TestEvaluateExpr_Literal(t *testing.T) {
	tests := []struct {
		name     string
		literal  interface{}
		expected interface{}
	}{
		{
			name:     "string literal",
			literal:  "hello world",
			expected: "hello world",
		},
		{
			name:     "integer literal",
			literal:  42,
			expected: 42,
		},
		{
			name:     "float literal",
			literal:  3.14,
			expected: 3.14,
		},
		{
			name:     "boolean literal",
			literal:  true,
			expected: true,
		},
		{
			name:     "null literal",
			literal:  nil,
			expected: nil,
		},
		{
			name:     "list literal",
			literal:  []interface{}{1, 2, 3},
			expected: []interface{}{1, 2, 3},
		},
		{
			name: "map literal",
			literal: map[interface{}]interface{}{
				"key": "value",
			},
			expected: map[interface{}]interface{}{
				"key": "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := &Expr{
				Type:    Literal,
				Literal: tt.literal,
			}

			ev := &Evaluator{
				Tree: make(map[interface{}]interface{}),
			}

			resp, err := EvaluateExpr(expr, ev)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.Type != Replace {
				t.Errorf("expected Replace response type, got %v", resp.Type)
			}

			if !deepEqual(resp.Value, tt.expected) {
				t.Errorf("expected value %v, got %v", tt.expected, resp.Value)
			}
		})
	}
}

// TestEvaluateExpr_Reference tests reference expression evaluation
func TestEvaluateExpr_Reference(t *testing.T) {
	testData := map[interface{}]interface{}{
		"simple": "value",
		"nested": map[interface{}]interface{}{
			"key": "nested value",
			"deep": map[interface{}]interface{}{
				"key": "deep value",
			},
		},
		"list": []interface{}{"first", "second", "third"},
	}

	tests := []struct {
		name      string
		path      string
		expected  interface{}
		wantError bool
	}{
		{
			name:     "simple reference",
			path:     "simple",
			expected: "value",
		},
		{
			name:     "nested reference",
			path:     "nested.key",
			expected: "nested value",
		},
		{
			name:     "deep nested reference",
			path:     "nested.deep.key",
			expected: "deep value",
		},
		{
			name:     "list reference",
			path:     "list.1",
			expected: "second",
		},
		{
			name:      "missing reference",
			path:      "missing.path",
			wantError: true,
		},
		{
			name:      "invalid list index",
			path:      "list.10",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor, err := tree.ParseCursor(tt.path)
			if err != nil {
				t.Fatalf("failed to parse cursor: %v", err)
			}

			expr := &Expr{
				Type:      Reference,
				Reference: cursor,
			}

			ev := &Evaluator{
				Tree: testData,
			}

			resp, err := EvaluateExpr(expr, ev)

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.Type != Replace {
				t.Errorf("expected Replace response type, got %v", resp.Type)
			}

			if !deepEqual(resp.Value, tt.expected) {
				t.Errorf("expected value %v, got %v", tt.expected, resp.Value)
			}
		})
	}
}

// TestEvaluateExpr_EnvVar tests environment variable expression evaluation
func TestEvaluateExpr_EnvVar(t *testing.T) {
	// Save and restore environment
	oldVars := make(map[string]string)
	testVars := map[string]string{
		"TEST_STRING":  "test value",
		"TEST_NUMBER":  "42",
		"TEST_BOOL":    "true",
		"TEST_JSON":    `{"key": "value"}`,
		"TEST_YAML":    "key: value",
		"TEST_LIST":    "[1, 2, 3]",
		"TEST_MULTILINE": "line1\nline2\nline3",
	}

	for k, v := range testVars {
		oldVars[k] = os.Getenv(k)
		os.Setenv(k, v)
	}

	defer func() {
		for k, v := range oldVars {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	tests := []struct {
		name     string
		envVar   string
		expected interface{}
	}{
		{
			name:     "string env var",
			envVar:   "TEST_STRING",
			expected: "test value",
		},
		{
			name:     "numeric string stays string",
			envVar:   "TEST_NUMBER",
			expected: "42",
		},
		{
			name:     "boolean env var",
			envVar:   "TEST_BOOL",
			expected: true,
		},
		{
			name:   "JSON env var",
			envVar: "TEST_JSON",
			expected: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name:     "YAML env var",
			envVar:   "TEST_YAML",
			expected: "key: value", // YAML parsing only happens for specific patterns
		},
		{
			name:     "list env var",
			envVar:   "TEST_LIST",
			expected: []interface{}{1, 2, 3},
		},
		{
			name:     "multiline string",
			envVar:   "TEST_MULTILINE",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "missing env var",
			envVar:   "MISSING_VAR",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := &Expr{
				Type: EnvVar,
				Name: tt.envVar,
			}

			ev := &Evaluator{
				Tree: make(map[interface{}]interface{}),
			}

			resp, err := EvaluateExpr(expr, ev)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.Type != Replace {
				t.Errorf("expected Replace response type, got %v", resp.Type)
			}

			if !deepEqual(resp.Value, tt.expected) {
				t.Errorf("expected value %#v, got %#v", tt.expected, resp.Value)
			}
		})
	}
}

// TestEvaluateExpr_LogicalOr tests logical OR (fallback) expression evaluation
func TestEvaluateExpr_LogicalOr(t *testing.T) {
	testData := map[interface{}]interface{}{
		"existing": "found value",
		"nested": map[interface{}]interface{}{
			"key": "nested value",
		},
	}

	tests := []struct {
		name        string
		leftPath    string
		rightValue  interface{}
		expected    interface{}
		description string
	}{
		{
			name:        "left succeeds",
			leftPath:    "existing",
			rightValue:  "fallback",
			expected:    "found value",
			description: "should return left value when it exists",
		},
		{
			name:        "left fails, right used",
			leftPath:    "missing",
			rightValue:  "fallback",
			expected:    "fallback",
			description: "should return right value when left fails",
		},
		{
			name:        "nested path succeeds",
			leftPath:    "nested.key",
			rightValue:  "fallback",
			expected:    "nested value",
			description: "should work with nested paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leftCursor, _ := tree.ParseCursor(tt.leftPath)
			
			expr := &Expr{
				Type: LogicalOr,
				Left: &Expr{
					Type:      Reference,
					Reference: leftCursor,
				},
				Right: &Expr{
					Type:    Literal,
					Literal: tt.rightValue,
				},
			}

			ev := &Evaluator{
				Tree: testData,
			}

			resp, err := EvaluateExpr(expr, ev)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !deepEqual(resp.Value, tt.expected) {
				t.Errorf("%s: expected %v, got %v", tt.description, tt.expected, resp.Value)
			}
		})
	}
}

// TestEvaluateExpr_OperatorCall tests nested operator call evaluation
func TestEvaluateExpr_OperatorCall(t *testing.T) {
	// This requires setting up operators, so we'll create a simple test
	t.Run("operator call evaluation", func(t *testing.T) {
		// Create expression with operator call
		expr := &Expr{
			Type: OperatorCall,
			Call: &Opcall{
				op: &TestExprOperator{
					runFunc: func(ev *Evaluator, args []*Expr) (*Response, error) {
						return &Response{
							Type:  Replace,
							Value: "operator result",
						}, nil
					},
				},
				args: []*Expr{},
			},
		}

		ev := &Evaluator{
			Tree: make(map[interface{}]interface{}),
		}

		resp, err := EvaluateExpr(expr, ev)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.Value != "operator result" {
			t.Errorf("expected 'operator result', got %v", resp.Value)
		}
	})
}

// TestEvaluateExpr_Errors tests error handling in expression evaluation
func TestEvaluateExpr_Errors(t *testing.T) {
	tests := []struct {
		name        string
		expr        *Expr
		expectError string
	}{
		{
			name:        "nil expression",
			expr:        nil,
			expectError: "nil expression",
		},
		{
			name: "unknown expression type",
			expr: &Expr{
				Type: ExprType(999),
			},
			expectError: "unknown expression type",
		},
		{
			name: "reference error",
			expr: func() *Expr {
				cursor, _ := tree.ParseCursor("missing")
				return &Expr{
					Type:      Reference,
					Reference: cursor,
				}
			}(),
			expectError: "could not be found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := &Evaluator{
				Tree: make(map[interface{}]interface{}),
			}

			_, err := EvaluateExpr(tt.expr, ev)
			if err == nil {
				t.Error("expected error but got none")
				return
			}

			if !contains(err.Error(), tt.expectError) {
				t.Errorf("expected error containing '%s', got '%s'", tt.expectError, err.Error())
			}
		})
	}
}

// TestEvaluateExpr_ComplexNesting tests deeply nested expressions
func TestEvaluateExpr_ComplexNesting(t *testing.T) {
	testData := map[interface{}]interface{}{
		"level1": map[interface{}]interface{}{
			"level2": map[interface{}]interface{}{
				"level3": map[interface{}]interface{}{
					"value": "deep value",
				},
			},
		},
		"switch": "level1",
	}

	t.Run("nested OR expressions", func(t *testing.T) {
		// (( grab missing1 || grab missing2 || grab level1.level2.level3.value ))
		cursor1, _ := tree.ParseCursor("missing1")
		cursor2, _ := tree.ParseCursor("missing2")
		cursor3, _ := tree.ParseCursor("level1.level2.level3.value")

		expr := &Expr{
			Type: LogicalOr,
			Left: &Expr{
				Type:      Reference,
				Reference: cursor1,
			},
			Right: &Expr{
				Type: LogicalOr,
				Left: &Expr{
					Type:      Reference,
					Reference: cursor2,
				},
				Right: &Expr{
					Type:      Reference,
					Reference: cursor3,
				},
			},
		}

		ev := &Evaluator{
			Tree: testData,
		}

		resp, err := EvaluateExpr(expr, ev)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.Value != "deep value" {
			t.Errorf("expected 'deep value', got %v", resp.Value)
		}
	})
}

// TestLooksLikeBooleanExpr tests the boolean expression detection
func TestLooksLikeBooleanExpr(t *testing.T) {
	tests := []struct {
		name     string
		expr     *Expr
		expected bool
	}{
		{
			name:     "nil expression",
			expr:     nil,
			expected: false,
		},
		{
			name: "boolean literal",
			expr: &Expr{
				Type:    Literal,
				Literal: true,
			},
			expected: true,
		},
		{
			name: "non-boolean literal",
			expr: &Expr{
				Type:    Literal,
				Literal: "string",
			},
			expected: false,
		},
		{
			name: "boolean operator &&",
			expr: &Expr{
				Type:     OperatorCall,
				Operator: "&&",
			},
			expected: true,
		},
		{
			name: "comparison operator ==",
			expr: &Expr{
				Type:     OperatorCall,
				Operator: "==",
			},
			expected: true,
		},
		{
			name: "non-boolean operator",
			expr: &Expr{
				Type:     OperatorCall,
				Operator: "concat",
			},
			expected: false,
		},
		{
			name: "nested boolean OR",
			expr: &Expr{
				Type: LogicalOr,
				Left: &Expr{
					Type:    Literal,
					Literal: true,
				},
				Right: &Expr{
					Type:    Literal,
					Literal: false,
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := looksLikeBooleanExpr(tt.expr)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestEvaluateExpr_Performance tests expression evaluation performance
func TestEvaluateExpr_Performance(t *testing.T) {
	// Create a large tree for performance testing
	largeTree := make(map[interface{}]interface{})
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key%d", i)
		largeTree[key] = map[interface{}]interface{}{
			"value": fmt.Sprintf("value%d", i),
			"nested": map[interface{}]interface{}{
				"deep": fmt.Sprintf("deep%d", i),
			},
		}
	}

	ev := &Evaluator{
		Tree: largeTree,
	}

	t.Run("simple reference performance", func(t *testing.T) {
		cursor, _ := tree.ParseCursor("key500.value")
		expr := &Expr{
			Type:      Reference,
			Reference: cursor,
		}

		start := time.Now()
		iterations := 10000

		for i := 0; i < iterations; i++ {
			_, err := EvaluateExpr(expr, ev)
			if err != nil {
				t.Fatal(err)
			}
		}

		elapsed := time.Since(start)
		opsPerSecond := float64(iterations) / elapsed.Seconds()

		t.Logf("Simple reference: %.0f ops/sec", opsPerSecond)

		// Ensure reasonable performance
		if opsPerSecond < 100000 {
			t.Errorf("Performance below threshold: %.0f ops/sec", opsPerSecond)
		}
	})

	t.Run("nested reference performance", func(t *testing.T) {
		cursor, _ := tree.ParseCursor("key500.nested.deep")
		expr := &Expr{
			Type:      Reference,
			Reference: cursor,
		}

		start := time.Now()
		iterations := 10000

		for i := 0; i < iterations; i++ {
			_, err := EvaluateExpr(expr, ev)
			if err != nil {
				t.Fatal(err)
			}
		}

		elapsed := time.Since(start)
		opsPerSecond := float64(iterations) / elapsed.Seconds()

		t.Logf("Nested reference: %.0f ops/sec", opsPerSecond)
	})
}

// TestEvaluateExpr_CircularReference tests circular reference detection
func TestEvaluateExpr_CircularReference(t *testing.T) {
	// Note: Circular reference detection would typically be handled
	// at a higher level (in the evaluator), not in expression evaluation
	// This test documents the expected behavior

	testData := map[interface{}]interface{}{
		"a": "(( grab b ))",
		"b": "(( grab c ))",
		"c": "(( grab a ))",
	}

	cursor, _ := tree.ParseCursor("a")
	expr := &Expr{
		Type:      Reference,
		Reference: cursor,
	}

	ev := &Evaluator{
		Tree: testData,
	}

	// This would return the raw operator string since we're not
	// actually evaluating operators here
	resp, err := EvaluateExpr(expr, ev)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The expression evaluation itself doesn't detect cycles,
	// it just returns the value at the path
	if resp.Value != "(( grab b ))" {
		t.Errorf("expected raw operator string, got %v", resp.Value)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsAt(s, substr)
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmarks

func BenchmarkEvaluateExpr_Literal(b *testing.B) {
	expr := &Expr{
		Type:    Literal,
		Literal: "test value",
	}
	ev := &Evaluator{
		Tree: make(map[interface{}]interface{}),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EvaluateExpr(expr, ev)
	}
}

func BenchmarkEvaluateExpr_Reference(b *testing.B) {
	testData := map[interface{}]interface{}{
		"test": map[interface{}]interface{}{
			"nested": map[interface{}]interface{}{
				"value": "benchmark value",
			},
		},
	}

	cursor, _ := tree.ParseCursor("test.nested.value")
	expr := &Expr{
		Type:      Reference,
		Reference: cursor,
	}
	ev := &Evaluator{
		Tree: testData,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EvaluateExpr(expr, ev)
	}
}

func BenchmarkEvaluateExpr_EnvVar(b *testing.B) {
	os.Setenv("BENCH_VAR", "benchmark value")
	defer os.Unsetenv("BENCH_VAR")

	expr := &Expr{
		Type: EnvVar,
		Name: "BENCH_VAR",
	}
	ev := &Evaluator{
		Tree: make(map[interface{}]interface{}),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EvaluateExpr(expr, ev)
	}
}

func BenchmarkEvaluateExpr_LogicalOr(b *testing.B) {
	testData := map[interface{}]interface{}{
		"existing": "value",
	}

	cursor1, _ := tree.ParseCursor("missing")
	cursor2, _ := tree.ParseCursor("existing")

	expr := &Expr{
		Type: LogicalOr,
		Left: &Expr{
			Type:      Reference,
			Reference: cursor1,
		},
		Right: &Expr{
			Type:      Reference,
			Reference: cursor2,
		},
	}
	ev := &Evaluator{
		Tree: testData,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EvaluateExpr(expr, ev)
	}
}