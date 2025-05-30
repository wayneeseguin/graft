package operators

import (
	"testing"
	
	. "github.com/smartystreets/goconvey/convey"
)

func TestMixedTypeHandler(t *testing.T) {
	Convey("MixedTypeHandler", t, func() {
		handler := NewMixedTypeHandler()
		
		Convey("CanHandle", func() {
			Convey("should handle any type combination", func() {
				// Test various type combinations
				So(handler.CanHandle(TypeString, TypeInt), ShouldBeTrue)
				So(handler.CanHandle(TypeBool, TypeFloat), ShouldBeTrue)
				So(handler.CanHandle(TypeMap, TypeList), ShouldBeTrue)
				So(handler.CanHandle(TypeNull, TypeString), ShouldBeTrue)
			})
		})
		
		Convey("Add (mixed-type addition)", func() {
			Convey("should handle null-safe operations", func() {
				// null + anything = anything
				result, err := handler.Add(nil, "hello")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "hello")
				
				result, err = handler.Add(42, nil)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, 42)
				
				result, err = handler.Add(nil, nil)
				So(err, ShouldBeNil)
				So(result, ShouldBeNil)
			})
			
			Convey("should fallback to string concatenation", func() {
				// int + string
				result, err := handler.Add(42, " items")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "42 items")
				
				// bool + string
				result, err = handler.Add(true, " value")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "true value")
				
				// float + string
				result, err = handler.Add(3.14, " pi")
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "3.14 pi")
			})
			
			Convey("should handle complex type conversions", func() {
				// map + list (both converted to strings)
				mapVal := map[interface{}]interface{}{"key": "value"}
				listVal := []interface{}{1, 2, 3}
				
				result, err := handler.Add(mapVal, listVal)
				So(err, ShouldBeNil)
				So(result, ShouldHaveSameTypeAs, "")
				// Result should be string representation of both
			})
		})
		
		Convey("Subtract (mixed-type subtraction)", func() {
			Convey("should handle null-safe operations", func() {
				// anything - null = anything
				result, err := handler.Subtract(42, nil)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, 42)
				
				// null - number = -number
				result, err = handler.Subtract(nil, int64(10))
				So(err, ShouldBeNil)
				So(result, ShouldEqual, int64(-10))
				
				result, err = handler.Subtract(nil, 3.14)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, -3.14)
				
				// null - null = null
				result, err = handler.Subtract(nil, nil)
				So(err, ShouldBeNil)
				So(result, ShouldBeNil)
			})
			
			Convey("should reject non-numeric mixed subtraction", func() {
				_, err := handler.Subtract("hello", "world")
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "subtract operation not supported")
			})
		})
		
		Convey("Multiply (mixed-type multiplication)", func() {
			Convey("should handle null-safe operations", func() {
				// anything * null = null
				result, err := handler.Multiply(42, nil)
				So(err, ShouldBeNil)
				So(result, ShouldBeNil)
				
				result, err = handler.Multiply(nil, "hello")
				So(err, ShouldBeNil)
				So(result, ShouldBeNil)
			})
			
			Convey("should reject non-null mixed multiplication", func() {
				_, err := handler.Multiply("hello", 42)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "multiply operation not supported")
			})
		})
		
		Convey("Divide (mixed-type division)", func() {
			Convey("should handle null-safe operations", func() {
				// null / anything = null
				result, err := handler.Divide(nil, 42)
				So(err, ShouldBeNil)
				So(result, ShouldBeNil)
				
				// anything / null = error
				_, err = handler.Divide(42, nil)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "division by null")
			})
		})
		
		Convey("Equal (universal equality)", func() {
			Convey("should handle null comparisons", func() {
				result, err := handler.Equal(nil, nil)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.Equal(nil, "hello")
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
				
				result, err = handler.Equal(42, nil)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
			
			Convey("should handle numeric type coercion", func() {
				// Different numeric types that represent the same value
				result, err := handler.Equal(int64(42), 42.0)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.Equal(int32(10), int64(10))
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
			})
			
			Convey("should fallback to string comparison", func() {
				// When numeric coercion doesn't apply
				result, err := handler.Equal("42", "42")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.Equal("hello", "world")
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
			
			Convey("should use deep equality as final fallback", func() {
				list1 := []interface{}{1, 2, 3}
				list2 := []interface{}{1, 2, 3}
				list3 := []interface{}{1, 2, 4}
				
				result, err := handler.Equal(list1, list2)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.Equal(list1, list3)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
		})
		
		Convey("NotEqual (universal inequality)", func() {
			Convey("should return opposite of Equal", func() {
				result, err := handler.NotEqual(nil, nil)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
				
				result, err = handler.NotEqual("hello", "world")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
			})
		})
		
		Convey("Less (mixed-type less-than)", func() {
			Convey("should handle null comparisons", func() {
				// null < anything (except null) = true
				result, err := handler.Less(nil, "hello")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				// anything < null = false
				result, err = handler.Less("hello", nil)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
				
				// null < null = false
				result, err = handler.Less(nil, nil)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
			
			Convey("should handle numeric comparisons", func() {
				result, err := handler.Less(int64(10), 20.5)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.Less(30.0, int64(25))
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
			
			Convey("should fallback to string comparison", func() {
				result, err := handler.Less("apple", "banana")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.Less("zebra", "apple")
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
		})
		
		Convey("Greater (mixed-type greater-than)", func() {
			Convey("should handle null comparisons", func() {
				// anything > null (except null) = true
				result, err := handler.Greater("hello", nil)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				// null > anything = false
				result, err = handler.Greater(nil, "hello")
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
			
			Convey("should handle numeric comparisons", func() {
				result, err := handler.Greater(30.0, int64(25))
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				result, err = handler.Greater(int64(10), 20.5)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
		})
		
		Convey("LessOrEqual", func() {
			Convey("should combine Less and Equal logic", func() {
				// Equal case
				result, err := handler.LessOrEqual(int64(42), 42.0)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				// Less case
				result, err = handler.LessOrEqual(int64(10), 20.5)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				// Greater case
				result, err = handler.LessOrEqual(30.0, int64(25))
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
		})
		
		Convey("GreaterOrEqual", func() {
			Convey("should combine Greater and Equal logic", func() {
				// Equal case
				result, err := handler.GreaterOrEqual(int64(42), 42.0)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				// Greater case  
				result, err = handler.GreaterOrEqual(30.0, int64(25))
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
				
				// Less case
				result, err = handler.GreaterOrEqual(int64(10), 20.5)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
		})
		
		Convey("Priority", func() {
			Convey("should have low priority (fallback handler)", func() {
				So(handler.Priority(), ShouldEqual, 10)
			})
		})
	})
}

func TestMixedTypeHelperFunctions(t *testing.T) {
	Convey("Helper functions", t, func() {
		Convey("toString", func() {
			Convey("should convert various types to strings", func() {
				So(toString(nil), ShouldEqual, "")
				So(toString("hello"), ShouldEqual, "hello")
				So(toString(true), ShouldEqual, "true")
				So(toString(false), ShouldEqual, "false")
				So(toString(42), ShouldEqual, "42")
				So(toString(int64(42)), ShouldEqual, "42")
				So(toString(3.14), ShouldEqual, "3.14")
				So(toString(float32(3.14)), ShouldContainSubstring, "3.14")
			})
			
			Convey("should handle complex types", func() {
				result := toString([]interface{}{1, 2, 3})
				So(result, ShouldNotBeEmpty)
				So(result, ShouldContainSubstring, "1")
			})
		})
		
		Convey("canCoerceToNumber", func() {
			Convey("should identify coercible types", func() {
				So(canCoerceToNumber(42), ShouldBeTrue)
				So(canCoerceToNumber(int64(42)), ShouldBeTrue)
				So(canCoerceToNumber(3.14), ShouldBeTrue)
				So(canCoerceToNumber(float32(3.14)), ShouldBeTrue)
				So(canCoerceToNumber(true), ShouldBeTrue)
				So(canCoerceToNumber(false), ShouldBeTrue)
				So(canCoerceToNumber("42"), ShouldBeTrue)
				So(canCoerceToNumber("3.14"), ShouldBeTrue)
			})
			
			Convey("should identify non-coercible types", func() {
				So(canCoerceToNumber(nil), ShouldBeFalse)
				So(canCoerceToNumber("hello"), ShouldBeFalse)
				So(canCoerceToNumber([]interface{}{1, 2, 3}), ShouldBeFalse)
				So(canCoerceToNumber(map[string]interface{}{"a": 1}), ShouldBeFalse)
			})
		})
		
		Convey("canCoerceToString", func() {
			Convey("should return true for all types", func() {
				So(canCoerceToString(nil), ShouldBeTrue)
				So(canCoerceToString(42), ShouldBeTrue)
				So(canCoerceToString("hello"), ShouldBeTrue)
				So(canCoerceToString(true), ShouldBeTrue)
				So(canCoerceToString([]interface{}{1, 2, 3}), ShouldBeTrue)
			})
		})
		
		Convey("compareNumbers", func() {
			Convey("should compare numeric values correctly", func() {
				So(compareNumbers(10, 20), ShouldEqual, -1)
				So(compareNumbers(20, 10), ShouldEqual, 1)
				So(compareNumbers(15, 15), ShouldEqual, 0)
				So(compareNumbers(int64(10), 10.0), ShouldEqual, 0)
				// float32(3.14) converted to float64 is slightly different than float64(3.14)
				// due to precision differences, so it's actually greater
				So(compareNumbers(float32(3.14), float64(3.14)), ShouldEqual, 1)
			})
		})
		
		Convey("convertToFloat64", func() {
			Convey("should convert various numeric types", func() {
				So(convertToFloat64(42), ShouldEqual, 42.0)
				So(convertToFloat64(int64(42)), ShouldEqual, 42.0)
				So(convertToFloat64(float32(3.14)), ShouldAlmostEqual, 3.14, 0.01)
				So(convertToFloat64(3.14), ShouldEqual, 3.14)
				So(convertToFloat64(uint32(42)), ShouldEqual, 42.0)
			})
			
			Convey("should return 0 for non-numeric types", func() {
				So(convertToFloat64("hello"), ShouldEqual, 0)
				So(convertToFloat64(true), ShouldEqual, 0)
				So(convertToFloat64(nil), ShouldEqual, 0)
			})
		})
	})
}