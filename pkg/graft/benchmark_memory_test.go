package graft_test

import (
	"testing"
	
	"github.com/wayneeseguin/graft/internal/utils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
	"github.com/wayneeseguin/graft/pkg/graft/operators"
)

// BenchmarkConcatOperatorMemory benchmarks memory allocations in concat operator
func BenchmarkConcatOperatorMemory(b *testing.B) {
	ev := &graft.Evaluator{
		Tree: map[interface{}]interface{}{
			"name": "test",
			"value": "data",
		},
		Here: func() *tree.Cursor {
			c, _ := tree.ParseCursor("$")
			return c
		}(),
	}
	
	// Create test expressions
	args := []*graft.Expr{
		{Type: graft.Literal, Literal: "prefix-"},
		{Type: graft.Reference, Reference: func() *tree.Cursor {
			c, _ := tree.ParseCursor("name")
			return c
		}()},
		{Type: graft.Literal, Literal: "-"},
		{Type: graft.Reference, Reference: func() *tree.Cursor {
			c, _ := tree.ParseCursor("value")
			return c
		}()},
		{Type: graft.Literal, Literal: "-suffix"},
	}
	
	op := operators.ConcatOperator{}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, _ = op.Run(ev, args)
	}
}

// BenchmarkJoinOperatorMemory benchmarks memory allocations in join operator
func BenchmarkJoinOperatorMemory(b *testing.B) {
	ev := &graft.Evaluator{
		Tree: map[interface{}]interface{}{
			"items": []interface{}{"one", "two", "three", "four", "five"},
		},
		Here: func() *tree.Cursor {
			c, _ := tree.ParseCursor("$")
			return c
		}(),
	}
	
	// Create test expressions
	args := []*graft.Expr{
		{Type: graft.Literal, Literal: ", "},
		{Type: graft.Reference, Reference: func() *tree.Cursor {
			c, _ := tree.ParseCursor("items")
			return c
		}()},
	}
	
	op := operators.JoinOperator{}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, _ = op.Run(ev, args)
	}
}

// BenchmarkTokenizerMemory benchmarks memory allocations in tokenizer
// TODO: Fix after refactoring
/*
func BenchmarkTokenizerMemory(b *testing.B) {
	expressions := []string{
		"(( grab meta.property.name ))",
		"(( concat \"prefix-\" name \"-\" value \"-suffix\" ))",
		"(( vault \"secret/path:key\" ))",
		"(( static_ips 0 1 2 ))",
		"(( calc \"2 + 2 * 3\" ))",
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, expr := range expressions {
			tokenizer := NewEnhancedTokenizer(expr)
			_ = tokenizer.Tokenize()
		}
	}
}
*/

// BenchmarkParserMemory benchmarks memory allocations in parser
// TODO: Fix after refactoring
/*
func BenchmarkParserMemory(b *testing.B) {
	expressions := []string{
		"grab meta.property.name",
		"concat \"prefix-\" name \"-\" value \"-suffix\"",
		"vault \"secret/path:key\"",
		"static_ips 0 1 2",
		"calc \"2 + 2 * 3\"",
	}
	
	registry := createTestRegistry()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, expr := range expressions {
			tokens := TokenizeExpression(expr)
			parser := parser.NewEnhancedParser(tokens, registry)
			_, _ = parser.Parse()
		}
	}
}
*/

// BenchmarkStringInterning benchmarks the effect of string interning
// TODO: Fix after refactoring
/*
func BenchmarkStringInterning(b *testing.B) {
	operators := []string{
		"grab", "concat", "vault", "static_ips", "calc",
		"join", "keys", "sort", "prune", "param",
	}
	
	b.Run("WithoutInterning", func(b *testing.B) {
		m := make(map[string]int)
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			for _, op := range operators {
				// Simulate creating new strings each time
				key := strings.ToLower(op)
				m[key] = i
			}
		}
	})
	
	b.Run("WithInterning", func(b *testing.B) {
		m := make(map[string]int)
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			for _, op := range operators {
				// Use interned strings
				key := InternString(op)
				m[key] = i
			}
		}
	})
}

// BenchmarkLargeDocumentParsing benchmarks parsing a large YAML-like structure
func BenchmarkLargeDocumentParsing(b *testing.B) {
	// Build a large expression with many operators
	var sb strings.Builder
	sb.WriteString("concat ")
	
	for i := 0; i < 100; i++ {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString("\"part")
		sb.WriteString(string(rune('0' + (i % 10))))
		sb.WriteString("\"")
	}
	
	expr := sb.String()
	registry := createTestRegistry()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		tokens := TokenizeExpression(expr)
		parser := parser.NewEnhancedParser(tokens, registry)
		_, _ = parser.Parse()
	}
}

*/
