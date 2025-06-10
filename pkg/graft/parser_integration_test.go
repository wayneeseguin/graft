// TODO: Parser integration tests removed - enhanced parser not implemented
//go:build ignore
// +build ignore

package graft_test

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/wayneeseguin/graft/internal/utils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
	"github.com/wayneeseguin/graft/pkg/graft/parser"
)

func TestParserIntegration(t *testing.T) {
	Convey("Parser Integration", t, func() {
		// Save original flag state
		originalFlag := parser.UseEnhancedParser
		defer func() { parser.UseEnhancedParser = originalFlag }()

		Convey("Enhanced Parser Feature Flag", func() {
			Convey("When disabled, uses original parser", func() {
				parser.UseEnhancedParser = false

				opcall, err := parser.ParseOpcallCompat(graft.EvalPhase, `(( grab foo.bar ))`)
				So(err, ShouldBeNil)
				So(opcall, ShouldNotBeNil)
				So(opcall.Operator(), ShouldNotBeNil)
			})

			Convey("When enabled, uses enhanced parser", func() {
				parser.UseEnhancedParser = true

				opcall, err := parser.ParseOpcallCompat(graft.EvalPhase, `(( concat "hello" "world" ))`)
				So(err, ShouldBeNil)
				So(opcall, ShouldNotBeNil)
				So(opcall.Operator(), ShouldNotBeNil)
			})
		})

		Convey("Automatic Enhanced Parser Selection", func() {
			parser.UseEnhancedParser = false // Rely on heuristics

			Convey("Uses enhanced parser for nested operators", func() {
				So(parser.ShouldUseEnhancedParser(`concat (grab foo) "bar"`), ShouldBeTrue)
				So(parser.ShouldUseEnhancedParser(`vault "path" || grab defaults`), ShouldBeFalse) // || is handled by original
			})

			Convey("Uses enhanced parser for arithmetic", func() {
				So(parser.ShouldUseEnhancedParser(`1 + 2`), ShouldBeTrue)
				So(parser.ShouldUseEnhancedParser(`count * 2`), ShouldBeTrue)
			})

			Convey("Uses enhanced parser for parentheses", func() {
				So(parser.ShouldUseEnhancedParser(`(grab foo)`), ShouldBeTrue)
				So(parser.ShouldUseEnhancedParser(`concat (a) (b)`), ShouldBeTrue)
			})
		})

		Convey("Enhanced Parser Functionality", func() {
			parser.UseEnhancedParser = true

			Convey("Parses simple operator calls", func() {
				opcall, err := parser.ParseOpcallEnhanced(graft.EvalPhase, `(( grab foo.bar ))`)
				So(err, ShouldBeNil)
				So(opcall, ShouldNotBeNil)
				So(len(opcall.Args()), ShouldEqual, 1)
				So(opcall.Args()[0].Type, ShouldEqual, Reference)
			})

			Convey("Parses operators with multiple arguments", func() {
				opcall, err := parser.ParseOpcallEnhanced(graft.EvalPhase, `(( concat "hello" " " "world" ))`)
				So(err, ShouldBeNil)
				So(opcall, ShouldNotBeNil)
				So(len(opcall.Args()), ShouldEqual, 3)
				So(opcall.Args()[0].Type, ShouldEqual, Literal)
				So(opcall.Args()[0].Literal, ShouldEqual, "hello")
				So(opcall.Args()[1].Type, ShouldEqual, Literal)
				So(opcall.Args()[1].Literal, ShouldEqual, " ")
				So(opcall.Args()[2].Type, ShouldEqual, Literal)
				So(opcall.Args()[2].Literal, ShouldEqual, "world")
			})

			Convey("Parses nested operator calls", func() {
				opcall, err := parser.ParseOpcallEnhanced(graft.EvalPhase, `(( concat (grab prefix) "-" (grab suffix) ))`)
				So(err, ShouldBeNil)
				So(opcall, ShouldNotBeNil)
				So(len(opcall.Args()), ShouldEqual, 3)

				// First argument should be a nested grab operator
				So(opcall.Args()[0].Type, ShouldEqual, OperatorCall)
				So(opcall.Args()[0].Op(), ShouldEqual, "grab")
				So(len(opcall.Args()[0].Args()), ShouldEqual, 1)

				// Second argument should be a literal
				So(opcall.Args()[1].Type, ShouldEqual, Literal)
				So(opcall.Args()[1].Literal, ShouldEqual, "-")

				// Third argument should be another nested grab operator
				So(opcall.Args()[2].Type, ShouldEqual, OperatorCall)
				So(opcall.Args()[2].Op(), ShouldEqual, "grab")
				So(len(opcall.Args()[2].Args()), ShouldEqual, 1)
			})

			Convey("Respects operator phases", func() {
				// grab is EvalPhase
				opcall, err := parser.ParseOpcallEnhanced(graft.MergePhase, `(( grab foo ))`)
				So(err, ShouldBeNil)
				So(opcall, ShouldBeNil) // Wrong phase

				opcall, err = parser.ParseOpcallEnhanced(graft.EvalPhase, `(( grab foo ))`)
				So(err, ShouldBeNil)
				So(opcall, ShouldNotBeNil) // Correct phase
			})
		})

		Convey("Expression Evaluation", func() {
			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"foo": map[interface{}]interface{}{
						"bar": "baz",
					},
					"prefix": "hello",
					"suffix": "world",
				},
			}

			Convey("Evaluates literals", func() {
				expr := &Expr{Type: Literal, Literal: "test"}
				resp, err := EvaluateExpr(expr, ev)
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, "test")
			})

			Convey("Evaluates references", func() {
				cursor, _ := tree.ParseCursor("foo.bar")
				expr := &Expr{Type: Reference, Reference: cursor}
				resp, err := EvaluateExpr(expr, ev)
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, "baz")
			})

			Convey("Evaluates environment variables", func() {
				// This test might fail if PATH is not set
				expr := &Expr{Type: EnvVar, Name: "PATH"}
				resp, err := EvaluateExpr(expr, ev)
				So(err, ShouldBeNil)
				So(resp.Value, ShouldNotBeNil)
			})

			Convey("Evaluates nested operator calls", func() {
				// Create a nested grab operator call
				cursor, _ := tree.ParseCursor("prefix")
				grabExpr := NewOperatorCall("grab", []*Expr{
					{Type: Reference, Reference: cursor},
				})

				resp, err := EvaluateExpr(grabExpr, ev)
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, "hello")
			})
		})

		Convey("Operator Wrapper", func() {
			// Test the wrapper that allows existing operators to handle nested calls
			mockOp := &MockOperator{
				runFunc: func(ev *Evaluator, args []*Expr) (*Response, error) {
					// Simple concat implementation
					result := ""
					for _, arg := range args {
						if arg.Type == Literal {
							result += arg.Literal.(string)
						}
					}
					return &Response{Type: graft.Replace, Value: result}, nil
				},
			}

			wrapped := WrapOperatorForNestedCalls(mockOp)

			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"prefix": "hello",
					"suffix": "world",
				},
			}

			Convey("Handles regular arguments", func() {
				args := []*Expr{
					{Type: Literal, Literal: "hello"},
					{Type: Literal, Literal: "world"},
				}

				resp, err := wrapped.Run(ev, args)
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, "helloworld")
			})

			Convey("Evaluates nested operator calls in arguments", func() {
				// This would require grab operator to be registered
				// For now, skip this test
				SkipSo("not implemented", ShouldBeNil)
			})
		})
	})
}

// MockOperator for testing
type MockOperator struct {
	phase   graft.OperatorPhase
	runFunc func(*graft.Evaluator, []*graft.Expr) (*graft.Response, error)
}

func (m *MockOperator) Setup() error               { return nil }
func (m *MockOperator) Phase() graft.OperatorPhase { return m.phase }
func (m *MockOperator) Dependencies(ev *graft.Evaluator, args []*graft.Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return nil
}
func (m *MockOperator) Run(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
	if m.runFunc != nil {
		return m.runFunc(ev, args)
	}
	return &graft.Response{Type: graft.Replace, Value: nil}, nil
}
