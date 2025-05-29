package operators

import (
	"testing"
	
	. "github.com/smartystreets/goconvey/convey"
)

func TestListTypeHandler(t *testing.T) {
	Convey("ListTypeHandler", t, func() {
		handler := NewListTypeHandler()
		
		Convey("CanHandle", func() {
			Convey("should handle list-list operations", func() {
				So(handler.CanHandle(TypeList, TypeList), ShouldBeTrue)
			})
			
			Convey("should handle list-int operations for repetition", func() {
				So(handler.CanHandle(TypeList, TypeInt), ShouldBeTrue)
				So(handler.CanHandle(TypeInt, TypeList), ShouldBeTrue)
			})
			
			Convey("should not handle non-list operations", func() {
				So(handler.CanHandle(TypeList, TypeString), ShouldBeFalse)
				So(handler.CanHandle(TypeString, TypeList), ShouldBeFalse)
				So(handler.CanHandle(TypeString, TypeString), ShouldBeFalse)
			})
		})
		
		Convey("Add (list concatenation)", func() {
			Convey("should concatenate two lists", func() {
				listA := []interface{}{1, 2, 3}
				listB := []interface{}{4, 5, 6}
				
				result, err := handler.Add(listA, listB)
				So(err, ShouldBeNil)
				
				expected := []interface{}{1, 2, 3, 4, 5, 6}
				So(result, ShouldResemble, expected)
			})
			
			Convey("should handle empty lists", func() {
				listA := []interface{}{}
				listB := []interface{}{1, 2}
				
				result, err := handler.Add(listA, listB)
				So(err, ShouldBeNil)
				So(result, ShouldResemble, listB)
				
				// Reverse order
				result, err = handler.Add(listB, listA)
				So(err, ShouldBeNil)
				So(result, ShouldResemble, listB)
			})
			
			Convey("should handle mixed types in lists", func() {
				listA := []interface{}{1, "hello", true}
				listB := []interface{}{3.14, false, "world"}
				
				result, err := handler.Add(listA, listB)
				So(err, ShouldBeNil)
				
				expected := []interface{}{1, "hello", true, 3.14, false, "world"}
				So(result, ShouldResemble, expected)
			})
			
			Convey("should return error for non-list operands", func() {
				_, err := handler.Add("not a list", []interface{}{})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "add operation not supported")
			})
		})
		
		Convey("Multiply (list repetition)", func() {
			Convey("should repeat list n times", func() {
				list := []interface{}{1, 2}
				times := int64(3)
				
				result, err := handler.Multiply(list, times)
				So(err, ShouldBeNil)
				
				expected := []interface{}{1, 2, 1, 2, 1, 2}
				So(result, ShouldResemble, expected)
			})
			
			Convey("should work with reverse order (int * list)", func() {
				list := []interface{}{"a", "b"}
				times := int64(2)
				
				result, err := handler.Multiply(times, list)
				So(err, ShouldBeNil)
				
				expected := []interface{}{"a", "b", "a", "b"}
				So(result, ShouldResemble, expected)
			})
			
			Convey("should handle zero repetitions", func() {
				list := []interface{}{1, 2, 3}
				times := int64(0)
				
				result, err := handler.Multiply(list, times)
				So(err, ShouldBeNil)
				
				expected := []interface{}{}
				So(result, ShouldResemble, expected)
			})
			
			Convey("should handle one repetition", func() {
				list := []interface{}{1, 2, 3}
				times := int64(1)
				
				result, err := handler.Multiply(list, times)
				So(err, ShouldBeNil)
				So(result, ShouldResemble, list)
			})
			
			Convey("should handle empty list", func() {
				list := []interface{}{}
				times := int64(5)
				
				result, err := handler.Multiply(list, times)
				So(err, ShouldBeNil)
				
				expected := []interface{}{}
				So(result, ShouldResemble, expected)
			})
			
			Convey("should reject negative repetitions", func() {
				list := []interface{}{1, 2}
				times := int64(-1)
				
				_, err := handler.Multiply(list, times)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "cannot repeat list negative times")
			})
			
			Convey("should reject excessively large repetitions", func() {
				list := []interface{}{1, 2, 3, 4, 5}
				times := int64(10000) // Would create 50,000 elements
				
				_, err := handler.Multiply(list, times)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "list repetition too large")
			})
			
			Convey("should handle different integer types", func() {
				list := []interface{}{1, 2}
				
				// Test with different int types
				result, err := handler.Multiply(list, int(2))
				So(err, ShouldBeNil)
				So(len(result.([]interface{})), ShouldEqual, 4)
				
				result, err = handler.Multiply(list, int32(2))
				So(err, ShouldBeNil)
				So(len(result.([]interface{})), ShouldEqual, 4)
			})
			
			Convey("should return error for non-list or non-int operands", func() {
				_, err := handler.Multiply("not a list", 3)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "multiply operation not supported")
				
				_, err = handler.Multiply([]interface{}{1, 2}, "not a number")
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "multiply operation not supported")
			})
		})
		
		Convey("Equal", func() {
			Convey("should return true for identical lists", func() {
				listA := []interface{}{1, "hello", true}
				listB := []interface{}{1, "hello", true}
				
				result, err := handler.Equal(listA, listB)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
			})
			
			Convey("should return false for lists with different elements", func() {
				listA := []interface{}{1, 2, 3}
				listB := []interface{}{1, 2, 4} // Different last element
				
				result, err := handler.Equal(listA, listB)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
			
			Convey("should return false for lists with different lengths", func() {
				listA := []interface{}{1, 2}
				listB := []interface{}{1, 2, 3}
				
				result, err := handler.Equal(listA, listB)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
			
			Convey("should return false for lists with same elements in different order", func() {
				listA := []interface{}{1, 2, 3}
				listB := []interface{}{3, 2, 1}
				
				result, err := handler.Equal(listA, listB)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
			
			Convey("should handle empty lists", func() {
				listA := []interface{}{}
				listB := []interface{}{}
				
				result, err := handler.Equal(listA, listB)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
			})
			
			Convey("should handle nested lists", func() {
				listA := []interface{}{
					[]interface{}{1, 2},
					[]interface{}{3, 4},
				}
				listB := []interface{}{
					[]interface{}{1, 2},
					[]interface{}{3, 4},
				}
				
				result, err := handler.Equal(listA, listB)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
			})
			
			Convey("should handle nested maps", func() {
				listA := []interface{}{
					map[interface{}]interface{}{"x": 1},
					map[interface{}]interface{}{"y": 2},
				}
				listB := []interface{}{
					map[interface{}]interface{}{"x": 1},
					map[interface{}]interface{}{"y": 2},
				}
				
				result, err := handler.Equal(listA, listB)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
			})
		})
		
		Convey("NotEqual", func() {
			Convey("should return false for identical lists", func() {
				listA := []interface{}{1, 2, 3}
				listB := []interface{}{1, 2, 3}
				
				result, err := handler.NotEqual(listA, listB)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})
			
			Convey("should return true for different lists", func() {
				listA := []interface{}{1, 2, 3}
				listB := []interface{}{1, 2, 4}
				
				result, err := handler.NotEqual(listA, listB)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
			})
		})
		
		Convey("Unsupported operations", func() {
			listA := []interface{}{1, 2}
			listB := []interface{}{3, 4}
			
			Convey("Subtract should return error", func() {
				_, err := handler.Subtract(listA, listB)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "subtract operation not supported")
			})
			
			Convey("Divide should return error", func() {
				_, err := handler.Divide(listA, listB)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "divide operation not supported")
			})
			
			Convey("Modulo should return error", func() {
				_, err := handler.Modulo(listA, listB)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "modulo operation not supported")
			})
			
			Convey("Less should return error", func() {
				_, err := handler.Less(listA, listB)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "less operation not supported")
			})
			
			Convey("Greater should return error", func() {
				_, err := handler.Greater(listA, listB)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "greater operation not supported")
			})
		})
		
		Convey("Priority", func() {
			Convey("should have high priority", func() {
				So(handler.Priority(), ShouldEqual, 70)
			})
		})
	})
}

func TestConvertToList(t *testing.T) {
	Convey("convertToList function", t, func() {
		Convey("should handle []interface{}", func() {
			input := []interface{}{1, 2, 3}
			result, ok := convertToList(input)
			So(ok, ShouldBeTrue)
			So(result, ShouldResemble, input)
		})
		
		Convey("should convert []int", func() {
			input := []int{1, 2, 3}
			result, ok := convertToList(input)
			So(ok, ShouldBeTrue)
			expected := []interface{}{1, 2, 3}
			So(result, ShouldResemble, expected)
		})
		
		Convey("should convert []string", func() {
			input := []string{"a", "b", "c"}
			result, ok := convertToList(input)
			So(ok, ShouldBeTrue)
			expected := []interface{}{"a", "b", "c"}
			So(result, ShouldResemble, expected)
		})
		
		Convey("should handle arrays", func() {
			input := [3]int{1, 2, 3}
			result, ok := convertToList(input)
			So(ok, ShouldBeTrue)
			expected := []interface{}{1, 2, 3}
			So(result, ShouldResemble, expected)
		})
		
		Convey("should return false for non-list types", func() {
			_, ok := convertToList("not a list")
			So(ok, ShouldBeFalse)
			
			_, ok = convertToList(42)
			So(ok, ShouldBeFalse)
			
			_, ok = convertToList(map[string]interface{}{"a": 1})
			So(ok, ShouldBeFalse)
		})
		
		Convey("should return false for nil", func() {
			_, ok := convertToList(nil)
			So(ok, ShouldBeFalse)
		})
	})
}

func TestConvertToInt(t *testing.T) {
	Convey("convertToInt function", t, func() {
		Convey("should handle various integer types", func() {
			tests := []struct {
				input    interface{}
				expected int64
				shouldOk bool
			}{
				{int(42), 42, true},
				{int8(42), 42, true},
				{int16(42), 42, true},
				{int32(42), 42, true},
				{int64(42), 42, true},
				{uint(42), 42, true},
				{uint8(42), 42, true},
				{uint16(42), 42, true},
				{uint32(42), 42, true},
				{uint64(42), 42, true},
				{float64(42.5), 0, false}, // floats should not convert
				{"42", 0, false},          // strings should not convert
				{true, 0, false},          // booleans should not convert
			}
			
			for _, test := range tests {
				result, ok := convertToInt(test.input)
				So(ok, ShouldEqual, test.shouldOk)
				if test.shouldOk {
					So(result, ShouldEqual, test.expected)
				}
			}
		})
		
		Convey("should handle large uint64 values correctly", func() {
			// Value within int64 range
			result, ok := convertToInt(uint64(9223372036854775807))
			So(ok, ShouldBeTrue)
			So(result, ShouldEqual, int64(9223372036854775807))
			
			// Value outside int64 range
			result, ok = convertToInt(uint64(9223372036854775808))
			So(ok, ShouldBeFalse)
		})
	})
}