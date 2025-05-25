package spruce

import (
	"testing"
	
	"github.com/starkandwayne/goutils/tree"
	. "github.com/smartystreets/goconvey/convey"
)

func TestExprErrorFormatting(t *testing.T) {
	Convey("Expression Errors", t, func() {
		Convey("should format syntax errors with source context", func() {
			err := NewSyntaxError("expected closing parenthesis", Position{Line: 2, Column: 15}).
				WithSource("value: (( grab meta.foo\nother: value").
				WithContext("parentheses must be balanced")
			
			msg := err.Error()
			So(msg, ShouldContainSubstring, "Syntax Error")
			So(msg, ShouldContainSubstring, "2:15")
			So(msg, ShouldContainSubstring, "expected closing parenthesis")
			So(msg, ShouldContainSubstring, "parentheses must be balanced")
			So(msg, ShouldContainSubstring, "^")
		})
		
		Convey("should format type errors", func() {
			err := NewTypeError("cannot add string to number", Position{Line: 1, Column: 10})
			
			msg := err.Error()
			So(msg, ShouldContainSubstring, "Type Error")
			So(msg, ShouldContainSubstring, "1:10")
			So(msg, ShouldContainSubstring, "cannot add string to number")
		})
		
		Convey("should format reference errors", func() {
			err := NewReferenceError("undefined reference 'meta.invalid'", Position{Line: 3, Column: 5})
			
			msg := err.Error()
			So(msg, ShouldContainSubstring, "Reference Error")
			So(msg, ShouldContainSubstring, "3:5")
			So(msg, ShouldContainSubstring, "undefined reference 'meta.invalid'")
		})
		
		Convey("should wrap nested errors", func() {
			inner := NewSyntaxError("invalid operator", Position{Line: 2, Column: 10})
			outer := NewEvaluationError("failed to evaluate expression", Position{Line: 1, Column: 5}).
				WithNested(inner)
			
			msg := outer.Error()
			So(msg, ShouldContainSubstring, "Evaluation Error")
			So(msg, ShouldContainSubstring, "failed to evaluate expression")
			So(msg, ShouldContainSubstring, "caused by:")
			So(msg, ShouldContainSubstring, "invalid operator")
		})
		
		Convey("should collect multiple errors", func() {
			list := &ExprErrorList{}
			list.Add(NewSyntaxError("first error", Position{Line: 1, Column: 5}))
			list.Add(NewSyntaxError("second error", Position{Line: 3, Column: 10}))
			
			msg := list.Error()
			So(msg, ShouldContainSubstring, "Found 2 errors")
			So(msg, ShouldContainSubstring, "[1]")
			So(msg, ShouldContainSubstring, "[2]")
			So(msg, ShouldContainSubstring, "first error")
			So(msg, ShouldContainSubstring, "second error")
		})
	})
}

func TestErrorRecovery(t *testing.T) {
	Convey("Error Recovery", t, func() {
		Convey("should stop on first error when configured", func() {
			ctx := NewErrorRecoveryContext(10)
			ctx.StopOnFirst = true
			
			shouldContinue := ctx.RecordError(NewSyntaxError("error 1", Position{}))
			So(shouldContinue, ShouldBeFalse)
			So(len(ctx.Errors.Errors), ShouldEqual, 1)
		})
		
		Convey("should collect multiple errors up to limit", func() {
			ctx := NewErrorRecoveryContext(3)
			
			So(ctx.RecordError(NewSyntaxError("error 1", Position{})), ShouldBeTrue)
			So(ctx.RecordError(NewSyntaxError("error 2", Position{})), ShouldBeTrue)
			So(ctx.RecordError(NewSyntaxError("error 3", Position{})), ShouldBeFalse)
			
			So(len(ctx.Errors.Errors), ShouldEqual, 3)
		})
	})
}

func TestEnhancedParserErrors(t *testing.T) {
	Convey("Enhanced Parser Error Messages", t, func() {
		UseEnhancedParser = true
		defer func() { UseEnhancedParser = true }()
		
		Convey("should provide helpful error for unclosed parentheses", func() {
			src := "(( grab meta.foo"
			_, err := ParseOpcallEnhanced(EvalPhase, src)
			So(err, ShouldNotBeNil)
			
			if exprErr, ok := err.(*ExprError); ok {
				So(exprErr.Type, ShouldEqual, SyntaxError)
				So(exprErr.Message, ShouldContainSubstring, "parenthesis")
			}
		})
		
		Convey("should provide context for operator errors", func() {
			src := "(( 5 + + 3 ))"
			_, err := ParseOpcallEnhanced(EvalPhase, src)
			So(err, ShouldNotBeNil)
			
			if exprErr, ok := err.(*ExprError); ok {
				So(exprErr.Type, ShouldEqual, SyntaxError)
				So(exprErr.Context, ShouldContainSubstring, "between operands")
			}
		})
		
		Convey("should track position through nested expressions", func() {
			
			src := "(( concat \"a\" (grab meta.invalid) \"b\" ))"
			opcall, err := ParseOpcallEnhanced(EvalPhase, src)
			So(err, ShouldBeNil)
			So(opcall, ShouldNotBeNil)
			
			// The error would come during evaluation when meta.invalid is resolved
			args := opcall.args
			So(len(args), ShouldEqual, 3)
			So(args[1], ShouldNotBeNil)
			So(args[1].Type, ShouldEqual, OperatorCall)
			So(args[1].Pos.Column, ShouldBeGreaterThan, 0)
		})
		
		Convey("should provide helpful error for invalid ternary syntax", func() {
			src := "(( condition ? true_val ))"
			_, err := ParseOpcallEnhanced(EvalPhase, src)
			So(err, ShouldNotBeNil)
			
			if exprErr, ok := err.(*ExprError); ok {
				So(exprErr.Type, ShouldEqual, SyntaxError)
				So(exprErr.Message, ShouldContainSubstring, "':'")
				So(exprErr.Context, ShouldContainSubstring, "ternary")
			}
		})
	})
}

func TestNestedExpressionErrors(t *testing.T) {
	Convey("Nested Expression Error Tracking", t, func() {
		Convey("should track errors through operator evaluation", func() {
			ev := &Evaluator{}
			
			// Create a nested expression: concat("a", grab(invalid.ref), "b")
			ref, _ := tree.ParseCursor("invalid.ref")
			innerExpr := NewOperatorCallWithPos("grab", []*Expr{
				{Type: Reference, Reference: ref, Pos: Position{Line: 1, Column: 20}},
			}, Position{Line: 1, Column: 15})
			
			outerExpr := NewOperatorCallWithPos("concat", []*Expr{
				{Type: Literal, Literal: "a", Pos: Position{Line: 1, Column: 10}},
				innerExpr,
				{Type: Literal, Literal: "b", Pos: Position{Line: 1, Column: 35}},
			}, Position{Line: 1, Column: 5})
			
			_, err := EvaluateExpr(outerExpr, ev)
			So(err, ShouldNotBeNil)
			
			// Check that we have position information
			if exprErr, ok := err.(*ExprError); ok {
				So(exprErr.Position.Column, ShouldBeGreaterThan, 0)
			}
		})
	})
}


