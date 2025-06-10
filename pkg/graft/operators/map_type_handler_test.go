package operators

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMapTypeHandler(t *testing.T) {
	Convey("MapTypeHandler", t, func() {
		handler := NewMapTypeHandler()

		Convey("CanHandle", func() {
			Convey("should handle map-map operations", func() {
				So(handler.CanHandle(TypeMap, TypeMap), ShouldBeTrue)
			})

			Convey("should not handle non-map operations", func() {
				So(handler.CanHandle(TypeMap, TypeString), ShouldBeFalse)
				So(handler.CanHandle(TypeInt, TypeMap), ShouldBeFalse)
				So(handler.CanHandle(TypeString, TypeString), ShouldBeFalse)
			})
		})

		Convey("Add (map merging)", func() {
			Convey("should merge two maps", func() {
				mapA := map[interface{}]interface{}{
					"a": 1,
					"b": 2,
				}
				mapB := map[interface{}]interface{}{
					"c": 3,
					"d": 4,
				}

				result, err := handler.Add(mapA, mapB)
				So(err, ShouldBeNil)

				expected := map[interface{}]interface{}{
					"a": 1,
					"b": 2,
					"c": 3,
					"d": 4,
				}
				So(result, ShouldResemble, expected)
			})

			Convey("should handle overlapping keys (second map wins)", func() {
				mapA := map[interface{}]interface{}{
					"a": 1,
					"b": 2,
				}
				mapB := map[interface{}]interface{}{
					"b": 20,
					"c": 3,
				}

				result, err := handler.Add(mapA, mapB)
				So(err, ShouldBeNil)

				expected := map[interface{}]interface{}{
					"a": 1,
					"b": 20, // Value from mapB
					"c": 3,
				}
				So(result, ShouldResemble, expected)
			})

			Convey("should handle empty maps", func() {
				mapA := map[interface{}]interface{}{}
				mapB := map[interface{}]interface{}{
					"a": 1,
				}

				result, err := handler.Add(mapA, mapB)
				So(err, ShouldBeNil)
				So(result, ShouldResemble, mapB)

				// Reverse order
				result, err = handler.Add(mapB, mapA)
				So(err, ShouldBeNil)
				So(result, ShouldResemble, mapB)
			})

			Convey("should handle map[string]interface{} conversion", func() {
				mapA := map[string]interface{}{
					"name":  "test",
					"value": 42,
				}
				mapB := map[interface{}]interface{}{
					"active": true,
				}

				result, err := handler.Add(mapA, mapB)
				So(err, ShouldBeNil)

				expected := map[interface{}]interface{}{
					"name":   "test",
					"value":  42,
					"active": true,
				}
				So(result, ShouldResemble, expected)
			})

			Convey("should return error for non-map operands", func() {
				_, err := handler.Add("not a map", map[interface{}]interface{}{})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "add operation not supported")
			})
		})

		Convey("Equal", func() {
			Convey("should return true for identical maps", func() {
				mapA := map[interface{}]interface{}{
					"a": 1,
					"b": "hello",
					"c": true,
				}
				mapB := map[interface{}]interface{}{
					"a": 1,
					"b": "hello",
					"c": true,
				}

				result, err := handler.Equal(mapA, mapB)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
			})

			Convey("should return true for maps with same content but different order", func() {
				mapA := map[interface{}]interface{}{
					"a": 1,
					"b": 2,
				}
				mapB := map[interface{}]interface{}{
					"b": 2,
					"a": 1,
				}

				result, err := handler.Equal(mapA, mapB)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
			})

			Convey("should return false for maps with different values", func() {
				mapA := map[interface{}]interface{}{
					"a": 1,
					"b": 2,
				}
				mapB := map[interface{}]interface{}{
					"a": 1,
					"b": 3, // Different value
				}

				result, err := handler.Equal(mapA, mapB)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})

			Convey("should return false for maps with different keys", func() {
				mapA := map[interface{}]interface{}{
					"a": 1,
					"b": 2,
				}
				mapB := map[interface{}]interface{}{
					"a": 1,
					"c": 2, // Different key
				}

				result, err := handler.Equal(mapA, mapB)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})

			Convey("should return false for maps with different sizes", func() {
				mapA := map[interface{}]interface{}{
					"a": 1,
				}
				mapB := map[interface{}]interface{}{
					"a": 1,
					"b": 2,
				}

				result, err := handler.Equal(mapA, mapB)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})

			Convey("should handle empty maps", func() {
				mapA := map[interface{}]interface{}{}
				mapB := map[interface{}]interface{}{}

				result, err := handler.Equal(mapA, mapB)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
			})

			Convey("should handle nested maps", func() {
				mapA := map[interface{}]interface{}{
					"nested": map[interface{}]interface{}{
						"x": 1,
						"y": 2,
					},
				}
				mapB := map[interface{}]interface{}{
					"nested": map[interface{}]interface{}{
						"x": 1,
						"y": 2,
					},
				}

				result, err := handler.Equal(mapA, mapB)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
			})
		})

		Convey("NotEqual", func() {
			Convey("should return false for identical maps", func() {
				mapA := map[interface{}]interface{}{"a": 1}
				mapB := map[interface{}]interface{}{"a": 1}

				result, err := handler.NotEqual(mapA, mapB)
				So(err, ShouldBeNil)
				So(result, ShouldBeFalse)
			})

			Convey("should return true for different maps", func() {
				mapA := map[interface{}]interface{}{"a": 1}
				mapB := map[interface{}]interface{}{"a": 2}

				result, err := handler.NotEqual(mapA, mapB)
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
			})
		})

		Convey("Unsupported operations", func() {
			mapA := map[interface{}]interface{}{"a": 1}
			mapB := map[interface{}]interface{}{"b": 2}

			Convey("Subtract should return error", func() {
				_, err := handler.Subtract(mapA, mapB)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "subtract operation not supported")
			})

			Convey("Multiply should return error", func() {
				_, err := handler.Multiply(mapA, mapB)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "multiply operation not supported")
			})

			Convey("Divide should return error", func() {
				_, err := handler.Divide(mapA, mapB)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "divide operation not supported")
			})

			Convey("Modulo should return error", func() {
				_, err := handler.Modulo(mapA, mapB)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "modulo operation not supported")
			})

			Convey("Less should return error", func() {
				_, err := handler.Less(mapA, mapB)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "less operation not supported")
			})

			Convey("Greater should return error", func() {
				_, err := handler.Greater(mapA, mapB)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "greater operation not supported")
			})
		})

		Convey("Priority", func() {
			Convey("should have high priority", func() {
				So(handler.Priority(), ShouldEqual, 70)
			})
		})

		Convey("Comparison operations error handling", func() {
			handler := &MapTypeHandler{}

			Convey("LessOrEqual should return not implemented error", func() {
				mapA := map[interface{}]interface{}{"a": 1, "b": 2}
				mapB := map[interface{}]interface{}{"c": 3, "d": 4}

				_, err := handler.LessOrEqual(mapA, mapB)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "lessOrEqual operation not supported")
			})

			Convey("GreaterOrEqual should return not implemented error", func() {
				mapA := map[interface{}]interface{}{"a": 1, "b": 2}
				mapB := map[interface{}]interface{}{"c": 3, "d": 4}

				_, err := handler.GreaterOrEqual(mapA, mapB)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "greaterOrEqual operation not supported")
			})
		})
	})
}

func TestConvertToMap(t *testing.T) {
	Convey("convertToMap function", t, func() {
		Convey("should handle map[interface{}]interface{}", func() {
			input := map[interface{}]interface{}{"a": 1, "b": 2}
			result, ok := convertToMap(input)
			So(ok, ShouldBeTrue)
			So(result, ShouldResemble, input)
		})

		Convey("should convert map[string]interface{}", func() {
			input := map[string]interface{}{"a": 1, "b": 2}
			result, ok := convertToMap(input)
			So(ok, ShouldBeTrue)
			expected := map[interface{}]interface{}{"a": 1, "b": 2}
			So(result, ShouldResemble, expected)
		})

		Convey("should return false for non-map types", func() {
			_, ok := convertToMap("not a map")
			So(ok, ShouldBeFalse)

			_, ok = convertToMap(42)
			So(ok, ShouldBeFalse)

			_, ok = convertToMap([]interface{}{1, 2, 3})
			So(ok, ShouldBeFalse)
		})

		Convey("should return false for nil", func() {
			_, ok := convertToMap(nil)
			So(ok, ShouldBeFalse)
		})
	})

}
