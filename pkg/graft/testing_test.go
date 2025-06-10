package graft

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTestHelper_BasicUsage(t *testing.T) {
	Convey("Given a test helper", t, func() {
		helper := NewTestHelper(t)

		Convey("When parsing YAML", func() {
			doc := helper.ParseYAMLString(`
name: test
config:
  enabled: true
  count: 42
`)

			Convey("Then assertions should work", func() {
				helper.AssertPathString(doc, "name", "test")
				helper.AssertPathBool(doc, "config.enabled", true)
				helper.AssertPathInt(doc, "config.count", 42)
			})
		})

		Convey("When parsing JSON", func() {
			doc := helper.ParseJSONString(`{
				"name": "test",
				"config": {
					"enabled": false,
					"timeout": 30
				}
			}`)

			Convey("Then assertions should work", func() {
				helper.AssertPathString(doc, "name", "test")
				helper.AssertPathBool(doc, "config.enabled", false)
				helper.AssertPathInt(doc, "config.timeout", 30)
			})
		})
	})
}

func TestTestHelper_MergeOperations(t *testing.T) {
	Convey("Given documents to merge", t, func() {
		helper := NewTestHelper(t)

		base := helper.ParseYAMLString(`
name: base
config:
  enabled: true
  timeout: 30
`)

		override := helper.ParseYAMLString(`
name: override
config:
  timeout: 60
  retries: 3
`)

		Convey("When merging documents", func() {
			result := helper.MustMerge(base, override)

			Convey("Then the result should be correctly merged", func() {
				helper.AssertPathString(result, "name", "override")
				helper.AssertPathBool(result, "config.enabled", true)
				helper.AssertPathInt(result, "config.timeout", 60)
				helper.AssertPathInt(result, "config.retries", 3)
			})
		})

		Convey("When merging with pruning", func() {
			result := helper.MustMergeWithPrune([]string{"config.enabled"}, base, override)

			Convey("Then pruned fields should be removed", func() {
				helper.AssertPathNotExists(result, "config.enabled")
				helper.AssertPathInt(result, "config.timeout", 60)
			})
		})
	})
}

func TestTestHelper_EvaluationOperations(t *testing.T) {
	Convey("Given a document with operators", t, func() {
		helper := NewTestHelper(t)

		doc := helper.ParseYAMLString(`
meta:
  app_name: "myapp"
  version: "1.0"

name: (( concat meta.app_name "-" meta.version ))
full_name: (( grab name ))
`)

		Convey("When evaluating", func() {
			result := helper.MustEvaluate(doc)

			Convey("Then operators should be evaluated", func() {
				helper.AssertPathString(result, "name", "myapp-1.0")
				helper.AssertPathString(result, "full_name", "myapp-1.0")
			})
		})

		Convey("When merging and evaluating", func() {
			override := helper.ParseYAMLString(`
meta:
  version: "2.0"
`)

			result := helper.MustMergeAndEvaluate(doc, override)

			Convey("Then the result should reflect the override", func() {
				helper.AssertPathString(result, "name", "myapp-2.0")
				helper.AssertPathString(result, "full_name", "myapp-2.0")
			})
		})
	})
}

func TestTestHelper_ErrorHandling(t *testing.T) {
	Convey("Given a test helper", t, func() {
		helper := NewTestHelper(t)

		Convey("When testing error scenarios", func() {
			doc := helper.ParseYAMLString(`
result: (( grab nonexistent.path ))
`)

			_, err := helper.engine.Evaluate(context.Background(), doc)

			Convey("Then error assertions should work", func() {
				helper.AssertError(err, EvaluationError)
			})
		})

		Convey("When testing non-existent paths", func() {
			doc := helper.ParseYAMLString(`
name: test
`)

			Convey("Then path assertions should detect missing paths", func() {
				helper.AssertPathNotExists(doc, "nonexistent")
				helper.AssertPathString(doc, "name", "test")
			})
		})
	})
}

func TestTestHelper_DocumentComparison(t *testing.T) {
	Convey("Given two similar documents", t, func() {
		helper := NewTestHelper(t)

		doc1 := helper.ParseYAMLString(`
name: test
config:
  enabled: true
`)

		doc2 := helper.ParseYAMLString(`
name: test
config:
  enabled: true
`)

		doc3 := helper.ParseYAMLString(`
name: test
config:
  enabled: false
`)

		Convey("When comparing identical documents", func() {
			Convey("Then they should be equal", func() {
				helper.AssertDocumentsEqual(doc1, doc2)
			})
		})

		Convey("When comparing different documents", func() {
			differences := helper.CompareDocuments(doc1, doc3)

			Convey("Then differences should be detected", func() {
				So(len(differences), ShouldBeGreaterThan, 0)
			})
		})
	})
}

func TestTestHelper_CreateTestDocument(t *testing.T) {
	Convey("Given a test helper", t, func() {
		helper := NewTestHelper(t)

		Convey("When creating a test document from map", func() {
			data := map[string]interface{}{
				"name": "test",
				"config": map[string]interface{}{
					"enabled": true,
					"count":   42,
				},
				"items": []interface{}{
					"item1",
					map[string]interface{}{
						"name": "item2",
					},
				},
			}

			doc := helper.CreateTestDocument(data)

			Convey("Then the document should be properly structured", func() {
				helper.AssertPathString(doc, "name", "test")
				helper.AssertPathBool(doc, "config.enabled", true)
				helper.AssertPathInt(doc, "config.count", 42)
				helper.AssertPath(doc, "items.0", "item1")
			})
		})
	})
}

func TestTestHelper_WithOptions(t *testing.T) {
	Convey("Given a test helper with custom options", t, func() {
		helper := NewTestHelperWithOptions(t,
			WithDebugLogging(true),
			WithConcurrency(5),
		)

		Convey("When using the helper", func() {
			doc := helper.ParseYAMLString(`
name: test
`)

			Convey("Then it should work normally", func() {
				helper.AssertPathString(doc, "name", "test")
			})
		})
	})
}
