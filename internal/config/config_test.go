package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	if cfg.Engine.DataflowOrder != "breadth-first" {
		t.Errorf("Expected dataflow order 'breadth-first', got '%s'", cfg.Engine.DataflowOrder)
	}
	
	if cfg.Engine.OutputFormat != "yaml" {
		t.Errorf("Expected output format 'yaml', got '%s'", cfg.Engine.OutputFormat)
	}
	
	if !cfg.Engine.ColorOutput {
		t.Error("Expected color output to be true")
	}
	
	if cfg.Engine.StrictMode {
		t.Error("Expected strict mode to be false")
	}
	
	if !cfg.Performance.EnableCaching {
		t.Error("Expected caching to be enabled")
	}
	
	if !cfg.Performance.EnableParallel {
		t.Error("Expected parallel processing to be enabled")
	}
	
	if cfg.Performance.Cache.ExpressionCacheSize != 10000 {
		t.Errorf("Expected expression cache size 10000, got %d", cfg.Performance.Cache.ExpressionCacheSize)
	}
	
	if cfg.Performance.Concurrency.MaxWorkers != 0 {
		t.Errorf("Expected max workers 0 (auto), got %d", cfg.Performance.Concurrency.MaxWorkers)
	}
	
	if cfg.Logging.Level != "info" {
		t.Errorf("Expected log level 'info', got '%s'", cfg.Logging.Level)
	}
	
	if cfg.Logging.Format != "text" {
		t.Errorf("Expected log format 'text', got '%s'", cfg.Logging.Format)
	}
	
	if cfg.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", cfg.Version)
	}
	
	if cfg.Profile != "default" {
		t.Errorf("Expected profile 'default', got '%s'", cfg.Profile)
	}
	
	if cfg.Features == nil {
		t.Error("Expected features map to be initialized")
	}
}

func TestNewManager(t *testing.T) {
	manager := NewManager()
	
	if manager == nil {
		t.Fatal("Expected manager to be created")
	}
	
	cfg := manager.Get()
	if cfg == nil {
		t.Fatal("Expected config to be available")
	}
	
	if cfg.Profile != "default" {
		t.Errorf("Expected default profile, got '%s'", cfg.Profile)
	}
}

func TestManagerLoad(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")
	
	configContent := `
version: "1.0"
profile: "test"
engine:
  dataflow_order: "depth-first"
  output_format: "json"
  color_output: false
performance:
  enable_caching: false
  cache:
    expression_cache_size: 5000
logging:
  level: "debug"
features:
  test_feature: true
`
	
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	
	manager := NewManager()
	err = manager.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	cfg := manager.Get()
	if cfg.Profile != "test" {
		t.Errorf("Expected profile 'test', got '%s'", cfg.Profile)
	}
	
	if cfg.Engine.DataflowOrder != "depth-first" {
		t.Errorf("Expected dataflow order 'depth-first', got '%s'", cfg.Engine.DataflowOrder)
	}
	
	if cfg.Engine.OutputFormat != "json" {
		t.Errorf("Expected output format 'json', got '%s'", cfg.Engine.OutputFormat)
	}
	
	if cfg.Engine.ColorOutput {
		t.Error("Expected color output to be false")
	}
	
	if cfg.Performance.EnableCaching {
		t.Error("Expected caching to be disabled")
	}
	
	if cfg.Performance.Cache.ExpressionCacheSize != 5000 {
		t.Errorf("Expected cache size 5000, got %d", cfg.Performance.Cache.ExpressionCacheSize)
	}
	
	if cfg.Logging.Level != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", cfg.Logging.Level)
	}
	
	if !cfg.Features["test_feature"] {
		t.Error("Expected test_feature to be true")
	}
}

func TestManagerUpdate(t *testing.T) {
	manager := NewManager()
	
	err := manager.Update(func(cfg *Config) {
		cfg.Engine.DataflowOrder = "depth-first"
		cfg.Logging.Level = "error"
	})
	
	if err != nil {
		t.Fatalf("Unexpected error updating config: %v", err)
	}
	
	cfg := manager.Get()
	if cfg.Engine.DataflowOrder != "depth-first" {
		t.Errorf("Expected dataflow order 'depth-first', got '%s'", cfg.Engine.DataflowOrder)
	}
	
	if cfg.Logging.Level != "error" {
		t.Errorf("Expected log level 'error', got '%s'", cfg.Logging.Level)
	}
}

func TestManagerInvalidConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid_config.yaml")
	
	invalidContent := `
version: "1.0"
profile: "test"
engine:
  dataflow_order: "invalid_order"
  output_format: "invalid_format"
`
	
	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	
	manager := NewManager()
	err = manager.Load(configPath)
	if err == nil {
		t.Error("Expected error loading invalid config")
	}
}

func TestConfigSerialization(t *testing.T) {
	original := DefaultConfig()
	original.Engine.DataflowOrder = "depth-first"
	original.Performance.Cache.ExpressionCacheSize = 20000
	original.SetFeature("test_feature", true)
	
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("Error marshaling config: %v", err)
	}
	
	var restored Config
	err = yaml.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Error unmarshaling config: %v", err)
	}
	
	if original.Engine.DataflowOrder != restored.Engine.DataflowOrder {
		t.Errorf("Dataflow order not preserved: expected '%s', got '%s'", 
			original.Engine.DataflowOrder, restored.Engine.DataflowOrder)
	}
	
	if original.Performance.Cache.ExpressionCacheSize != restored.Performance.Cache.ExpressionCacheSize {
		t.Errorf("Cache size not preserved: expected %d, got %d",
			original.Performance.Cache.ExpressionCacheSize, restored.Performance.Cache.ExpressionCacheSize)
	}
}