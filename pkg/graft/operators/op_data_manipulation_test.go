package operators

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/wayneeseguin/graft/internal/utils/tree"
)

// TestGrabOperator tests the grab operator functionality
func TestGrabOperator(t *testing.T) {
	t.Run("basic path resolution", func(t *testing.T) {
		tests := []struct {
			name     string
			tree     map[interface{}]interface{}
			path     string
			expected interface{}
			wantErr  bool
		}{
			{
				name: "simple path",
				tree: map[interface{}]interface{}{
					"key": "value",
				},
				path:     "key",
				expected: "value",
			},
			{
				name: "nested path",
				tree: map[interface{}]interface{}{
					"parent": map[interface{}]interface{}{
						"child": "nested value",
					},
				},
				path:     "parent.child",
				expected: "nested value",
			},
			{
				name: "array access",
				tree: map[interface{}]interface{}{
					"list": []interface{}{"first", "second", "third"},
				},
				path:     "list.1",
				expected: "second",
			},
			{
				name: "deep nesting",
				tree: map[interface{}]interface{}{
					"a": map[interface{}]interface{}{
						"b": map[interface{}]interface{}{
							"c": map[interface{}]interface{}{
								"d": "deep value",
							},
						},
					},
				},
				path:     "a.b.c.d",
				expected: "deep value",
			},
			{
				name: "missing path",
				tree: map[interface{}]interface{}{
					"key": "value",
				},
				path:    "missing.path",
				wantErr: true,
			},
			{
				name: "nil value",
				tree: map[interface{}]interface{}{
					"key": nil,
				},
				path:     "key",
				expected: nil,
			},
			{
				name: "boolean value",
				tree: map[interface{}]interface{}{
					"enabled": true,
				},
				path:     "enabled",
				expected: true,
			},
			{
				name: "numeric value",
				tree: map[interface{}]interface{}{
					"count": int64(42),
				},
				path:     "count",
				expected: int64(42),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				op := GrabOperator{}
				ev := &Evaluator{Tree: tt.tree}

				cursor, err := tree.ParseCursor(tt.path)
				if err != nil {
					t.Fatalf("failed to parse cursor: %v", err)
				}

				args := []*Expr{
					{Type: Reference, Reference: cursor},
				}

				resp, err := op.Run(ev, args)

				if tt.wantErr {
					if err == nil {
						t.Error("expected error but got none")
					}
					return
				}

				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if resp.Value != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, resp.Value)
				}
			})
		}
	})

	t.Run("multiple arguments", func(t *testing.T) {
		// Grab with multiple arguments flattens them into an array
		op := GrabOperator{}
		ev := &Evaluator{Tree: map[interface{}]interface{}{
			"key1": "value1",
			"key2": "value2",
		}}

		cursor1, _ := tree.ParseCursor("key1")
		cursor2, _ := tree.ParseCursor("key2")

		args := []*Expr{
			{Type: Reference, Reference: cursor1},
			{Type: Reference, Reference: cursor2},
			{Type: Literal, Literal: "literal"},
		}

		resp, err := op.Run(ev, args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, ok := resp.Value.([]interface{})
		if !ok {
			t.Fatalf("expected array, got %T", resp.Value)
		}

		expected := []interface{}{"value1", "value2", "literal"}
		if len(result) != len(expected) {
			t.Errorf("expected length %d, got %d", len(expected), len(result))
		}

		for i, v := range expected {
			if i < len(result) && result[i] != v {
				t.Errorf("at index %d: expected %v, got %v", i, v, result[i])
			}
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("empty path", func(t *testing.T) {
			op := GrabOperator{}
			ev := &Evaluator{Tree: map[interface{}]interface{}{}}

			_, err := op.Run(ev, []*Expr{})
			if err == nil {
				t.Error("expected error for empty arguments")
			}
		})

		t.Run("literal arguments pass through", func(t *testing.T) {
			op := GrabOperator{}
			ev := &Evaluator{Tree: map[interface{}]interface{}{}}

			args := []*Expr{
				{Type: Literal, Literal: "literal value"},
			}

			resp, err := op.Run(ev, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.Value != "literal value" {
				t.Errorf("expected literal value to pass through, got %v", resp.Value)
			}
		})

		t.Run("too many arguments", func(t *testing.T) {
			op := GrabOperator{}
			ev := &Evaluator{Tree: map[interface{}]interface{}{}}

			cursor, _ := tree.ParseCursor("path")
			args := []*Expr{
				{Type: Reference, Reference: cursor},
				{Type: Literal, Literal: "fallback1"},
				{Type: Literal, Literal: "fallback2"},
			}

			_, err := op.Run(ev, args)
			if err == nil {
				t.Error("expected error for too many arguments")
			}
		})
	})

	t.Run("dependencies", func(t *testing.T) {
		op := GrabOperator{}
		ev := &Evaluator{}

		cursor1, _ := tree.ParseCursor("foo.bar")
		cursor2, _ := tree.ParseCursor("baz.qux")

		args := []*Expr{
			{Type: Reference, Reference: cursor1},
		}

		locs := []*tree.Cursor{}
		auto := []*tree.Cursor{cursor2}

		deps := op.Dependencies(ev, args, locs, auto)

		// Should include the referenced path in dependencies
		found := false
		for _, dep := range deps {
			if dep.String() == "foo.bar" {
				found = true
				break
			}
		}

		if !found {
			t.Error("expected foo.bar in dependencies")
		}
	})
}

// TestConcatOperator tests the concat operator functionality
func TestConcatOperator(t *testing.T) {
	t.Run("string concatenation", func(t *testing.T) {
		tests := []struct {
			name     string
			args     []interface{}
			expected string
		}{
			{
				name:     "two strings",
				args:     []interface{}{"hello", "world"},
				expected: "helloworld",
			},
			{
				name:     "multiple strings",
				args:     []interface{}{"a", "b", "c", "d"},
				expected: "abcd",
			},
			{
				name:     "strings with spaces",
				args:     []interface{}{"hello ", "world"},
				expected: "hello world",
			},
			{
				name:     "empty strings",
				args:     []interface{}{"", "test", ""},
				expected: "test",
			},
			// Single string test removed - concat requires at least 2 args
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				op := ConcatOperator{}
				ev := &Evaluator{}

				args := make([]*Expr, len(tt.args))
				for i, arg := range tt.args {
					args[i] = &Expr{Type: Literal, Literal: arg}
				}

				resp, err := op.Run(ev, args)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if resp.Value != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, resp.Value)
				}
			})
		}
	})

	t.Run("mixed types", func(t *testing.T) {
		tests := []struct {
			name     string
			args     []interface{}
			expected string
		}{
			{
				name:     "string and number",
				args:     []interface{}{"value: ", int64(42)},
				expected: "value: 42",
			},
			{
				name:     "string and boolean",
				args:     []interface{}{"enabled: ", true},
				expected: "enabled: true",
			},
			{
				name:     "multiple types",
				args:     []interface{}{"test", int64(123), "-", false},
				expected: "test123-false",
			},
			{
				name:     "with nil",
				args:     []interface{}{"before", nil, "after"},
				expected: "before<nil>after",
			},
			{
				name:     "float values",
				args:     []interface{}{"pi: ", 3.14159},
				expected: "pi: 3.14159",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				op := ConcatOperator{}
				ev := &Evaluator{}

				args := make([]*Expr, len(tt.args))
				for i, arg := range tt.args {
					args[i] = &Expr{Type: Literal, Literal: arg}
				}

				resp, err := op.Run(ev, args)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if resp.Value != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, resp.Value)
				}
			})
		}
	})

	t.Run("array to string conversion", func(t *testing.T) {
		tests := []struct {
			name     string
			args     []interface{}
			expected string
		}{
			{
				name: "array elements joined as string",
				args: []interface{}{
					[]interface{}{1, 2, 3},
					"suffix",
				},
				expected: "123suffix",
			},
			{
				name: "mixed array and string",
				args: []interface{}{
					"prefix",
					[]interface{}{"a", "b"},
				},
				expected: "prefixab",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				op := ConcatOperator{}
				ev := &Evaluator{}

				args := make([]*Expr, len(tt.args))
				for i, arg := range tt.args {
					args[i] = &Expr{Type: Literal, Literal: arg}
				}

				resp, err := op.Run(ev, args)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if resp.Value != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, resp.Value)
				}
			})
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("no arguments", func(t *testing.T) {
			op := ConcatOperator{}
			ev := &Evaluator{}

			_, err := op.Run(ev, []*Expr{})
			if err == nil {
				t.Error("expected error for no arguments")
			}
		})

		t.Run("maps converted to string", func(t *testing.T) {
			op := ConcatOperator{}
			ev := &Evaluator{}

			args := []*Expr{
				{Type: Literal, Literal: map[interface{}]interface{}{"a": 1}},
				{Type: Literal, Literal: "string"},
			}

			resp, err := op.Run(ev, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Maps get converted to string representation
			result := resp.Value.(string)
			if !strings.Contains(result, "string") {
				t.Errorf("expected result to contain 'string', got %q", result)
			}
		})
	})
}

// TestInjectOperator tests the inject operator functionality
func TestInjectOperator(t *testing.T) {
	t.Run("basic injection", func(t *testing.T) {
		t.Run("inject valid map", func(t *testing.T) {
			op := InjectOperator{}
			ev := &Evaluator{Tree: map[interface{}]interface{}{}}

			args := []*Expr{
				{Type: Literal, Literal: map[interface{}]interface{}{"key": "value"}},
			}

			_, err := op.Run(ev, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})

		t.Run("inject nil causes error", func(t *testing.T) {
			op := InjectOperator{}
			ev := &Evaluator{Tree: map[interface{}]interface{}{}}

			args := []*Expr{
				{Type: Literal, Literal: nil},
			}

			_, err := op.Run(ev, args)
			if err == nil {
				t.Error("expected error for nil argument")
			}
		})

		t.Run("inject non-map value causes error", func(t *testing.T) {
			op := InjectOperator{}
			ev := &Evaluator{Tree: map[interface{}]interface{}{}}

			args := []*Expr{
				{Type: Literal, Literal: "not a map"},
			}

			_, err := op.Run(ev, args)
			if err == nil {
				t.Error("expected error for non-map argument")
			}
		})
	})

	t.Run("argument validation", func(t *testing.T) {
		op := InjectOperator{}
		ev := &Evaluator{}

		// No arguments
		_, err := op.Run(ev, []*Expr{})
		if err == nil {
			t.Error("expected error for no arguments")
		}
	})
}

// TestDeferOperator tests the defer operator functionality
func TestDeferOperator(t *testing.T) {
	t.Run("basic deferral", func(t *testing.T) {
		tests := []struct {
			name     string
			value    interface{}
			expected string
		}{
			{
				name:     "defer string",
				value:    "hello",
				expected: `(( "hello" ))`,
			},
			{
				name:     "defer integer",
				value:    int64(42),
				expected: "(( 42 ))",
			},
			{
				name:     "defer float",
				value:    3.14,
				expected: "(( 3.14 ))",
			},
			{
				name:     "defer boolean",
				value:    true,
				expected: "(( true ))",
			},
			{
				name:     "defer nil",
				value:    nil,
				expected: "(( nil ))",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				op := DeferOperator{}
				ev := &Evaluator{}

				args := []*Expr{
					{Type: Literal, Literal: tt.value},
				}

				resp, err := op.Run(ev, args)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Defer operator returns the YAML representation
				result, ok := resp.Value.(string)
				if !ok {
					t.Fatalf("expected string, got %T", resp.Value)
				}

				if result != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			})
		}
	})

	t.Run("complex structures", func(t *testing.T) {
		t.Run("defer map", func(t *testing.T) {
			op := DeferOperator{}
			ev := &Evaluator{}

			args := []*Expr{
				{Type: Literal, Literal: map[interface{}]interface{}{
					"key": "value",
					"num": int64(42),
				}},
			}

			resp, err := op.Run(ev, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			result, ok := resp.Value.(string)
			if !ok {
				t.Fatalf("expected string, got %T", resp.Value)
			}

			// Should be a graft expression
			if !strings.HasPrefix(result, "((") || !strings.HasSuffix(result, "))") {
				t.Errorf("expected graft expression format, got %q", result)
			}
		})

		t.Run("defer array", func(t *testing.T) {
			op := DeferOperator{}
			ev := &Evaluator{}

			args := []*Expr{
				{Type: Literal, Literal: []interface{}{"a", "b", "c"}},
			}

			resp, err := op.Run(ev, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			result, ok := resp.Value.(string)
			if !ok {
				t.Fatalf("expected string, got %T", resp.Value)
			}

			// Should be a graft expression
			if !strings.HasPrefix(result, "((") || !strings.HasSuffix(result, "))") {
				t.Errorf("expected graft expression format, got %q", result)
			}
		})
	})

	t.Run("defer expressions", func(t *testing.T) {
		t.Run("defer operator expression", func(t *testing.T) {
			op := DeferOperator{}
			ev := &Evaluator{}

			// Create an expression that looks like an operator
			args := []*Expr{
				{Type: Literal, Literal: "(( grab meta.value ))"},
			}

			resp, err := op.Run(ev, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Should wrap the string in a defer expression
			expected := `(( "(( grab meta.value ))" ))`
			if resp.Value != expected {
				t.Errorf("expected %q, got %q", expected, resp.Value)
			}
		})
	})

	t.Run("argument validation", func(t *testing.T) {
		op := DeferOperator{}
		ev := &Evaluator{}

		// No arguments
		_, err := op.Run(ev, []*Expr{})
		if err == nil {
			t.Error("expected error for no arguments")
		}

		// Multiple arguments are allowed for defer
		_, err = op.Run(ev, []*Expr{
			{Type: Literal, Literal: "value"},
			{Type: Literal, Literal: "extra"},
		})
		if err != nil {
			t.Errorf("unexpected error for multiple arguments: %v", err)
		}
	})
}

// TestDataOperators_Performance tests performance of data manipulation operators
func TestDataOperators_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance tests in short mode")
	}

	t.Run("grab performance", func(t *testing.T) {
		// Create a deep tree structure
		data := make(map[interface{}]interface{})
		current := data
		for i := 0; i < 10; i++ {
			next := make(map[interface{}]interface{})
			current[fmt.Sprintf("level%d", i)] = next
			current = next
		}
		current["value"] = "deep value"

		op := GrabOperator{}
		ev := &Evaluator{Tree: data}

		cursor, _ := tree.ParseCursor("level0.level1.level2.level3.level4.level5.level6.level7.level8.level9.value")
		args := []*Expr{
			{Type: Reference, Reference: cursor},
		}

		// Warm up
		for i := 0; i < 100; i++ {
			_, _ = op.Run(ev, args)
		}

		// Simple timing measurement
		iterations := 10000
		start := time.Now()
		for i := 0; i < iterations; i++ {
			_, err := op.Run(ev, args)
			if err != nil {
				t.Fatal(err)
			}
		}
		elapsed := time.Since(start)

		t.Logf("grab deep path: %d ns/op", elapsed.Nanoseconds()/int64(iterations))
	})

	t.Run("concat performance", func(t *testing.T) {
		op := ConcatOperator{}
		ev := &Evaluator{}

		// Test concatenating many strings
		args := make([]*Expr, 100)
		for i := 0; i < 100; i++ {
			args[i] = &Expr{Type: Literal, Literal: fmt.Sprintf("part%d", i)}
		}

		// Warm up
		for i := 0; i < 100; i++ {
			_, _ = op.Run(ev, args)
		}

		// Simple timing measurement
		iterations := 10000
		start := time.Now()
		for i := 0; i < iterations; i++ {
			_, err := op.Run(ev, args)
			if err != nil {
				t.Fatal(err)
			}
		}
		elapsed := time.Since(start)

		t.Logf("concat 100 strings: %d ns/op", elapsed.Nanoseconds()/int64(iterations))
	})
}
