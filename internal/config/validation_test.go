package config

import (
	"testing"
	"time"
)

func TestValidateValidConfig(t *testing.T) {
	cfg := DefaultConfig()
	err := Validate(cfg)
	if err != nil {
		t.Errorf("Valid config should not have validation errors: %v", err)
	}
}

func TestValidateEmptyVersion(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Version = ""
	
	err := Validate(cfg)
	if err == nil {
		t.Error("Expected validation error for empty version")
	}
	
	if !containsError(err, "version cannot be empty") {
		t.Errorf("Expected 'version cannot be empty' error, got: %v", err)
	}
}

func TestValidateInvalidDataflowOrder(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Engine.DataflowOrder = "invalid"
	
	err := Validate(cfg)
	if err == nil {
		t.Error("Expected validation error for invalid dataflow order")
	}
	
	if !containsError(err, "must be one of") {
		t.Errorf("Expected 'must be one of' error, got: %v", err)
	}
}

func TestValidateInvalidOutputFormat(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Engine.OutputFormat = "invalid"
	
	err := Validate(cfg)
	if err == nil {
		t.Error("Expected validation error for invalid output format")
	}
	
	if !containsError(err, "must be one of") {
		t.Errorf("Expected 'must be one of' error, got: %v", err)
	}
}

func TestValidateNegativeCacheSize(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Performance.Cache.ExpressionCacheSize = -1
	
	err := Validate(cfg)
	if err == nil {
		t.Error("Expected validation error for negative cache size")
	}
	
	if !containsError(err, "cannot be negative") {
		t.Errorf("Expected 'cannot be negative' error, got: %v", err)
	}
}

func TestValidateNegativeTTL(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Performance.Cache.TTL = -1 * time.Second
	
	err := Validate(cfg)
	if err == nil {
		t.Error("Expected validation error for negative TTL")
	}
	
	if !containsError(err, "cannot be negative") {
		t.Errorf("Expected 'cannot be negative' error, got: %v", err)
	}
}

func TestValidateZeroQueueSize(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Performance.Concurrency.QueueSize = 0
	
	err := Validate(cfg)
	if err == nil {
		t.Error("Expected validation error for zero queue size")
	}
	
	if !containsError(err, "must be greater than 0") {
		t.Errorf("Expected 'must be greater than 0' error, got: %v", err)
	}
}

func TestValidateInvalidLogLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logging.Level = "invalid"
	
	err := Validate(cfg)
	if err == nil {
		t.Error("Expected validation error for invalid log level")
	}
	
	if !containsError(err, "must be one of") {
		t.Errorf("Expected 'must be one of' error, got: %v", err)
	}
}

func TestValidateInvalidLogFormat(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Logging.Format = "invalid"
	
	err := Validate(cfg)
	if err == nil {
		t.Error("Expected validation error for invalid log format")
	}
	
	if !containsError(err, "must be one of") {
		t.Errorf("Expected 'must be one of' error, got: %v", err)
	}
}

func TestValidateVaultAddress(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Engine.Vault.Address = "not-a-url"
	
	err := Validate(cfg)
	if err == nil {
		t.Error("Expected validation error for invalid vault address")
	}
	
	if !containsError(err, "must have scheme and host") {
		t.Errorf("Expected 'must have scheme and host' error, got: %v", err)
	}
}

func TestValidateVaultTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Engine.Vault.Timeout = "invalid-duration"
	
	err := Validate(cfg)
	if err == nil {
		t.Error("Expected validation error for invalid vault timeout")
	}
	
	if !containsError(err, "invalid duration") {
		t.Errorf("Expected 'invalid duration' error, got: %v", err)
	}
}

func TestValidateAWSRegion(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Engine.AWS.Region = "invalid"
	
	err := Validate(cfg)
	if err == nil {
		t.Error("Expected validation error for invalid AWS region")
	}
	
	if !containsError(err, "invalid AWS region format") {
		t.Errorf("Expected 'invalid AWS region format' error, got: %v", err)
	}
}

func TestValidateAWSEndpoint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Engine.AWS.Endpoint = "not-a-url"
	
	err := Validate(cfg)
	if err == nil {
		t.Error("Expected validation error for invalid AWS endpoint")
	}
	
	if !containsError(err, "must have scheme and host") {
		t.Errorf("Expected 'must have scheme and host' error, got: %v", err)
	}
}

func TestValidateParserMaxDocumentSize(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Engine.Parser.MaxDocumentSize = 0
	
	err := Validate(cfg)
	if err == nil {
		t.Error("Expected validation error for zero max document size")
	}
	
	if !containsError(err, "must be greater than 0") {
		t.Errorf("Expected 'must be greater than 0' error, got: %v", err)
	}
}

func TestValidateParserLargeDocumentSize(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Engine.Parser.MaxDocumentSize = 200 * 1024 * 1024 // 200MB
	
	err := Validate(cfg)
	if err == nil {
		t.Error("Expected validation warning for very large document size")
	}
	
	if !containsError(err, "warning: very large document size") {
		t.Errorf("Expected large document size warning, got: %v", err)
	}
}

func TestValidationErrors(t *testing.T) {
	var errors ValidationErrors
	errors = append(errors, ValidationError{
		Field:   "test1",
		Value:   "value1",
		Message: "error1",
	})
	errors = append(errors, ValidationError{
		Field:   "test2", 
		Value:   "value2",
		Message: "error2",
	})
	
	errorStr := errors.Error()
	if !containsSubstring(errorStr, "test1") {
		t.Error("Error string should contain test1")
	}
	if !containsSubstring(errorStr, "error1") {
		t.Error("Error string should contain error1")
	}
	if !containsSubstring(errorStr, "test2") {
		t.Error("Error string should contain test2")
	}
	if !containsSubstring(errorStr, "error2") {
		t.Error("Error string should contain error2")
	}
	
	var emptyErrors ValidationErrors
	if emptyErrors.Error() != "" {
		t.Error("Empty validation errors should return empty string")
	}
}

// Helper functions
func containsError(err error, substr string) bool {
	if err == nil {
		return false
	}
	return containsSubstring(err.Error(), substr)
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
		 (s[:len(substr)] == substr || 
		  s[len(s)-len(substr):] == substr ||
		  containsSubstringHelper(s, substr))))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}