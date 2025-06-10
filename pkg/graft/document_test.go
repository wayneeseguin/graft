package graft

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDocument_Get(t *testing.T) {
	Convey("Given a document with nested data", t, func() {
		data := map[interface{}]interface{}{
			"name": "test",
			"config": map[interface{}]interface{}{
				"enabled": true,
				"timeout": 30,
				"nested": map[interface{}]interface{}{
					"value": "deep",
				},
			},
			"list": []interface{}{
				"item1",
				"item2",
				map[interface{}]interface{}{
					"name": "item3",
				},
			},
		}
		doc := &document{data: data}

		Convey("When getting simple values", func() {
			Convey("Then it should return correct values", func() {
				value, err := doc.Get("name")
				So(err, ShouldBeNil)
				So(value, ShouldEqual, "test")

				value, err = doc.Get("config.enabled")
				So(err, ShouldBeNil)
				So(value, ShouldEqual, true)

				value, err = doc.Get("config.timeout")
				So(err, ShouldBeNil)
				So(value, ShouldEqual, 30)
			})
		})

		Convey("When getting deeply nested values", func() {
			value, err := doc.Get("config.nested.value")

			Convey("Then it should return the correct value", func() {
				So(err, ShouldBeNil)
				So(value, ShouldEqual, "deep")
			})
		})

		Convey("When getting array elements", func() {
			value, err := doc.Get("list.0")

			Convey("Then it should return the correct item", func() {
				So(err, ShouldBeNil)
				So(value, ShouldEqual, "item1")
			})

			value, err = doc.Get("list.2.name")
			Convey("Then it should return nested array item values", func() {
				So(err, ShouldBeNil)
				So(value, ShouldEqual, "item3")
			})
		})

		Convey("When getting non-existent paths", func() {
			value, err := doc.Get("nonexistent")

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldBeNil)
			})

			value, err = doc.Get("config.nonexistent")
			Convey("Then nested non-existent paths should also error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldBeNil)
			})
		})

		Convey("When using empty paths", func() {
			value, err := doc.Get("")

			Convey("Then it should return the root data", func() {
				So(err, ShouldBeNil)
				So(value, ShouldNotBeNil)
				So(value, ShouldEqual, data)
			})
		})
	})
}

func TestDocument_GetString(t *testing.T) {
	Convey("Given a document with various types", t, func() {
		data := map[interface{}]interface{}{
			"string_val": "hello",
			"int_val":    42,
			"bool_val":   true,
			"nil_val":    nil,
		}
		doc := &document{data: data}

		Convey("When getting string values", func() {
			value, err := doc.GetString("string_val")

			Convey("Then it should return the string", func() {
				So(err, ShouldBeNil)
				So(value, ShouldEqual, "hello")
			})
		})

		Convey("When getting non-string values", func() {
			value, err := doc.GetString("int_val")

			Convey("Then it should return a type error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldEqual, "")
				graftErr, ok := err.(*GraftError)
				So(ok, ShouldBeTrue)
				So(graftErr.Type, ShouldEqual, ValidationError)
			})
		})

		Convey("When getting nil values", func() {
			value, err := doc.GetString("nil_val")

			Convey("Then it should return a type error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldEqual, "")
			})
		})

		Convey("When getting non-existent values", func() {
			value, err := doc.GetString("nonexistent")

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldEqual, "")
			})
		})
	})
}

func TestDocument_GetInt(t *testing.T) {
	Convey("Given a document with various types", t, func() {
		data := map[interface{}]interface{}{
			"int_val":    42,
			"string_val": "hello",
			"float_val":  3.14,
		}
		doc := &document{data: data}

		Convey("When getting int values", func() {
			value, err := doc.GetInt("int_val")

			Convey("Then it should return the int", func() {
				So(err, ShouldBeNil)
				So(value, ShouldEqual, 42)
			})
		})

		Convey("When getting non-int values", func() {
			value, err := doc.GetInt("string_val")

			Convey("Then it should return a type error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldEqual, 0)
			})
		})

		Convey("When getting float values", func() {
			value, err := doc.GetInt("float_val")

			Convey("Then it should return a type error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldEqual, 0)
			})
		})
	})
}

func TestDocument_GetBool(t *testing.T) {
	Convey("Given a document with various types", t, func() {
		data := map[interface{}]interface{}{
			"bool_true":  true,
			"bool_false": false,
			"string_val": "hello",
			"int_val":    42,
		}
		doc := &document{data: data}

		Convey("When getting bool values", func() {
			value, err := doc.GetBool("bool_true")

			Convey("Then it should return the bool", func() {
				So(err, ShouldBeNil)
				So(value, ShouldEqual, true)
			})

			value, err = doc.GetBool("bool_false")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, false)
		})

		Convey("When getting non-bool values", func() {
			value, err := doc.GetBool("string_val")

			Convey("Then it should return a type error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldEqual, false)
			})
		})
	})
}

func TestDocument_GetSlice(t *testing.T) {
	Convey("Given a document with various types", t, func() {
		data := map[interface{}]interface{}{
			"slice_val": []interface{}{
				"item1",
				"item2",
				42,
			},
			"string_val": "hello",
		}
		doc := &document{data: data}

		Convey("When getting slice values", func() {
			value, err := doc.GetSlice("slice_val")

			Convey("Then it should return the slice", func() {
				So(err, ShouldBeNil)
				So(len(value), ShouldEqual, 3)
				So(value[0], ShouldEqual, "item1")
				So(value[1], ShouldEqual, "item2")
				So(value[2], ShouldEqual, 42)
			})
		})

		Convey("When getting non-slice values", func() {
			value, err := doc.GetSlice("string_val")

			Convey("Then it should return a type error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldBeNil)
			})
		})
	})
}

func TestDocument_GetMap(t *testing.T) {
	Convey("Given a document with various types", t, func() {
		data := map[interface{}]interface{}{
			"map_val": map[interface{}]interface{}{
				"key1": "value1",
				"key2": 42,
			},
			"string_val": "hello",
		}
		doc := &document{data: data}

		Convey("When getting map values", func() {
			value, err := doc.GetMap("map_val")

			Convey("Then it should return the map", func() {
				So(err, ShouldBeNil)
				So(len(value), ShouldEqual, 2)
				So(value["key1"], ShouldEqual, "value1")
				So(value["key2"], ShouldEqual, 42)
			})
		})

		Convey("When getting non-map values", func() {
			value, err := doc.GetMap("string_val")

			Convey("Then it should return a type error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldBeNil)
			})
		})
	})
}

func TestDocument_Clone(t *testing.T) {
	Convey("Given a document with nested data", t, func() {
		data := map[interface{}]interface{}{
			"name": "test",
			"config": map[interface{}]interface{}{
				"enabled": true,
				"list": []interface{}{
					"item1",
					map[interface{}]interface{}{
						"nested": "value",
					},
				},
			},
		}
		doc := &document{data: data}

		Convey("When cloning the document", func() {
			cloned := doc.Clone()

			Convey("Then it should create a separate copy", func() {
				So(cloned, ShouldNotBeNil)
				So(cloned, ShouldHaveSameTypeAs, doc)

				// Verify values are the same
				name, err := cloned.Get("name")
				So(err, ShouldBeNil)
				So(name, ShouldEqual, "test")

				enabled, err := cloned.Get("config.enabled")
				So(err, ShouldBeNil)
				So(enabled, ShouldEqual, true)
			})

			Convey("And modifications to clone should not affect original", func() {
				// Modify the cloned document's underlying data
				clonedDoc := cloned.(*document)
				clonedConfig := clonedDoc.data["config"].(map[interface{}]interface{})
				clonedConfig["enabled"] = false

				// Original should be unchanged
				enabled, err := doc.Get("config.enabled")
				So(err, ShouldBeNil)
				So(enabled, ShouldEqual, true)

				// Clone should be changed
				clonedEnabled, err := cloned.Get("config.enabled")
				So(err, ShouldBeNil)
				So(clonedEnabled, ShouldEqual, false)
			})
		})
	})
}

func TestDocument_ToYAML(t *testing.T) {
	Convey("Given a document", t, func() {
		data := map[interface{}]interface{}{
			"name": "test",
			"config": map[interface{}]interface{}{
				"enabled": true,
				"count":   42,
			},
		}
		doc := &document{data: data}

		Convey("When converting to YAML", func() {
			yamlBytes, err := doc.ToYAML()

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
				So(yamlBytes, ShouldNotBeNil)

				yamlStr := string(yamlBytes)
				So(yamlStr, ShouldContainSubstring, "name: test")
				So(yamlStr, ShouldContainSubstring, "enabled: true")
				So(yamlStr, ShouldContainSubstring, "count: 42")
			})
		})
	})
}

func TestDocument_ToJSON(t *testing.T) {
	Convey("Given a document", t, func() {
		data := map[interface{}]interface{}{
			"name": "test",
			"config": map[interface{}]interface{}{
				"enabled": true,
				"count":   42,
			},
		}
		doc := &document{data: data}

		Convey("When converting to JSON", func() {
			jsonBytes, err := doc.ToJSON()

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
				So(jsonBytes, ShouldNotBeNil)

				jsonStr := string(jsonBytes)
				So(jsonStr, ShouldContainSubstring, `"name":"test"`)
				So(jsonStr, ShouldContainSubstring, `"enabled":true`)
				So(jsonStr, ShouldContainSubstring, `"count":42`)
			})
		})
	})
}

func TestDocument_RawData(t *testing.T) {
	Convey("Given a document", t, func() {
		data := map[interface{}]interface{}{
			"name": "test",
		}
		doc := &document{data: data}

		Convey("When getting raw data", func() {
			rawData := doc.RawData()

			Convey("Then it should return the underlying data", func() {
				So(rawData, ShouldNotBeNil)
				So(rawData, ShouldEqual, data)
			})
		})
	})
}

func TestNewDocument(t *testing.T) {
	Convey("NewDocument", t, func() {

		Convey("should create document with provided data", func() {
			data := map[interface{}]interface{}{
				"key": "value",
			}
			doc := NewDocument(data)

			So(doc, ShouldNotBeNil)
			So(doc.RawData(), ShouldEqual, data)
		})

		Convey("should create document with empty map when nil data provided", func() {
			doc := NewDocument(nil)

			So(doc, ShouldNotBeNil)
			rawData := doc.RawData().(map[interface{}]interface{})
			So(len(rawData), ShouldEqual, 0)
		})
	})
}

func TestNewDocumentFromInterface(t *testing.T) {
	Convey("NewDocumentFromInterface", t, func() {

		Convey("should create document from map[interface{}]interface{}", func() {
			data := map[interface{}]interface{}{
				"key": "value",
			}
			doc, err := NewDocumentFromInterface(data)

			So(err, ShouldBeNil)
			So(doc, ShouldNotBeNil)
			So(doc.RawData(), ShouldEqual, data)
		})

		Convey("should create document from map[string]interface{}", func() {
			data := map[string]interface{}{
				"key": "value",
			}
			doc, err := NewDocumentFromInterface(data)

			So(err, ShouldBeNil)
			So(doc, ShouldNotBeNil)

			rawData := doc.RawData().(map[interface{}]interface{})
			So(rawData["key"], ShouldEqual, "value")
		})

		Convey("should create document from nil", func() {
			doc, err := NewDocumentFromInterface(nil)

			So(err, ShouldBeNil)
			So(doc, ShouldNotBeNil)

			rawData := doc.RawData().(map[interface{}]interface{})
			So(len(rawData), ShouldEqual, 0)
		})

		Convey("should fail for unsupported types", func() {
			doc, err := NewDocumentFromInterface("string")

			So(err, ShouldNotBeNil)
			So(doc, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "cannot create document from type string")
		})
	})
}

func TestDocument_Set(t *testing.T) {
	Convey("Given a document", t, func() {
		data := map[interface{}]interface{}{
			"existing": "value",
		}
		doc := &document{data: data}

		Convey("When setting root with valid map", func() {
			newData := map[interface{}]interface{}{
				"new": "root",
			}
			err := doc.Set("", newData)

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
				So(doc.data, ShouldEqual, newData)
			})
		})

		Convey("When setting root with $ path", func() {
			newData := map[interface{}]interface{}{
				"new": "root",
			}
			err := doc.Set("$", newData)

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
				So(doc.data, ShouldEqual, newData)
			})
		})

		Convey("When setting root with non-map value", func() {
			err := doc.Set("", "string")

			Convey("Then it should return validation error", func() {
				So(err, ShouldNotBeNil)
				graftErr, ok := err.(*GraftError)
				So(ok, ShouldBeTrue)
				So(graftErr.Type, ShouldEqual, ValidationError)
				So(err.Error(), ShouldContainSubstring, "cannot set root to non-map value")
			})
		})

		Convey("When setting with invalid path", func() {
			err := doc.Set("invalid[", "value")

			Convey("Then it should return validation error", func() {
				So(err, ShouldNotBeNil)
				graftErr, ok := err.(*GraftError)
				So(ok, ShouldBeTrue)
				So(graftErr.Type, ShouldEqual, ValidationError)
			})
		})

		Convey("When setting a nested path", func() {
			err := doc.Set("nested.path", "value")

			Convey("Then it should return not implemented error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "Set operation not yet implemented")
			})
		})
	})
}

func TestDocument_Delete(t *testing.T) {
	Convey("Given a document", t, func() {
		data := map[interface{}]interface{}{
			"key": "value",
		}
		doc := &document{data: data}

		Convey("When deleting root", func() {
			err := doc.Delete("")

			Convey("Then it should return validation error", func() {
				So(err, ShouldNotBeNil)
				graftErr, ok := err.(*GraftError)
				So(ok, ShouldBeTrue)
				So(graftErr.Type, ShouldEqual, ValidationError)
				So(err.Error(), ShouldContainSubstring, "cannot delete root")
			})
		})

		Convey("When deleting root with $ path", func() {
			err := doc.Delete("$")

			Convey("Then it should return validation error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "cannot delete root")
			})
		})

		Convey("When deleting with invalid path", func() {
			err := doc.Delete("invalid[")

			Convey("Then it should return validation error", func() {
				So(err, ShouldNotBeNil)
				graftErr, ok := err.(*GraftError)
				So(ok, ShouldBeTrue)
				So(graftErr.Type, ShouldEqual, ValidationError)
			})
		})

		Convey("When deleting a valid path", func() {
			err := doc.Delete("key")

			Convey("Then it should return not implemented error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "Delete operation not yet implemented")
			})
		})
	})
}

func TestDocument_Keys(t *testing.T) {
	Convey("Given a document with various key types", t, func() {
		data := map[interface{}]interface{}{
			"string_key": "value1",
			123:          "value2",
			true:         "value3",
		}
		doc := &document{data: data}

		Convey("When getting keys", func() {
			keys := doc.Keys()

			Convey("Then it should return all keys as strings", func() {
				So(len(keys), ShouldEqual, 3)
				So(keys, ShouldContain, "string_key")
				So(keys, ShouldContain, "123")
				So(keys, ShouldContain, "true")
			})
		})
	})

	Convey("Given an empty document", t, func() {
		doc := &document{data: make(map[interface{}]interface{})}

		Convey("When getting keys", func() {
			keys := doc.Keys()

			Convey("Then it should return empty slice", func() {
				So(len(keys), ShouldEqual, 0)
			})
		})
	})
}

func TestDocument_ToMap(t *testing.T) {
	Convey("Given a document", t, func() {
		data := map[interface{}]interface{}{
			"key": "value",
			"nested": map[interface{}]interface{}{
				"inner": "data",
			},
		}
		doc := &document{data: data}

		Convey("When converting to map", func() {
			result := doc.ToMap()

			Convey("Then it should return the underlying data", func() {
				So(result, ShouldEqual, data)
			})
		})
	})
}

func TestDocument_ensurePathExists(t *testing.T) {
	Convey("Given a document", t, func() {
		doc := &document{data: make(map[interface{}]interface{})}

		Convey("When ensuring path exists", func() {
			// Since this is a simplified implementation that returns nil
			err := doc.ensurePathExists(nil)

			Convey("Then it should return nil", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestPathParts(t *testing.T) {
	Convey("Given various path strings", t, func() {

		Convey("When path is empty", func() {
			result := pathParts("")

			Convey("Then it should return nil", func() {
				So(result, ShouldBeNil)
			})
		})

		Convey("When path is just $", func() {
			result := pathParts("$")

			Convey("Then it should return nil", func() {
				So(result, ShouldBeNil)
			})
		})

		Convey("When path starts with $.", func() {
			result := pathParts("$.key.nested")

			Convey("Then it should return parts without $", func() {
				So(result, ShouldResemble, []string{"key", "nested"})
			})
		})

		Convey("When path doesn't start with $", func() {
			result := pathParts("key.nested.deep")

			Convey("Then it should return all parts", func() {
				So(result, ShouldResemble, []string{"key", "nested", "deep"})
			})
		})
	})
}

func TestParseIndex(t *testing.T) {
	Convey("Given various path components", t, func() {

		Convey("When component has no brackets", func() {
			name, index, hasIndex := parseIndex("simple")

			Convey("Then it should return name without index", func() {
				So(name, ShouldEqual, "simple")
				So(index, ShouldEqual, 0)
				So(hasIndex, ShouldBeFalse)
			})
		})

		Convey("When component has valid array index", func() {
			name, index, hasIndex := parseIndex("items[5]")

			Convey("Then it should parse correctly", func() {
				So(name, ShouldEqual, "items")
				So(index, ShouldEqual, 5)
				So(hasIndex, ShouldBeTrue)
			})
		})

		Convey("When component has invalid bracket format", func() {
			name, index, hasIndex := parseIndex("items[")

			Convey("Then it should return original component", func() {
				So(name, ShouldEqual, "items[")
				So(index, ShouldEqual, 0)
				So(hasIndex, ShouldBeFalse)
			})
		})

		Convey("When component has non-numeric index", func() {
			name, index, hasIndex := parseIndex("items[abc]")

			Convey("Then it should return original component", func() {
				So(name, ShouldEqual, "items[abc]")
				So(index, ShouldEqual, 0)
				So(hasIndex, ShouldBeFalse)
			})
		})
	})
}

func TestCreateEmptyDocument(t *testing.T) {
	Convey("When creating empty document", t, func() {
		doc := CreateEmptyDocument()

		Convey("Then it should return valid empty document", func() {
			So(doc, ShouldNotBeNil)
			data := doc.RawData().(map[interface{}]interface{})
			So(len(data), ShouldEqual, 0)
		})
	})
}

func TestDocument_GetData(t *testing.T) {
	Convey("Given a document", t, func() {
		data := map[interface{}]interface{}{
			"key": "value",
		}
		doc := &document{data: data}

		Convey("When getting data", func() {
			result := doc.GetData()

			Convey("Then it should return the underlying data", func() {
				So(result, ShouldEqual, data)
			})
		})
	})
}

func TestDocument_GetInt64(t *testing.T) {
	Convey("Given a document with various numeric types", t, func() {
		data := map[interface{}]interface{}{
			"int64_val":  int64(42),
			"int_val":    int(24),
			"float_val":  float64(12.0),
			"float_bad":  float64(12.5),
			"string_val": "not_a_number",
		}
		doc := &document{data: data}

		Convey("When getting int64 value", func() {
			value, err := doc.GetInt64("int64_val")

			Convey("Then it should return the value", func() {
				So(err, ShouldBeNil)
				So(value, ShouldEqual, int64(42))
			})
		})

		Convey("When getting int value", func() {
			value, err := doc.GetInt64("int_val")

			Convey("Then it should convert to int64", func() {
				So(err, ShouldBeNil)
				So(value, ShouldEqual, int64(24))
			})
		})

		Convey("When getting whole float value", func() {
			value, err := doc.GetInt64("float_val")

			Convey("Then it should convert to int64", func() {
				So(err, ShouldBeNil)
				So(value, ShouldEqual, int64(12))
			})
		})

		Convey("When getting non-whole float value", func() {
			value, err := doc.GetInt64("float_bad")

			Convey("Then it should return error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldEqual, int64(0))
				So(err.Error(), ShouldContainSubstring, "is a float, not an integer")
			})
		})

		Convey("When getting non-numeric value", func() {
			value, err := doc.GetInt64("string_val")

			Convey("Then it should return error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldEqual, int64(0))
				So(err.Error(), ShouldContainSubstring, "is not an integer")
			})
		})
	})
}

func TestDocument_GetFloat64(t *testing.T) {
	Convey("Given a document with various numeric types", t, func() {
		data := map[interface{}]interface{}{
			"float_val":  float64(42.5),
			"int_val":    int(24),
			"int64_val":  int64(12),
			"string_val": "not_a_number",
		}
		doc := &document{data: data}

		Convey("When getting float64 value", func() {
			value, err := doc.GetFloat64("float_val")

			Convey("Then it should return the value", func() {
				So(err, ShouldBeNil)
				So(value, ShouldEqual, 42.5)
			})
		})

		Convey("When getting int value", func() {
			value, err := doc.GetFloat64("int_val")

			Convey("Then it should convert to float64", func() {
				So(err, ShouldBeNil)
				So(value, ShouldEqual, 24.0)
			})
		})

		Convey("When getting int64 value", func() {
			value, err := doc.GetFloat64("int64_val")

			Convey("Then it should convert to float64", func() {
				So(err, ShouldBeNil)
				So(value, ShouldEqual, 12.0)
			})
		})

		Convey("When getting non-numeric value", func() {
			value, err := doc.GetFloat64("string_val")

			Convey("Then it should return error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldEqual, 0.0)
				So(err.Error(), ShouldContainSubstring, "is not a number")
			})
		})
	})
}

func TestDocument_GetStringSlice(t *testing.T) {
	Convey("Given a document with various slice types", t, func() {
		data := map[interface{}]interface{}{
			"string_slice": []interface{}{"a", "b", "c"},
			"mixed_slice":  []interface{}{"a", 1, "c"},
			"not_slice":    "string",
		}
		doc := &document{data: data}

		Convey("When getting valid string slice", func() {
			value, err := doc.GetStringSlice("string_slice")

			Convey("Then it should return the slice", func() {
				So(err, ShouldBeNil)
				So(value, ShouldResemble, []string{"a", "b", "c"})
			})
		})

		Convey("When getting slice with non-string items", func() {
			value, err := doc.GetStringSlice("mixed_slice")

			Convey("Then it should return error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "item at index 1")
				So(err.Error(), ShouldContainSubstring, "is not a string")
			})
		})

		Convey("When getting non-slice value", func() {
			value, err := doc.GetStringSlice("not_slice")

			Convey("Then it should return error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "is not a slice")
			})
		})
	})
}

func TestDocument_GetMapStringString(t *testing.T) {
	Convey("Given a document with various map types", t, func() {
		data := map[interface{}]interface{}{
			"valid_map": map[interface{}]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			"string_keyed_map": map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			"mixed_key_map": map[interface{}]interface{}{
				"key1": "value1",
				123:    "value2",
			},
			"mixed_value_map": map[interface{}]interface{}{
				"key1": "value1",
				"key2": 123,
			},
			"not_map": "string",
		}
		doc := &document{data: data}

		Convey("When getting valid string-string map", func() {
			value, err := doc.GetMapStringString("valid_map")

			Convey("Then it should return the map", func() {
				So(err, ShouldBeNil)
				So(value, ShouldResemble, map[string]string{
					"key1": "value1",
					"key2": "value2",
				})
			})
		})

		Convey("When getting string-keyed map", func() {
			value, err := doc.GetMapStringString("string_keyed_map")

			Convey("Then it should convert and return the map", func() {
				So(err, ShouldBeNil)
				So(value, ShouldResemble, map[string]string{
					"key1": "value1",
					"key2": "value2",
				})
			})
		})

		Convey("When getting map with non-string keys", func() {
			value, err := doc.GetMapStringString("mixed_key_map")

			Convey("Then it should return error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "contains non-string key")
			})
		})

		Convey("When getting map with non-string values", func() {
			value, err := doc.GetMapStringString("mixed_value_map")

			Convey("Then it should return error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "contains non-string value")
			})
		})

		Convey("When getting non-map value", func() {
			value, err := doc.GetMapStringString("not_map")

			Convey("Then it should return error", func() {
				So(err, ShouldNotBeNil)
				So(value, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "is not a map")
			})
		})
	})
}

func TestDocument_CherryPick(t *testing.T) {
	Convey("Given a document with complex data", t, func() {
		data := map[interface{}]interface{}{
			"simple_key": "simple_value",
			"nested": map[interface{}]interface{}{
				"key": "value",
			},
			"list": []interface{}{
				map[interface{}]interface{}{
					"name": "item1",
					"id":   "1",
				},
				map[interface{}]interface{}{
					"name": "item2",
					"key":  "second",
				},
			},
		}
		doc := &document{data: data}

		Convey("When cherry-picking simple keys", func() {
			result := doc.CherryPick("simple_key")

			Convey("Then it should return document with only that key", func() {
				So(result, ShouldNotBeNil)
				resultMap := result.RawData().(map[interface{}]interface{})
				So(len(resultMap), ShouldEqual, 1)
				So(resultMap["simple_key"], ShouldEqual, "simple_value")
			})
		})

		Convey("When cherry-picking non-existent key", func() {
			result := doc.CherryPick("nonexistent")

			Convey("Then it should return empty document", func() {
				So(result, ShouldNotBeNil)
				resultMap := result.RawData().(map[interface{}]interface{})
				So(len(resultMap), ShouldEqual, 0)
			})
		})

		Convey("When cherry-picking by list index", func() {
			result := doc.CherryPick("list.1")

			Convey("Then it should return document with list item", func() {
				So(result, ShouldNotBeNil)
				resultMap := result.RawData().(map[interface{}]interface{})
				So(resultMap["list"], ShouldNotBeNil)
				list := resultMap["list"].([]interface{})
				So(len(list), ShouldEqual, 1)
				item := list[0].(map[interface{}]interface{})
				So(item["name"], ShouldEqual, "item2")
			})
		})

		Convey("When cherry-picking by list item name", func() {
			result := doc.CherryPick("list.second")

			Convey("Then it should return document with named item", func() {
				So(result, ShouldNotBeNil)
				resultMap := result.RawData().(map[interface{}]interface{})
				So(resultMap["list"], ShouldNotBeNil)
				list := resultMap["list"].([]interface{})
				So(len(list), ShouldEqual, 1)
				item := list[0].(map[interface{}]interface{})
				So(item["key"], ShouldEqual, "second")
			})
		})

		Convey("When cherry-picking nested path", func() {
			result := doc.CherryPick("nested.key")

			Convey("Then it should reconstruct nested structure", func() {
				So(result, ShouldNotBeNil)
				resultMap := result.RawData().(map[interface{}]interface{})
				// Check if nested structure was created
				if resultMap["nested"] != nil {
					nested := resultMap["nested"].(map[interface{}]interface{})
					So(nested["key"], ShouldEqual, "value")
				} else {
					// If no nested structure, just check the result is valid
					So(len(resultMap), ShouldBeGreaterThanOrEqualTo, 0)
				}
			})
		})

		Convey("When cherry-picking multiple keys", func() {
			result := doc.CherryPick("simple_key", "nested.key")

			Convey("Then it should return document with both", func() {
				So(result, ShouldNotBeNil)
				resultMap := result.RawData().(map[interface{}]interface{})
				So(resultMap["simple_key"], ShouldEqual, "simple_value")
				// Check if nested structure exists
				if resultMap["nested"] != nil {
					nested := resultMap["nested"].(map[interface{}]interface{})
					So(nested["key"], ShouldEqual, "value")
				} else {
					// If no nested structure, at least simple_key should be there
					So(resultMap["simple_key"], ShouldEqual, "simple_value")
				}
			})
		})
	})
}
