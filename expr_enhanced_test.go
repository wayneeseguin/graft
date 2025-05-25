package spruce

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEnhancedExpressionModel(t *testing.T) {
	Convey("Enhanced Expression Model", t, func() {
		Convey("Operator Expression Creation", func() {
			expr := NewOperatorExpr("grab", []*Expr{})
			So(expr, ShouldNotBeNil)
			So(expr.Type, ShouldEqual, OperatorCall)
			So(expr.Name, ShouldEqual, "grab")
			So(expr.IsOperator(), ShouldBeTrue)
			So(expr.IsOperatorNamed("grab"), ShouldBeTrue)
			So(expr.IsOperatorNamed("vault"), ShouldBeFalse)
			So(expr.GetOperatorName(), ShouldEqual, "grab")
		})

		Convey("Operator Info Registry", func() {
			Convey("should contain vault operator", func() {
				info, ok := GetOperatorInfo("vault")
				So(ok, ShouldBeTrue)
				So(info.Name, ShouldEqual, "vault")
				So(info.Precedence, ShouldEqual, PrecedenceCall)
				So(info.Phase, ShouldEqual, EvalPhase)
				So(info.MinArgs, ShouldEqual, 1)
				So(info.MaxArgs, ShouldEqual, -1)
			})

			Convey("should contain grab operator", func() {
				info, ok := GetOperatorInfo("grab")
				So(ok, ShouldBeTrue)
				So(info.Name, ShouldEqual, "grab")
				So(info.MinArgs, ShouldEqual, 1)
				So(info.MaxArgs, ShouldEqual, 1)
			})

			Convey("should return false for unknown operator", func() {
				_, ok := GetOperatorInfo("unknown")
				So(ok, ShouldBeFalse)
			})
		})

		Convey("Operator Validation", func() {
			Convey("should validate vault args", func() {
				err := ValidateOperatorArgs("vault", 1)
				So(err, ShouldBeNil)
				
				err = ValidateOperatorArgs("vault", 3)
				So(err, ShouldBeNil)
				
				err = ValidateOperatorArgs("vault", 0)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "at least 1 argument")
			})

			Convey("should validate grab args", func() {
				err := ValidateOperatorArgs("grab", 1)
				So(err, ShouldBeNil)
				
				err = ValidateOperatorArgs("grab", 2)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "at most 1 argument")
			})

			Convey("should validate vault-try args", func() {
				err := ValidateOperatorArgs("vault-try", 1)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "at least 2 arguments")
				
				err = ValidateOperatorArgs("vault-try", 2)
				So(err, ShouldBeNil)
				
				err = ValidateOperatorArgs("vault-try", 5)
				So(err, ShouldBeNil)
			})

			Convey("should reject unknown operator", func() {
				err := ValidateOperatorArgs("unknown", 1)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "unknown operator")
			})
		})

		Convey("IsRegisteredOperator", func() {
			So(IsRegisteredOperator("vault"), ShouldBeTrue)
			So(IsRegisteredOperator("grab"), ShouldBeTrue)
			So(IsRegisteredOperator("concat"), ShouldBeTrue)
			So(IsRegisteredOperator("unknown"), ShouldBeFalse)
		})

		Convey("Precedence Levels", func() {
			// Verify precedence ordering
			So(PrecedenceLowest, ShouldBeLessThan, PrecedenceOr)
			So(PrecedenceOr, ShouldBeLessThan, PrecedenceAnd)
			So(PrecedenceAnd, ShouldBeLessThan, PrecedenceEquality)
			So(PrecedenceEquality, ShouldBeLessThan, PrecedenceComparison)
			So(PrecedenceComparison, ShouldBeLessThan, PrecedenceAdditive)
			So(PrecedenceAdditive, ShouldBeLessThan, PrecedenceMultiplicative)
			So(PrecedenceMultiplicative, ShouldBeLessThan, PrecedenceUnary)
			So(PrecedenceUnary, ShouldBeLessThan, PrecedenceCall)
			So(PrecedenceCall, ShouldBeLessThan, PrecedenceHighest)
		})
	})
}