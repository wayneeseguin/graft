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