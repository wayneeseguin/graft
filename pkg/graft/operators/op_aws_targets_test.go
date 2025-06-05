package operators

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

func TestAwsClientPool(t *testing.T) {
	// Test the AwsClientPool functionality
	
	// Set up environment variables for a test target
	os.Setenv("AWS_PRODUCTION_REGION", "us-east-1")
	os.Setenv("AWS_PRODUCTION_PROFILE", "production")
	os.Setenv("AWS_PRODUCTION_ROLE", "arn:aws:iam::123456789012:role/GraftRole")
	os.Setenv("AWS_PRODUCTION_ACCESS_KEY_ID", "AKIATEST12345")
	os.Setenv("AWS_PRODUCTION_SECRET_ACCESS_KEY", "secret-key-123")
	os.Setenv("AWS_PRODUCTION_MAX_RETRIES", "5")
	os.Setenv("AWS_PRODUCTION_HTTP_TIMEOUT", "60s")
	os.Setenv("AWS_PRODUCTION_CACHE_TTL", "15m")
	os.Setenv("AWS_PRODUCTION_AUDIT_LOGGING", "true")
	defer func() {
		os.Unsetenv("AWS_PRODUCTION_REGION")
		os.Unsetenv("AWS_PRODUCTION_PROFILE")
		os.Unsetenv("AWS_PRODUCTION_ROLE")
		os.Unsetenv("AWS_PRODUCTION_ACCESS_KEY_ID")
		os.Unsetenv("AWS_PRODUCTION_SECRET_ACCESS_KEY")
		os.Unsetenv("AWS_PRODUCTION_MAX_RETRIES")
		os.Unsetenv("AWS_PRODUCTION_HTTP_TIMEOUT")
		os.Unsetenv("AWS_PRODUCTION_CACHE_TTL")
		os.Unsetenv("AWS_PRODUCTION_AUDIT_LOGGING")
	}()
	
	// Create a new client pool
	pool := &AwsClientPool{
		sessions:             make(map[string]*session.Session),
		secretsManagerClients: make(map[string]secretsmanageriface.SecretsManagerAPI),
		parameterStoreClients: make(map[string]ssmiface.SSMAPI),
		configs:              make(map[string]*AwsTarget),
		secretsCache:         make(map[string]map[string]string),
		paramsCache:          make(map[string]map[string]string),
	}
	
	// Test target config retrieval
	config, err := pool.getTargetConfig("production")
	if err != nil {
		t.Fatalf("Expected to get target config, got error: %v", err)
	}
	
	if config.Region != "us-east-1" {
		t.Errorf("Expected region 'us-east-1', got '%s'", config.Region)
	}
	
	if config.Profile != "production" {
		t.Errorf("Expected profile 'production', got '%s'", config.Profile)
	}
	
	if config.Role != "arn:aws:iam::123456789012:role/GraftRole" {
		t.Errorf("Expected role 'arn:aws:iam::123456789012:role/GraftRole', got '%s'", config.Role)
	}
	
	if config.AccessKeyID != "AKIATEST12345" {
		t.Errorf("Expected access key ID 'AKIATEST12345', got '%s'", config.AccessKeyID)
	}
	
	if config.SecretAccessKey != "secret-key-123" {
		t.Errorf("Expected secret access key 'secret-key-123', got '%s'", config.SecretAccessKey)
	}
	
	if config.MaxRetries != 5 {
		t.Errorf("Expected max retries 5, got %d", config.MaxRetries)
	}
	
	if config.HTTPTimeout != 60*time.Second {
		t.Errorf("Expected HTTP timeout 60s, got %v", config.HTTPTimeout)
	}
	
	if config.CacheTTL != 15*time.Minute {
		t.Errorf("Expected cache TTL 15m, got %v", config.CacheTTL)
	}
	
	if !config.AuditLogging {
		t.Error("Expected audit logging to be true")
	}
}

func TestAwsClientPoolMissingConfig(t *testing.T) {
	// Test error handling when target config is missing
	
	pool := &AwsClientPool{
		sessions:             make(map[string]*session.Session),
		secretsManagerClients: make(map[string]secretsmanageriface.SecretsManagerAPI),
		parameterStoreClients: make(map[string]ssmiface.SSMAPI),
		configs:              make(map[string]*AwsTarget),
		secretsCache:         make(map[string]map[string]string),
		paramsCache:          make(map[string]map[string]string),
	}
	
	// Try to get config for non-existent target
	_, err := pool.getTargetConfig("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent target, got nil")
	}
	
	expectedErrMsg := "AWS target 'nonexistent' configuration incomplete"
	if err != nil && err.Error()[:len(expectedErrMsg)] != expectedErrMsg {
		t.Errorf("Expected error message to start with '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestAwsOperatorCacheKey(t *testing.T) {
	// Test cache key generation
	
	op := AwsOperator{variant: "awsparam"}
	
	// Test without target
	key1 := op.getCacheKey("", "awsparam", "/app/database/password")
	if key1 != "awsparam:/app/database/password" {
		t.Errorf("Expected cache key 'awsparam:/app/database/password', got '%s'", key1)
	}
	
	// Test with target
	key2 := op.getCacheKey("production", "awsparam", "/app/database/password")
	if key2 != "production@awsparam:/app/database/password" {
		t.Errorf("Expected cache key 'production@awsparam:/app/database/password', got '%s'", key2)
	}
	
	// Test secrets operator
	op = AwsOperator{variant: "awssecret"}
	key3 := op.getCacheKey("staging", "awssecret", "database-credentials")
	if key3 != "staging@awssecret:database-credentials" {
		t.Errorf("Expected cache key 'staging@awssecret:database-credentials', got '%s'", key3)
	}
}

func TestAwsTargetConfiguration(t *testing.T) {
	// Test AwsTarget structure
	
	target := &AwsTarget{
		Region:             "us-west-2",
		Profile:            "staging",
		Role:               "arn:aws:iam::987654321098:role/StagingRole",
		AccessKeyID:        "AKIASTAGING789",
		SecretAccessKey:    "staging-secret-key",
		SessionToken:       "staging-session-token",
		Endpoint:           "https://custom.aws.endpoint.com",
		S3ForcePathStyle:   true,
		DisableSSL:         false,
		MaxRetries:         10,
		HTTPTimeout:        45 * time.Second,
		CacheTTL:           20 * time.Minute,
		AssumeRoleDuration: 2 * time.Hour,
		ExternalID:         "external-id-123",
		SessionName:        "graft-staging",
		MfaSerial:          "arn:aws:iam::987654321098:mfa/user",
		AuditLogging:       true,
	}
	
	if target.Region != "us-west-2" {
		t.Errorf("Expected region 'us-west-2', got '%s'", target.Region)
	}
	
	if target.Profile != "staging" {
		t.Errorf("Expected profile 'staging', got '%s'", target.Profile)
	}
	
	if target.Role != "arn:aws:iam::987654321098:role/StagingRole" {
		t.Errorf("Expected role 'arn:aws:iam::987654321098:role/StagingRole', got '%s'", target.Role)
	}
	
	if target.AccessKeyID != "AKIASTAGING789" {
		t.Errorf("Expected access key ID 'AKIASTAGING789', got '%s'", target.AccessKeyID)
	}
	
	if target.SecretAccessKey != "staging-secret-key" {
		t.Errorf("Expected secret access key 'staging-secret-key', got '%s'", target.SecretAccessKey)
	}
	
	if target.SessionToken != "staging-session-token" {
		t.Errorf("Expected session token 'staging-session-token', got '%s'", target.SessionToken)
	}
	
	if target.Endpoint != "https://custom.aws.endpoint.com" {
		t.Errorf("Expected endpoint 'https://custom.aws.endpoint.com', got '%s'", target.Endpoint)
	}
	
	if !target.S3ForcePathStyle {
		t.Error("Expected S3ForcePathStyle to be true")
	}
	
	if target.DisableSSL {
		t.Error("Expected DisableSSL to be false")
	}
	
	if target.MaxRetries != 10 {
		t.Errorf("Expected max retries 10, got %d", target.MaxRetries)
	}
	
	if target.HTTPTimeout != 45*time.Second {
		t.Errorf("Expected HTTP timeout 45s, got %v", target.HTTPTimeout)
	}
	
	if target.CacheTTL != 20*time.Minute {
		t.Errorf("Expected cache TTL 20m, got %v", target.CacheTTL)
	}
	
	if target.AssumeRoleDuration != 2*time.Hour {
		t.Errorf("Expected assume role duration 2h, got %v", target.AssumeRoleDuration)
	}
	
	if target.ExternalID != "external-id-123" {
		t.Errorf("Expected external ID 'external-id-123', got '%s'", target.ExternalID)
	}
	
	if target.SessionName != "graft-staging" {
		t.Errorf("Expected session name 'graft-staging', got '%s'", target.SessionName)
	}
	
	if target.MfaSerial != "arn:aws:iam::987654321098:mfa/user" {
		t.Errorf("Expected MFA serial 'arn:aws:iam::987654321098:mfa/user', got '%s'", target.MfaSerial)
	}
	
	if !target.AuditLogging {
		t.Error("Expected audit logging to be true")
	}
}

// TestAwsOperatorTargetExtraction tests the target extraction mechanism
func TestAwsOperatorTargetExtraction(t *testing.T) {
	op := AwsOperator{variant: "awsparam"}
	
	// For now, the extractTarget method returns empty string (placeholder)
	// In the full implementation, this would extract target from the operator call
	target := op.extractTarget(nil, nil)
	if target != "" {
		t.Errorf("Expected empty target (placeholder implementation), got '%s'", target)
	}
}

// Test environment variable parsing helpers
func TestAwsEnvParsingHelpers(t *testing.T) {
	// Test getEnvOrDefault
	os.Setenv("TEST_AWS_VAR", "test_value")
	defer os.Unsetenv("TEST_AWS_VAR")
	
	result := getEnvOrDefault("TEST_AWS_VAR", "default")
	if result != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", result)
	}
	
	result = getEnvOrDefault("NONEXISTENT_AWS_VAR", "default")
	if result != "default" {
		t.Errorf("Expected 'default', got '%s'", result)
	}
	
	// Test parseDurationOrDefault
	duration := parseDurationOrDefault("45s", 30*time.Second)
	if duration != 45*time.Second {
		t.Errorf("Expected 45s, got %v", duration)
	}
	
	duration = parseDurationOrDefault("invalid", 30*time.Second)
	if duration != 30*time.Second {
		t.Errorf("Expected 30s (default), got %v", duration)
	}
	
	// Test parseIntOrDefault
	intVal := parseIntOrDefault("100", 0)
	if intVal != 100 {
		t.Errorf("Expected 100, got %d", intVal)
	}
	
	intVal = parseIntOrDefault("invalid", 50)
	if intVal != 50 {
		t.Errorf("Expected 50 (default), got %d", intVal)
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
}

func TestAwsClientPoolCaching(t *testing.T) {
	// Test cache functionality
	
	pool := &AwsClientPool{
		sessions:             make(map[string]*session.Session),
		secretsManagerClients: make(map[string]secretsmanageriface.SecretsManagerAPI),
		parameterStoreClients: make(map[string]ssmiface.SSMAPI),
		configs:              make(map[string]*AwsTarget),
		secretsCache:         make(map[string]map[string]string),
		paramsCache:          make(map[string]map[string]string),
	}
	
	targetName := "test"
	
	// Test secrets cache
	pool.SetSecretCache(targetName, "test-secret", "secret-value")
	cache := pool.GetSecretCache(targetName)
	if val, ok := cache["test-secret"]; !ok || val != "secret-value" {
		t.Errorf("Expected secret cache to contain 'secret-value', got '%s'", val)
	}
	
	// Test params cache
	pool.SetParamCache(targetName, "test-param", "param-value")
	cache = pool.GetParamCache(targetName)
	if val, ok := cache["test-param"]; !ok || val != "param-value" {
		t.Errorf("Expected param cache to contain 'param-value', got '%s'", val)
	}
}

// BenchmarkAwsCacheKey benchmarks cache key generation
func BenchmarkAwsCacheKey(b *testing.B) {
	op := AwsOperator{variant: "awsparam"}
	
	b.Run("NoTarget", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = op.getCacheKey("", "awsparam", "/app/database/password")
		}
	})
	
	b.Run("WithTarget", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = op.getCacheKey("production", "awsparam", "/app/database/password")
		}
	})
	
	b.Run("SecretOperator", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = op.getCacheKey("staging", "awssecret", "database-credentials")
		}
	})
}

// Test AWS operators variants
func TestAwsOperatorVariants(t *testing.T) {
	paramOp := NewAwsParamOperator()
	if paramOp.variant != "awsparam" {
		t.Errorf("Expected awsparam variant, got '%s'", paramOp.variant)
	}
	
	secretOp := NewAwsSecretOperator()
	if secretOp.variant != "awssecret" {
		t.Errorf("Expected awssecret variant, got '%s'", secretOp.variant)
	}
}

// Example of how the AWS operators with targets would be used
func ExampleAwsOperator_withTargets() {
	// This example shows the intended usage of AWS operators with targets
	
	// Set up target configuration (normally done via config file or environment)
	os.Setenv("AWS_PRODUCTION_REGION", "us-east-1")
	os.Setenv("AWS_PRODUCTION_PROFILE", "production")
	os.Setenv("AWS_PRODUCTION_ROLE", "arn:aws:iam::123456789012:role/ProdRole")
	os.Setenv("AWS_PRODUCTION_AUDIT_LOGGING", "true")
	
	os.Setenv("AWS_STAGING_REGION", "us-west-2")
	os.Setenv("AWS_STAGING_PROFILE", "staging")
	os.Setenv("AWS_STAGING_MAX_RETRIES", "10")
	
	// In YAML template, you would use:
	// production_db_password: (( awsparam@production "/app/prod/db/password" ))
	// staging_db_password: (( awsparam@staging "/app/staging/db/password" ))
	// production_secret: (( awssecret@production "database-credentials" ))
	// staging_secret: (( awssecret@staging "database-credentials" ))
	// default_param: (( awsparam "/app/dev/db/password" ))  // uses default configuration
	
	fmt.Println("AWS targets configured for production and staging")
	// Output: AWS targets configured for production and staging
}