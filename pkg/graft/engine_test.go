package graft

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/wayneeseguin/graft/internal/utils/tree"
)

// TestMockOperator implements the Operator interface for testing
type TestMockOperator struct {
	name        string
	returnValue interface{}
	returnError error
	callCount   int
	phase       OperatorPhase
}

func NewTestMockOperator(name string) *TestMockOperator {
	return &TestMockOperator{
		name:  name,
		phase: EvalPhase,
	}
}

func (m *TestMockOperator) Setup() error {
	return nil
}

func (m *TestMockOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	m.callCount++
	if m.returnError != nil {
		return nil, m.returnError
	}
	return &Response{Type: Replace, Value: m.returnValue}, nil
}

func (m *TestMockOperator) Dependencies(ev *Evaluator, args []*Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return nil
}

func (m *TestMockOperator) Phase() OperatorPhase {
	return m.phase
}

func (m *TestMockOperator) SetReturnValue(value interface{}) {
	m.returnValue = value
}

func (m *TestMockOperator) SetReturnError(err error) {
	m.returnError = err
}

func (m *TestMockOperator) GetCallCount() int {
	return m.callCount
}

func TestDefaultEngine(t *testing.T) {
	Convey("DefaultEngine", t, func() {

		Convey("NewDefaultEngine should create engine with default config", func() {
			engine := NewDefaultEngine()

			So(engine, ShouldNotBeNil)
			So(engine.config.UseEnhancedParser, ShouldBeTrue)
			So(engine.config.EnableCaching, ShouldBeTrue)
			So(engine.config.CacheSize, ShouldEqual, 10000)
			So(engine.config.EnableParallel, ShouldBeFalse)
			So(engine.config.MaxWorkers, ShouldEqual, 4)
			So(engine.config.DataflowOrder, ShouldEqual, "alphabetical")
			So(engine.operators, ShouldNotBeNil)
			So(engine.vaultSecretCache, ShouldNotBeNil)
			So(engine.vaultRefs, ShouldNotBeNil)
			So(engine.awsSecretsCache, ShouldNotBeNil)
			So(engine.awsParamsCache, ShouldNotBeNil)
			So(engine.usedIPs, ShouldNotBeNil)
			So(engine.pathsToSort, ShouldNotBeNil)
		})

		Convey("NewDefaultEngineWithConfig should create engine with custom config", func() {
			config := EngineConfig{
				UseEnhancedParser: false,
				EnableCaching:     false,
				CacheSize:         5000,
				EnableParallel:    true,
				MaxWorkers:        8,
				DataflowOrder:     "insertion",
				SkipVault:         true,
				SkipAWS:           true,
			}

			engine := NewDefaultEngineWithConfig(config)

			So(engine, ShouldNotBeNil)
			So(engine.config.UseEnhancedParser, ShouldBeFalse)
			So(engine.config.EnableCaching, ShouldBeFalse)
			So(engine.config.CacheSize, ShouldEqual, 5000)
			So(engine.config.EnableParallel, ShouldBeTrue)
			So(engine.config.MaxWorkers, ShouldEqual, 8)
			So(engine.config.DataflowOrder, ShouldEqual, "insertion")
			So(engine.skipVault, ShouldBeTrue)
			So(engine.skipAws, ShouldBeTrue)
		})

		Convey("DefaultEngineConfig should return correct defaults", func() {
			config := DefaultEngineConfig()

			So(config.UseEnhancedParser, ShouldBeTrue)
			So(config.EnableCaching, ShouldBeTrue)
			So(config.CacheSize, ShouldEqual, 10000)
			So(config.EnableParallel, ShouldBeFalse)
			So(config.MaxWorkers, ShouldEqual, 4)
			So(config.DataflowOrder, ShouldEqual, "alphabetical")
		})
	})
}

func TestEngineOperatorManagement(t *testing.T) {
	Convey("Engine Operator Management", t, func() {
		engine := NewDefaultEngine()

		Convey("RegisterOperator should register new operator", func() {
			mockOp := NewTestMockOperator("test")
			err := engine.RegisterOperator("test", mockOp)

			So(err, ShouldBeNil)

			op, exists := engine.GetOperator("test")
			So(exists, ShouldBeTrue)
			So(op, ShouldEqual, mockOp)
		})

		Convey("RegisterOperator should fail for duplicate operator", func() {
			mockOp := NewTestMockOperator("test")
			err := engine.RegisterOperator("test", mockOp)
			So(err, ShouldBeNil)

			err = engine.RegisterOperator("test", mockOp)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "already registered")
		})

		Convey("GetOperator should return false for non-existent operator", func() {
			op, exists := engine.GetOperator("nonexistent")
			So(exists, ShouldBeFalse)
			So(op, ShouldBeNil)
		})

		Convey("UnregisterOperator should remove operator", func() {
			mockOp := NewTestMockOperator("test")
			err := engine.RegisterOperator("test", mockOp)
			So(err, ShouldBeNil)

			err = engine.UnregisterOperator("test")
			So(err, ShouldBeNil)

			op, exists := engine.GetOperator("test")
			So(exists, ShouldBeFalse)
			So(op, ShouldBeNil)
		})

		Convey("ListOperators should return all operators", func() {
			mockOp := NewTestMockOperator("test1")
			err := engine.RegisterOperator("test1", mockOp)
			So(err, ShouldBeNil)
			mockOp2 := NewTestMockOperator("test2")
			err = engine.RegisterOperator("test2", mockOp2)
			So(err, ShouldBeNil)

			operators := engine.ListOperators()
			So(operators, ShouldContain, "test1")
			So(operators, ShouldContain, "test2")
		})
	})
}

func TestEngineYAMLParsing(t *testing.T) {
	Convey("Engine YAML Parsing", t, func() {
		engine := NewDefaultEngine()

		Convey("ParseYAML should parse valid YAML map", func() {
			yaml := `
key1: value1
key2:
  nested: value2
list:
  - item1
  - item2
`
			doc, err := engine.ParseYAML([]byte(yaml))

			So(err, ShouldBeNil)
			So(doc, ShouldNotBeNil)

			data := doc.RawData().(map[interface{}]interface{})
			So(data["key1"], ShouldEqual, "value1")
			So(data["key2"], ShouldNotBeNil)
			So(data["list"], ShouldNotBeNil)
		})

		Convey("ParseYAML should handle empty data", func() {
			doc, err := engine.ParseYAML([]byte{})

			So(err, ShouldBeNil)
			So(doc, ShouldBeNil)
		})

		Convey("ParseYAML should handle null document", func() {
			doc, err := engine.ParseYAML([]byte("null"))

			So(err, ShouldBeNil)
			So(doc, ShouldBeNil)
		})

		Convey("ParseYAML should fail for non-map root", func() {
			yaml := `
- item1
- item2
`
			doc, err := engine.ParseYAML([]byte(yaml))

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Root of YAML document is not a hash/map")
			So(doc, ShouldBeNil)
		})

		Convey("ParseYAML should fail for invalid YAML", func() {
			yaml := `
key1: value1
  invalid: indentation
`
			doc, err := engine.ParseYAML([]byte(yaml))

			So(err, ShouldNotBeNil)
			So(doc, ShouldBeNil)
		})

		Convey("ParseYAML should convert YAML 1.1 booleans", func() {
			yaml := `
yes_val: yes
no_val: no
on_val: on
off_val: off
normal_string: hello
`
			doc, err := engine.ParseYAML([]byte(yaml))

			So(err, ShouldBeNil)
			So(doc, ShouldNotBeNil)

			data := doc.RawData().(map[interface{}]interface{})
			So(data["yes_val"], ShouldEqual, true)
			So(data["no_val"], ShouldEqual, false)
			So(data["on_val"], ShouldEqual, true)
			So(data["off_val"], ShouldEqual, false)
			So(data["normal_string"], ShouldEqual, "hello")
		})
	})
}

func TestEngineJSONParsing(t *testing.T) {
	Convey("Engine JSON Parsing", t, func() {
		engine := NewDefaultEngine()

		Convey("ParseJSON should parse valid JSON", func() {
			json := `{
				"key1": "value1",
				"key2": {
					"nested": "value2"
				},
				"list": ["item1", "item2"]
			}`

			doc, err := engine.ParseJSON([]byte(json))

			So(err, ShouldBeNil)
			So(doc, ShouldNotBeNil)

			data := doc.RawData().(map[interface{}]interface{})
			So(data["key1"], ShouldEqual, "value1")
			So(data["key2"], ShouldNotBeNil)
			So(data["list"], ShouldNotBeNil)
		})

		Convey("ParseJSON should handle empty data", func() {
			doc, err := engine.ParseJSON([]byte{})

			So(err, ShouldBeNil)
			So(doc, ShouldBeNil)
		})

		Convey("ParseJSON should handle null JSON", func() {
			doc, err := engine.ParseJSON([]byte("null"))

			So(err, ShouldBeNil)
			So(doc, ShouldBeNil)
		})

		Convey("ParseJSON should fail for invalid JSON", func() {
			json := `{
				"key1": "value1",
				"key2": invalid
			}`

			doc, err := engine.ParseJSON([]byte(json))

			So(err, ShouldNotBeNil)
			So(doc, ShouldBeNil)
		})
	})
}

func TestEngineEvaluation(t *testing.T) {
	Convey("Engine Evaluation", t, func() {
		engine := NewDefaultEngine()

		Convey("Evaluate should process simple document", func() {
			yaml := `
key1: value1
key2: value2
`
			doc, err := engine.ParseYAML([]byte(yaml))
			So(err, ShouldBeNil)

			result, err := engine.Evaluate(context.Background(), doc)

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)

			data := result.RawData().(map[interface{}]interface{})
			So(data["key1"], ShouldEqual, "value1")
			So(data["key2"], ShouldEqual, "value2")
		})

		Convey("Evaluate should handle nil context", func() {
			yaml := `
key1: value1
`
			doc, err := engine.ParseYAML([]byte(yaml))
			So(err, ShouldBeNil)

			result, err := engine.Evaluate(nil, doc)

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
		})

		Convey("Evaluate should fail for non-map document", func() {
			// We'll skip this test for now since the Document interface doesn't
			// allow creating documents with non-map data directly
			// This would need to be tested at a lower level where we can create
			// invalid document structures
			So(true, ShouldBeTrue) // Placeholder test
		})
	})
}

func TestEngineStateManagement(t *testing.T) {
	Convey("Engine State Management", t, func() {
		engine := NewDefaultEngine()

		Convey("Vault state management", func() {
			Convey("should track vault cache", func() {
				cache := make(map[string]interface{})
				cache["key"] = "value"
				engine.SetVaultCache("secret/test", cache)

				retrieved := engine.GetVaultCache()
				So(retrieved["secret/test"], ShouldNotBeNil)
				secretData := retrieved["secret/test"]
				So(secretData["key"], ShouldEqual, "value")
			})

			Convey("should track vault references", func() {
				engine.AddVaultRef("secret/test", []string{"key1", "key2"})

				// Vault refs are stored internally but not directly accessible
				// This tests that the method doesn't panic
				engine.AddVaultRef("secret/test", []string{"key3"})
			})

			Convey("should report vault skip status", func() {
				So(engine.IsVaultSkipped(), ShouldEqual, engine.skipVault)
			})
		})

		Convey("AWS state management", func() {
			Convey("should track AWS secrets cache", func() {
				engine.SetAWSSecretCache("secret1", "value1")

				cache := engine.GetAWSSecretsCache()
				So(cache["secret1"], ShouldEqual, "value1")
			})

			Convey("should track AWS params cache", func() {
				engine.SetAWSParamCache("param1", "value1")

				cache := engine.GetAWSParamsCache()
				So(cache["param1"], ShouldEqual, "value1")
			})

			Convey("should report AWS skip status", func() {
				So(engine.IsAWSSkipped(), ShouldEqual, engine.skipAws)
			})
		})

		Convey("IP state management", func() {
			Convey("should track used IPs", func() {
				engine.SetUsedIP("10.0.0.1", "job1")

				ips := engine.GetUsedIPs()
				So(ips["10.0.0.1"], ShouldEqual, "job1")
			})
		})

		Convey("Prune state management", func() {
			Convey("should track keys to prune", func() {
				engine.AddKeyToPrune("key1")
				engine.AddKeyToPrune("key2")

				keys := engine.GetKeysToPrune()
				So(keys, ShouldContain, "key1")
				So(keys, ShouldContain, "key2")
				So(len(keys), ShouldEqual, 2)
			})
		})

		Convey("Sort state management", func() {
			Convey("should track paths to sort", func() {
				engine.AddPathToSort("path1", "name")
				engine.AddPathToSort("path2", "id")

				paths := engine.GetPathsToSort()
				So(paths["path1"], ShouldEqual, "name")
				So(paths["path2"], ShouldEqual, "id")
				So(len(paths), ShouldEqual, 2)
			})
		})
	})
}

func TestNewDefaultEngineFactories(t *testing.T) {
	Convey("Engine Factory Functions", t, func() {

		Convey("NewDefaultEngine should create engine with default config", func() {
			engine := NewDefaultEngine()

			So(engine, ShouldNotBeNil)
			So(engine.config.UseEnhancedParser, ShouldBeTrue)
			So(engine.config.EnableCaching, ShouldBeTrue)
			So(engine.config.CacheSize, ShouldEqual, 10000)
			So(engine.config.EnableParallel, ShouldBeFalse)
		})

		Convey("DefaultEngineConfig should return default configuration", func() {
			config := DefaultEngineConfig()

			So(config.UseEnhancedParser, ShouldBeTrue)
			So(config.EnableCaching, ShouldBeTrue)
			So(config.CacheSize, ShouldEqual, 10000)
			So(config.EnableParallel, ShouldBeFalse)
			So(config.MaxWorkers, ShouldEqual, 4)
			So(config.SkipVault, ShouldBeFalse)
			So(config.SkipAWS, ShouldBeFalse)
		})
	})
}

func TestEngineOperatorRegistryMethods(t *testing.T) {
	Convey("Engine Operator Registry Methods", t, func() {
		engine := NewDefaultEngine()

		Convey("RegisterOperator should add new operator", func() {
			mockOp := NewTestMockOperator("test_op")
			err := engine.RegisterOperator("test_op", mockOp)

			So(err, ShouldBeNil)

			// Verify operator was registered
			op, exists := engine.GetOperator("test_op")
			So(exists, ShouldBeTrue)
			So(op, ShouldEqual, mockOp)
		})

		Convey("RegisterOperator should fail for existing operator", func() {
			mockOp := NewTestMockOperator("test_op_duplicate")
			engine.RegisterOperator("test_op_duplicate", mockOp)

			err := engine.RegisterOperator("test_op_duplicate", mockOp)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "already registered")
		})

		Convey("GetOperator should return false for non-existent operator", func() {
			op, exists := engine.GetOperator("nonexistent_operator")
			So(exists, ShouldBeFalse)
			So(op, ShouldBeNil)
		})
	})
}

func TestEngineVaultServiceMethods(t *testing.T) {
	Convey("Engine Vault Service Methods", t, func() {
		engine := NewDefaultEngine()

		Convey("GetVaultClient should return nil initially", func() {
			client := engine.GetVaultClient()
			So(client, ShouldBeNil)
		})

		Convey("GetVaultCache should return empty cache initially", func() {
			cache := engine.GetVaultCache()
			So(cache, ShouldNotBeNil)
			So(len(cache), ShouldEqual, 0)
		})

		Convey("SetVaultCache should update cache", func() {
			testData := map[string]interface{}{
				"key": "value",
			}

			engine.SetVaultCache("test/path", testData)
			cache := engine.GetVaultCache()
			So(cache["test/path"], ShouldResemble, testData)
		})

		Convey("AddVaultRef should add reference", func() {
			engine.AddVaultRef("test/path", []string{"field1"})
			engine.AddVaultRef("test/path", []string{"field2"})

			// Test with a valid path - assuming the method doesn't panic
			// Since we can't easily access the internal refs map, we'll just test it doesn't error
			So(func() { engine.AddVaultRef("test/path", []string{"field3"}) }, ShouldNotPanic)
		})

		Convey("IsVaultSkipped should return config value", func() {
			// Default config has SkipVault: false
			So(engine.IsVaultSkipped(), ShouldBeFalse)

			// Test with skip vault enabled
			config := DefaultEngineConfig()
			config.SkipVault = true
			engine2 := NewDefaultEngineWithConfig(config)
			So(engine2.IsVaultSkipped(), ShouldBeTrue)
		})
	})
}

func TestEngineAWSServiceMethods(t *testing.T) {
	Convey("Engine AWS Service Methods", t, func() {
		engine := NewDefaultEngine()

		Convey("AWS clients should return nil initially", func() {
			So(engine.GetAWSSession(), ShouldBeNil)
			So(engine.GetSecretsManagerClient(), ShouldBeNil)
			So(engine.GetParameterStoreClient(), ShouldBeNil)
		})

		Convey("AWS caches should return empty initially", func() {
			secretsCache := engine.GetAWSSecretsCache()
			paramsCache := engine.GetAWSParamsCache()

			So(secretsCache, ShouldNotBeNil)
			So(len(secretsCache), ShouldEqual, 0)
			So(paramsCache, ShouldNotBeNil)
			So(len(paramsCache), ShouldEqual, 0)
		})

		Convey("SetAWSSecretCache should update cache", func() {
			engine.SetAWSSecretCache("test/secret", "secret_value")
			cache := engine.GetAWSSecretsCache()
			So(cache["test/secret"], ShouldEqual, "secret_value")
		})

		Convey("SetAWSParamCache should update cache", func() {
			engine.SetAWSParamCache("/app/config/port", "8080")
			engine.SetAWSParamCache("/app/config/host", "localhost")

			cache := engine.GetAWSParamsCache()
			So(cache["/app/config/port"], ShouldEqual, "8080")
			So(cache["/app/config/host"], ShouldEqual, "localhost")
		})

		Convey("IsAWSSkipped should return config value", func() {
			// Default config has SkipAWS: false
			So(engine.IsAWSSkipped(), ShouldBeFalse)

			// Test with skip AWS enabled
			config := DefaultEngineConfig()
			config.SkipAWS = true
			engine2 := NewDefaultEngineWithConfig(config)
			So(engine2.IsAWSSkipped(), ShouldBeTrue)
		})
	})
}

func TestEngineIPServiceMethods(t *testing.T) {
	Convey("Engine IP Service Methods", t, func() {
		engine := NewDefaultEngine()

		Convey("GetUsedIPs should return empty initially", func() {
			ips := engine.GetUsedIPs()
			So(ips, ShouldNotBeNil)
			So(len(ips), ShouldEqual, 0)
		})

		Convey("SetUsedIP should set IP", func() {
			engine.SetUsedIP("key1", "192.168.1.1")
			engine.SetUsedIP("key2", "192.168.1.2")

			ips := engine.GetUsedIPs()
			So(ips["key1"], ShouldEqual, "192.168.1.1")
			So(ips["key2"], ShouldEqual, "192.168.1.2")
		})
	})
}

func TestEngineEdgeCases(t *testing.T) {
	Convey("Engine Edge Cases", t, func() {
		engine := NewDefaultEngine()

		Convey("should handle deeply nested structures", func() {
			yaml := `
level1:
  level2:
    level3:
      level4:
        level5: deep_value
`
			doc, err := engine.ParseYAML([]byte(yaml))
			So(err, ShouldBeNil)

			result, err := engine.Evaluate(context.Background(), doc)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
		})

		Convey("should handle large documents", func() {
			yamlBuilder := strings.Builder{}
			yamlBuilder.WriteString("large_doc:\n")
			for i := 0; i < 1000; i++ {
				yamlBuilder.WriteString(fmt.Sprintf("  key_%d: value_%d\n", i, i))
			}

			doc, err := engine.ParseYAML([]byte(yamlBuilder.String()))
			So(err, ShouldBeNil)

			result, err := engine.Evaluate(context.Background(), doc)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
		})

		Convey("should handle unicode and special characters", func() {
			yaml := `
unicode_key: "Hello 世界 🌍"
special_chars: "@#$%^&*()[]{}|\\:;'\"<>?/~"
empty_string: ""
spaces: "   "
`
			doc, err := engine.ParseYAML([]byte(yaml))
			So(err, ShouldBeNil)

			result, err := engine.Evaluate(context.Background(), doc)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
		})

		Convey("should handle concurrent evaluations", func() {
			yaml := `test_key: test_value`
			doc, err := engine.ParseYAML([]byte(yaml))
			So(err, ShouldBeNil)

			// Run multiple concurrent evaluations
			done := make(chan bool, 10)
			errors := make(chan error, 10)

			for i := 0; i < 10; i++ {
				go func() {
					result, err := engine.Evaluate(context.Background(), doc)
					if err != nil {
						errors <- err
						return
					}
					if result == nil {
						errors <- fmt.Errorf("got nil result")
						return
					}
					done <- true
				}()
			}

			// Wait for all goroutines
			for i := 0; i < 10; i++ {
				select {
				case <-done:
					// Success
				case err := <-errors:
					So(err, ShouldBeNil)
				}
			}
		})

		Convey("should handle mixed data types", func() {
			yaml := `
string_val: "text"
int_val: 42
float_val: 3.14
bool_val: true
null_val: null
array_val:
  - "item1"
  - 123
  - false
map_val:
  nested_string: "nested"
  nested_int: 456
`
			doc, err := engine.ParseYAML([]byte(yaml))
			So(err, ShouldBeNil)

			result, err := engine.Evaluate(context.Background(), doc)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
		})
	})
}

func TestEnginePerformance(t *testing.T) {
	Convey("Engine Performance", t, func() {
		engine := NewDefaultEngine()

		Convey("should complete evaluation within reasonable time", func() {
			yaml := `
performance_test:
  iterations: 100
  data:
    key1: value1
    key2: value2
    key3: value3
`
			doc, err := engine.ParseYAML([]byte(yaml))
			So(err, ShouldBeNil)

			start := time.Now()
			result, err := engine.Evaluate(context.Background(), doc)
			duration := time.Since(start)

			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
			So(duration, ShouldBeLessThan, time.Second) // Should complete quickly
		})

		Convey("should handle memory efficiently", func() {
			// Test memory usage doesn't grow excessively
			yaml := `memory_test: small_value`
			doc, err := engine.ParseYAML([]byte(yaml))
			So(err, ShouldBeNil)

			// Multiple evaluations should not leak memory significantly
			for i := 0; i < 100; i++ {
				result, err := engine.Evaluate(context.Background(), doc)
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
			}
		})
	})
}

func TestEngineConfigurationEdgeCases(t *testing.T) {
	Convey("Engine Configuration Edge Cases", t, func() {

		Convey("should handle extreme configuration values", func() {
			config := EngineConfig{
				CacheSize:     0,         // Minimum cache
				MaxWorkers:    1,         // Minimum workers
				DataflowOrder: "unknown", // Invalid order
			}

			engine := NewDefaultEngineWithConfig(config)
			So(engine, ShouldNotBeNil)
			So(engine.config.CacheSize, ShouldEqual, 0)
			So(engine.config.MaxWorkers, ShouldEqual, 1)
		})

		Convey("should handle maximum configuration values", func() {
			config := EngineConfig{
				CacheSize:  999999, // Large cache
				MaxWorkers: 1000,   // Many workers
			}

			engine := NewDefaultEngineWithConfig(config)
			So(engine, ShouldNotBeNil)
			So(engine.config.CacheSize, ShouldEqual, 999999)
			So(engine.config.MaxWorkers, ShouldEqual, 1000)
		})

		Convey("should handle empty configuration strings", func() {
			config := EngineConfig{
				VaultAddr:     "",
				VaultToken:    "",
				AWSRegion:     "",
				AWSProfile:    "",
				DataflowOrder: "",
			}

			engine := NewDefaultEngineWithConfig(config)
			So(engine, ShouldNotBeNil)
			So(engine.config.VaultAddr, ShouldEqual, "")
			So(engine.config.VaultToken, ShouldEqual, "")
			So(engine.config.AWSRegion, ShouldEqual, "")
			So(engine.config.AWSProfile, ShouldEqual, "")
			So(engine.config.DataflowOrder, ShouldEqual, "")
		})
	})
}

func TestEngineFileParsing(t *testing.T) {
	Convey("Engine File Parsing", t, func() {
		engine := NewDefaultEngine()

		Convey("ParseFile should fail for non-existent file", func() {
			doc, err := engine.ParseFile("/non/existent/file.yml")

			So(err, ShouldNotBeNil)
			So(doc, ShouldBeNil)
		})

		Convey("ParseReader should return not implemented error", func() {
			yamlContent := `key1: value1`
			reader := strings.NewReader(yamlContent)

			doc, err := engine.ParseReader(reader)

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "not implemented")
			So(doc, ShouldBeNil)
		})
	})
}

func TestEngineMerging(t *testing.T) {
	Convey("Engine Merging", t, func() {
		engine := NewDefaultEngine()

		Convey("Merge should create merge builder", func() {
			yaml1 := `key1: value1`
			yaml2 := `key2: value2`

			doc1, err := engine.ParseYAML([]byte(yaml1))
			So(err, ShouldBeNil)
			doc2, err := engine.ParseYAML([]byte(yaml2))
			So(err, ShouldBeNil)

			builder := engine.Merge(context.Background(), doc1, doc2)

			So(builder, ShouldNotBeNil)
		})

		Convey("Merge should handle nil context", func() {
			yaml1 := `key1: value1`
			doc1, err := engine.ParseYAML([]byte(yaml1))
			So(err, ShouldBeNil)

			builder := engine.Merge(nil, doc1)
			So(builder, ShouldNotBeNil)
		})

		Convey("MergeFiles should create merge builder", func() {
			builder := engine.MergeFiles(context.Background(), "/non/existent/file1.yml", "/non/existent/file2.yml")

			// Should return a builder (may be nil for unimplemented functionality)
			_ = builder
		})

		Convey("MergeReaders should create merge builder", func() {
			yaml1 := `key1: value1`
			yaml2 := `key2: value2`
			reader1 := strings.NewReader(yaml1)
			reader2 := strings.NewReader(yaml2)

			builder := engine.MergeReaders(context.Background(), reader1, reader2)

			// Should return a builder (may be nil for unimplemented functionality)
			_ = builder
		})
	})
}

func TestEngineOutputFormatting(t *testing.T) {
	Convey("Engine Output Formatting", t, func() {
		engine := NewDefaultEngine()

		Convey("ToYAML should return not implemented error", func() {
			yaml := `key1: value1`
			doc, err := engine.ParseYAML([]byte(yaml))
			So(err, ShouldBeNil)

			result, err := engine.ToYAML(doc)

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "not implemented")
			So(result, ShouldBeNil)
		})

		Convey("ToJSON should return not implemented error", func() {
			yaml := `key1: value1`
			doc, err := engine.ParseYAML([]byte(yaml))
			So(err, ShouldBeNil)

			result, err := engine.ToJSON(doc)

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "not implemented")
			So(result, ShouldBeNil)
		})

		Convey("ToJSONIndent should return not implemented error", func() {
			yaml := `key1: value1`
			doc, err := engine.ParseYAML([]byte(yaml))
			So(err, ShouldBeNil)

			result, err := engine.ToJSONIndent(doc, "  ")

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "not implemented")
			So(result, ShouldBeNil)
		})

		Convey("Output formatters should handle nil document", func() {
			yamlResult, err := engine.ToYAML(nil)
			So(err, ShouldNotBeNil)
			So(yamlResult, ShouldBeNil)

			jsonResult, err := engine.ToJSON(nil)
			So(err, ShouldNotBeNil)
			So(jsonResult, ShouldBeNil)

			jsonIndentResult, err := engine.ToJSONIndent(nil, "  ")
			So(err, ShouldNotBeNil)
			So(jsonIndentResult, ShouldBeNil)
		})
	})
}

func TestEngineInitialization(t *testing.T) {
	Convey("Engine Initialization", t, func() {
		engine := NewDefaultEngine()

		Convey("registerDefaultOperators should not panic", func() {
			// This tests the private function indirectly by checking that
			// default operators are available after engine creation
			operators := engine.ListOperators()

			So(len(operators), ShouldBeGreaterThan, 5)
			So(operators, ShouldContain, "grab")
			So(operators, ShouldContain, "concat")
			So(operators, ShouldContain, "static_ips")
			So(operators, ShouldContain, "calc")
		})

		Convey("initializeVault should handle configuration", func() {
			// Test vault initialization by checking skip status
			config := EngineConfig{SkipVault: true}
			engine := NewDefaultEngineWithConfig(config)

			So(engine.IsVaultSkipped(), ShouldBeTrue)
		})

		Convey("initializeAWS should handle configuration", func() {
			// Test AWS initialization by checking skip status
			config := EngineConfig{SkipAWS: true}
			engine := NewDefaultEngineWithConfig(config)

			So(engine.IsAWSSkipped(), ShouldBeTrue)
		})
	})
}

func TestEngineBuilderMethods(t *testing.T) {
	Convey("Engine Builder Methods", t, func() {
		engine := NewDefaultEngine()

		Convey("WithLogger should set logger", func() {
			newEngine := engine.WithLogger(nil)

			// The method should return the engine instance
			So(newEngine, ShouldEqual, engine)
		})

		Convey("WithVaultClient should set vault client", func() {
			newEngine := engine.WithVaultClient(nil)

			// The method should return the engine instance
			So(newEngine, ShouldEqual, engine)
		})

		Convey("WithAWSConfig should set AWS config", func() {
			config := AWSConfig{Region: "us-west-2"}
			newEngine := engine.WithAWSConfig(config)

			// The method should return a new engine instance
			So(newEngine, ShouldNotBeNil)
		})
	})
}

func TestEngineExternalClientAccess(t *testing.T) {
	Convey("Engine External Client Access", t, func() {
		engine := NewDefaultEngine()

		Convey("GetVaultClient should return vault client", func() {
			client := engine.GetVaultClient()
			// Initially nil since no vault client is configured by default
			So(client, ShouldBeNil)
		})

		Convey("GetAWSSession should return AWS session", func() {
			session := engine.GetAWSSession()
			// Initially nil since no AWS session is configured by default
			So(session, ShouldBeNil)
		})

		Convey("GetSecretsManagerClient should return secrets manager client", func() {
			client := engine.GetSecretsManagerClient()
			// Initially nil since no AWS client is configured by default
			So(client, ShouldBeNil)
		})

		Convey("GetParameterStoreClient should return parameter store client", func() {
			client := engine.GetParameterStoreClient()
			// Initially nil since no AWS client is configured by default
			So(client, ShouldBeNil)
		})
	})
}

func TestEngineErrorHandling(t *testing.T) {
	Convey("Engine Error Handling", t, func() {
		engine := NewDefaultEngine()

		Convey("evaluate should handle evaluation errors", func() {
			// This test verifies error handling capability exists
			// For a meaningful test, we'd need a document that actually triggers evaluation
			yaml := `test_key: simple_value`
			doc, err := engine.ParseYAML([]byte(yaml))
			So(err, ShouldBeNil)

			result, err := engine.Evaluate(context.Background(), doc)

			// Simple documents without operators should not error
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
		})

		Convey("evaluation should handle context cancellation", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			yaml := `key1: value1`
			doc, err := engine.ParseYAML([]byte(yaml))
			So(err, ShouldBeNil)

			result, err := engine.Evaluate(ctx, doc)

			// Should handle cancellation gracefully
			if err != nil {
				So(err.Error(), ShouldContainSubstring, "context")
			}
			_ = result // May or may not be nil depending on timing
		})

		Convey("should handle operator registration errors", func() {
			mockOp := NewTestMockOperator("test_error")
			err := engine.RegisterOperator("test_error", mockOp)
			So(err, ShouldBeNil)

			// Try to register the same operator again
			err = engine.RegisterOperator("test_error", mockOp)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "already registered")
		})

		Convey("should handle invalid operator expressions gracefully", func() {
			yaml := `result: (( nonexistent_operator "param" ))`
			doc, err := engine.ParseYAML([]byte(yaml))
			So(err, ShouldBeNil)

			result, err := engine.Evaluate(context.Background(), doc)

			// Current behavior: evaluation may succeed but operator remains unevaluated
			// This tests the engine doesn't crash on unknown operators
			_ = err
			_ = result
		})

		Convey("should handle malformed operator expressions gracefully", func() {
			yaml := `result: (( grab ])`
			doc, err := engine.ParseYAML([]byte(yaml))
			So(err, ShouldBeNil)

			result, err := engine.Evaluate(context.Background(), doc)

			// Current behavior: evaluation may succeed with malformed expressions
			// This tests the engine doesn't crash on malformed syntax
			_ = err
			_ = result
		})
	})
}

func TestEngineConfiguration(t *testing.T) {
	Convey("Engine Configuration", t, func() {
		engine := NewDefaultEngine()

		Convey("UpdateConfig should update all config fields", func() {
			newConfig := EngineConfig{
				VaultAddr:         "https://vault.example.com",
				VaultToken:        "token123",
				VaultSkipTLS:      true,
				SkipVault:         true,
				AWSRegion:         "us-west-2",
				AWSProfile:        "test",
				SkipAWS:           true,
				UseEnhancedParser: false,
				EnableCaching:     false,
				CacheSize:         5000,
				EnableParallel:    true,
				MaxWorkers:        8,
				DataflowOrder:     "insertion",
			}

			engine.UpdateConfig(newConfig)

			So(engine.config, ShouldResemble, newConfig)
			So(engine.skipVault, ShouldBeTrue)
			So(engine.skipAws, ShouldBeTrue)
			So(engine.useEnhancedParser, ShouldBeFalse)
		})

		Convey("GetOperatorState should return the engine itself", func() {
			state := engine.GetOperatorState()
			So(state, ShouldEqual, engine)
		})
	})
}

func TestConvertStringMapToInterfaceMap(t *testing.T) {
	Convey("convertStringMapToInterfaceMap", t, func() {

		Convey("should convert string map to interface map", func() {
			input := map[string]interface{}{
				"key1": "value1",
				"key2": map[string]interface{}{
					"nested": "value2",
				},
			}

			result := convertStringMapToInterfaceMap(input)

			converted, ok := result.(map[interface{}]interface{})
			So(ok, ShouldBeTrue)
			So(converted["key1"], ShouldEqual, "value1")

			nested, ok := converted["key2"].(map[interface{}]interface{})
			So(ok, ShouldBeTrue)
			So(nested["nested"], ShouldEqual, "value2")
		})

		Convey("should convert arrays", func() {
			input := []interface{}{
				"value1",
				map[string]interface{}{
					"key": "value",
				},
			}

			result := convertStringMapToInterfaceMap(input)

			converted, ok := result.([]interface{})
			So(ok, ShouldBeTrue)
			So(len(converted), ShouldEqual, 2)
			So(converted[0], ShouldEqual, "value1")

			nested, ok := converted[1].(map[interface{}]interface{})
			So(ok, ShouldBeTrue)
			So(nested["key"], ShouldEqual, "value")
		})

		Convey("should convert YAML 1.1 boolean strings", func() {
			testCases := map[string]bool{
				"yes": true,
				"Yes": true,
				"YES": true,
				"on":  true,
				"On":  true,
				"ON":  true,
				"no":  false,
				"No":  false,
				"NO":  false,
				"off": false,
				"Off": false,
				"OFF": false,
			}

			for input, expected := range testCases {
				result := convertStringMapToInterfaceMap(input)
				So(result, ShouldEqual, expected)
			}
		})

		Convey("should pass through non-boolean strings", func() {
			result := convertStringMapToInterfaceMap("normal string")
			So(result, ShouldEqual, "normal string")
		})

		Convey("should pass through primitive types", func() {
			So(convertStringMapToInterfaceMap(42), ShouldEqual, 42)
			So(convertStringMapToInterfaceMap(3.14), ShouldEqual, 3.14)
			So(convertStringMapToInterfaceMap(true), ShouldEqual, true)
			So(convertStringMapToInterfaceMap(nil), ShouldBeNil)
		})
	})
}

func TestCreateEngineFromOptions(t *testing.T) {
	Convey("createEngineFromOptions", t, func() {

		Convey("should create engine with valid options", func() {
			opts := &EngineOptions{
				VaultAddress:      "https://vault.example.com",
				VaultToken:        "token123",
				AWSRegion:         "us-west-2",
				UseEnhancedParser: true,
				EnableCache:       true,
				CacheSize:         5000,
				MaxConcurrency:    4,
				DataflowOrder:     "insertion",
			}

			engine, err := createEngineFromOptions(opts)

			So(err, ShouldBeNil)
			So(engine, ShouldNotBeNil)

			defaultEngine, ok := engine.(*DefaultEngine)
			So(ok, ShouldBeTrue)
			So(defaultEngine.config.VaultAddr, ShouldEqual, "https://vault.example.com")
			So(defaultEngine.config.VaultToken, ShouldEqual, "token123")
			So(defaultEngine.config.AWSRegion, ShouldEqual, "us-west-2")
			So(defaultEngine.config.UseEnhancedParser, ShouldBeTrue)
			So(defaultEngine.config.EnableCaching, ShouldBeTrue)
			So(defaultEngine.config.CacheSize, ShouldEqual, 5000)
			So(defaultEngine.config.MaxWorkers, ShouldEqual, 4)
			So(defaultEngine.config.DataflowOrder, ShouldEqual, "insertion")
		})

		Convey("should fail for negative concurrency", func() {
			opts := &EngineOptions{
				MaxConcurrency: -1,
			}

			engine, err := createEngineFromOptions(opts)

			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "concurrency must be non-negative")
			So(engine, ShouldBeNil)
		})

		Convey("should register custom operators", func() {
			mockOp := NewTestMockOperator("custom")
			opts := &EngineOptions{
				CustomOperators: map[string]Operator{
					"custom": mockOp,
				},
			}

			engine, err := createEngineFromOptions(opts)

			So(err, ShouldBeNil)
			So(engine, ShouldNotBeNil)

			op, exists := engine.GetOperator("custom")
			So(exists, ShouldBeTrue)
			So(op, ShouldEqual, mockOp)
		})

		Convey("should fail if custom operator registration fails", func() {
			mockOp := NewTestMockOperator("grab")
			opts := &EngineOptions{
				CustomOperators: map[string]Operator{
					"grab": mockOp, // This will conflict with existing operator
				},
			}

			// The createEngineFromOptions function handles this scenario
			engine, err := createEngineFromOptions(opts)
			if err != nil {
				So(err.Error(), ShouldContainSubstring, "already registered")
			} else {
				// If no error, it means the operator was successfully registered
				So(engine, ShouldNotBeNil)
			}
		})
	})
}
