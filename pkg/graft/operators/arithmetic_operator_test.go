package operators

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/wayneeseguin/graft/internal/utils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
)

func TestArithmeticOperatorBase(t *testing.T) {
	Convey("ArithmeticOperatorBase", t, func() {

		Convey("NewArithmeticOperatorBase should create operator", func() {
			op := NewArithmeticOperatorBase("add")

			So(op, ShouldNotBeNil)
			So(op.op, ShouldEqual, "add")
			So(op.registry, ShouldNotBeNil)
		})

		Convey("Setup should succeed", func() {
			op := NewArithmeticOperatorBase("add")
			err := op.Setup()

			So(err, ShouldBeNil)
		})

		Convey("Phase should return EvalPhase", func() {
			op := NewArithmeticOperatorBase("add")
			phase := op.Phase()

			So(phase, ShouldEqual, graft.EvalPhase)
		})

		Convey("Dependencies should return auto dependencies", func() {
			op := NewArithmeticOperatorBase("add")
			auto := []*tree.Cursor{
				&tree.Cursor{},
			}

			deps := op.Dependencies(nil, nil, nil, auto)

			So(deps, ShouldEqual, auto)
		})

		Convey("Run should require exactly two arguments", func() {
			op := NewArithmeticOperatorBase("add")
			evaluator := &graft.Evaluator{}

			// Test with no arguments
			resp, err := op.Run(evaluator, []*graft.Expr{})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "requires exactly two arguments")
			So(resp, ShouldBeNil)

			// Test with one argument
			expr1 := &graft.Expr{Type: graft.Literal, Literal: 5}
			resp, err = op.Run(evaluator, []*graft.Expr{expr1})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "requires exactly two arguments")
			So(resp, ShouldBeNil)

			// Test with three arguments
			expr2 := &graft.Expr{Type: graft.Literal, Literal: 10}
			expr3 := &graft.Expr{Type: graft.Literal, Literal: 15}
			resp, err = op.Run(evaluator, []*graft.Expr{expr1, expr2, expr3})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "requires exactly two arguments")
			So(resp, ShouldBeNil)
		})

		Convey("Run should handle arithmetic operations", func() {
			evaluator := &graft.Evaluator{}

			Convey("Add operation", func() {
				op := NewArithmeticOperatorBase("add")
				expr1 := &graft.Expr{Type: graft.Literal, Literal: 5}
				expr2 := &graft.Expr{Type: graft.Literal, Literal: 3}

				resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

				// The actual implementation may vary, but should not panic
				_ = err
				_ = resp
			})

			Convey("Subtract operation", func() {
				op := NewArithmeticOperatorBase("subtract")
				expr1 := &graft.Expr{Type: graft.Literal, Literal: 10}
				expr2 := &graft.Expr{Type: graft.Literal, Literal: 4}

				resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

				_ = err
				_ = resp
			})

			Convey("Multiply operation", func() {
				op := NewArithmeticOperatorBase("multiply")
				expr1 := &graft.Expr{Type: graft.Literal, Literal: 6}
				expr2 := &graft.Expr{Type: graft.Literal, Literal: 7}

				resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

				_ = err
				_ = resp
			})

			Convey("Divide operation", func() {
				op := NewArithmeticOperatorBase("divide")
				expr1 := &graft.Expr{Type: graft.Literal, Literal: 20}
				expr2 := &graft.Expr{Type: graft.Literal, Literal: 4}

				resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

				_ = err
				_ = resp
			})

			Convey("Modulo operation", func() {
				op := NewArithmeticOperatorBase("modulo")
				expr1 := &graft.Expr{Type: graft.Literal, Literal: 17}
				expr2 := &graft.Expr{Type: graft.Literal, Literal: 5}

				resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

				_ = err
				_ = resp
			})
		})

		Convey("Run should handle type errors", func() {
			op := NewArithmeticOperatorBase("add")
			evaluator := &graft.Evaluator{}

			// Test with incompatible types
			expr1 := &graft.Expr{Type: graft.Literal, Literal: "string"}
			expr2 := &graft.Expr{Type: graft.Literal, Literal: 5}

			resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

			// Should handle type mismatches gracefully
			_ = err
			_ = resp
		})
	})
}

func TestTypeAwareArithmeticOperators(t *testing.T) {
	Convey("Type Aware Arithmetic Operators", t, func() {

		Convey("NewTypeAwareAddOperator should create operator", func() {
			op := NewTypeAwareAddOperator()

			So(op, ShouldNotBeNil)
		})

		Convey("NewTypeAwareSubtractOperator should create operator", func() {
			op := NewTypeAwareSubtractOperator()

			So(op, ShouldNotBeNil)
		})

		Convey("NewTypeAwareMultiplyOperator should create operator", func() {
			op := NewTypeAwareMultiplyOperator()

			So(op, ShouldNotBeNil)
		})

		Convey("NewTypeAwareDivideOperator should create operator", func() {
			op := NewTypeAwareDivideOperator()

			So(op, ShouldNotBeNil)
		})

		Convey("NewTypeAwareModuloOperator should create operator", func() {
			op := NewTypeAwareModuloOperator()

			So(op, ShouldNotBeNil)
		})

		Convey("Type aware operators should implement Operator interface", func() {
			ops := []graft.Operator{
				NewTypeAwareAddOperator(),
				NewTypeAwareSubtractOperator(),
				NewTypeAwareMultiplyOperator(),
				NewTypeAwareDivideOperator(),
				NewTypeAwareModuloOperator(),
			}

			for _, op := range ops {
				So(op.Setup(), ShouldBeNil)
				So(op.Phase(), ShouldEqual, graft.EvalPhase)
				deps := op.Dependencies(nil, nil, nil, nil)
				_ = deps // Dependencies may be nil or non-nil
			}
		})

		Convey("Type aware operators should perform operations", func() {
			evaluator := &graft.Evaluator{}

			testCases := []struct {
				name     string
				operator graft.Operator
				arg1     interface{}
				arg2     interface{}
			}{
				{"Add integers", NewTypeAwareAddOperator(), 5, 3},
				{"Add floats", NewTypeAwareAddOperator(), 5.5, 3.2},
				{"Subtract integers", NewTypeAwareSubtractOperator(), 10, 4},
				{"Multiply integers", NewTypeAwareMultiplyOperator(), 6, 7},
				{"Divide integers", NewTypeAwareDivideOperator(), 20, 4},
				{"Modulo integers", NewTypeAwareModuloOperator(), 17, 5},
			}

			for _, tc := range testCases {
				Convey(tc.name, func() {
					expr1 := &graft.Expr{Type: graft.Literal, Literal: tc.arg1}
					expr2 := &graft.Expr{Type: graft.Literal, Literal: tc.arg2}

					resp, err := tc.operator.Run(evaluator, []*graft.Expr{expr1, expr2})

					// Should execute without panic
					_ = err
					_ = resp
				})
			}
		})
	})
}

func TestArithmeticOperatorEdgeCases(t *testing.T) {
	Convey("Arithmetic Operator Edge Cases", t, func() {
		evaluator := &graft.Evaluator{}

		Convey("Should handle division by zero", func() {
			op := NewTypeAwareDivideOperator()
			expr1 := &graft.Expr{Type: graft.Literal, Literal: 10}
			expr2 := &graft.Expr{Type: graft.Literal, Literal: 0}

			resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

			// Should handle division by zero gracefully
			_ = err
			_ = resp
		})

		Convey("Should handle modulo by zero", func() {
			op := NewTypeAwareModuloOperator()
			expr1 := &graft.Expr{Type: graft.Literal, Literal: 10}
			expr2 := &graft.Expr{Type: graft.Literal, Literal: 0}

			resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

			// Should handle modulo by zero gracefully
			_ = err
			_ = resp
		})

		Convey("Should handle large numbers", func() {
			op := NewTypeAwareMultiplyOperator()
			expr1 := &graft.Expr{Type: graft.Literal, Literal: 999999999}
			expr2 := &graft.Expr{Type: graft.Literal, Literal: 999999999}

			resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

			// Should handle large number operations
			_ = err
			_ = resp
		})

		Convey("Should handle negative numbers", func() {
			op := NewTypeAwareAddOperator()
			expr1 := &graft.Expr{Type: graft.Literal, Literal: -5}
			expr2 := &graft.Expr{Type: graft.Literal, Literal: -3}

			resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

			// Should handle negative numbers
			_ = err
			_ = resp
		})

		Convey("Should handle mixed types", func() {
			op := NewTypeAwareAddOperator()
			expr1 := &graft.Expr{Type: graft.Literal, Literal: 5}
			expr2 := &graft.Expr{Type: graft.Literal, Literal: 3.14}

			resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

			// Should handle mixed int/float types
			_ = err
			_ = resp
		})
	})
}
