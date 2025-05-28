package graft

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func BenchmarkEngineV2_ParseYAML(b *testing.B) {
	engine, err := NewEngineV2()
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}

	smallYAML := []byte(`
name: test
config:
  enabled: true
  count: 42
`)

	largeYAML := generateLargeYAML(1000)

	b.Run("Small", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.ParseYAML(smallYAML)
			if err != nil {
				b.Fatalf("Parse error: %v", err)
			}
		}
	})

	b.Run("Large", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.ParseYAML(largeYAML)
			if err != nil {
				b.Fatalf("Parse error: %v", err)
			}
		}
	})
}

func BenchmarkEngineV2_ParseJSON(b *testing.B) {
	engine, err := NewEngineV2()
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}

	smallJSON := []byte(`{
		"name": "test",
		"config": {
			"enabled": true,
			"count": 42
		}
	}`)

	largeJSON := generateLargeJSON(1000)

	b.Run("Small", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.ParseJSON(smallJSON)
			if err != nil {
				b.Fatalf("Parse error: %v", err)
			}
		}
	})

	b.Run("Large", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.ParseJSON(largeJSON)
			if err != nil {
				b.Fatalf("Parse error: %v", err)
			}
		}
	})
}

func BenchmarkEngineV2_Merge(b *testing.B) {
	engine, err := NewEngineV2()
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}

	ctx := context.Background()

	// Small documents
	smallBase, _ := engine.ParseYAML([]byte(`
name: base
config:
  enabled: true
  timeout: 30
`))
	smallOverride, _ := engine.ParseYAML([]byte(`
name: override
config:
  timeout: 60
  retries: 3
`))

	// Large documents
	largeBase, _ := engine.ParseYAML(generateLargeYAML(500))
	largeOverride, _ := engine.ParseYAML(generateLargeYAML(500))

	b.Run("SmallDocuments", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.Merge(ctx, smallBase, smallOverride).Execute()
			if err != nil {
				b.Fatalf("Merge error: %v", err)
			}
		}
	})

	b.Run("LargeDocuments", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.Merge(ctx, largeBase, largeOverride).Execute()
			if err != nil {
				b.Fatalf("Merge error: %v", err)
			}
		}
	})

	b.Run("MultipleSmallDocuments", func(b *testing.B) {
		docs := make([]DocumentV2, 10)
		for i := 0; i < 10; i++ {
			docs[i], _ = engine.ParseYAML([]byte(fmt.Sprintf(`
name: doc%d
value: %d
`, i, i)))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.Merge(ctx, docs...).Execute()
			if err != nil {
				b.Fatalf("Merge error: %v", err)
			}
		}
	})
}

func BenchmarkEngineV2_Evaluate(b *testing.B) {
	engine, err := NewEngineV2()
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}

	ctx := context.Background()

	// Simple operators
	simpleDoc, _ := engine.ParseYAML([]byte(`
meta:
  app_name: "myapp"
  version: "1.0"

name: (( concat meta.app_name "-" meta.version ))
full_name: (( grab name ))
`))

	// Complex operators
	complexDoc, _ := engine.ParseYAML([]byte(`
meta:
  app_name: "myapp"
  version: "1.0"
  environment: "production"

name: (( concat meta.app_name "-" meta.version ))
database:
  name: (( concat meta.app_name "_" meta.environment "_db" ))
  pool_size: (( calc meta.environment == "production" ? 20 : 5 ))

features:
  enabled: (( calc meta.environment == "production" || meta.environment == "staging" ))
  debug: (( calc meta.environment == "development" ))

config:
  app_name: (( grab meta.app_name ))
  db_name: (( grab database.name ))
  pool_size: (( grab database.pool_size ))
`))

	// Document with many operators
	manyOperatorsDoc := generateDocumentWithManyOperators(100)

	b.Run("SimpleOperators", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.Evaluate(ctx, simpleDoc)
			if err != nil {
				b.Fatalf("Evaluate error: %v", err)
			}
		}
	})

	b.Run("ComplexOperators", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.Evaluate(ctx, complexDoc)
			if err != nil {
				b.Fatalf("Evaluate error: %v", err)
			}
		}
	})

	b.Run("ManyOperators", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.Evaluate(ctx, manyOperatorsDoc)
			if err != nil {
				b.Fatalf("Evaluate error: %v", err)
			}
		}
	})
}

func BenchmarkDocument_Operations(b *testing.B) {
	engine, err := NewEngineV2()
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}

	largeDoc, _ := engine.ParseYAML(generateLargeYAML(1000))

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := largeDoc.Get("item_500.config.enabled")
			if err != nil {
				b.Fatalf("Get error: %v", err)
			}
		}
	})

	b.Run("Clone", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = largeDoc.Clone()
		}
	})

	b.Run("ToYAML", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := largeDoc.ToYAML()
			if err != nil {
				b.Fatalf("ToYAML error: %v", err)
			}
		}
	})

	b.Run("ToJSON", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := largeDoc.ToJSON()
			if err != nil {
				b.Fatalf("ToJSON error: %v", err)
			}
		}
	})
}

func BenchmarkEngineV2_MergeAndEvaluate(b *testing.B) {
	engine, err := NewEngineV2()
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}

	ctx := context.Background()

	base, _ := engine.ParseYAML([]byte(`
meta:
  app_name: "myapp"
  version: "1.0"
  environment: "production"

database:
  host: "localhost"
  port: 5432
`))

	override, _ := engine.ParseYAML([]byte(`
database:
  host: "prod.example.com"
  ssl: true
  name: (( concat meta.app_name "_" meta.environment "_db" ))

deployment:
  replicas: (( calc meta.environment == "production" ? 3 : 1 ))
  image: (( concat meta.app_name ":" meta.version ))
`))

	b.Run("MergeAndEvaluate", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			merged, err := engine.Merge(ctx, base, override).Execute()
			if err != nil {
				b.Fatalf("Merge error: %v", err)
			}

			_, err = engine.Evaluate(ctx, merged)
			if err != nil {
				b.Fatalf("Evaluate error: %v", err)
			}
		}
	})
}

func BenchmarkEngineV2_Concurrency(b *testing.B) {
	engine, err := NewEngineV2(WithConcurrency(10))
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}

	ctx := context.Background()
	doc := generateDocumentWithManyOperators(200)

	b.Run("Sequential", func(b *testing.B) {
		sequentialEngine, _ := NewEngineV2(WithConcurrency(1))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := sequentialEngine.Evaluate(ctx, doc)
			if err != nil {
				b.Fatalf("Evaluate error: %v", err)
			}
		}
	})

	b.Run("Concurrent", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.Evaluate(ctx, doc)
			if err != nil {
				b.Fatalf("Evaluate error: %v", err)
			}
		}
	})
}

// Helper functions for generating test data

func generateLargeYAML(itemCount int) []byte {
	var builder strings.Builder
	builder.WriteString("items:\n")

	for i := 0; i < itemCount; i++ {
		builder.WriteString(fmt.Sprintf(`  item_%d:
    name: "item_%d"
    value: %d
    config:
      enabled: %t
      timeout: %d
      tags:
        - "tag_%d_1"
        - "tag_%d_2"
        - "tag_%d_3"
`, i, i, i, i%2 == 0, 30+i%100, i, i, i))
	}

	return []byte(builder.String())
}

func generateLargeJSON(itemCount int) []byte {
	var builder strings.Builder
	builder.WriteString(`{"items":{`)

	for i := 0; i < itemCount; i++ {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(fmt.Sprintf(`"item_%d":{"name":"item_%d","value":%d,"config":{"enabled":%t,"timeout":%d,"tags":["tag_%d_1","tag_%d_2","tag_%d_3"]}}`,
			i, i, i, i%2 == 0, 30+i%100, i, i, i))
	}

	builder.WriteString("}}")
	return []byte(builder.String())
}

func generateDocumentWithManyOperators(operatorCount int) DocumentV2 {
	engine, _ := NewEngineV2()

	var builder strings.Builder
	builder.WriteString("meta:\n")
	builder.WriteString("  base_name: \"app\"\n")
	builder.WriteString("  version: \"1.0\"\n")
	builder.WriteString("  count: 100\n\n")

	builder.WriteString("values:\n")
	for i := 0; i < operatorCount; i++ {
		switch i % 4 {
		case 0:
			builder.WriteString(fmt.Sprintf("  value_%d: (( concat meta.base_name \"_%d\" ))\n", i, i))
		case 1:
			builder.WriteString(fmt.Sprintf("  value_%d: (( grab meta.version ))\n", i))
		case 2:
			builder.WriteString(fmt.Sprintf("  value_%d: (( calc meta.count + %d ))\n", i, i))
		case 3:
			builder.WriteString(fmt.Sprintf("  value_%d: (( calc %d > 50 ? \"large\" : \"small\" ))\n", i, i))
		}
	}

	doc, _ := engine.ParseYAML([]byte(builder.String()))
	return doc
}