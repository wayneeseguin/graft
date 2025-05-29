package operators

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestStringTypeHandler(t *testing.T) {
	Convey("StringTypeHandler", t, func() {
		handler := NewStringTypeHandler()
		
		Convey("supports correct type combinations", func() {
			So(handler.CanHandle(TypeString, TypeString), ShouldBeTrue)
			So(handler.CanHandle(TypeString, TypeInt), ShouldBeTrue)
			So(handler.CanHandle(TypeString, TypeFloat), ShouldBeTrue)
			So(handler.CanHandle(TypeInt, TypeString), ShouldBeTrue)
			So(handler.CanHandle(TypeString, TypeBool), ShouldBeTrue) // For concatenation
			
			// Should handle any type with string
			So(handler.CanHandle(TypeString, TypeMap), ShouldBeTrue)
			So(handler.CanHandle(TypeList, TypeString), ShouldBeTrue)
		})
		
		Convey("Addition (concatenation)", func() {
			Convey("string + string", func() {
				result, err := handler.Add("hello", "world")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "helloworld")
			})
			
			Convey("string + string with spaces", func() {
				result, err := handler.Add("hello ", "world")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "hello world")
			})
			
			Convey("empty string concatenation", func() {
				result, err := handler.Add("", "test")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "test")
				
				result, err = handler.Add("test", "")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "test")
			})
			
			Convey("string + number", func() {
				result, err := handler.Add("value: ", int64(42))
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "value: 42")
				
				result, err = handler.Add("pi: ", 3.14159)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "pi: 3.14159")
			})
			
			Convey("string + bool", func() {
				result, err := handler.Add("enabled: ", true)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "enabled: true")
			})
		})
		
		Convey("Multiplication (repetition)", func() {
			Convey("string * positive int", func() {
				result, err := handler.Multiply("ab", int64(3))
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "ababab")
			})
			
			Convey("int * string (commutative)", func() {
				result, err := handler.Multiply(int64(3), "xy")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "xyxyxy")
			})
			
			Convey("string * zero", func() {
				result, err := handler.Multiply("test", int64(0))
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "")
			})
			
			Convey("string * one", func() {
				result, err := handler.Multiply("test", int64(1))
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "test")
			})
			
			Convey("string * negative int", func() {
				result, err := handler.Multiply("test", int64(-5))
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "cannot repeat string negative times")
				So(result, ShouldBeNil)
			})
			
			Convey("string * very large int", func() {
				result, err := handler.Multiply("x", int64(1000001))
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "string repetition count too large")
				So(result, ShouldBeNil)
			})
			
			Convey("string * float (truncated to int)", func() {
				result, err := handler.Multiply("x", 3.7)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "xxx")
			})
		})
		
		Convey("Unsupported operations", func() {
			Convey("subtraction", func() {
				result, err := handler.Subtract("hello", "world")
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "subtract operation not supported")
				So(result, ShouldBeNil)
			})
			
			Convey("division", func() {
				result, err := handler.Divide("hello", "world")
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "divide operation not supported")
				So(result, ShouldBeNil)
			})
			
			Convey("modulo", func() {
				result, err := handler.Modulo("hello", "world")
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "modulo operation not supported")
				So(result, ShouldBeNil)
			})
		})
		
		Convey("Comparisons", func() {
			Convey("Equal", func() {
				result, err := handler.Equal("hello", "hello")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.Equal("hello", "world")
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
				
				result, err = handler.Equal("", "")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				// String vs non-string
				result, err = handler.Equal("42", int64(42))
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
			
			Convey("NotEqual", func() {
				result, err := handler.NotEqual("hello", "world")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.NotEqual("hello", "hello")
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
			
			Convey("Less (lexicographic)", func() {
				result, err := handler.Less("apple", "banana")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.Less("banana", "apple")
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
				
				result, err = handler.Less("a", "aa")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.Less("10", "2")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue) // Lexicographic, not numeric
			})
			
			Convey("Greater (lexicographic)", func() {
				result, err := handler.Greater("banana", "apple")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.Greater("apple", "banana")
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
				
				result, err = handler.Greater("aa", "a")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
			})
			
			Convey("LessOrEqual", func() {
				result, err := handler.LessOrEqual("apple", "banana")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.LessOrEqual("apple", "apple")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.LessOrEqual("banana", "apple")
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
			
			Convey("GreaterOrEqual", func() {
				result, err := handler.GreaterOrEqual("banana", "apple")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.GreaterOrEqual("apple", "apple")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.GreaterOrEqual("apple", "banana")
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
		})
		
		Convey("Edge cases", func() {
			Convey("very long string repetition", func() {
				// Should work up to the limit
				result, err := handler.Multiply("x", int64(1000))
				So(err, ShouldBeNil)
				So(len(result.(string)), ShouldEqual, 1000)
				So(result, ShouldEqual, strings.Repeat("x", 1000))
			})
			
			Convey("unicode string operations", func() {
				result, err := handler.Add("Hello ", "世界")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "Hello 世界")
				
				result, err = handler.Multiply("🎉", int64(3))
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "🎉🎉🎉")
				
				result, err = handler.Less("α", "β")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
			})
		})
	})
}