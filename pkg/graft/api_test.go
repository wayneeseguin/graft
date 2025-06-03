package graft

import (
	"context"
	"strings"
	"testing"

	"github.com/starkandwayne/goutils/tree"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEngine_ParseYAML(t *testing.T) {
	Convey("Given an Engine instance", t, func() {
		engine, err := NewEngine()
		So(err, ShouldBeNil)
		So(engine, ShouldNotBeNil)

		Convey("When parsing valid YAML", func() {
			yamlData := []byte(`
name: test
values:
  - one
  - two
config:
  enabled: true
  count: 42
`)
			doc, err := engine.ParseYAML(yamlData)

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
				So(doc, ShouldNotBeNil)
			})

			Convey("And the document should contain expected values", func() {
				name, err := doc.Get("name")
				So(err, ShouldBeNil)
				So(name, ShouldEqual, "test")

				enabled, err := doc.Get("config.enabled")
				So(err, ShouldBeNil)
				So(enabled, ShouldEqual, true)

				count, err := doc.Get("config.count")
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 42)
			})
		})

		Convey("When parsing invalid YAML", func() {
			yamlData := []byte(`
name: test
  - invalid: indentation
`)
			doc, err := engine.ParseYAML(yamlData)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(doc, ShouldBeNil)
			})

			Convey("And the error should be a ParseError", func() {
				graftErr, ok := err.(*GraftError)
				So(ok, ShouldBeTrue)
				So(graftErr.Type, ShouldEqual, ParseError)
			})
		})

		Convey("When parsing empty YAML", func() {
			yamlData := []byte("")
			doc, err := engine.ParseYAML(yamlData)

			Convey("Then it should succeed with nil document", func() {
				So(err, ShouldBeNil)
				So(doc, ShouldBeNil)
			})
		})
	})
}

func TestEngine_ParseJSON(t *testing.T) {
	Convey("Given an Engine instance", t, func() {
		engine, err := NewEngine()
		So(err, ShouldBeNil)

		Convey("When parsing valid JSON", func() {
			jsonData := []byte(`{
				"name": "test",
				"values": ["one", "two"],
				"config": {
					"enabled": true,
					"count": 42
				}
			}`)
			doc, err := engine.ParseJSON(jsonData)

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
				So(doc, ShouldNotBeNil)
			})

			Convey("And the document should contain expected values", func() {
				name, err := doc.Get("name")
				So(err, ShouldBeNil)
				So(name, ShouldEqual, "test")

				enabled, err := doc.Get("config.enabled")
				So(err, ShouldBeNil)
				So(enabled, ShouldEqual, true)
			})
		})

		Convey("When parsing invalid JSON", func() {
			jsonData := []byte(`{
				"name": "test",
				"invalid": json
			}`)
			doc, err := engine.ParseJSON(jsonData)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(doc, ShouldBeNil)
			})
		})
	})
}

func TestEngine_Merge(t *testing.T) {
	Convey("Given an Engine instance and documents", t, func() {
		engine, err := NewEngine()
		So(err, ShouldBeNil)

		doc1, err := engine.ParseYAML([]byte(`
name: base
config:
  enabled: true
  timeout: 30
`))
		So(err, ShouldBeNil)

		doc2, err := engine.ParseYAML([]byte(`
name: override
config:
  timeout: 60
  retries: 3
new_field: added
`))
		So(err, ShouldBeNil)

		Convey("When merging documents", func() {
			ctx := context.Background()
			result, err := engine.Merge(ctx, doc1, doc2).Execute()

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
			})

			Convey("And values should be merged correctly", func() {
				name, err := result.Get("name")
				So(err, ShouldBeNil)
				So(name, ShouldEqual, "override")

				enabled, err := result.Get("config.enabled")
				So(err, ShouldBeNil)
				So(enabled, ShouldEqual, true)

				timeout, err := result.Get("config.timeout")
				So(err, ShouldBeNil)
				So(timeout, ShouldEqual, 60)

				retries, err := result.Get("config.retries")
				So(err, ShouldBeNil)
				So(retries, ShouldEqual, 3)

				newField, err := result.Get("new_field")
				So(err, ShouldBeNil)
				So(newField, ShouldEqual, "added")
			})
		})

		Convey("When merging with prune option", func() {
			ctx := context.Background()
			result, err := engine.Merge(ctx, doc1, doc2).WithPrune("config.enabled").Execute()

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})

			Convey("And pruned fields should be removed", func() {
				_, err := result.Get("config.enabled")
				So(err, ShouldNotBeNil)

				timeout, err := result.Get("config.timeout")
				So(err, ShouldBeNil)
				So(timeout, ShouldEqual, 60)
			})
		})
	})
}

func TestEngine_Evaluate(t *testing.T) {
	Convey("Given an Engine instance", t, func() {
		engine, err := NewEngine()
		So(err, ShouldBeNil)
		
		// Save original operators
		originalConcat := OpRegistry["concat"]
		originalGrab := OpRegistry["grab"]
		
		// Restore original operators when test completes
		defer func() {
			if originalConcat != nil {
				OpRegistry["concat"] = originalConcat
			}
			if originalGrab != nil {
				OpRegistry["grab"] = originalGrab
			}
		}()
		
		// Register test operators
		OpRegistry["concat"] = TestConcatOperator{}
		OpRegistry["grab"] = TestGrabOperator{}
		
		// Also register with the engine
		engine.RegisterOperator("concat", TestConcatOperator{})
		engine.RegisterOperator("grab", TestGrabOperator{})
		

		Convey("When evaluating a document with operators", func() {
			doc, err := engine.ParseYAML([]byte(`
meta:
  base_name: "app"
  version: "1.0"

name: (( concat meta.base_name "-" meta.version ))
config:
  full_name: (( grab name ))
  version: (( grab meta.version ))
`))
			So(err, ShouldBeNil)

			ctx := context.Background()
			result, err := engine.Evaluate(ctx, doc)

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
			})

			Convey("And operators should be evaluated", func() {
				name, err := result.Get("name")
				So(err, ShouldBeNil)
				So(name, ShouldEqual, "app-1.0")

				fullName, err := result.Get("config.full_name")
				So(err, ShouldBeNil)
				So(fullName, ShouldEqual, "app-1.0")

				version, err := result.Get("config.version")
				So(err, ShouldBeNil)
				So(version, ShouldEqual, "1.0")
			})
		})

		Convey("When evaluating a document with unknown operators", func() {
			doc, err := engine.ParseYAML([]byte(`
name: (( unknown_operator "test" ))
`))
			So(err, ShouldBeNil)

			ctx := context.Background()
			result, err := engine.Evaluate(ctx, doc)

			Convey("Then it should succeed without error", func() {
				So(err, ShouldBeNil)
				So(result, ShouldNotBeNil)
			})

			Convey("And the unknown operator should remain unevaluated", func() {
				name, err := result.Get("name")
				So(err, ShouldBeNil)
				So(name, ShouldEqual, "(( unknown_operator \"test\" ))")
			})
		})
	})
}

func TestEngine_WithOptions(t *testing.T) {
	Convey("Given various engine options", t, func() {
		Convey("When creating engine with vault configuration", func() {
			engine, err := NewEngine(
				WithVaultConfig("https://vault.example.com", "mytoken"),
				WithDebugLogging(true),
			)

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
				So(engine, ShouldNotBeNil)
			})
		})

		Convey("When creating engine with AWS configuration", func() {
			engine, err := NewEngine(
				WithAWSRegion("us-west-2"),
				WithConcurrency(10),
			)

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
				So(engine, ShouldNotBeNil)
			})
		})

		Convey("When creating engine with invalid options", func() {
			engine, err := NewEngine(
				WithConcurrency(-1),
			)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(engine, ShouldBeNil)
			})
		})
	})
}

func TestEngine_Context(t *testing.T) {
	Convey("Given an Engine instance", t, func() {
		engine, err := NewEngine()
		So(err, ShouldBeNil)
		
		// Save original operator
		originalConcat := OpRegistry["concat"]
		
		// Restore original operator when test completes
		defer func() {
			if originalConcat != nil {
				OpRegistry["concat"] = originalConcat
			}
		}()
		
		// Register test operators
		OpRegistry["concat"] = TestConcatOperator{}
		
		// Also register with the engine  
		engine.RegisterOperator("concat", TestConcatOperator{})

		doc, err := engine.ParseYAML([]byte(`
name: (( concat "app-" "test" ))
`))
		So(err, ShouldBeNil)

		Convey("When context is cancelled during evaluation", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			result, err := engine.Evaluate(ctx, doc)

			Convey("Then it should return a context error", func() {
				So(err, ShouldNotBeNil)
				So(result, ShouldBeNil)
				So(err, ShouldEqual, context.Canceled)
			})
		})
	})
}
// Test operators for the API tests

// TestConcatOperator is a simple concat operator for testing
type TestConcatOperator struct{}

func (op TestConcatOperator) Setup() error {
	return nil
}

func (op TestConcatOperator) Phase() OperatorPhase {
	return EvalPhase
}

func (op TestConcatOperator) Dependencies(ev *Evaluator, args []*Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	deps := []*tree.Cursor{}
	for _, arg := range args {
		if arg.Type == Reference && arg.Reference != nil {
			deps = append(deps, arg.Reference)
		}
	}
	return deps
}

func (op TestConcatOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	var parts []string
	for _, arg := range args {
		val, err := arg.Evaluate(ev.Tree)
		if err != nil {
			return nil, err
		}
		parts = append(parts, ToString(val))
	}
	
	return &Response{
		Type:  Replace,
		Value: strings.Join(parts, ""),
	}, nil
}

// TestGrabOperator is a simple grab operator for testing  
type TestGrabOperator struct{}

func (op TestGrabOperator) Setup() error {
	return nil
}

func (op TestGrabOperator) Phase() OperatorPhase {
	return EvalPhase
}

func (op TestGrabOperator) Dependencies(ev *Evaluator, args []*Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	deps := []*tree.Cursor{}
	for _, arg := range args {
		if arg.Type == Reference && arg.Reference != nil {
			deps = append(deps, arg.Reference)
		}
	}
	return deps
}

func (op TestGrabOperator) Run(ev *Evaluator, args []*Expr) (*Response, error) {
	if len(args) != 1 {
		return nil, NewOperatorError("grab", "requires exactly 1 argument", nil)
	}
	
	val, err := args[0].Evaluate(ev.Tree)
	if err != nil {
		return nil, err
	}
	
	return &Response{
		Type:  Replace,
		Value: val,
	}, nil
}

// ToString converts a value to string
func ToString(val interface{}) string {
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}