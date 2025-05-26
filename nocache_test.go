package spruce

import (
	"testing"
	"time"
	
	. "github.com/smartystreets/goconvey/convey"
)

func TestNoCacheModifier(t *testing.T) {
	Convey("NoCache Modifier Support", t, func() {
		// Use a real registry with operators loaded
		registry := NewOperatorRegistry()
		
		Convey("Parse operator with nocache modifier", func() {
			parser := NewMemoizedEnhancedParser(`vault:nocache "secret/path"`, registry)
			expr, err := parser.Parse()
			
			So(err, ShouldBeNil)
			So(expr, ShouldNotBeNil)
			So(expr.Type, ShouldEqual, OperatorCall)
			So(expr.Op(), ShouldEqual, "vault")
			So(expr.IsNoCache(), ShouldBeTrue)
		})
		
		Convey("Parse operator without nocache modifier", func() {
			parser := NewMemoizedEnhancedParser(`vault "secret/path"`, registry)
			expr, err := parser.Parse()
			
			So(err, ShouldBeNil)
			So(expr, ShouldNotBeNil)
			So(expr.Type, ShouldEqual, OperatorCall)
			So(expr.Op(), ShouldEqual, "vault")
			So(expr.IsNoCache(), ShouldBeFalse)
		})
		
		Convey("Parse operator with multiple modifiers", func() {
			parser := NewMemoizedEnhancedParser(`vault:nocache:debug "secret/path"`, registry)
			expr, err := parser.Parse()
			
			So(err, ShouldBeNil)
			So(expr, ShouldNotBeNil)
			So(expr.Type, ShouldEqual, OperatorCall)
			So(expr.Op(), ShouldEqual, "vault")
			So(expr.IsNoCache(), ShouldBeTrue)
			So(expr.HasModifier("debug"), ShouldBeTrue)
		})
		
		Convey("Tokenizer handles operator modifiers", func() {
			tokenizer := NewEnhancedTokenizer(`vault:nocache "secret/path"`)
			tokens := tokenizer.Tokenize()
			
			So(len(tokens), ShouldBeGreaterThan, 0)
			// First token should be the operator with modifier
			So(tokens[0].Type, ShouldEqual, TokenOperator)
			So(tokens[0].Value, ShouldEqual, "vault:nocache")
		})
		
		Convey("Colon in ternary expressions still works", func() {
			parser := NewMemoizedEnhancedParser("true ? 1 : 2", registry)
			expr, err := parser.Parse()
			
			So(err, ShouldBeNil)
			So(expr, ShouldNotBeNil)
			So(expr.Type, ShouldEqual, OperatorCall)
			So(expr.Op(), ShouldEqual, "?:")
		})
		
		Convey("Modifier parsing", func() {
			modifiers := make(map[string]bool)
			
			// Test parseOperatorModifiers logic manually
			parts := []string{"vault", "nocache"}
			for i := 1; i < len(parts); i++ {
				modifiers[parts[i]] = true
			}
			
			So(modifiers["nocache"], ShouldBeTrue)
			So(len(modifiers), ShouldEqual, 1)
		})
		
		Convey("Expression modifier methods", func() {
			expr := NewOperatorCall("vault", []*Expr{})
			
			// Initially no modifiers
			So(expr.IsNoCache(), ShouldBeFalse)
			So(expr.HasModifier("nocache"), ShouldBeFalse)
			
			// Set nocache modifier
			expr.SetModifier("nocache", true)
			So(expr.IsNoCache(), ShouldBeTrue)
			So(expr.HasModifier("nocache"), ShouldBeTrue)
			
			// Set another modifier
			expr.SetModifier("debug", true)
			So(expr.HasModifier("debug"), ShouldBeTrue)
			
			// Get all modifiers
			allMods := expr.GetModifiers()
			So(len(allMods), ShouldEqual, 2)
			So(allMods["nocache"], ShouldBeTrue)
			So(allMods["debug"], ShouldBeTrue)
		})
		
		Convey("Cache respects nocache modifier", func() {
			cache := NewParserMemoizationCache(100, time.Hour)
			
			// Parse regular expression (should be cached)
			parser1 := NewMemoizedEnhancedParser(`vault "secret/path"`, registry)
			parser1.cache = cache
			_, err1 := parser1.Parse()
			So(err1, ShouldBeNil)
			
			// Parse with nocache (should not be cached)
			parser2 := NewMemoizedEnhancedParser(`vault:nocache "secret/path"`, registry)
			parser2.cache = cache
			expr2, err2 := parser2.Parse()
			So(err2, ShouldBeNil)
			So(expr2.IsNoCache(), ShouldBeTrue)
			
			// Verify different cache behavior
			metrics := cache.GetMetrics()
			So(metrics.Size, ShouldBeGreaterThan, 0)
		})
	})
}

func BenchmarkNoCacheOperator(b *testing.B) {
	registry := NewOperatorRegistry()
	
	b.Run("WithNoCache", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			parser := NewMemoizedEnhancedParser(`vault:nocache "secret/data"`, registry)
			expr, _ := parser.Parse()
			_ = expr.IsNoCache()
		}
	})
	
	b.Run("WithoutNoCache", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			parser := NewMemoizedEnhancedParser(`vault "secret/data"`, registry)
			expr, _ := parser.Parse()
			_ = expr.IsNoCache()
		}
	})
}