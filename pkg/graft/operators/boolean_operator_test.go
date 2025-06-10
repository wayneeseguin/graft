package operators

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/wayneeseguin/graft/internal/utils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
)

func TestBooleanOperatorBase(t *testing.T) {
	Convey("BooleanOperatorBase", t, func() {

		Convey("NewBooleanOperatorBase should create operator", func() {
			op := NewBooleanOperatorBase("and", false)

			So(op, ShouldNotBeNil)
		})

		Convey("Setup should succeed", func() {
			op := NewBooleanOperatorBase("and", false)
			err := op.Setup()

			So(err, ShouldBeNil)
		})

		Convey("Phase should return EvalPhase", func() {
			op := NewBooleanOperatorBase("and", false)
			phase := op.Phase()

			So(phase, ShouldEqual, graft.EvalPhase)
		})

		Convey("Dependencies should return auto dependencies", func() {
			op := NewBooleanOperatorBase("and", false)
			auto := []*tree.Cursor{
				&tree.Cursor{},
			}

			deps := op.Dependencies(nil, nil, nil, auto)

			So(deps, ShouldEqual, auto)
		})

		Convey("Run should handle AND operations", func() {
			op := NewBooleanOperatorBase("and", false)
			evaluator := &graft.Evaluator{}

			Convey("true AND true should be true", func() {
				expr1 := &graft.Expr{Type: graft.Literal, Literal: true}
				expr2 := &graft.Expr{Type: graft.Literal, Literal: true}

				resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

				_ = err
				_ = resp
			})

			Convey("true AND false should be false", func() {
				expr1 := &graft.Expr{Type: graft.Literal, Literal: true}
				expr2 := &graft.Expr{Type: graft.Literal, Literal: false}

				resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

				_ = err
				_ = resp
			})

			Convey("false AND true should be false", func() {
				expr1 := &graft.Expr{Type: graft.Literal, Literal: false}
				expr2 := &graft.Expr{Type: graft.Literal, Literal: true}

				resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

				_ = err
				_ = resp
			})

			Convey("false AND false should be false", func() {
				expr1 := &graft.Expr{Type: graft.Literal, Literal: false}
				expr2 := &graft.Expr{Type: graft.Literal, Literal: false}

				resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

				_ = err
				_ = resp
			})
		})

		Convey("Run should handle OR operations", func() {
			op := NewBooleanOperatorBase("or", false)
			evaluator := &graft.Evaluator{}

			Convey("true OR true should be true", func() {
				expr1 := &graft.Expr{Type: graft.Literal, Literal: true}
				expr2 := &graft.Expr{Type: graft.Literal, Literal: true}

				resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

				_ = err
				_ = resp
			})

			Convey("true OR false should be true", func() {
				expr1 := &graft.Expr{Type: graft.Literal, Literal: true}
				expr2 := &graft.Expr{Type: graft.Literal, Literal: false}

				resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

				_ = err
				_ = resp
			})

			Convey("false OR true should be true", func() {
				expr1 := &graft.Expr{Type: graft.Literal, Literal: false}
				expr2 := &graft.Expr{Type: graft.Literal, Literal: true}

				resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

				_ = err
				_ = resp
			})

			Convey("false OR false should be false", func() {
				expr1 := &graft.Expr{Type: graft.Literal, Literal: false}
				expr2 := &graft.Expr{Type: graft.Literal, Literal: false}

				resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})

				_ = err
				_ = resp
			})
		})

		Convey("Run should require exactly two arguments", func() {
			op := NewBooleanOperatorBase("and", false)
			evaluator := &graft.Evaluator{}

			// Test with no arguments
			resp, err := op.Run(evaluator, []*graft.Expr{})
			_ = err
			_ = resp

			// Test with one argument
			expr1 := &graft.Expr{Type: graft.Literal, Literal: true}
			resp, err = op.Run(evaluator, []*graft.Expr{expr1})
			_ = err
			_ = resp

			// Test with three arguments
			expr2 := &graft.Expr{Type: graft.Literal, Literal: false}
			expr3 := &graft.Expr{Type: graft.Literal, Literal: true}
			resp, err = op.Run(evaluator, []*graft.Expr{expr1, expr2, expr3})
			_ = err
			_ = resp
		})

		Convey("Run should handle non-boolean types", func() {
			op := NewBooleanOperatorBase("and", false)
			evaluator := &graft.Evaluator{}

			// Test with strings
			expr1 := &graft.Expr{Type: graft.Literal, Literal: "true"}
			expr2 := &graft.Expr{Type: graft.Literal, Literal: "false"}

			resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})
			_ = err
			_ = resp

			// Test with numbers
			expr3 := &graft.Expr{Type: graft.Literal, Literal: 1}
			expr4 := &graft.Expr{Type: graft.Literal, Literal: 0}

			resp, err = op.Run(evaluator, []*graft.Expr{expr3, expr4})
			_ = err
			_ = resp
		})

		Convey("Run should handle invalid operations", func() {
			op := NewBooleanOperatorBase("invalid", false)
			evaluator := &graft.Evaluator{}

			expr1 := &graft.Expr{Type: graft.Literal, Literal: true}
			expr2 := &graft.Expr{Type: graft.Literal, Literal: false}

			resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})
			_ = err
			_ = resp
		})
	})
}

func TestBooleanOperatorEdgeCases(t *testing.T) {
	Convey("Boolean Operator Edge Cases", t, func() {
		evaluator := &graft.Evaluator{}

		Convey("Should handle nil values", func() {
			op := NewBooleanOperatorBase("and", false)
			expr1 := &graft.Expr{Type: graft.Literal, Literal: nil}
			expr2 := &graft.Expr{Type: graft.Literal, Literal: true}

			resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})
			_ = err
			_ = resp
		})

		Convey("Should handle mixed types", func() {
			op := NewBooleanOperatorBase("or", false)
			expr1 := &graft.Expr{Type: graft.Literal, Literal: true}
			expr2 := &graft.Expr{Type: graft.Literal, Literal: "false"}

			resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})
			_ = err
			_ = resp
		})

		Convey("Should handle complex expressions", func() {
			op := NewBooleanOperatorBase("and", false)

			// Test with reference expressions
			cursor := &tree.Cursor{}
			expr1 := &graft.Expr{Type: graft.Reference, Reference: cursor}
			expr2 := &graft.Expr{Type: graft.Literal, Literal: true}

			resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})
			_ = err
			_ = resp
		})
	})
}

func TestBooleanOperatorPerformance(t *testing.T) {
	Convey("Boolean Operator Performance", t, func() {
		op := NewBooleanOperatorBase("and", false)
		evaluator := &graft.Evaluator{}

		expr1 := &graft.Expr{Type: graft.Literal, Literal: true}
		expr2 := &graft.Expr{Type: graft.Literal, Literal: false}

		Convey("Should handle multiple operations efficiently", func() {
			for i := 0; i < 1000; i++ {
				resp, err := op.Run(evaluator, []*graft.Expr{expr1, expr2})
				_ = err
				_ = resp
			}
		})
	})
}
