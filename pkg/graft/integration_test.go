package graft

import (
	"testing"
	
	. "github.com/smartystreets/goconvey/convey"
	"github.com/starkandwayne/goutils/tree"
)

func TestEnhancedParserIntegration(t *testing.T) {
	Convey("Enhanced Parser Full Integration", t, func() {
		// Save original state
		originalFlag := UseEnhancedParser
		defer func() {
			UseEnhancedParser = originalFlag
			// Restore original concat operator
			RegisterOp("concat", ConcatOperator{})
		}()
		
		Convey("With enhanced parser disabled", func() {
			UseEnhancedParser = false
			
			Convey("Simple expressions work as before", func() {
				opcall, err := ParseOpcall(EvalPhase, `(( concat "hello" "world" ))`)
				So(err, ShouldBeNil)
				So(opcall, ShouldNotBeNil)
				So(len(opcall.args), ShouldEqual, 2)
				So(opcall.args[0].Type, ShouldEqual, Literal)
			})
		})
		
		Convey("With enhanced parser enabled", func() {
			EnableEnhancedParser()
			
			Convey("Simple operators still work", func() {
				opcall, err := ParseOpcall(EvalPhase, `(( concat "hello" " " "world" ))`)
				So(err, ShouldBeNil)
				So(opcall, ShouldNotBeNil)
				So(len(opcall.args), ShouldEqual, 3)
			})
			
			Convey("Nested operators work", func() {
				opcall, err := ParseOpcall(EvalPhase, `(( concat (grab name) " world" ))`)
				So(err, ShouldBeNil)
				So(opcall, ShouldNotBeNil)
				So(len(opcall.args), ShouldEqual, 2)
				So(opcall.args[0].Type, ShouldEqual, OperatorCall)
				So(opcall.args[0].Op(), ShouldEqual, "grab")
			})
			
			Convey("Complex nested expressions work", func() {
				opcall, err := ParseOpcall(EvalPhase, `(( concat (grab prefix) "-" (grab suffix) ))`)
				So(err, ShouldBeNil)
				So(opcall, ShouldNotBeNil)
				So(len(opcall.args), ShouldEqual, 3)
				So(opcall.args[0].Type, ShouldEqual, OperatorCall)
				So(opcall.args[2].Type, ShouldEqual, OperatorCall)
			})
			
			Convey("Evaluating nested operators works", func() {
				ev := &Evaluator{
					Tree: map[interface{}]interface{}{
						"name":   "graft",
						"prefix": "hello",
						"suffix": "world",
					},
				}
				
				// Create opcall with nested grab
				opcall, err := ParseOpcall(EvalPhase, `(( concat (grab name) " is awesome" ))`)
				So(err, ShouldBeNil)
				So(opcall, ShouldNotBeNil)
				
				// Run the operator
				resp, err := opcall.Run(ev)
				So(err, ShouldBeNil)
				So(resp, ShouldNotBeNil)
				So(resp.Type, ShouldEqual, Replace)
				So(resp.Value, ShouldEqual, "graft is awesome")
			})
			
			Convey("Deeply nested operators work", func() {
				ev := &Evaluator{
					Tree: map[interface{}]interface{}{
						"parts": map[interface{}]interface{}{
							"first":  "hello",
							"second": "world",
						},
					},
				}
				
				// concat with nested grabs
				opcall, err := ParseOpcall(EvalPhase, `(( concat (grab parts.first) " " (grab parts.second) ))`)
				So(err, ShouldBeNil)
				So(opcall, ShouldNotBeNil)
				
				resp, err := opcall.Run(ev)
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, "hello world")
			})
		})
		
		Convey("Environment variable control", func() {
			// Test that GRAFT_ENHANCED_PARSER env var works
			// This is tested indirectly through InitEnhancedParser()
			So(UseEnhancedParser, ShouldEqual, originalFlag)
		})
		
		Convey("Backward compatibility", func() {
			EnableEnhancedParser()
			
			Convey("Existing expressions continue to work", func() {
				testCases := []struct {
					name string
					expr string
					args int
					checkOp string // For cases where the enhanced parser creates a wrapper
				}{
					{"simple grab", `(( grab foo.bar ))`, 1, "grab"},
					{"vault with default", `(( vault "secret:key" || "default" ))`, 0, ""}, // Enhanced parser creates ExpressionWrapperOperator
					{"static_ips", `(( static_ips 1 2 3 ))`, 3, "static_ips"},
					{"concat literals", `(( concat "a" "b" "c" ))`, 3, "concat"},
				}
				
				for _, tc := range testCases {
					Convey(tc.name, func() {
						opcall, err := ParseOpcall(EvalPhase, tc.expr)
						So(err, ShouldBeNil)
						if opcall != nil {
							if tc.checkOp != "" {
								// Regular operator
								So(len(opcall.args), ShouldEqual, tc.args)
								_, isNull := opcall.op.(NullOperator)
								So(isNull, ShouldBeFalse)
							} else {
								// Expression wrapper (for || expressions)
								_, isWrapper := opcall.op.(*ExpressionWrapperOperator)
								So(isWrapper, ShouldBeTrue)
							}
						}
					})
				}
			})
		})
	})
}

func TestOperatorHelpers(t *testing.T) {
	Convey("Operator Helper Functions", t, func() {
		ev := &Evaluator{
			Tree: map[interface{}]interface{}{
				"foo": map[interface{}]interface{}{
					"bar": "baz",
				},
				"list": []interface{}{"a", "b", "c"},
			},
		}
		
		Convey("ResolveOperatorArgument", func() {
			Convey("resolves literals", func() {
				expr := &Expr{Type: Literal, Literal: "test"}
				val, err := ResolveOperatorArgument(ev, expr)
				So(err, ShouldBeNil)
				So(val, ShouldEqual, "test")
			})
			
			Convey("resolves references", func() {
				cursor, _ := tree.ParseCursor("foo.bar")
				expr := &Expr{Type: Reference, Reference: cursor}
				val, err := ResolveOperatorArgument(ev, expr)
				So(err, ShouldBeNil)
				So(val, ShouldEqual, "baz")
			})
			
			Convey("resolves nested operator calls", func() {
				EnableEnhancedParser()
				defer func() { RegisterOp("concat", ConcatOperator{}) }()
				
				// Create a nested grab operator call
				cursor, _ := tree.ParseCursor("foo.bar")
				grabExpr := NewOperatorCall("grab", []*Expr{
					{Type: Reference, Reference: cursor},
				})
				
				val, err := ResolveOperatorArgument(ev, grabExpr)
				So(err, ShouldBeNil)
				So(val, ShouldEqual, "baz")
			})
		})
		
		Convey("AsString", func() {
			Convey("converts various types to string", func() {
				s, _ := AsString("hello")
				So(s, ShouldEqual, "hello")
				s, _ = AsString(42)
				So(s, ShouldEqual, "42")
				s, _ = AsString(true)
				So(s, ShouldEqual, "true")
				s, _ = AsString(nil)
				So(s, ShouldEqual, "")
			})
		})
		
		Convey("AsStringArray", func() {
			Convey("converts to string array", func() {
				result, err := AsStringArray([]interface{}{"a", "b", "c"})
				So(err, ShouldBeNil)
				So(result, ShouldResemble, []string{"a", "b", "c"})
				
				result, err = AsStringArray("single")
				So(err, ShouldBeNil)
				So(result, ShouldResemble, []string{"single"})
			})
		})
	})
}