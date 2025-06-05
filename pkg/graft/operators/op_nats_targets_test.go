package operators

import (
	"os"
	"testing"
	"time"
)

func TestNatsClientPool(t *testing.T) {
	// Test the NatsClientPool functionality
	
	// Set up environment variables for a test target
	os.Setenv("NATS_PRODUCTION_URL", "nats://nats.production.example.com:4222")
	os.Setenv("NATS_PRODUCTION_TIMEOUT", "10s")
	os.Setenv("NATS_PRODUCTION_RETRIES", "5")
	os.Setenv("NATS_PRODUCTION_TLS", "true")
	os.Setenv("NATS_PRODUCTION_CACHE_TTL", "10m")
	defer func() {
		os.Unsetenv("NATS_PRODUCTION_URL")
		os.Unsetenv("NATS_PRODUCTION_TIMEOUT")
		os.Unsetenv("NATS_PRODUCTION_RETRIES")
		os.Unsetenv("NATS_PRODUCTION_TLS")
		os.Unsetenv("NATS_PRODUCTION_CACHE_TTL")
	}()
	
	// Create a new client pool
	pool := &NatsClientPool{
		connections: make(map[string]*pooledConnection),
		configs:     make(map[string]*NatsTarget),
	}
	
	// Test target config retrieval
	config, err := pool.getTargetConfig("production")
	if err != nil {
		t.Fatalf("Expected to get target config, got error: %v", err)
	}
	
	if config.URL != "nats://nats.production.example.com:4222" {
		t.Errorf("Expected URL 'nats://nats.production.example.com:4222', got '%s'", config.URL)
	}
	
	if config.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", config.Timeout)
	}
	
	if config.Retries != 5 {
		t.Errorf("Expected retries 5, got %d", config.Retries)
	}
	
	if !config.TLS {
		t.Error("Expected TLS to be true")
	}
	
	if config.CacheTTL != 10*time.Minute {
		t.Errorf("Expected cache TTL 10m, got %v", config.CacheTTL)
	}
}

func TestNatsClientPoolMissingConfig(t *testing.T) {
	// Test error handling when target config is missing
	
	pool := &NatsClientPool{
		connections: make(map[string]*pooledConnection),
		configs:     make(map[string]*NatsTarget),
	}
	
	// Try to get config for non-existent target
	_, err := pool.getTargetConfig("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent target, got nil")
	}
	
	expectedErrMsg := "NATS target 'nonexistent' configuration incomplete"
	if err != nil && err.Error()[:len(expectedErrMsg)] != expectedErrMsg {
		t.Errorf("Expected error message to start with '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestNatsOperatorCacheKey(t *testing.T) {
	// Test cache key generation
	
	op := NatsOperator{}
	
	// Test without target
	key1 := op.getCacheKey("", "kv", "store/key")
	if key1 != "kv:store/key" {
		t.Errorf("Expected cache key 'kv:store/key', got '%s'", key1)
	}
	
	// Test with target
	key2 := op.getCacheKey("production", "kv", "store/key")
	if key2 != "production@kv:store/key" {
		t.Errorf("Expected cache key 'production@kv:store/key', got '%s'", key2)
	}
	
	// Test object store
	key3 := op.getCacheKey("staging", "obj", "bucket/object")
	if key3 != "staging@obj:bucket/object" {
		t.Errorf("Expected cache key 'staging@obj:bucket/object', got '%s'", key3)
	}
}

func TestNatsTargetConfiguration(t *testing.T) {
	// Test NatsTarget structure
	
	target := &NatsTarget{
		URL:                "nats://nats.example.com:4222",
		Timeout:            15 * time.Second,
		Retries:            3,
		RetryInterval:      2 * time.Second,
		RetryBackoff:       1.5,
		MaxRetryInterval:   60 * time.Second,
		TLS:                true,
		CertFile:           "/path/to/cert.pem",
		KeyFile:            "/path/to/key.pem",
		CAFile:             "/path/to/ca.pem",
		InsecureSkipVerify: false,
		CacheTTL:           15 * time.Minute,
		StreamingThreshold: 5 * 1024 * 1024, // 5MB
		AuditLogging:       true,
	}
	
	if target.URL != "nats://nats.example.com:4222" {
		t.Errorf("Expected URL 'nats://nats.example.com:4222', got '%s'", target.URL)
	}
	
	if target.Timeout != 15*time.Second {
		t.Errorf("Expected timeout 15s, got %v", target.Timeout)
	}
	
	if target.Retries != 3 {
		t.Errorf("Expected retries 3, got %d", target.Retries)
	}
	
	if target.RetryInterval != 2*time.Second {
		t.Errorf("Expected retry interval 2s, got %v", target.RetryInterval)
	}
	
	if target.RetryBackoff != 1.5 {
		t.Errorf("Expected retry backoff 1.5, got %f", target.RetryBackoff)
	}
	
	if target.MaxRetryInterval != 60*time.Second {
		t.Errorf("Expected max retry interval 60s, got %v", target.MaxRetryInterval)
	}
	
	if !target.TLS {
		t.Error("Expected TLS to be true")
	}
	
	if target.CertFile != "/path/to/cert.pem" {
		t.Errorf("Expected cert file '/path/to/cert.pem', got '%s'", target.CertFile)
	}
	
	if target.KeyFile != "/path/to/key.pem" {
		t.Errorf("Expected key file '/path/to/key.pem', got '%s'", target.KeyFile)
	}
	
	if target.CAFile != "/path/to/ca.pem" {
		t.Errorf("Expected CA file '/path/to/ca.pem', got '%s'", target.CAFile)
	}
	
	if target.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be false")
	}
	
	if target.CacheTTL != 15*time.Minute {
		t.Errorf("Expected cache TTL 15m, got %v", target.CacheTTL)
	}
	
	if target.StreamingThreshold != 5*1024*1024 {
		t.Errorf("Expected streaming threshold 5MB, got %d", target.StreamingThreshold)
	}
	
	if !target.AuditLogging {
		t.Error("Expected AuditLogging to be true")
	}
}

// TestNatsOperatorTargetExtraction tests the target extraction mechanism
func TestNatsOperatorTargetExtraction(t *testing.T) {
	op := NatsOperator{}
	
	// For now, the extractTarget method returns empty string (placeholder)
	// In the full implementation, this would extract target from the operator call
	target := op.extractTarget(nil, nil)
	if target != "" {
		t.Errorf("Expected empty target (placeholder implementation), got '%s'", target)
	}
}

// Test environment variable parsing helpers
func TestEnvParsingHelpers(t *testing.T) {
	// Test getEnvOrDefault
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")
	
	result := getEnvOrDefault("TEST_VAR", "default")
	if result != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", result)
	}
	
	result = getEnvOrDefault("NONEXISTENT_VAR", "default")
	if result != "default" {
		t.Errorf("Expected 'default', got '%s'", result)
	}
	
	// Test parseDurationOrDefault
	duration := parseDurationOrDefault("30s", 10*time.Second)
	if duration != 30*time.Second {
		t.Errorf("Expected 30s, got %v", duration)
	}
	
	duration = parseDurationOrDefault("invalid", 10*time.Second)
	if duration != 10*time.Second {
		t.Errorf("Expected 10s (default), got %v", duration)
	}
	
	// Test parseIntOrDefault
	intVal := parseIntOrDefault("42", 0)
	if intVal != 42 {
		t.Errorf("Expected 42, got %d", intVal)
	}
	
	intVal = parseIntOrDefault("invalid", 10)
	if intVal != 10 {
		t.Errorf("Expected 10 (default), got %d", intVal)
	}
	
	// Test parseBoolOrDefault
	boolVal := parseBoolOrDefault("true", false)
	if !boolVal {
		t.Error("Expected true, got false")
	}
	
	boolVal = parseBoolOrDefault("invalid", true)
	if !boolVal {
		t.Error("Expected true (default), got false")
	}
	
	// Test parseFloatOrDefault
	floatVal := parseFloatOrDefault("3.14", 0.0)
	if floatVal != 3.14 {
		t.Errorf("Expected 3.14, got %f", floatVal)
	}
	
	floatVal = parseFloatOrDefault("invalid", 2.0)
	if floatVal != 2.0 {
		t.Errorf("Expected 2.0 (default), got %f", floatVal)
	}
	
	// Test parseInt64OrDefault
	int64Val := parseInt64OrDefault("9223372036854775807", 0)
	if int64Val != 9223372036854775807 {
		t.Errorf("Expected 9223372036854775807, got %d", int64Val)
	}
	
	int64Val = parseInt64OrDefault("invalid", 100)
	if int64Val != 100 {
		t.Errorf("Expected 100 (default), got %d", int64Val)
	}
}

// BenchmarkNatsCacheKey benchmarks cache key generation
func BenchmarkNatsCacheKey(b *testing.B) {
	op := NatsOperator{}
	
	b.Run("NoTarget", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = op.getCacheKey("", "kv", "store/key/path/to/test")
		}
	})
	
	b.Run("WithTarget", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = op.getCacheKey("production", "kv", "store/key/path/to/test")
		}
	})
	
	b.Run("ObjectStore", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = op.getCacheKey("staging", "obj", "bucket/object/path/to/test")
		}
	})
}

// Example of how the NATS operator with targets would be used
func ExampleNatsOperator_withTargets() {
	// This example shows the intended usage of NATS operator with targets
	
	// Set up target configuration (normally done via config file or environment)
	os.Setenv("NATS_PRODUCTION_URL", "nats://nats.prod.example.com:4222")
	os.Setenv("NATS_PRODUCTION_TLS", "true")
	os.Setenv("NATS_PRODUCTION_CERT_FILE", "/etc/ssl/nats-prod.crt")
	os.Setenv("NATS_PRODUCTION_KEY_FILE", "/etc/ssl/nats-prod.key")
	
	os.Setenv("NATS_STAGING_URL", "nats://nats.staging.example.com:4222")
	os.Setenv("NATS_STAGING_TLS", "false")
	
	// In YAML template, you would use:
	// production_config: (( nats@production "kv:config/app" ))
	// staging_config: (( nats@staging "kv:config/app" ))
	// default_config: (( nats "kv:config/app" ))  // uses default configuration
	// binary_data: (( nats@production "obj:uploads/data.bin" ))
	
	// Output: NATS targets configured for production and staging
}