package graft

import (
	"math"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestValue(t *testing.T) {
	t.Run("String value", func(t *testing.T) {
		v := NewValue("hello")
		if v.Type() != StringValue {
			t.Errorf("expected StringValue, got %v", v.Type())
		}

		str, err := v.AsString()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if str != "hello" {
			t.Errorf("expected 'hello', got '%s'", str)
		}

		// Should fail to convert to int
		_, err = v.AsInt()
		if err == nil {
			t.Error("expected error when converting string to int")
		}
	})

	t.Run("Int value", func(t *testing.T) {
		v := NewValue(42)
		if v.Type() != IntValue {
			t.Errorf("expected IntValue, got %v", v.Type())
		}

		i, err := v.AsInt()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if i != 42 {
			t.Errorf("expected 42, got %d", i)
		}

		// Should convert to int64
		i64, err := v.AsInt64()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if i64 != 42 {
			t.Errorf("expected 42, got %d", i64)
		}

		// Should convert to float64
		f, err := v.AsFloat64()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if f != 42.0 {
			t.Errorf("expected 42.0, got %f", f)
		}

		// Should convert to string
		str, err := v.AsString()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if str != "42" {
			t.Errorf("expected '42', got '%s'", str)
		}
	})

	t.Run("Float64 value", func(t *testing.T) {
		v := NewValue(3.14)
		if v.Type() != Float64Value {
			t.Errorf("expected Float64Value, got %v", v.Type())
		}

		f, err := v.AsFloat64()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if f != 3.14 {
			t.Errorf("expected 3.14, got %f", f)
		}

		// Should not convert to int (not a whole number)
		_, err = v.AsInt()
		if err == nil {
			t.Error("expected error when converting float to int")
		}
	})

	t.Run("Bool value", func(t *testing.T) {
		v := NewValue(true)
		if v.Type() != BoolValue {
			t.Errorf("expected BoolValue, got %v", v.Type())
		}

		b, err := v.AsBool()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !b {
			t.Errorf("expected true, got %v", b)
		}

		// Should convert to string
		str, err := v.AsString()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if str != "true" {
			t.Errorf("expected 'true', got '%s'", str)
		}
	})

	t.Run("Slice value", func(t *testing.T) {
		original := []interface{}{"a", "b", "c"}
		v := NewValue(original)
		if v.Type() != SliceValue {
			t.Errorf("expected SliceValue, got %v", v.Type())
		}

		slice, err := v.AsSlice()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(slice) != 3 {
			t.Errorf("expected length 3, got %d", len(slice))
		}
		if slice[0] != "a" {
			t.Errorf("expected 'a', got '%v'", slice[0])
		}
	})

	t.Run("Map value", func(t *testing.T) {
		original := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		}
		v := NewValue(original)
		if v.Type() != MapValue {
			t.Errorf("expected MapValue, got %v", v.Type())
		}

		m, err := v.AsMap()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(m) != 2 {
			t.Errorf("expected length 2, got %d", len(m))
		}
		if m["key1"] != "value1" {
			t.Errorf("expected 'value1', got '%v'", m["key1"])
		}
	})

	t.Run("Interface map conversion", func(t *testing.T) {
		original := map[interface{}]interface{}{
			"key1": "value1",
			"key2": 42,
		}
		v := NewValue(original)
		if v.Type() != MapValue {
			t.Errorf("expected MapValue, got %v", v.Type())
		}

		m, err := v.AsMap()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(m) != 2 {
			t.Errorf("expected length 2, got %d", len(m))
		}
		if m["key1"] != "value1" {
			t.Errorf("expected 'value1', got '%v'", m["key1"])
		}
	})

	t.Run("Nil value", func(t *testing.T) {
		v := NewValue(nil)
		if v.Type() != NilValue {
			t.Errorf("expected NilValue, got %v", v.Type())
		}

		if !v.IsNil() {
			t.Error("expected IsNil to return true")
		}

		// All conversions should fail
		_, err := v.AsString()
		if err == nil {
			t.Error("expected error when converting nil to string")
		}

		_, err = v.AsInt()
		if err == nil {
			t.Error("expected error when converting nil to int")
		}
	})
}

func TestTypedResponse(t *testing.T) {
	t.Run("Create and convert", func(t *testing.T) {
		tr := NewTypedResponse(Replace, "hello")

		if tr.Type != Replace {
			t.Errorf("expected Replace action, got %v", tr.Type)
		}

		str, err := tr.Value.AsString()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if str != "hello" {
			t.Errorf("expected 'hello', got '%s'", str)
		}

		// Convert to legacy format
		legacy := tr.ToLegacyResponse()
		if legacy.Type != Replace {
			t.Errorf("expected Replace action, got %v", legacy.Type)
		}
		if legacy.Value != "hello" {
			t.Errorf("expected 'hello', got '%v'", legacy.Value)
		}

		// Convert back from legacy
		tr2 := NewResponseFromLegacy(legacy)
		str2, err := tr2.Value.AsString()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if str2 != "hello" {
			t.Errorf("expected 'hello', got '%s'", str2)
		}
	})
}

// Additional comprehensive tests using GoConvey to improve coverage
func TestValueType_String(t *testing.T) {
	Convey("ValueType String method", t, func() {

		Convey("should return correct string representations", func() {
			So(NilValue.String(), ShouldEqual, "nil")
			So(StringValue.String(), ShouldEqual, "string")
			So(IntValue.String(), ShouldEqual, "int")
			So(Int64Value.String(), ShouldEqual, "int64")
			So(Float64Value.String(), ShouldEqual, "float64")
			So(BoolValue.String(), ShouldEqual, "bool")
			So(SliceValue.String(), ShouldEqual, "slice")
			So(MapValue.String(), ShouldEqual, "map")
			So(UnknownValue.String(), ShouldEqual, "unknown")
		})

		Convey("should return 'unknown' for undefined value types", func() {
			// Test an out-of-range value type
			invalidType := ValueType(999)
			So(invalidType.String(), ShouldEqual, "unknown")
		})
	})
}

func TestValueImpl_String(t *testing.T) {
	Convey("valueImpl String method", t, func() {

		Convey("should return '<nil>' for nil values", func() {
			nilValue := NewValue(nil)
			So(nilValue.String(), ShouldEqual, "<nil>")
		})

		Convey("should return string representation for non-nil values", func() {
			So(NewValue("test").String(), ShouldEqual, "test")
			So(NewValue(42).String(), ShouldEqual, "42")
			So(NewValue(true).String(), ShouldEqual, "true")
			So(NewValue(3.14).String(), ShouldEqual, "3.14")
		})
	})
}

func TestNewValue_EdgeCases(t *testing.T) {
	Convey("NewValue edge cases", t, func() {

		Convey("should handle map[interface{}]interface{} with non-string keys", func() {
			input := map[interface{}]interface{}{
				123:    "numeric key",
				"test": "string key",
			}

			value := NewValue(input)
			So(value.Type(), ShouldEqual, UnknownValue)
		})

		Convey("should handle map[interface{}]interface{} with all string keys", func() {
			input := map[interface{}]interface{}{
				"key1": "value1",
				"key2": "value2",
			}

			value := NewValue(input)
			So(value.Type(), ShouldEqual, MapValue)

			mapValue, err := value.AsMap()
			So(err, ShouldBeNil)
			So(mapValue["key1"], ShouldEqual, "value1")
			So(mapValue["key2"], ShouldEqual, "value2")
		})

		Convey("should handle unknown types", func() {
			type CustomType struct {
				Field string
			}

			value := NewValue(CustomType{Field: "test"})
			So(value.Type(), ShouldEqual, UnknownValue)
		})
	})
}

func TestValueImpl_AsString_EdgeCases(t *testing.T) {
	Convey("AsString edge cases", t, func() {

		Convey("should handle all value types", func() {
			// Test conversions from all types
			testCases := []struct {
				input    interface{}
				expected string
			}{
				{42, "42"},
				{int64(42), "42"},
				{3.14, "3.14"},
				{true, "true"},
				{false, "false"},
			}

			for _, tc := range testCases {
				value := NewValue(tc.input)
				result, err := value.AsString()
				So(err, ShouldBeNil)
				So(result, ShouldEqual, tc.expected)
			}
		})

		Convey("should convert slice/map/unknown types to string", func() {
			slice := []interface{}{"a", "b"}
			value := NewValue(slice)
			result, err := value.AsString()
			So(err, ShouldBeNil)
			So(result, ShouldNotBeEmpty)
		})
	})
}

func TestValueImpl_AsInt_EdgeCases(t *testing.T) {
	Convey("AsInt edge cases", t, func() {

		Convey("should handle int64 overflow", func() {
			largeValue := NewValue(int64(math.MaxInt64))
			_, err := largeValue.AsInt()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "overflows int")
		})

		Convey("should handle int64 underflow", func() {
			smallValue := NewValue(int64(math.MinInt64))
			_, err := smallValue.AsInt()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "overflows int")
		})

		Convey("should handle non-integer float64 values", func() {
			floatValue := NewValue(3.14)
			_, err := floatValue.AsInt()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "is not an integer")
		})

		Convey("should handle string/bool/slice/map types", func() {
			testCases := []interface{}{
				"string",
				true,
				[]interface{}{1, 2, 3},
				map[string]interface{}{"key": "value"},
			}

			for _, tc := range testCases {
				value := NewValue(tc)
				_, err := value.AsInt()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "cannot convert")
			}
		})
	})
}

func TestValueImpl_AsInt64_EdgeCases(t *testing.T) {
	Convey("AsInt64 edge cases", t, func() {

		Convey("should handle non-integer float64 values", func() {
			floatValue := NewValue(3.14)
			_, err := floatValue.AsInt64()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "is not an integer")
		})

		Convey("should convert whole number floats correctly", func() {
			floatValue := NewValue(42.0)
			result, err := floatValue.AsInt64()
			So(err, ShouldBeNil)
			So(result, ShouldEqual, int64(42))
		})

		Convey("should handle unsupported types", func() {
			testCases := []interface{}{
				"string",
				true,
				[]interface{}{1, 2, 3},
				map[string]interface{}{"key": "value"},
			}

			for _, tc := range testCases {
				value := NewValue(tc)
				_, err := value.AsInt64()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "cannot convert")
			}
		})
	})
}

func TestValueImpl_AsFloat64_EdgeCases(t *testing.T) {
	Convey("AsFloat64 edge cases", t, func() {

		Convey("should convert all numeric types", func() {
			testCases := []struct {
				input    interface{}
				expected float64
			}{
				{42, 42.0},
				{int64(42), 42.0},
				{3.14, 3.14},
			}

			for _, tc := range testCases {
				value := NewValue(tc.input)
				result, err := value.AsFloat64()
				So(err, ShouldBeNil)
				So(result, ShouldEqual, tc.expected)
			}
		})

		Convey("should handle unsupported types", func() {
			testCases := []interface{}{
				"string",
				true,
				[]interface{}{1, 2, 3},
				map[string]interface{}{"key": "value"},
			}

			for _, tc := range testCases {
				value := NewValue(tc)
				_, err := value.AsFloat64()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "cannot convert")
			}
		})
	})
}

func TestValueImpl_AsBool_EdgeCases(t *testing.T) {
	Convey("AsBool edge cases", t, func() {

		Convey("should handle bool values", func() {
			trueValue := NewValue(true)
			result, err := trueValue.AsBool()
			So(err, ShouldBeNil)
			So(result, ShouldBeTrue)

			falseValue := NewValue(false)
			result, err = falseValue.AsBool()
			So(err, ShouldBeNil)
			So(result, ShouldBeFalse)
		})

		Convey("should handle unsupported types", func() {
			testCases := []interface{}{
				"string",
				42,
				[]interface{}{1, 2, 3},
				map[string]interface{}{"key": "value"},
			}

			for _, tc := range testCases {
				value := NewValue(tc)
				_, err := value.AsBool()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "cannot convert")
			}
		})
	})
}

func TestValueImpl_AsSlice_EdgeCases(t *testing.T) {
	Convey("AsSlice edge cases", t, func() {

		Convey("should handle slice values", func() {
			slice := []interface{}{"a", "b", "c"}
			value := NewValue(slice)
			result, err := value.AsSlice()
			So(err, ShouldBeNil)
			So(result, ShouldResemble, slice)
		})

		Convey("should handle unsupported types", func() {
			testCases := []interface{}{
				"string",
				42,
				true,
				map[string]interface{}{"key": "value"},
			}

			for _, tc := range testCases {
				value := NewValue(tc)
				_, err := value.AsSlice()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "cannot convert")
			}
		})
	})
}

func TestValueImpl_AsMap_EdgeCases(t *testing.T) {
	Convey("AsMap edge cases", t, func() {

		Convey("should handle map values", func() {
			mapData := map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			}
			value := NewValue(mapData)
			result, err := value.AsMap()
			So(err, ShouldBeNil)
			So(result, ShouldResemble, mapData)
		})

		Convey("should handle unsupported types", func() {
			testCases := []interface{}{
				"string",
				42,
				true,
				[]interface{}{"a", "b"},
			}

			for _, tc := range testCases {
				value := NewValue(tc)
				_, err := value.AsMap()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "cannot convert")
			}
		})
	})
}

func TestValueImpl_NilHandling(t *testing.T) {
	Convey("Nil value handling", t, func() {
		nilValue := NewValue(nil)

		Convey("should handle nil for AsString", func() {
			_, err := nilValue.AsString()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "cannot convert nil to string")
		})

		Convey("should handle nil for AsInt", func() {
			_, err := nilValue.AsInt()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "cannot convert nil to int")
		})

		Convey("should handle nil for AsInt64", func() {
			_, err := nilValue.AsInt64()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "cannot convert nil to int64")
		})

		Convey("should handle nil for AsFloat64", func() {
			_, err := nilValue.AsFloat64()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "cannot convert nil to float64")
		})

		Convey("should handle nil for AsBool", func() {
			_, err := nilValue.AsBool()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "cannot convert nil to bool")
		})

		Convey("should handle nil for AsSlice", func() {
			_, err := nilValue.AsSlice()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "cannot convert nil to slice")
		})

		Convey("should handle nil for AsMap", func() {
			_, err := nilValue.AsMap()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "cannot convert nil to map")
		})
	})
}
