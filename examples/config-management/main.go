package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/wayneeseguin/graft/pkg/graft"
)

// This example demonstrates using Graft as a configuration management library
// for a microservices application with environment-specific overrides

func main() {
	fmt.Println("=== Graft Configuration Management Example ===\n")

	// Create engine
	engine, err := graft.NewEngineV2()
	if err != nil {
		log.Fatal("Failed to create engine:", err)
	}

	// Simulate different environments
	environments := []string{"development", "staging", "production"}
	
	for _, env := range environments {
		fmt.Printf("Generating configuration for: %s\n", env)
		config := generateConfig(engine, env)
		
		// Save to file
		filename := fmt.Sprintf("config-%s.yaml", env)
		saveConfig(engine, config, filename)
		
		// Display key configuration values
		displayKeyConfig(config, env)
		fmt.Println()
	}
}

func generateConfig(engine graft.EngineV2, environment string) graft.DocumentV2 {
	// Base configuration
	baseConfig := `
application:
  name: user-service
  version: 1.0.0
  
database:
  driver: postgresql
  port: 5432
  ssl_mode: require
  
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  
logging:
  level: info
  format: json
  
monitoring:
  enabled: true
  port: 9090
`

	// Environment-specific configurations
	envConfigs := map[string]string{
		"development": `
database:
  host: localhost
  name: userservice_dev
  ssl_mode: disable
  
server:
  port: 3000
  
logging:
  level: debug
  
monitoring:
  enabled: false
  
features:
  debug_mode: true
  hot_reload: true
`,
		"staging": `
database:
  host: staging-db.internal
  name: userservice_staging
  
server:
  replicas: 2
  
features:
  debug_mode: false
  hot_reload: false
  
resources:
  cpu: 200m
  memory: 256Mi
`,
		"production": `
database:
  host: prod-db.internal
  name: userservice_production
  pool_size: 20
  
server:
  replicas: 5
  
resources:
  cpu: 500m
  memory: 512Mi
  
security:
  tls_enabled: true
  rate_limiting: true
  
features:
  debug_mode: false
  hot_reload: false
`,
	}

	// Template with operators for dynamic configuration
	template := `
metadata:
  environment: ` + environment + `
  generated_at: "2024-01-01T00:00:00Z"
  
application:
  full_name: (( concat application.name "-" metadata.environment ))
  
database:
  url: (( concat database.driver "://" database.host ":" database.port "/" database.name ))
  
server:
  bind_address: (( concat "0.0.0.0:" server.port ))
`

	// Parse all configurations
	baseDoc, _ := engine.ParseYAML([]byte(baseConfig))
	envDoc, _ := engine.ParseYAML([]byte(envConfigs[environment]))
	templateDoc, _ := engine.ParseYAML([]byte(template))

	ctx := context.Background()

	// Merge base + environment + template
	merged, err := engine.Merge(ctx, baseDoc, envDoc, templateDoc).Execute()
	if err != nil {
		log.Fatal("Failed to merge config:", err)
	}

	// Evaluate operators
	evaluated, err := engine.Evaluate(ctx, merged)
	if err != nil {
		log.Fatal("Failed to evaluate config:", err)
	}

	return evaluated
}

func saveConfig(engine graft.EngineV2, config graft.DocumentV2, filename string) {
	yamlBytes, err := engine.ToYAML(config)
	if err != nil {
		log.Printf("Failed to convert to YAML: %v", err)
		return
	}

	err = os.WriteFile(filename, yamlBytes, 0644)
	if err != nil {
		log.Printf("Failed to write file %s: %v", filename, err)
		return
	}

	fmt.Printf("  ✓ Saved to %s\n", filename)
}

func displayKeyConfig(config graft.DocumentV2, env string) {
	// Extract key configuration values
	appName, _ := config.GetString("application.full_name")
	dbUrl, _ := config.GetString("database.url")
	serverAddr, _ := config.GetString("server.bind_address")
	logLevel, _ := config.GetString("logging.level")
	
	fmt.Printf("  Application: %s\n", appName)
	fmt.Printf("  Database: %s\n", dbUrl)
	fmt.Printf("  Server: %s\n", serverAddr)
	fmt.Printf("  Log Level: %s\n", logLevel)
	
	// Show environment-specific features
	switch env {
	case "development":
		if debugMode, err := config.GetBool("features.debug_mode"); err == nil {
			fmt.Printf("  Debug Mode: %v\n", debugMode)
		}
		if hotReload, err := config.GetBool("features.hot_reload"); err == nil {
			fmt.Printf("  Hot Reload: %v\n", hotReload)
		}
	case "staging", "production":
		if replicas, err := config.GetInt("server.replicas"); err == nil {
			fmt.Printf("  Replicas: %d\n", replicas)
		}
		if cpu, err := config.GetString("resources.cpu"); err == nil {
			fmt.Printf("  CPU: %s\n", cpu)
		}
		if memory, err := config.GetString("resources.memory"); err == nil {
			fmt.Printf("  Memory: %s\n", memory)
		}
	}
}