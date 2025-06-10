package operators

import (
	"fmt"
	"github.com/cloudfoundry-community/vaultkv"
	"os"
	"testing"
)

func TestVaultClientPool(t *testing.T) {
	// Test the VaultClientPool functionality

	// Set up environment variables for a test target
	os.Setenv("VAULT_PRODUCTION_ADDR", "https://vault.production.example.com")
	os.Setenv("VAULT_PRODUCTION_TOKEN", "test-token-123")
	os.Setenv("VAULT_PRODUCTION_NAMESPACE", "production")
	defer func() {
		os.Unsetenv("VAULT_PRODUCTION_ADDR")
		os.Unsetenv("VAULT_PRODUCTION_TOKEN")
		os.Unsetenv("VAULT_PRODUCTION_NAMESPACE")
	}()

	// Create a new client pool
	pool := &VaultClientPool{
		clients: make(map[string]*vaultkv.KV),
		configs: make(map[string]*VaultTarget),
	}

	// Test target config retrieval
	config, err := pool.getTargetConfig("production", nil)
	if err != nil {
		t.Fatalf("Expected to get target config, got error: %v", err)
	}

	if config.URL != "https://vault.production.example.com" {
		t.Errorf("Expected URL 'https://vault.production.example.com', got '%s'", config.URL)
	}

	if config.Token != "test-token-123" {
		t.Errorf("Expected token 'test-token-123', got '%s'", config.Token)
	}

	if config.Namespace != "production" {
		t.Errorf("Expected namespace 'production', got '%s'", config.Namespace)
	}
}

func TestVaultClientPoolMissingConfig(t *testing.T) {
	// Test error handling when target config is missing

	pool := &VaultClientPool{
		clients: make(map[string]*vaultkv.KV),
		configs: make(map[string]*VaultTarget),
	}

	// Try to get config for non-existent target
	_, err := pool.getTargetConfig("nonexistent", nil)
	if err == nil {
		t.Error("Expected error for non-existent target, got nil")
	}

	expectedErrMsg := "vault target 'nonexistent' configuration not found"
	if err != nil && err.Error()[:len(expectedErrMsg)] != expectedErrMsg {
		t.Errorf("Expected error message to start with '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestVaultOperatorCacheKey(t *testing.T) {
	// Test cache key generation

	op := VaultOperator{}

	// Test without target
	key1 := op.getCacheKey("", "secret/path")
	if key1 != "secret/path" {
		t.Errorf("Expected cache key 'secret/path', got '%s'", key1)
	}

	// Test with target
	key2 := op.getCacheKey("production", "secret/path")
	if key2 != "production@secret/path" {
		t.Errorf("Expected cache key 'production@secret/path', got '%s'", key2)
	}
}

func TestVaultTargetConfiguration(t *testing.T) {
	// Test VaultTarget structure

	target := &VaultTarget{
		URL:        "https://vault.example.com",
		Token:      "test-token",
		Namespace:  "test-namespace",
		SkipVerify: true,
	}

	if target.URL != "https://vault.example.com" {
		t.Errorf("Expected URL 'https://vault.example.com', got '%s'", target.URL)
	}

	if target.Token != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", target.Token)
	}

	if target.Namespace != "test-namespace" {
		t.Errorf("Expected namespace 'test-namespace', got '%s'", target.Namespace)
	}

	if !target.SkipVerify {
		t.Error("Expected SkipVerify to be true")
	}
}

// TestVaultOperatorTargetExtraction tests the target extraction mechanism
func TestVaultOperatorTargetExtraction(t *testing.T) {
	op := VaultOperator{}

	// For now, the extractTarget method returns empty string (placeholder)
	// In the full implementation, this would extract target from the operator call
	target := op.extractTarget(nil, nil)
	if target != "" {
		t.Errorf("Expected empty target (placeholder implementation), got '%s'", target)
	}
}

// BenchmarkVaultCacheKey benchmarks cache key generation
func BenchmarkVaultCacheKey(b *testing.B) {
	op := VaultOperator{}

	b.Run("NoTarget", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = op.getCacheKey("", "secret/path/to/test")
		}
	})

	b.Run("WithTarget", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = op.getCacheKey("production", "secret/path/to/test")
		}
	})
}

// Example of how the vault operator with targets would be used
func ExampleVaultOperator_withTargets() {
	// This example shows the intended usage of vault operator with targets

	// Set up target configuration (normally done via config file or environment)
	os.Setenv("VAULT_PRODUCTION_ADDR", "https://vault.prod.example.com")
	os.Setenv("VAULT_PRODUCTION_TOKEN", "prod-token")

	os.Setenv("VAULT_STAGING_ADDR", "https://vault.staging.example.com")
	os.Setenv("VAULT_STAGING_TOKEN", "staging-token")

	// In YAML template, you would use:
	// production_password: (( vault@production "secret/app:password" ))
	// staging_password: (( vault@staging "secret/app:password" ))
	// default_password: (( vault "secret/app:password" ))  // uses default configuration

	fmt.Println("Vault targets configured for production and staging")

	// Output:
	// Vault targets configured for production and staging
}
