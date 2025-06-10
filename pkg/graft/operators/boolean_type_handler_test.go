package operators

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBooleanTypeHandler(t *testing.T) {
	Convey("BooleanTypeHandler", t, func() {
		handler := NewBooleanTypeHandler()

		Convey("supports correct type combinations", func() {
			So(handler.CanHandle(TypeBool, TypeBool), ShouldBeTrue)
			So(handler.CanHandle(TypeBool, TypeInt), ShouldBeTrue)
			So(handler.CanHandle(TypeBool, TypeString), ShouldBeTrue)
			So(handler.CanHandle(TypeBool, TypeNull), ShouldBeTrue)
			So(handler.CanHandle(TypeInt, TypeBool), ShouldBeTrue)

			// Should NOT handle types not explicitly supported
			So(handler.CanHandle(TypeBool, TypeFloat), ShouldBeFalse)
			So(handler.CanHandle(TypeMap, TypeBool), ShouldBeFalse)
		})

		Convey("Addition (logical OR)", func() {
			Convey("bool + bool", func() {
				result, err := handler.Add(true, true)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, true)

				result, err = handler.Add(true, false)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, true)

				result, err = handler.Add(false, true)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, true)

				result, err = handler.Add(false, false)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, false)
			})
		})

		Convey("Multiplication (logical AND)", func() {
			Convey("bool * bool", func() {
				result, err := handler.Multiply(true, true)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, true)

				result, err = handler.Multiply(true, false)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, false)

				result, err = handler.Multiply(false, true)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, false)

				result, err = handler.Multiply(false, false)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, false)
			})
		})

		Convey("Unsupported operations", func() {
			Convey("subtraction", func() {
				result, err := handler.Subtract(true, false)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "subtract operation not supported")
				So(result, ShouldBeNil)
			})

			Convey("division", func() {
				result, err := handler.Divide(true, false)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "divide operation not supported")
				So(result, ShouldBeNil)
			})

			Convey("modulo", func() {
				result, err := handler.Modulo(true, false)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "modulo operation not supported")
				So(result, ShouldBeNil)
			})
		})

		Convey("Comparisons", func() {
			Convey("Equal", func() {
				result, err := handler.Equal(true, true)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)

				result, err = handler.Equal(false, false)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)

				result, err = handler.Equal(true, false)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)

				// Bool vs non-bool values (strict type checking)
				result, err = handler.Equal(true, int64(1))
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse) // Different types, not equal

				result, err = handler.Equal(false, int64(0))
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse) // Different types, not equal

				result, err = handler.Equal(true, "hello")
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse) // Different types, not equal

				result, err = handler.Equal(false, "")
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse) // Different types, not equal
			})

			Convey("NotEqual", func() {
				result, err := handler.NotEqual(true, false)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)

				result, err = handler.NotEqual(true, true)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})

			Convey("Less (false < true)", func() {
				result, err := handler.Less(false, true)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)

				result, err = handler.Less(true, false)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)

				result, err = handler.Less(true, true)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)

				result, err = handler.Less(false, false)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})

			Convey("Greater (true > false)", func() {
				result, err := handler.Greater(true, false)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)

				result, err = handler.Greater(false, true)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)

				result, err = handler.Greater(true, true)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)

				result, err = handler.Greater(false, false)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})

			Convey("LessOrEqual", func() {
				result, err := handler.LessOrEqual(false, true)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)

				result, err = handler.LessOrEqual(true, true)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)

				result, err = handler.LessOrEqual(false, false)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)

				result, err = handler.LessOrEqual(true, false)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})

			Convey("GreaterOrEqual", func() {
				result, err := handler.GreaterOrEqual(true, false)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)

				result, err = handler.GreaterOrEqual(true, true)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)

				result, err = handler.GreaterOrEqual(false, false)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)

				result, err = handler.GreaterOrEqual(false, true)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
		})
	})
}

func TestBooleanLogicalOps(t *testing.T) {
	Convey("BooleanLogicalOps", t, func() {
		handler := NewBooleanTypeHandler()
		ops := NewBooleanLogicalOps(handler)

		Convey("And operation", func() {
			result, err := ops.And(true, true)
			So(err, ShouldBeNil)
			So(result, ShouldBeTrue)

			result, err = ops.And(true, false)
			So(err, ShouldBeNil)
			So(result, ShouldBeFalse)

			result, err = ops.And(false, true)
			So(err, ShouldBeNil)
			So(result, ShouldBeFalse)

			result, err = ops.And(false, false)
			So(err, ShouldBeNil)
			So(result, ShouldBeFalse)

			// With truthy values
			result, err = ops.And(int64(1), "hello")
			So(err, ShouldBeNil)
			So(result, ShouldBeTrue)

			result, err = ops.And(int64(0), "hello")
			So(err, ShouldBeNil)
			So(result, ShouldBeFalse)
		})

		Convey("Or operation", func() {
			result, err := ops.Or(true, true)
			So(err, ShouldBeNil)
			So(result, ShouldBeTrue)

			result, err = ops.Or(true, false)
			So(err, ShouldBeNil)
			So(result, ShouldBeTrue)

			result, err = ops.Or(false, true)
			So(err, ShouldBeNil)
			So(result, ShouldBeTrue)

			result, err = ops.Or(false, false)
			So(err, ShouldBeNil)
			So(result, ShouldBeFalse)

			// With truthy values
			result, err = ops.Or(int64(0), "")
			So(err, ShouldBeNil)
			So(result, ShouldBeFalse)

			result, err = ops.Or(int64(0), "hello")
			So(err, ShouldBeNil)
			So(result, ShouldBeTrue)
		})

		Convey("Not operation", func() {
			result, err := ops.Not(true)
			So(err, ShouldBeNil)
			So(result, ShouldBeFalse)

			result, err = ops.Not(false)
			So(err, ShouldBeNil)
			So(result, ShouldBeTrue)

			// With truthy values
			result, err = ops.Not(int64(0))
			So(err, ShouldBeNil)
			So(result, ShouldBeTrue)

			result, err = ops.Not(int64(42))
			So(err, ShouldBeNil)
			So(result, ShouldBeFalse)

			result, err = ops.Not("")
			So(err, ShouldBeNil)
			So(result, ShouldBeTrue)

			result, err = ops.Not("hello")
			So(err, ShouldBeNil)
			So(result, ShouldBeFalse)

			result, err = ops.Not(nil)
			So(err, ShouldBeNil)
			So(result, ShouldBeTrue)
		})

		Convey("Xor operation", func() {
			result, err := ops.Xor(true, true)
			So(err, ShouldBeNil)
			So(result, ShouldBeFalse)

			result, err = ops.Xor(true, false)
			So(err, ShouldBeNil)
			So(result, ShouldBeTrue)

			result, err = ops.Xor(false, true)
			So(err, ShouldBeNil)
			So(result, ShouldBeTrue)

			result, err = ops.Xor(false, false)
			So(err, ShouldBeNil)
			So(result, ShouldBeFalse)

			// With truthy values
			result, err = ops.Xor(int64(1), "")
			So(err, ShouldBeNil)
			So(result, ShouldBeTrue) // truthy XOR falsy = true

			result, err = ops.Xor(int64(0), nil)
			So(err, ShouldBeNil)
			So(result, ShouldBeFalse) // falsy XOR falsy = false
		})
	})
}
