package graft

import (
	"context"
	"testing"
)

// TestMockEngineV2 demonstrates how to use the mock engine for testing
func TestMockEngineV2(t *testing.T) {
	// Create a mock engine
	mockEngine := NewMockEngineV2()
	
	// Test that it implements the interface
	var _ EngineV2 = mockEngine
	
	// Test basic functionality
	doc, err := mockEngine.ParseYAML([]byte("test: value"))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if doc == nil {
		t.Error("Expected document, got nil")
	}
	
	// Verify call tracking
	if len(mockEngine.ParseYAMLCalls) != 1 {
		t.Errorf("Expected 1 ParseYAML call, got %d", len(mockEngine.ParseYAMLCalls))
	}
	if string(mockEngine.ParseYAMLCalls[0]) != "test: value" {
		t.Errorf("Expected 'test: value', got %s", string(mockEngine.ParseYAMLCalls[0]))
	}
}

// TestMockEngineV2CustomBehavior demonstrates customizing mock behavior
func TestMockEngineV2CustomBehavior(t *testing.T) {
	mockEngine := NewMockEngineV2()
	
	// Customize behavior to return an error
	mockEngine.ParseYAMLFunc = func(data []byte) (DocumentV2, error) {
		return nil, NewParseError("mock error", nil)
	}
	
	// Test the custom behavior
	doc, err := mockEngine.ParseYAML([]byte("invalid"))
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if doc != nil {
		t.Error("Expected nil document, got non-nil")
	}
	
	// Verify the error type
	if _, ok := err.(*GraftError); !ok {
		t.Errorf("Expected GraftError, got %T", err)
	}
}

// TestMockDocumentV2 demonstrates how to use the mock document
func TestMockDocumentV2(t *testing.T) {
	mockDoc := NewMockDocumentV2()
	
	// Test that it implements the interface
	var _ DocumentV2 = mockDoc
	
	// Test basic functionality
	err := mockDoc.Set("test.path", "value")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	value, err := mockDoc.Get("test.path")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if value != nil {
		// Default mock returns nil, but this shows it's working
	}
	
	// Verify call tracking
	if len(mockDoc.SetCalls) != 1 {
		t.Errorf("Expected 1 Set call, got %d", len(mockDoc.SetCalls))
	}
	if mockDoc.SetCalls[0].Path != "test.path" {
		t.Errorf("Expected 'test.path', got %s", mockDoc.SetCalls[0].Path)
	}
	if mockDoc.SetCalls[0].Value != "value" {
		t.Errorf("Expected 'value', got %v", mockDoc.SetCalls[0].Value)
	}
	
	if len(mockDoc.GetCalls) != 1 {
		t.Errorf("Expected 1 Get call, got %d", len(mockDoc.GetCalls))
	}
}

// TestMockDocumentV2CustomBehavior demonstrates customizing mock document behavior
func TestMockDocumentV2CustomBehavior(t *testing.T) {
	mockDoc := NewMockDocumentV2()
	
	// Set up test data
	mockDoc.TestData["name"] = "test-app"
	mockDoc.TestData["port"] = 8080
	
	// Customize behavior to return test data
	mockDoc.GetStringFunc = func(path string) (string, error) {
		if path == "name" {
			return mockDoc.TestData["name"].(string), nil
		}
		return "", NewValidationError("path not found: " + path)
	}
	
	mockDoc.GetIntFunc = func(path string) (int, error) {
		if path == "port" {
			return mockDoc.TestData["port"].(int), nil
		}
		return 0, NewValidationError("path not found: " + path)
	}
	
	// Test the custom behavior
	name, err := mockDoc.GetString("name")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if name != "test-app" {
		t.Errorf("Expected 'test-app', got %s", name)
	}
	
	port, err := mockDoc.GetInt("port")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if port != 8080 {
		t.Errorf("Expected 8080, got %d", port)
	}
	
	// Test error case
	_, err = mockDoc.GetString("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent path")
	}
}

// TestMockMergeBuilder demonstrates how to use the mock merge builder
func TestMockMergeBuilder(t *testing.T) {
	mockBuilder := NewMockMergeBuilder()
	
	// Test that it implements the interface
	var _ MergeBuilder = mockBuilder
	
	// Test fluent interface
	result, err := mockBuilder.
		WithPrune("secret").
		WithCherryPick("config").
		SkipEvaluation().
		Execute()
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Error("Expected result, got nil")
	}
	
	// Verify call tracking
	if len(mockBuilder.WithPruneCalls) != 1 {
		t.Errorf("Expected 1 WithPrune call, got %d", len(mockBuilder.WithPruneCalls))
	}
	if len(mockBuilder.WithPruneCalls[0]) != 1 || mockBuilder.WithPruneCalls[0][0] != "secret" {
		t.Errorf("Expected WithPrune('secret'), got %v", mockBuilder.WithPruneCalls[0])
	}
	
	if len(mockBuilder.WithCherryPickCalls) != 1 {
		t.Errorf("Expected 1 WithCherryPick call, got %d", len(mockBuilder.WithCherryPickCalls))
	}
	
	if mockBuilder.SkipEvaluationCalls != 1 {
		t.Errorf("Expected 1 SkipEvaluation call, got %d", mockBuilder.SkipEvaluationCalls)
	}
	
	if mockBuilder.ExecuteCalls != 1 {
		t.Errorf("Expected 1 Execute call, got %d", mockBuilder.ExecuteCalls)
	}
}

// TestMockUsage demonstrates practical usage of mocks in testing
func TestMockUsage(t *testing.T) {
	// This example shows how to test a service that uses the graft library
	
	// Create a mock engine for testing
	mockEngine := NewMockEngineV2()
	
	// Set up expected behavior
	expectedDoc := NewMockDocumentV2()
	expectedDoc.GetStringFunc = func(path string) (string, error) {
		if path == "database.url" {
			return "postgresql://localhost:5432/test", nil
		}
		return "", NewValidationError("path not found")
	}
	
	mockEngine.ParseFileFunc = func(path string) (DocumentV2, error) {
		return expectedDoc, nil
	}
	
	mockEngine.EvaluateFunc = func(ctx context.Context, doc DocumentV2) (DocumentV2, error) {
		return doc, nil // Just return the document unchanged
	}
	
	// Use the mock in your service
	service := &ConfigService{engine: mockEngine}
	
	config, err := service.LoadConfig("config.yaml")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Verify the service called the engine correctly
	if len(mockEngine.ParseFileCalls) != 1 {
		t.Errorf("Expected 1 ParseFile call, got %d", len(mockEngine.ParseFileCalls))
	}
	if mockEngine.ParseFileCalls[0] != "config.yaml" {
		t.Errorf("Expected 'config.yaml', got %s", mockEngine.ParseFileCalls[0])
	}
	
	// Verify the service processed the config correctly
	if config.DatabaseURL != "postgresql://localhost:5432/test" {
		t.Errorf("Expected database URL to be set correctly, got %s", config.DatabaseURL)
	}
}

// ConfigService is an example service that uses the graft library
type ConfigService struct {
	engine EngineV2
}

// TestConfig represents the application configuration for testing
type TestConfig struct {
	DatabaseURL string
}

// LoadConfig loads and parses a configuration file
func (s *ConfigService) LoadConfig(path string) (*TestConfig, error) {
	// Parse the config file
	doc, err := s.engine.ParseFile(path)
	if err != nil {
		return nil, err
	}
	
	// Evaluate any operators
	evaluated, err := s.engine.Evaluate(context.Background(), doc)
	if err != nil {
		return nil, err
	}
	
	// Extract configuration values
	dbURL, err := evaluated.GetString("database.url")
	if err != nil {
		return nil, err
	}
	
	return &TestConfig{
		DatabaseURL: dbURL,
	}, nil
}