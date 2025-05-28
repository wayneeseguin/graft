package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/wayneeseguin/graft/pkg/graft"
)

func main() {
	fmt.Println("=== Graft Library Basic Usage Example ===")

	// Create a new engine
	engine, err := graft.NewEngine()
	if err != nil {
		log.Fatal("Failed to create engine:", err)
	}

	// Example 1: Simple YAML merging
	fmt.Println("1. Basic YAML Merging:")
	runBasicMerging(engine)

	// Example 2: Operator evaluation
	fmt.Println("\n2. Operator Evaluation:")
	runOperatorEvaluation(engine)

	// Example 3: Document access patterns
	fmt.Println("\n3. Document Access Patterns:")
	runDocumentAccess(engine)

	// Example 4: Multi-document merging
	fmt.Println("\n4. Multi-Document Merging:")
	runMultiDocumentMerging(engine)
}

func runBasicMerging(engine graft.Engine) {
	base := `
name: my-app
config:
  port: 8080
  debug: false
`

	override := `
config:
  port: 3000
  database: postgresql
`

	// Parse documents
	baseDoc, _ := engine.ParseYAML([]byte(base))
	overrideDoc, _ := engine.ParseYAML([]byte(override))

	// Merge documents
	ctx := context.Background()
	result, err := engine.Merge(ctx, baseDoc, overrideDoc).Execute()
	if err != nil {
		log.Fatal("Failed to merge:", err)
	}

	// Output result
	yamlOut, _ := engine.ToYAML(result)
	fmt.Printf("Merged result:\n%s", yamlOut)
}

func runOperatorEvaluation(engine graft.Engine) {
	yamlContent := `
app:
  name: web-service
  environment: production
  
config:
  database_url: (( concat app.name "-" app.environment ))
  service_name: (( grab app.name ))
  port: 8080
`

	// Parse and evaluate
	doc, _ := engine.ParseYAML([]byte(yamlContent))
	ctx := context.Background()
	evaluated, err := engine.Evaluate(ctx, doc)
	if err != nil {
		log.Fatal("Failed to evaluate:", err)
	}

	// Show results
	yamlOut, _ := engine.ToYAML(evaluated)
	fmt.Printf("Evaluated result:\n%s", yamlOut)
}

func runDocumentAccess(engine graft.Engine) {
	yamlContent := `
application:
  name: api-server
  version: 1.2.3
  enabled: true
  replicas: 3
  configs:
    - name: dev
      url: dev.example.com
    - name: prod
      url: prod.example.com
`

	doc, _ := engine.ParseYAML([]byte(yamlContent))

	// Demonstrate type-safe access
	name, _ := doc.GetString("application.name")
	version, _ := doc.GetString("application.version")
	enabled, _ := doc.GetBool("application.enabled")
	replicas, _ := doc.GetInt("application.replicas")

	fmt.Printf("Application: %s v%s\n", name, version)
	fmt.Printf("Enabled: %v, Replicas: %d\n", enabled, replicas)

	// Access nested arrays
	if configs, err := doc.Get("application.configs"); err == nil {
		if configList, ok := configs.([]interface{}); ok {
			fmt.Printf("Configurations:\n")
			for _, config := range configList {
				if configMap, ok := config.(map[interface{}]interface{}); ok {
					fmt.Printf("  - %v: %v\n", configMap["name"], configMap["url"])
				}
			}
		}
	}
}

func runMultiDocumentMerging(engine graft.Engine) {
	// Base configuration
	base := `
service:
  name: web-app
  port: 8080
resources:
  cpu: 100m
  memory: 128Mi
`

	// Environment-specific overrides
	development := `
service:
  debug: true
resources:
  cpu: 50m
environment: development
`

	production := `
service:
  replicas: 3
resources:
  cpu: 500m
  memory: 512Mi
environment: production
`

	// Parse all documents
	baseDoc, _ := engine.ParseYAML([]byte(base))
	devDoc, _ := engine.ParseYAML([]byte(development))
	prodDoc, _ := engine.ParseYAML([]byte(production))

	ctx := context.Background()

	// Create development configuration
	devConfig, _ := engine.Merge(ctx, baseDoc, devDoc).Execute()
	
	// Create production configuration  
	prodConfig, _ := engine.Merge(ctx, baseDoc, prodDoc).Execute()

	fmt.Println("Development configuration:")
	devYaml, _ := engine.ToYAML(devConfig)
	fmt.Printf("%s\n", indentOutput(devYaml))

	fmt.Println("Production configuration:")
	prodYaml, _ := engine.ToYAML(prodConfig)
	fmt.Printf("%s\n", indentOutput(prodYaml))
}

func indentOutput(output []byte) []byte {
	lines := strings.Split(string(output), "\n")
	var indented []string
	for _, line := range lines {
		if line != "" {
			indented = append(indented, "  "+line)
		}
	}
	return []byte(strings.Join(indented, "\n"))
}