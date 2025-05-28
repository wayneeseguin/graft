package graft

import (
	"testing"
)

func TestDocumentTypedMethods(t *testing.T) {
	// Setup test data
	data := map[interface{}]interface{}{
		"string":   "hello",
		"int":      42,
		"int64":    int64(9223372036854775807),
		"float":    3.14,
		"bool":     true,
		"slice":    []interface{}{"a", "b", "c"},
		"strSlice": []interface{}{"one", "two", "three"},
		"map": map[interface{}]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
		"strMap": map[interface{}]interface{}{
			"foo": "bar",
			"baz": "qux",
		},
	}
	doc := NewDocument(data)

	// Test GetString
	t.Run("GetString", func(t *testing.T) {
		val, err := doc.GetString("string")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != "hello" {
			t.Errorf("expected 'hello', got '%s'", val)
		}

		// Test error case
		_, err = doc.GetString("int")
		if err == nil {
			t.Error("expected error for non-string value")
		}
	})

	// Test GetInt
	t.Run("GetInt", func(t *testing.T) {
		val, err := doc.GetInt("int")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != 42 {
			t.Errorf("expected 42, got %d", val)
		}

		// Test error case
		_, err = doc.GetInt("string")
		if err == nil {
			t.Error("expected error for non-integer value")
		}
	})

	// Test GetInt64
	t.Run("GetInt64", func(t *testing.T) {
		val, err := doc.GetInt64("int64")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != int64(9223372036854775807) {
			t.Errorf("expected 9223372036854775807, got %d", val)
		}

		// Test conversion from int
		val, err = doc.GetInt64("int")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != int64(42) {
			t.Errorf("expected 42, got %d", val)
		}
	})

	// Test GetFloat64
	t.Run("GetFloat64", func(t *testing.T) {
		val, err := doc.GetFloat64("float")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != 3.14 {
			t.Errorf("expected 3.14, got %f", val)
		}

		// Test conversion from int
		val, err = doc.GetFloat64("int")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != 42.0 {
			t.Errorf("expected 42.0, got %f", val)
		}
	})

	// Test GetBool
	t.Run("GetBool", func(t *testing.T) {
		val, err := doc.GetBool("bool")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != true {
			t.Errorf("expected true, got %v", val)
		}

		// Test error case
		_, err = doc.GetBool("string")
		if err == nil {
			t.Error("expected error for non-boolean value")
		}
	})

	// Test GetMap
	t.Run("GetMap", func(t *testing.T) {
		val, err := doc.GetMap("map")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(val) != 2 {
			t.Errorf("expected map with 2 elements, got %d", len(val))
		}
		if val["key1"] != "value1" {
			t.Errorf("expected 'value1' for key1, got '%v'", val["key1"])
		}

		// Test error case
		_, err = doc.GetMap("string")
		if err == nil {
			t.Error("expected error for non-map value")
		}
	})

	// Test GetSlice
	t.Run("GetSlice", func(t *testing.T) {
		val, err := doc.GetSlice("slice")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(val) != 3 {
			t.Errorf("expected slice with 3 elements, got %d", len(val))
		}
		if val[0] != "a" {
			t.Errorf("expected 'a' as first element, got '%v'", val[0])
		}

		// Test error case
		_, err = doc.GetSlice("string")
		if err == nil {
			t.Error("expected error for non-slice value")
		}
	})

	// Test GetStringSlice
	t.Run("GetStringSlice", func(t *testing.T) {
		val, err := doc.GetStringSlice("strSlice")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(val) != 3 {
			t.Errorf("expected slice with 3 elements, got %d", len(val))
		}
		if val[0] != "one" {
			t.Errorf("expected 'one' as first element, got '%s'", val[0])
		}

		// Test error case with mixed types
		mixedData := map[interface{}]interface{}{
			"mixed": []interface{}{"string", 123, true},
		}
		mixedDoc := NewDocument(mixedData)
		_, err = mixedDoc.GetStringSlice("mixed")
		if err == nil {
			t.Error("expected error for slice with non-string elements")
		}
	})

	// Test GetMapStringString
	t.Run("GetMapStringString", func(t *testing.T) {
		val, err := doc.GetMapStringString("strMap")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(val) != 2 {
			t.Errorf("expected map with 2 elements, got %d", len(val))
		}
		if val["foo"] != "bar" {
			t.Errorf("expected 'bar' for key 'foo', got '%s'", val["foo"])
		}

		// Test error case with mixed value types
		mixedData := map[interface{}]interface{}{
			"mixed": map[interface{}]interface{}{
				"str": "value",
				"num": 123,
			},
		}
		mixedDoc := NewDocument(mixedData)
		_, err = mixedDoc.GetMapStringString("mixed")
		if err == nil {
			t.Error("expected error for map with non-string values")
		}
	})
}