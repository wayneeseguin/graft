package graft

import (
	"context"
	"testing"

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

		Convey("When evaluating a document with invalid operators", func() {
			doc, err := engine.ParseYAML([]byte(`
name: (( unknown_operator "test" ))
`))
			So(err, ShouldBeNil)

			ctx := context.Background()
			result, err := engine.Evaluate(ctx, doc)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(result, ShouldBeNil)
			})

			Convey("And the error should be an OperatorError", func() {
				graftErr, ok := err.(*GraftError)
				So(ok, ShouldBeTrue)
				So(graftErr.Type, ShouldEqual, OperatorError)
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