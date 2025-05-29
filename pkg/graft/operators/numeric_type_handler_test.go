package operators

import (
	"math"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNumericTypeHandler(t *testing.T) {
	Convey("NumericTypeHandler", t, func() {
		handler := NewNumericTypeHandler()
		
		Convey("supports correct type combinations", func() {
			So(handler.CanHandle(TypeInt, TypeInt), ShouldBeTrue)
			So(handler.CanHandle(TypeInt, TypeFloat), ShouldBeTrue)
			So(handler.CanHandle(TypeFloat, TypeInt), ShouldBeTrue)
			So(handler.CanHandle(TypeFloat, TypeFloat), ShouldBeTrue)
			
			So(handler.CanHandle(TypeString, TypeInt), ShouldBeFalse)
			So(handler.CanHandle(TypeBool, TypeFloat), ShouldBeFalse)
		})
		
		Convey("Addition", func() {
			Convey("int + int", func() {
				result, err := handler.Add(int64(5), int64(3))
				So(err, ShouldBeNil)
				So(result, ShouldEqual, int64(8))
			})
			
			Convey("int + float", func() {
				result, err := handler.Add(int64(5), 3.5)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, 8.5)
			})
			
			Convey("float + int", func() {
				result, err := handler.Add(5.5, int64(3))
				So(err, ShouldBeNil)
				So(result, ShouldEqual, 8.5)
			})
			
			Convey("float + float", func() {
				result, err := handler.Add(5.5, 3.5)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, 9.0)
			})
			
			Convey("integer overflow", func() {
				result, err := handler.Add(int64(math.MaxInt64), int64(1))
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "integer overflow")
				So(result, ShouldBeNil)
			})
		})
		
		Convey("Subtraction", func() {
			Convey("int - int", func() {
				result, err := handler.Subtract(int64(10), int64(3))
				So(err, ShouldBeNil)
				So(result, ShouldEqual, int64(7))
			})
			
			Convey("int - float", func() {
				result, err := handler.Subtract(int64(10), 3.5)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, 6.5)
			})
			
			Convey("float - int", func() {
				result, err := handler.Subtract(10.5, int64(3))
				So(err, ShouldBeNil)
				So(result, ShouldEqual, 7.5)
			})
			
			Convey("float - float", func() {
				result, err := handler.Subtract(10.5, 3.5)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, 7.0)
			})
			
			Convey("integer overflow", func() {
				result, err := handler.Subtract(int64(math.MinInt64), int64(1))
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "integer overflow")
				So(result, ShouldBeNil)
			})
		})
		
		Convey("Multiplication", func() {
			Convey("int * int", func() {
				result, err := handler.Multiply(int64(5), int64(3))
				So(err, ShouldBeNil)
				So(result, ShouldEqual, int64(15))
			})
			
			Convey("int * float", func() {
				result, err := handler.Multiply(int64(5), 3.5)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, 17.5)
			})
			
			Convey("float * int", func() {
				result, err := handler.Multiply(5.5, int64(3))
				So(err, ShouldBeNil)
				So(result, ShouldEqual, 16.5)
			})
			
			Convey("float * float", func() {
				result, err := handler.Multiply(5.5, 3.5)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, 19.25)
			})
			
			Convey("multiply by zero", func() {
				result, err := handler.Multiply(int64(100), int64(0))
				So(err, ShouldBeNil)
				So(result, ShouldEqual, int64(0))
			})
			
			Convey("integer overflow", func() {
				result, err := handler.Multiply(int64(math.MaxInt64/2), int64(3))
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "integer overflow")
				So(result, ShouldBeNil)
			})
		})
		
		Convey("Division", func() {
			Convey("int / int", func() {
				result, err := handler.Divide(int64(10), int64(2))
				So(err, ShouldBeNil)
				So(result, ShouldEqual, 5.0)
			})
			
			Convey("int / float", func() {
				result, err := handler.Divide(int64(10), 2.5)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, 4.0)
			})
			
			Convey("float / int", func() {
				result, err := handler.Divide(10.5, int64(2))
				So(err, ShouldBeNil)
				So(result, ShouldEqual, 5.25)
			})
			
			Convey("float / float", func() {
				result, err := handler.Divide(10.5, 2.5)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, 4.2)
			})
			
			Convey("division by zero", func() {
				result, err := handler.Divide(int64(10), int64(0))
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "division by zero")
				So(result, ShouldBeNil)
			})
		})
		
		Convey("Modulo", func() {
			Convey("int % int", func() {
				result, err := handler.Modulo(int64(10), int64(3))
				So(err, ShouldBeNil)
				So(result, ShouldEqual, int64(1))
			})
			
			Convey("float % int (truncated)", func() {
				result, err := handler.Modulo(10.7, int64(3))
				So(err, ShouldBeNil)
				So(result, ShouldEqual, int64(1))
			})
			
			Convey("int % float (truncated)", func() {
				result, err := handler.Modulo(int64(10), 3.7)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, int64(1))
			})
			
			Convey("modulo by zero", func() {
				result, err := handler.Modulo(int64(10), int64(0))
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "modulo by zero")
				So(result, ShouldBeNil)
			})
		})
		
		Convey("Comparisons", func() {
			Convey("Equal", func() {
				Convey("int == int", func() {
					result, err := handler.Equal(int64(5), int64(5))
					So(err, ShouldBeNil)
					So(result, ShouldBeTrue)
					
					result, err = handler.Equal(int64(5), int64(3))
					So(err, ShouldBeNil)
					So(result, ShouldBeFalse)
				})
				
				Convey("int == float", func() {
					result, err := handler.Equal(int64(5), 5.0)
					So(err, ShouldBeNil)
					So(result, ShouldBeTrue)
					
					result, err = handler.Equal(int64(5), 5.1)
					So(err, ShouldBeNil)
					So(result, ShouldBeFalse)
				})
			})
			
			Convey("NotEqual", func() {
				result, err := handler.NotEqual(int64(5), int64(3))
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.NotEqual(int64(5), int64(5))
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
			
			Convey("Less", func() {
				result, err := handler.Less(int64(3), int64(5))
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.Less(int64(5), int64(3))
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
				
				result, err = handler.Less(3.5, int64(4))
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
			})
			
			Convey("Greater", func() {
				result, err := handler.Greater(int64(5), int64(3))
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.Greater(int64(3), int64(5))
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
				
				result, err = handler.Greater(4.5, int64(4))
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
			})
			
			Convey("LessOrEqual", func() {
				result, err := handler.LessOrEqual(int64(3), int64(5))
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.LessOrEqual(int64(5), int64(5))
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.LessOrEqual(int64(7), int64(5))
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
			
			Convey("GreaterOrEqual", func() {
				result, err := handler.GreaterOrEqual(int64(5), int64(3))
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.GreaterOrEqual(int64(5), int64(5))
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.GreaterOrEqual(int64(3), int64(5))
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
		})
	})
}