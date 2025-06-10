package operators_test

import (
	"testing"

	"github.com/wayneeseguin/graft/pkg/graft"
	_ "github.com/wayneeseguin/graft/pkg/graft/operators"
)

func TestArithmeticWithParentheses(t *testing.T) {
	evaluator := &graft.Evaluator{
		Tree: map[interface{}]interface{}{
			"base":       10,
			"multiplier": 5,
			"addend":     7,
			"x":          2,
			"y":          3,
			"z":          4,
		},
	}

	testCases := []struct {
		name       string
		expression string
		expected   interface{}
	}{
		// Basic arithmetic
		{"simple addition", "(( 2 + 3 ))", int64(5)},
		{"simple multiplication", "(( 4 * 5 ))", int64(20)},
		{"simple subtraction", "(( 10 - 3 ))", int64(7)},
		{"simple division", "(( 20 / 4 ))", float64(5)},
		{"simple modulo", "(( 10 % 3 ))", int64(1)},

		// Parentheses for precedence
		{"multiplication in parentheses", "(( 2 + (3 * 4) ))", int64(14)},
		{"addition in parentheses", "(( (2 + 3) * 4 ))", int64(20)},
		{"subtraction with parentheses", "(( 10 - (2 + 3) ))", int64(5)},
		{"subtraction in parentheses", "(( (10 - 2) + 3 ))", int64(11)},

		// Nested parentheses
		{"nested parentheses", "(( ((2 + 3) * 4) + 5 ))", int64(25)},
		{"complex nested parentheses", "(( 2 * ((3 + 4) - 5) ))", int64(4)},

		// With references
		{"reference addition", "(( base + addend ))", int64(17)},
		{"reference multiplication", "(( base * multiplier ))", int64(50)},
		{"complex reference expression", "(( (base * multiplier) + addend ))", int64(57)},
		{"reference with arithmetic", "(( base + (multiplier * 2) ))", int64(20)},

		// Mixed literals and references
		{"all references with parentheses", "(( (x + y) * z ))", int64(20)},
		{"references with precedence", "(( x + (y * z) ))", int64(14)},
		{"complex mixed expression", "(( (base / 2) + (multiplier * 3) ))", float64(20)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opcall, err := graft.ParseOpcallCompat(graft.EvalPhase, tc.expression)
			if err != nil {
				t.Fatalf("Failed to parse expression %s: %v", tc.expression, err)
			}
			if opcall == nil {
				t.Fatalf("ParseOpcallCompat returned nil for expression %s", tc.expression)
			}

			result, err := opcall.Run(evaluator)
			if err != nil {
				t.Fatalf("Failed to evaluate expression %s: %v", tc.expression, err)
			}
			if result == nil {
				t.Fatalf("Run returned nil result for expression %s", tc.expression)
			}
			if result.Type != graft.Replace {
				t.Fatalf("Expected Replace response type, got %v for expression %s", result.Type, tc.expression)
			}
			if result.Value != tc.expected {
				t.Fatalf("Expected %v (%T), got %v (%T) for expression %s",
					tc.expected, tc.expected, result.Value, result.Value, tc.expression)
			}
		})
	}
}

func TestDeeplyNestedParentheses(t *testing.T) {
	evaluator := &graft.Evaluator{
		Tree: map[interface{}]interface{}{},
	}

	expr := "(( ((((1 + 1) + 1) + 1) + 1) + 1 ))"
	opcall, err := graft.ParseOpcallCompat(graft.EvalPhase, expr)
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}
	if opcall == nil {
		t.Fatal("ParseOpcallCompat returned nil")
	}

	result, err := opcall.Run(evaluator)
	if err != nil {
		t.Fatalf("Failed to evaluate expression: %v", err)
	}
	if result.Value != int64(6) {
		t.Fatalf("Expected 6, got %v", result.Value)
	}
}

func TestComplexParenthesesWithAllOperators(t *testing.T) {
	evaluator := &graft.Evaluator{
		Tree: map[interface{}]interface{}{},
	}

	expr := "(( ((10 + 5) * 2) - ((20 / 4) % 3) ))"
	opcall, err := graft.ParseOpcallCompat(graft.EvalPhase, expr)
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}
	if opcall == nil {
		t.Fatal("ParseOpcallCompat returned nil")
	}

	result, err := opcall.Run(evaluator)
	if err != nil {
		t.Fatalf("Failed to evaluate expression: %v", err)
	}
	// ((10 + 5) * 2) = 30, ((20 / 4) % 3) = (5 % 3) = 2, 30 - 2 = 28
	if result.Value != int64(28) {
		t.Fatalf("Expected 28, got %v", result.Value)
	}
}
