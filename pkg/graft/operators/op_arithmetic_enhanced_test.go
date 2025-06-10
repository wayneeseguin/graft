package operators

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/wayneeseguin/graft/internal/utils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// TestArithmeticOperators_EdgeCases tests edge cases and boundary conditions
func TestArithmeticOperators_EdgeCases(t *testing.T) {
	t.Run("integer overflow", func(t *testing.T) {
		tests := []struct {
			name     string
			op       string
			a        interface{}
			b        interface{}
			expected interface{}
		}{
			{
				name:     "add max int64",
				op:       "+",
				a:        int64(math.MaxInt64 - 1),
				b:        int64(1),
				expected: int64(math.MaxInt64),
			},
			{
				name:     "add causes overflow",
				op:       "+",
				a:        int64(math.MaxInt64),
				b:        int64(1),
				expected: float64(math.MaxInt64) + 1, // Should convert to float
			},
			{
				name:     "subtract min int64",
				op:       "-",
				a:        int64(math.MinInt64 + 1),
				b:        int64(1),
				expected: int64(math.MinInt64),
			},
			{
				name:     "multiply large numbers",
				op:       "*",
				a:        int64(math.MaxInt32),
				b:        int64(2),
				expected: int64(math.MaxInt32) * 2,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var op graft.Operator
				switch tt.op {
				case "+":
					addOp := NewAddOperator()
					op = &addOp
				case "-":
					subOp := NewSubtractOperator()
					op = &subOp
				case "*":
					mulOp := NewMultiplyOperator()
					op = &mulOp
				}

				args := []*graft.Expr{
					{Type: graft.Literal, Literal: tt.a},
					{Type: graft.Literal, Literal: tt.b},
				}

				ev := &graft.Evaluator{}
				resp, err := op.Run(ev, args)

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Type assertion based on expected type
				switch expected := tt.expected.(type) {
				case int64:
					if result, ok := resp.Value.(int64); !ok || result != expected {
						t.Errorf("expected %v (int64), got %v (%T)", expected, resp.Value, resp.Value)
					}
				case float64:
					if result, ok := resp.Value.(float64); !ok || result != expected {
						t.Errorf("expected %v (float64), got %v (%T)", expected, resp.Value, resp.Value)
					}
				}
			})
		}
	})

	t.Run("float special values", func(t *testing.T) {
		tests := []struct {
			name        string
			op          string
			a           interface{}
			b           interface{}
			expectError bool
			checkFunc   func(interface{}) bool
		}{
			{
				name: "add infinity",
				op:   "+",
				a:    math.Inf(1),
				b:    1.0,
				checkFunc: func(v interface{}) bool {
					f, ok := v.(float64)
					return ok && math.IsInf(f, 1)
				},
			},
			{
				name: "subtract infinity",
				op:   "-",
				a:    math.Inf(1),
				b:    math.Inf(1),
				checkFunc: func(v interface{}) bool {
					f, ok := v.(float64)
					return ok && math.IsNaN(f)
				},
			},
			{
				name: "multiply by infinity",
				op:   "*",
				a:    0.0,
				b:    math.Inf(1),
				checkFunc: func(v interface{}) bool {
					f, ok := v.(float64)
					return ok && math.IsNaN(f)
				},
			},
			{
				name: "divide by infinity",
				op:   "/",
				a:    1.0,
				b:    math.Inf(1),
				checkFunc: func(v interface{}) bool {
					f, ok := v.(float64)
					return ok && f == 0.0
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var op graft.Operator
				switch tt.op {
				case "+":
					addOp := NewAddOperator()
					op = &addOp
				case "-":
					subOp := NewSubtractOperator()
					op = &subOp
				case "*":
					mulOp := NewMultiplyOperator()
					op = &mulOp
				case "/":
					divOp := NewDivideOperator()
					op = &divOp
				}

				args := []*graft.Expr{
					{Type: graft.Literal, Literal: tt.a},
					{Type: graft.Literal, Literal: tt.b},
				}

				ev := &graft.Evaluator{}
				resp, err := op.Run(ev, args)

				if tt.expectError {
					if err == nil {
						t.Error("expected error but got none")
					}
					return
				}

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if !tt.checkFunc(resp.Value) {
					t.Errorf("check function failed for result: %v (%T)", resp.Value, resp.Value)
				}
			})
		}
	})

	t.Run("type coercion edge cases", func(t *testing.T) {
		tests := []struct {
			name        string
			op          string
			a           interface{}
			b           interface{}
			expected    interface{}
			expectError bool
		}{
			{
				name:     "add string numbers",
				op:       "+",
				a:        "123",
				b:        "456",
				expected: "123456", // String concatenation via MixedTypeHandler
			},
			{
				name:        "add boolean to number",
				op:          "+",
				a:           true,
				b:           int64(1),
				expectError: true, // Booleans cannot be used in arithmetic
			},
			{
				name:        "multiply boolean",
				op:          "*",
				a:           false,
				b:           int64(100),
				expectError: true, // Booleans cannot be used in arithmetic
			},
			{
				name:        "divide with string number",
				op:          "/",
				a:           "10",
				b:           int64(2),
				expectError: true, // Strings cannot be used in arithmetic division
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var op graft.Operator
				switch tt.op {
				case "+":
					addOp := NewAddOperator()
					op = &addOp
				case "*":
					mulOp := NewMultiplyOperator()
					op = &mulOp
				case "/":
					divOp := NewDivideOperator()
					op = &divOp
				}

				args := []*graft.Expr{
					{Type: graft.Literal, Literal: tt.a},
					{Type: graft.Literal, Literal: tt.b},
				}

				ev := &graft.Evaluator{}
				resp, err := op.Run(ev, args)

				if tt.expectError {
					if err == nil {
						t.Error("expected error but got none")
					}
					return
				}

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if resp.Value != tt.expected {
					t.Errorf("expected %v (%T), got %v (%T)",
						tt.expected, tt.expected, resp.Value, resp.Value)
				}
			})
		}
	})

	t.Run("precision and rounding", func(t *testing.T) {
		tests := []struct {
			name      string
			op        string
			a         interface{}
			b         interface{}
			tolerance float64
			expected  float64
		}{
			{
				name:      "divide with precision",
				op:        "/",
				a:         1.0,
				b:         3.0,
				expected:  0.3333333333333333,
				tolerance: 1e-15,
			},
			{
				name:      "multiply small numbers",
				op:        "*",
				a:         0.1,
				b:         0.2,
				expected:  0.02,
				tolerance: 1e-15,
			},
			{
				name:      "subtract near zero",
				op:        "-",
				a:         0.1 + 0.2,
				b:         0.3,
				expected:  0.0,
				tolerance: 1e-10,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var op graft.Operator
				switch tt.op {
				case "/":
					divOp := NewDivideOperator()
					op = &divOp
				case "*":
					mulOp := NewMultiplyOperator()
					op = &mulOp
				case "-":
					subOp := NewSubtractOperator()
					op = &subOp
				}

				args := []*graft.Expr{
					{Type: graft.Literal, Literal: tt.a},
					{Type: graft.Literal, Literal: tt.b},
				}

				ev := &graft.Evaluator{}
				resp, err := op.Run(ev, args)

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				result, ok := resp.Value.(float64)
				if !ok {
					t.Fatalf("expected float64, got %T", resp.Value)
				}

				if math.Abs(result-tt.expected) > tt.tolerance {
					t.Errorf("expected %v±%v, got %v", tt.expected, tt.tolerance, result)
				}
			})
		}
	})
}

// TestArithmeticOperators_ErrorHandling tests comprehensive error scenarios
func TestArithmeticOperators_ErrorHandling(t *testing.T) {
	t.Run("invalid operand types", func(t *testing.T) {
		tests := []struct {
			name         string
			op           string
			a            interface{}
			b            interface{}
			expectError  string
			expectResult interface{} // For cases that don't error
		}{
			{
				name:         "add incompatible types",
				op:           "+",
				a:            []interface{}{1, 2, 3},
				b:            map[string]interface{}{"key": "value"},
				expectResult: "[1 2 3]map[key:value]", // MixedTypeHandler concatenates as strings
			},
			{
				name:        "subtract from string",
				op:          "-",
				a:           "hello",
				b:           int64(5),
				expectError: "cannot use string",
			},
			{
				name:        "multiply maps",
				op:          "*",
				a:           map[string]interface{}{"a": 1},
				b:           map[string]interface{}{"b": 2},
				expectError: "cannot convert map",
			},
			{
				name:        "modulo with float",
				op:          "%",
				a:           3.14,
				b:           2.0,
				expectError: "not an integer",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var op graft.Operator
				switch tt.op {
				case "+":
					addOp := NewAddOperator()
					op = &addOp
				case "-":
					subOp := NewSubtractOperator()
					op = &subOp
				case "*":
					mulOp := NewMultiplyOperator()
					op = &mulOp
				case "%":
					modOp := NewModuloOperator()
					op = &modOp
				}

				args := []*graft.Expr{
					{Type: graft.Literal, Literal: tt.a},
					{Type: graft.Literal, Literal: tt.b},
				}

				ev := &graft.Evaluator{}
				resp, err := op.Run(ev, args)

				if tt.expectError != "" {
					if err == nil {
						t.Error("expected error but got none")
						return
					}
					if !contains(err.Error(), tt.expectError) {
						t.Errorf("expected error containing %q, got %q", tt.expectError, err.Error())
					}
				} else {
					if err != nil {
						t.Errorf("unexpected error: %v", err)
						return
					}
					if resp.Value != tt.expectResult {
						t.Errorf("expected %v (%T), got %v (%T)",
							tt.expectResult, tt.expectResult, resp.Value, resp.Value)
					}
				}
			})
		}
	})

	t.Run("argument count validation", func(t *testing.T) {
		operators := []struct {
			name string
			op   graft.Operator
		}{
			{"add", func() graft.Operator { op := NewAddOperator(); return &op }()},
			{"subtract", func() graft.Operator { op := NewSubtractOperator(); return &op }()},
			{"multiply", func() graft.Operator { op := NewMultiplyOperator(); return &op }()},
			{"divide", func() graft.Operator { op := NewDivideOperator(); return &op }()},
			{"modulo", func() graft.Operator { op := NewModuloOperator(); return &op }()},
		}

		for _, opTest := range operators {
			t.Run(opTest.name, func(t *testing.T) {
				// Test with no arguments
				_, err := opTest.op.Run(&graft.Evaluator{}, []*graft.Expr{})
				if err == nil {
					t.Error("expected error for no arguments")
				}

				// Test with one argument
				_, err = opTest.op.Run(&graft.Evaluator{}, []*graft.Expr{
					{Type: graft.Literal, Literal: 1},
				})
				if err == nil {
					t.Error("expected error for single argument")
				}

				// Test with too many arguments
				_, err = opTest.op.Run(&graft.Evaluator{}, []*graft.Expr{
					{Type: graft.Literal, Literal: 1},
					{Type: graft.Literal, Literal: 2},
					{Type: graft.Literal, Literal: 3},
				})
				if err == nil {
					t.Error("expected error for too many arguments")
				}
			})
		}
	})
}

// TestArithmeticOperators_Performance tests performance characteristics
func TestArithmeticOperators_Performance(t *testing.T) {
	operators := []struct {
		name string
		op   graft.Operator
		a    interface{}
		b    interface{}
	}{
		{"add_int", func() graft.Operator { op := NewAddOperator(); return &op }(), int64(100), int64(200)},
		{"add_float", func() graft.Operator { op := NewAddOperator(); return &op }(), 100.5, 200.5},
		{"subtract", func() graft.Operator { op := NewSubtractOperator(); return &op }(), int64(1000), int64(500)},
		{"multiply", func() graft.Operator { op := NewMultiplyOperator(); return &op }(), int64(25), int64(4)},
		{"divide", func() graft.Operator { op := NewDivideOperator(); return &op }(), int64(1000), int64(10)},
		{"modulo", func() graft.Operator { op := NewModuloOperator(); return &op }(), int64(1000), int64(7)},
	}

	for _, opTest := range operators {
		t.Run(opTest.name, func(t *testing.T) {
			args := []*graft.Expr{
				{Type: graft.Literal, Literal: opTest.a},
				{Type: graft.Literal, Literal: opTest.b},
			}
			ev := &graft.Evaluator{}

			// Warm up
			for i := 0; i < 100; i++ {
				_, _ = opTest.op.Run(ev, args)
			}

			// Benchmark
			start := time.Now()
			iterations := 100000

			for i := 0; i < iterations; i++ {
				_, err := opTest.op.Run(ev, args)
				if err != nil {
					t.Fatal(err)
				}
			}

			elapsed := time.Since(start)
			opsPerSecond := float64(iterations) / elapsed.Seconds()

			t.Logf("%s: %.0f ops/sec", opTest.name, opsPerSecond)

			// Ensure reasonable performance (at least 1M ops/sec for simple arithmetic)
			if opsPerSecond < 1000000 {
				t.Errorf("Performance below threshold: %.0f ops/sec", opsPerSecond)
			}
		})
	}
}

// TestArithmeticOperators_ComplexExpressions tests nested and complex arithmetic
func TestArithmeticOperators_ComplexExpressions(t *testing.T) {
	t.Run("deeply nested operations", func(t *testing.T) {
		// Test expression like: ((((1 + 2) * 3) - 4) / 2)
		// Expected: (((3) * 3) - 4) / 2 = (9 - 4) / 2 = 5 / 2 = 2.5

		// Build nested expression programmatically
		// This tests that operators handle nested expressions correctly
		ev := &graft.Evaluator{
			Tree: map[interface{}]interface{}{
				"a": int64(1),
				"b": int64(2),
				"c": int64(3),
				"d": int64(4),
				"e": int64(2),
			},
		}

		// Simulate nested operation evaluation
		addOp := NewAddOperator()
		add := &addOp
		resp1, err := add.Run(ev, []*graft.Expr{
			{Type: graft.Reference, Reference: mustParseCursor("a")},
			{Type: graft.Reference, Reference: mustParseCursor("b")},
		})
		if err != nil {
			t.Fatal(err)
		}

		mulOp := NewMultiplyOperator()
		multiply := &mulOp
		resp2, err := multiply.Run(ev, []*graft.Expr{
			{Type: graft.Literal, Literal: resp1.Value},
			{Type: graft.Reference, Reference: mustParseCursor("c")},
		})
		if err != nil {
			t.Fatal(err)
		}

		subOp := NewSubtractOperator()
		subtract := &subOp
		resp3, err := subtract.Run(ev, []*graft.Expr{
			{Type: graft.Literal, Literal: resp2.Value},
			{Type: graft.Reference, Reference: mustParseCursor("d")},
		})
		if err != nil {
			t.Fatal(err)
		}

		divOp := NewDivideOperator()
		divide := &divOp
		resp4, err := divide.Run(ev, []*graft.Expr{
			{Type: graft.Literal, Literal: resp3.Value},
			{Type: graft.Reference, Reference: mustParseCursor("e")},
		})
		if err != nil {
			t.Fatal(err)
		}

		expected := 2.5
		if resp4.Value != expected {
			t.Errorf("expected %v, got %v", expected, resp4.Value)
		}
	})
}

// TestArithmeticOperators_Dependencies tests operator dependency tracking
func TestArithmeticOperators_Dependencies(t *testing.T) {
	operators := []struct {
		name string
		op   graft.Operator
	}{
		{"add", func() graft.Operator { op := NewAddOperator(); return &op }()},
		{"subtract", func() graft.Operator { op := NewSubtractOperator(); return &op }()},
		{"multiply", func() graft.Operator { op := NewMultiplyOperator(); return &op }()},
		{"divide", func() graft.Operator { op := NewDivideOperator(); return &op }()},
		{"modulo", func() graft.Operator { op := NewModuloOperator(); return &op }()},
	}

	for _, opTest := range operators {
		t.Run(opTest.name, func(t *testing.T) {
			ev := &graft.Evaluator{}
			args := []*graft.Expr{
				{Type: graft.Reference, Reference: mustParseCursor("foo.bar")},
				{Type: graft.Reference, Reference: mustParseCursor("baz.qux")},
			}

			locs := []*tree.Cursor{}
			auto := []*tree.Cursor{
				mustParseCursor("foo.bar"),
				mustParseCursor("baz.qux"),
			}

			deps := opTest.op.Dependencies(ev, args, locs, auto)

			// Should return auto dependencies
			if len(deps) != len(auto) {
				t.Errorf("expected %d dependencies, got %d", len(auto), len(deps))
			}
		})
	}
}

// Helper function to parse cursor (panic on error for tests)
func mustParseCursor(path string) *tree.Cursor {
	cursor, err := tree.ParseCursor(path)
	if err != nil {
		panic(fmt.Sprintf("failed to parse cursor %q: %v", path, err))
	}
	return cursor
}

// Helper function for string contains
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmarks

func BenchmarkAddOperator_Integers(b *testing.B) {
	opObj := NewAddOperator()
	op := &opObj
	args := []*graft.Expr{
		{Type: graft.Literal, Literal: int64(100)},
		{Type: graft.Literal, Literal: int64(200)},
	}
	ev := &graft.Evaluator{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = op.Run(ev, args)
	}
}

func BenchmarkAddOperator_Floats(b *testing.B) {
	opObj := NewAddOperator()
	op := &opObj
	args := []*graft.Expr{
		{Type: graft.Literal, Literal: 100.5},
		{Type: graft.Literal, Literal: 200.5},
	}
	ev := &graft.Evaluator{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = op.Run(ev, args)
	}
}

func BenchmarkAddOperator_Strings(b *testing.B) {
	opObj := NewAddOperator()
	op := &opObj
	args := []*graft.Expr{
		{Type: graft.Literal, Literal: "hello"},
		{Type: graft.Literal, Literal: "world"},
	}
	ev := &graft.Evaluator{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = op.Run(ev, args)
	}
}

func BenchmarkMultiplyOperator_LargeNumbers(b *testing.B) {
	opObj := NewMultiplyOperator()
	op := &opObj
	args := []*graft.Expr{
		{Type: graft.Literal, Literal: int64(12345678)},
		{Type: graft.Literal, Literal: int64(87654321)},
	}
	ev := &graft.Evaluator{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = op.Run(ev, args)
	}
}

func BenchmarkDivideOperator_Precision(b *testing.B) {
	opObj := NewDivideOperator()
	op := &opObj
	args := []*graft.Expr{
		{Type: graft.Literal, Literal: 1.0},
		{Type: graft.Literal, Literal: 3.0},
	}
	ev := &graft.Evaluator{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = op.Run(ev, args)
	}
}
