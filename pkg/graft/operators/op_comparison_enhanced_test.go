package operators

import (
	"math"
	"testing"
	"time"

	"github.com/wayneeseguin/graft/internal/utils/tree"
)

// TestComparisonOperators_Comprehensive tests all comparison operators with comprehensive cases
func TestComparisonOperators_Comprehensive(t *testing.T) {
	t.Run("equality operators", func(t *testing.T) {
		tests := []struct {
			name     string
			op       string
			a        interface{}
			b        interface{}
			expected bool
		}{
			// Basic equality tests
			{name: "equal integers", op: "==", a: int64(5), b: int64(5), expected: true},
			{name: "unequal integers", op: "==", a: int64(5), b: int64(3), expected: false},
			{name: "equal floats", op: "==", a: 3.14, b: 3.14, expected: true},
			{name: "unequal floats", op: "==", a: 3.14, b: 2.71, expected: false},
			{name: "equal strings", op: "==", a: "hello", b: "hello", expected: true},
			{name: "unequal strings", op: "==", a: "hello", b: "world", expected: false},
			{name: "equal booleans", op: "==", a: true, b: true, expected: true},
			{name: "unequal booleans", op: "==", a: true, b: false, expected: false},
			{name: "equal nil", op: "==", a: nil, b: nil, expected: true},
			{name: "nil vs value", op: "==", a: nil, b: "hello", expected: false},
			{name: "value vs nil", op: "==", a: "hello", b: nil, expected: false},
			
			// Numeric type coercion
			{name: "int64 vs float64", op: "==", a: int64(5), b: 5.0, expected: true},
			{name: "float64 vs int64", op: "==", a: 5.0, b: int64(5), expected: true},
			{name: "int vs int64", op: "==", a: int(5), b: int64(5), expected: true},
			{name: "float32 vs float64", op: "==", a: float32(3.14), b: float64(3.14), expected: false}, // precision difference
			
			// Array equality
			{name: "equal arrays", op: "==", a: []interface{}{1, 2, 3}, b: []interface{}{1, 2, 3}, expected: true},
			{name: "unequal arrays", op: "==", a: []interface{}{1, 2, 3}, b: []interface{}{1, 2, 4}, expected: false},
			{name: "arrays different length", op: "==", a: []interface{}{1, 2}, b: []interface{}{1, 2, 3}, expected: false},
			{name: "empty arrays", op: "==", a: []interface{}{}, b: []interface{}{}, expected: true},
			
			// Map equality
			{name: "equal maps", op: "==", a: map[interface{}]interface{}{"a": 1}, b: map[interface{}]interface{}{"a": 1}, expected: true},
			{name: "unequal maps", op: "==", a: map[interface{}]interface{}{"a": 1}, b: map[interface{}]interface{}{"a": 2}, expected: false},
			{name: "maps different keys", op: "==", a: map[interface{}]interface{}{"a": 1}, b: map[interface{}]interface{}{"b": 1}, expected: false},
			{name: "empty maps", op: "==", a: map[interface{}]interface{}{}, b: map[interface{}]interface{}{}, expected: true},
			
			// Not equal tests
			{name: "not equal integers", op: "!=", a: int64(5), b: int64(3), expected: true},
			{name: "not equal same values", op: "!=", a: int64(5), b: int64(5), expected: false},
			{name: "not equal nil", op: "!=", a: nil, b: nil, expected: false},
			{name: "not equal nil vs value", op: "!=", a: nil, b: "hello", expected: true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var op Operator
				switch tt.op {
				case "==":
					opObj := NewTypeAwareEqualOperator()
					op = opObj
				case "!=":
					opObj := NewTypeAwareNotEqualOperator()
					op = opObj
				}

				ev := &Evaluator{Tree: map[interface{}]interface{}{}}
				args := []*Expr{
					{Type: Literal, Literal: tt.a},
					{Type: Literal, Literal: tt.b},
				}

				resp, err := op.Run(ev, args)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				result, ok := resp.Value.(bool)
				if !ok {
					t.Fatalf("expected bool result, got %T", resp.Value)
				}

				if result != tt.expected {
					t.Errorf("expected %v, got %v for %v %s %v", tt.expected, result, tt.a, tt.op, tt.b)
				}
			})
		}
	})

	t.Run("ordering operators", func(t *testing.T) {
		tests := []struct {
			name     string
			op       string
			a        interface{}
			b        interface{}
			expected bool
		}{
			// Numeric comparisons
			{name: "5 < 10", op: "<", a: int64(5), b: int64(10), expected: true},
			{name: "10 < 5", op: "<", a: int64(10), b: int64(5), expected: false},
			{name: "5 < 5", op: "<", a: int64(5), b: int64(5), expected: false},
			{name: "5 > 3", op: ">", a: int64(5), b: int64(3), expected: true},
			{name: "3 > 5", op: ">", a: int64(3), b: int64(5), expected: false},
			{name: "5 > 5", op: ">", a: int64(5), b: int64(5), expected: false},
			{name: "5 <= 10", op: "<=", a: int64(5), b: int64(10), expected: true},
			{name: "5 <= 5", op: "<=", a: int64(5), b: int64(5), expected: true},
			{name: "10 <= 5", op: "<=", a: int64(10), b: int64(5), expected: false},
			{name: "10 >= 5", op: ">=", a: int64(10), b: int64(5), expected: true},
			{name: "5 >= 5", op: ">=", a: int64(5), b: int64(5), expected: true},
			{name: "3 >= 5", op: ">=", a: int64(3), b: int64(5), expected: false},

			// Float comparisons
			{name: "3.14 < 3.15", op: "<", a: 3.14, b: 3.15, expected: true},
			{name: "3.15 > 3.14", op: ">", a: 3.15, b: 3.14, expected: true},
			{name: "3.14 <= 3.14", op: "<=", a: 3.14, b: 3.14, expected: true},
			{name: "3.14 >= 3.14", op: ">=", a: 3.14, b: 3.14, expected: true},

			// Mixed numeric types
			{name: "int64 < float64", op: "<", a: int64(5), b: 5.1, expected: true},
			{name: "float64 > int64", op: ">", a: 5.1, b: int64(5), expected: true},
			{name: "int64 == float64", op: "<=", a: int64(5), b: 5.0, expected: true},
			{name: "float64 == int64", op: ">=", a: 5.0, b: int64(5), expected: true},

			// String comparisons (lexicographic)
			{name: "\"apple\" < \"banana\"", op: "<", a: "apple", b: "banana", expected: true},
			{name: "\"banana\" > \"apple\"", op: ">", a: "banana", b: "apple", expected: true},
			{name: "\"apple\" <= \"apple\"", op: "<=", a: "apple", b: "apple", expected: true},
			{name: "\"apple\" >= \"apple\"", op: ">=", a: "apple", b: "apple", expected: true},
			{name: "\"10\" < \"2\"", op: "<", a: "10", b: "2", expected: true}, // lexicographic, not numeric
			{name: "\"abc\" < \"abd\"", op: "<", a: "abc", b: "abd", expected: true},

			// Mixed type comparisons are not supported in ordering operations
			// These will be tested in error handling section

			// Nil comparisons
			{name: "nil < value", op: "<", a: nil, b: int64(5), expected: true},
			{name: "value > nil", op: ">", a: int64(5), b: nil, expected: true},
			{name: "nil <= nil", op: "<=", a: nil, b: nil, expected: true},
			{name: "nil >= nil", op: ">=", a: nil, b: nil, expected: true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var op Operator
				switch tt.op {
				case "<":
					opObj := NewTypeAwareLessOperator()
					op = opObj
				case ">":
					opObj := NewTypeAwareGreaterOperator()
					op = opObj
				case "<=":
					opObj := NewTypeAwareLessOrEqualOperator()
					op = opObj
				case ">=":
					opObj := NewTypeAwareGreaterOrEqualOperator()
					op = opObj
				}

				ev := &Evaluator{Tree: map[interface{}]interface{}{}}
				args := []*Expr{
					{Type: Literal, Literal: tt.a},
					{Type: Literal, Literal: tt.b},
				}

				resp, err := op.Run(ev, args)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				result, ok := resp.Value.(bool)
				if !ok {
					t.Fatalf("expected bool result, got %T", resp.Value)
				}

				if result != tt.expected {
					t.Errorf("expected %v, got %v for %v %s %v", tt.expected, result, tt.a, tt.op, tt.b)
				}
			})
		}
	})
}

// TestComparisonOperators_SpecialValues tests edge cases and special values
func TestComparisonOperators_SpecialValues(t *testing.T) {
	t.Run("floating point special values", func(t *testing.T) {
		tests := []struct {
			name     string
			op       string
			a        interface{}
			b        interface{}
			expected bool
		}{
			{name: "Inf == Inf", op: "==", a: math.Inf(1), b: math.Inf(1), expected: true},
			{name: "Inf != -Inf", op: "!=", a: math.Inf(1), b: math.Inf(-1), expected: true},
			{name: "NaN != NaN", op: "!=", a: math.NaN(), b: math.NaN(), expected: true}, // NaN is never equal to itself
			{name: "Inf > 1000", op: ">", a: math.Inf(1), b: 1000.0, expected: true},
			{name: "-Inf < -1000", op: "<", a: math.Inf(-1), b: -1000.0, expected: true},
			{name: "1000 < Inf", op: "<", a: 1000.0, b: math.Inf(1), expected: true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var op Operator
				switch tt.op {
				case "==":
					opObj := NewTypeAwareEqualOperator()
					op = opObj
				case "!=":
					opObj := NewTypeAwareNotEqualOperator()
					op = opObj
				case "<":
					opObj := NewTypeAwareLessOperator()
					op = opObj
				case ">":
					opObj := NewTypeAwareGreaterOperator()
					op = opObj
				}

				ev := &Evaluator{Tree: map[interface{}]interface{}{}}
				args := []*Expr{
					{Type: Literal, Literal: tt.a},
					{Type: Literal, Literal: tt.b},
				}

				resp, err := op.Run(ev, args)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				result, ok := resp.Value.(bool)
				if !ok {
					t.Fatalf("expected bool result, got %T", resp.Value)
				}

				if result != tt.expected {
					t.Errorf("expected %v, got %v for %v %s %v", tt.expected, result, tt.a, tt.op, tt.b)
				}
			})
		}
	})

	t.Run("precision and rounding", func(t *testing.T) {
		tests := []struct {
			name     string
			op       string
			a        interface{}
			b        interface{}
			expected bool
		}{
			{name: "floating point precision", op: "==", a: 0.1 + 0.2, b: 0.3, expected: true}, // Go's float64 precision
			{name: "very close floats !=", op: "!=", a: 1.0000000000000001, b: 1.0, expected: false}, // within float64 precision
			{name: "distinguishable floats", op: "!=", a: 1.1, b: 1.2, expected: true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var op Operator
				switch tt.op {
				case "==":
					opObj := NewTypeAwareEqualOperator()
					op = opObj
				case "!=":
					opObj := NewTypeAwareNotEqualOperator()
					op = opObj
				}

				ev := &Evaluator{Tree: map[interface{}]interface{}{}}
				args := []*Expr{
					{Type: Literal, Literal: tt.a},
					{Type: Literal, Literal: tt.b},
				}

				resp, err := op.Run(ev, args)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				result, ok := resp.Value.(bool)
				if !ok {
					t.Fatalf("expected bool result, got %T", resp.Value)
				}

				if result != tt.expected {
					t.Errorf("expected %v, got %v for %v %s %v", tt.expected, result, tt.a, tt.op, tt.b)
				}
			})
		}
	})

	t.Run("complex nested structures", func(t *testing.T) {
		tests := []struct {
			name     string
			op       string
			a        interface{}
			b        interface{}
			expected bool
		}{
			{
				name: "nested arrays equal",
				op:   "==",
				a:    []interface{}{[]interface{}{1, 2}, []interface{}{3, 4}},
				b:    []interface{}{[]interface{}{1, 2}, []interface{}{3, 4}},
				expected: true,
			},
			{
				name: "nested arrays unequal",
				op:   "!=",
				a:    []interface{}{[]interface{}{1, 2}, []interface{}{3, 4}},
				b:    []interface{}{[]interface{}{1, 2}, []interface{}{3, 5}},
				expected: true,
			},
			{
				name: "nested maps equal",
				op:   "==",
				a:    map[interface{}]interface{}{"nested": map[interface{}]interface{}{"key": "value"}},
				b:    map[interface{}]interface{}{"nested": map[interface{}]interface{}{"key": "value"}},
				expected: true,
			},
			{
				name: "mixed nested structures",
				op:   "==",
				a:    map[interface{}]interface{}{"array": []interface{}{1, 2, 3}},
				b:    map[interface{}]interface{}{"array": []interface{}{1, 2, 3}},
				expected: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var op Operator
				switch tt.op {
				case "==":
					opObj := NewTypeAwareEqualOperator()
					op = opObj
				case "!=":
					opObj := NewTypeAwareNotEqualOperator()
					op = opObj
				}

				ev := &Evaluator{Tree: map[interface{}]interface{}{}}
				args := []*Expr{
					{Type: Literal, Literal: tt.a},
					{Type: Literal, Literal: tt.b},
				}

				resp, err := op.Run(ev, args)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				result, ok := resp.Value.(bool)
				if !ok {
					t.Fatalf("expected bool result, got %T", resp.Value)
				}

				if result != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, result)
				}
			})
		}
	})
}

// TestComparisonOperators_ErrorHandling tests error cases and validation
func TestComparisonOperators_ErrorHandling(t *testing.T) {
	t.Run("argument validation", func(t *testing.T) {
		operators := []struct {
			name string
			op   Operator
		}{
			{"equal", func() Operator { op := NewTypeAwareEqualOperator(); return op }()},
			{"not_equal", func() Operator { op := NewTypeAwareNotEqualOperator(); return op }()},
			{"less", func() Operator { op := NewTypeAwareLessOperator(); return op }()},
			{"greater", func() Operator { op := NewTypeAwareGreaterOperator(); return op }()},
			{"less_equal", func() Operator { op := NewTypeAwareLessOrEqualOperator(); return op }()},
			{"greater_equal", func() Operator { op := NewTypeAwareGreaterOrEqualOperator(); return op }()},
		}

		for _, opTest := range operators {
			t.Run(opTest.name, func(t *testing.T) {
				ev := &Evaluator{Tree: map[interface{}]interface{}{}}

				// Test with no arguments
				_, err := opTest.op.Run(ev, []*Expr{})
				if err == nil {
					t.Error("expected error for no arguments")
				}

				// Test with one argument
				_, err = opTest.op.Run(ev, []*Expr{
					{Type: Literal, Literal: 1},
				})
				if err == nil {
					t.Error("expected error for single argument")
				}

				// Test with too many arguments
				_, err = opTest.op.Run(ev, []*Expr{
					{Type: Literal, Literal: 1},
					{Type: Literal, Literal: 2},
					{Type: Literal, Literal: 3},
				})
				if err == nil {
					t.Error("expected error for too many arguments")
				}
			})
		}
	})

	t.Run("incompatible type comparisons", func(t *testing.T) {
		tests := []struct {
			name string
			op   string
			a    interface{}
			b    interface{}
		}{
			{name: "array vs map ordering", op: "<", a: []interface{}{1, 2}, b: map[interface{}]interface{}{"a": 1}},
			{name: "bool vs array ordering", op: ">", a: true, b: []interface{}{1, 2}},
			{name: "map vs bool ordering", op: "<=", a: map[interface{}]interface{}{"a": 1}, b: true},
			{name: "string vs number ordering", op: "<", a: "10", b: int64(2)},
			{name: "number vs string ordering", op: ">", a: int64(2), b: "10"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var op Operator
				switch tt.op {
				case "<":
					opObj := NewTypeAwareLessOperator()
					op = opObj
				case ">":
					opObj := NewTypeAwareGreaterOperator()
					op = opObj
				case "<=":
					opObj := NewTypeAwareLessOrEqualOperator()
					op = opObj
				}

				ev := &Evaluator{Tree: map[interface{}]interface{}{}}
				args := []*Expr{
					{Type: Literal, Literal: tt.a},
					{Type: Literal, Literal: tt.b},
				}

				_, err := op.Run(ev, args)
				if err == nil {
					t.Error("expected error for incompatible type comparison")
				}
			})
		}
	})
}

// TestComparisonOperators_WithReferences tests comparisons using references
func TestComparisonOperators_WithReferences(t *testing.T) {
	t.Run("reference resolution", func(t *testing.T) {
		data := map[interface{}]interface{}{
			"a": int64(5),
			"b": int64(10),
			"nested": map[interface{}]interface{}{
				"value": "hello",
			},
			"array": []interface{}{1, 2, 3},
		}

		tests := []struct {
			name      string
			op        string
			aPath     string
			bPath     string
			expected  bool
		}{
			{name: "a < b", op: "<", aPath: "a", bPath: "b", expected: true},
			{name: "b > a", op: ">", aPath: "b", bPath: "a", expected: true},
			{name: "a <= a", op: "<=", aPath: "a", bPath: "a", expected: true},
			{name: "nested value == literal", op: "==", aPath: "nested.value", bPath: "", expected: true}, // vs literal
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var op Operator
				switch tt.op {
				case "==":
					opObj := NewTypeAwareEqualOperator()
					op = opObj
				case "<":
					opObj := NewTypeAwareLessOperator()
					op = opObj
				case ">":
					opObj := NewTypeAwareGreaterOperator()
					op = opObj
				case "<=":
					opObj := NewTypeAwareLessOrEqualOperator()
					op = opObj
				}

				ev := &Evaluator{Tree: data}
				
				args := make([]*Expr, 2)
				
				// First argument
				cursor1, err := tree.ParseCursor(tt.aPath)
				if err != nil {
					t.Fatalf("failed to parse cursor: %v", err)
				}
				args[0] = &Expr{Type: Reference, Reference: cursor1}
				
				// Second argument
				if tt.bPath != "" {
					cursor2, err := tree.ParseCursor(tt.bPath)
					if err != nil {
						t.Fatalf("failed to parse cursor: %v", err)
					}
					args[1] = &Expr{Type: Reference, Reference: cursor2}
				} else {
					// Use literal for nested.value test
					args[1] = &Expr{Type: Literal, Literal: "hello"}
				}

				resp, err := op.Run(ev, args)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				result, ok := resp.Value.(bool)
				if !ok {
					t.Fatalf("expected bool result, got %T", resp.Value)
				}

				if result != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, result)
				}
			})
		}
	})

	t.Run("dependencies", func(t *testing.T) {
		op := NewTypeAwareEqualOperator()
		ev := &Evaluator{}
		
		cursor1, _ := tree.ParseCursor("foo.bar")
		cursor2, _ := tree.ParseCursor("baz.qux")
		
		args := []*Expr{
			{Type: Reference, Reference: cursor1},
			{Type: Reference, Reference: cursor2},
		}
		
		locs := []*tree.Cursor{}
		auto := []*tree.Cursor{}
		
		deps := op.Dependencies(ev, args, locs, auto)
		
		// Should return auto dependencies (type-aware operators use auto deps)
		if len(deps) != len(auto) {
			t.Errorf("expected %d dependencies, got %d", len(auto), len(deps))
		}
	})
}

// TestComparisonOperators_Performance tests comparison operator performance
func TestComparisonOperators_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance tests in short mode")
	}

	operators := []struct {
		name string
		op   Operator
		a    interface{}
		b    interface{}
	}{
		{"equal_int", func() Operator { op := NewTypeAwareEqualOperator(); return op }(), int64(100), int64(200)},
		{"equal_float", func() Operator { op := NewTypeAwareEqualOperator(); return op }(), 100.5, 200.5},
		{"equal_string", func() Operator { op := NewTypeAwareEqualOperator(); return op }(), "hello", "world"},
		{"less_int", func() Operator { op := NewTypeAwareLessOperator(); return op }(), int64(100), int64(200)},
		{"less_float", func() Operator { op := NewTypeAwareLessOperator(); return op }(), 100.5, 200.5},
		{"less_string", func() Operator { op := NewTypeAwareLessOperator(); return op }(), "apple", "banana"},
	}

	for _, opTest := range operators {
		t.Run(opTest.name, func(t *testing.T) {
			args := []*Expr{
				{Type: Literal, Literal: opTest.a},
				{Type: Literal, Literal: opTest.b},
			}
			ev := &Evaluator{Tree: map[interface{}]interface{}{}}

			// Warm up
			for i := 0; i < 1000; i++ {
				_, _ = opTest.op.Run(ev, args)
			}

			// Simple timing measurement
			iterations := 100000
			start := time.Now()

			for i := 0; i < iterations; i++ {
				_, err := opTest.op.Run(ev, args)
				if err != nil {
					t.Fatal(err)
				}
			}

			elapsed := time.Since(start)
			opsPerSecond := float64(iterations) / elapsed.Seconds()

			t.Logf("%s: %.0f ops/sec", opTest.name, opsPerSecond)

			// Ensure reasonable performance (at least 500K ops/sec)
			if opsPerSecond < 500000 {
				t.Errorf("Performance below threshold: %.0f ops/sec", opsPerSecond)
			}
		})
	}
}