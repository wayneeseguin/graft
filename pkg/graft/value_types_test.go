package graft

import (
	"testing"
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
