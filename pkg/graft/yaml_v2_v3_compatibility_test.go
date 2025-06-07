package graft

import (
	"encoding/json"
	"testing"

	yamlv2 "gopkg.in/yaml.v2"
	yamlv3 "gopkg.in/yaml.v3"
	. "github.com/smartystreets/goconvey/convey"
)

// TestYAMLv2vsv3Compatibility tests compatibility between yaml.v2 and yaml.v3
func TestYAMLv2vsv3Compatibility(t *testing.T) {
	Convey("YAML v2 vs v3 Compatibility Analysis", t, func() {
		
		Convey("Map Type Differences", func() {
			yamlData := `
name: test
nested:
  key: value
  number: 42
`
			var v2Result interface{}
			var v3Result interface{}
			
			err := yamlv2.Unmarshal([]byte(yamlData), &v2Result)
			So(err, ShouldBeNil)
			
			err = yamlv3.Unmarshal([]byte(yamlData), &v3Result)
			So(err, ShouldBeNil)
			
			// v2 returns map[interface{}]interface{}
			v2Map, v2Ok := v2Result.(map[interface{}]interface{})
			So(v2Ok, ShouldBeTrue)
			
			// v3 returns map[string]interface{}
			v3Map, v3Ok := v3Result.(map[string]interface{})
			So(v3Ok, ShouldBeTrue)
			
			// Values should be the same when accessed properly
			So(v2Map["name"], ShouldEqual, v3Map["name"])
			
			v2Nested := v2Map["nested"].(map[interface{}]interface{})
			v3Nested := v3Map["nested"].(map[string]interface{})
			So(v2Nested["key"], ShouldEqual, v3Nested["key"])
			So(v2Nested["number"], ShouldEqual, v3Nested["number"])
		})
		
		Convey("Boolean Value Compatibility", func() {
			testCases := []struct {
				yaml string
				desc string
			}{
				{`value: true`, "YAML 1.2 true"},
				{`value: false`, "YAML 1.2 false"},
				{`value: yes`, "YAML 1.1 yes"},
				{`value: no`, "YAML 1.1 no"},
				{`value: on`, "YAML 1.1 on"},
				{`value: off`, "YAML 1.1 off"},
			}
			
			for _, tc := range testCases {
				Convey("Testing "+tc.desc, func() {
					var v2Result map[interface{}]interface{}
					var v3Result map[string]interface{}
					
					err := yamlv2.Unmarshal([]byte(tc.yaml), &v2Result)
					So(err, ShouldBeNil)
					
					err = yamlv3.Unmarshal([]byte(tc.yaml), &v3Result)
					So(err, ShouldBeNil)
					
					v2Value := v2Result["value"]
					v3Value := v3Result["value"]
					
					// Document any differences
					if v2Value != v3Value {
						t.Logf("DIFFERENCE in %s: v2=%v (%T), v3=%v (%T)", 
							tc.desc, v2Value, v2Value, v3Value, v3Value)
					}
					
					// For YAML 1.2, they should be the same
					if tc.yaml == `value: true` || tc.yaml == `value: false` {
						So(v2Value, ShouldEqual, v3Value)
					}
				})
			}
		})
		
		Convey("JSON Compatibility Improvement", func() {
			yamlData := `
name: test
config:
  enabled: true
  count: 42
`
			var v2Result interface{}
			var v3Result interface{}
			
			err := yamlv2.Unmarshal([]byte(yamlData), &v2Result)
			So(err, ShouldBeNil)
			
			err = yamlv3.Unmarshal([]byte(yamlData), &v3Result)
			So(err, ShouldBeNil)
			
			// v2 should fail direct JSON marshaling
			_, err = json.Marshal(v2Result)
			So(err, ShouldNotBeNil)
			
			// v3 should succeed with direct JSON marshaling
			jsonBytes, err := json.Marshal(v3Result)
			So(err, ShouldBeNil)
			So(jsonBytes, ShouldNotBeNil)
			
			// Verify JSON content is valid
			var jsonResult interface{}
			err = json.Unmarshal(jsonBytes, &jsonResult)
			So(err, ShouldBeNil)
		})
		
		Convey("Type Conversion Compatibility", func() {
			yamlData := `
number: 42
float: 3.14
string: "hello"
boolean: true
null_value: null
array: [1, 2, 3]
`
			var v2Result map[interface{}]interface{}
			var v3Result map[string]interface{}
			
			err := yamlv2.Unmarshal([]byte(yamlData), &v2Result)
			So(err, ShouldBeNil)
			
			err = yamlv3.Unmarshal([]byte(yamlData), &v3Result)
			So(err, ShouldBeNil)
			
			// Test that our conversion function can handle both
			v2Converted := convertToJSONCompatible(v2Result)
			v3Converted := convertToJSONCompatible(v3Result)
			
			// Both should be map[string]interface{} after conversion
			v2ConvertedMap, ok := v2Converted.(map[string]interface{})
			So(ok, ShouldBeTrue)
			
			v3ConvertedMap, ok := v3Converted.(map[string]interface{})
			So(ok, ShouldBeTrue)
			
			// Values should match after conversion
			So(v2ConvertedMap["number"], ShouldEqual, v3ConvertedMap["number"])
			So(v2ConvertedMap["float"], ShouldEqual, v3ConvertedMap["float"])
			So(v2ConvertedMap["string"], ShouldEqual, v3ConvertedMap["string"])
			So(v2ConvertedMap["boolean"], ShouldEqual, v3ConvertedMap["boolean"])
		})
		
		Convey("Error Handling Compatibility", func() {
			invalidYAML := `
name: test
  invalid: indentation
`
			var v2Result interface{}
			var v3Result interface{}
			
			v2Err := yamlv2.Unmarshal([]byte(invalidYAML), &v2Result)
			v3Err := yamlv3.Unmarshal([]byte(invalidYAML), &v3Result)
			
			// Both should return errors
			So(v2Err, ShouldNotBeNil)
			So(v3Err, ShouldNotBeNil)
			
			// Error messages might differ, but both should indicate YAML issues
			So(v2Err.Error(), ShouldContainSubstring, "yaml")
			So(v3Err.Error(), ShouldContainSubstring, "yaml")
		})
		
		Convey("Environment Variable Parsing Compatibility", func() {
			testCases := []string{
				`true`,
				`false`,
				`null`,
				`[1,2,3]`,
				`{"key":"value"}`,
				`plain string`,
			}
			
			for _, envValue := range testCases {
				Convey("Testing env value: "+envValue, func() {
					var v2Result interface{}
					var v3Result interface{}
					
					v2Err := yamlv2.Unmarshal([]byte(envValue), &v2Result)
					v3Err := yamlv3.Unmarshal([]byte(envValue), &v3Result)
					
					// Both should either succeed or fail consistently
					if v2Err != nil && v3Err != nil {
						// Both failed - that's fine for invalid YAML
						return
					}
					
					if v2Err == nil && v3Err == nil {
						// Both succeeded - compare results
						if v2Map, ok := v2Result.(map[interface{}]interface{}); ok {
							if v3Map, ok := v3Result.(map[string]interface{}); ok {
								// Compare map contents after conversion
								v2Conv := convertToJSONCompatible(v2Map)
								v3Conv := convertToJSONCompatible(v3Map)
								So(v2Conv, ShouldResemble, v3Conv)
								return
							}
						}
						
						// For non-map values, they should be identical
						So(v2Result, ShouldEqual, v3Result)
					} else {
						// One succeeded, one failed - document the difference
						t.Logf("PARSING DIFFERENCE for '%s': v2_err=%v, v3_err=%v", 
							envValue, v2Err, v3Err)
					}
				})
			}
		})
	})
}

// TestMigrationHelpers tests that our existing helper functions work with both versions
func TestMigrationHelpers(t *testing.T) {
	Convey("Migration Helper Function Compatibility", t, func() {
		
		Convey("convertToJSONCompatible works with both map types", func() {
			// Test with v2 style map
			v2Map := map[interface{}]interface{}{
				"string_key": "value",
				123:          "numeric_key",
			}
			
			// Test with v3 style map
			v3Map := map[string]interface{}{
				"string_key": "value",
				"123":        "numeric_key",
			}
			
			v2Result := convertToJSONCompatible(v2Map)
			v3Result := convertToJSONCompatible(v3Map)
			
			// Both should produce map[string]interface{}
			v2ResultMap, ok := v2Result.(map[string]interface{})
			So(ok, ShouldBeTrue)
			
			v3ResultMap, ok := v3Result.(map[string]interface{})
			So(ok, ShouldBeTrue)
			
			// Results should be equivalent
			So(v2ResultMap["string_key"], ShouldEqual, v3ResultMap["string_key"])
		})
		
		Convey("Document works with v3 maps through NewDocumentFromInterface", func() {
			yamlData := `
name: test
config:
  key: value
`
			// Test what happens when we use yaml.v3 data with current Document
			var v3Result map[string]interface{}
			err := yamlv3.Unmarshal([]byte(yamlData), &v3Result)
			So(err, ShouldBeNil)
			
			// Create a document with v3 data using the existing helper
			doc, err := NewDocumentFromInterface(v3Result)
			So(err, ShouldBeNil)
			
			// GetMap should work (it converts to map[string]interface{})
			resultMap, err := doc.GetMap("")
			So(err, ShouldBeNil)
			So(resultMap["name"], ShouldEqual, "test")
			
			configMap, err := doc.GetMap("config")
			So(err, ShouldBeNil)
			So(configMap["key"], ShouldEqual, "value")
		})
	})
}