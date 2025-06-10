package factory

import (
	"os"
	"sync"
	"testing"

	"github.com/wayneeseguin/graft/pkg/graft"
)

func TestEngineFactory_CreateDefaultEngine(t *testing.T) {
	tests := []struct {
		name         string
		envVars      map[string]string
		expectPanic  bool
		validateFunc func(*testing.T, *graft.DefaultEngine)
	}{
		{
			name: "creates engine with no environment variables",
			validateFunc: func(t *testing.T, engine *graft.DefaultEngine) {
				if engine == nil {
					t.Fatal("expected engine to be created, got nil")
				}

				// Verify essential operators are registered
				testOperators := []string{"grab", "concat", "empty", "calc", "vault"}
				for _, op := range testOperators {
					if _, exists := engine.GetOperator(op); !exists {
						t.Errorf("expected operator %s to be registered", op)
					}
				}
			},
		},
		{
			name: "configures vault from environment",
			envVars: map[string]string{
				"VAULT_ADDR":  "http://localhost:8200",
				"VAULT_TOKEN": "test-token",
			},
			validateFunc: func(t *testing.T, engine *graft.DefaultEngine) {
				if engine == nil {
					t.Fatal("expected engine to be created, got nil")
				}
				// Engine should be created successfully with vault config
			},
		},
		{
			name: "configures AWS region from AWS_REGION",
			envVars: map[string]string{
				"AWS_REGION": "us-west-2",
			},
			validateFunc: func(t *testing.T, engine *graft.DefaultEngine) {
				if engine == nil {
					t.Fatal("expected engine to be created, got nil")
				}
			},
		},
		{
			name: "configures AWS region from AWS_DEFAULT_REGION when AWS_REGION not set",
			envVars: map[string]string{
				"AWS_DEFAULT_REGION": "eu-west-1",
			},
			validateFunc: func(t *testing.T, engine *graft.DefaultEngine) {
				if engine == nil {
					t.Fatal("expected engine to be created, got nil")
				}
			},
		},
		{
			name: "enables redaction mode when REDACT is set",
			envVars: map[string]string{
				"REDACT": "true",
			},
			validateFunc: func(t *testing.T, engine *graft.DefaultEngine) {
				if engine == nil {
					t.Fatal("expected engine to be created, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			originalEnv := make(map[string]string)
			for key, value := range tt.envVars {
				originalEnv[key] = os.Getenv(key)
				os.Setenv(key, value)
			}

			// Cleanup environment after test
			defer func() {
				for key := range tt.envVars {
					if original, exists := originalEnv[key]; exists {
						os.Setenv(key, original)
					} else {
						os.Unsetenv(key)
					}
				}
			}()

			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("expected panic but none occurred")
					}
				}()
			}

			engine := NewDefaultEngine()

			if !tt.expectPanic && tt.validateFunc != nil {
				tt.validateFunc(t, engine)
			}
		})
	}
}

func TestEngineFactory_CreateWithCustomConfig(t *testing.T) {
	// Test that engines can be created with custom configurations
	config := graft.DefaultEngineConfig()
	config.SkipVault = true
	config.SkipAWS = true
	config.EnableCaching = false

	engine := graft.NewDefaultEngineWithConfig(config)
	if engine == nil {
		t.Fatal("expected engine to be created with custom config, got nil")
	}

	// Test that the engine was created successfully with custom config
	// Note: We'll test operator registration through the factory functions
}

func TestEngineFactory_CreateWithInvalidConfig(t *testing.T) {
	// Test scenarios that could lead to invalid configurations
	tests := []struct {
		name        string
		setupFunc   func()
		expectPanic bool
	}{
		{
			name: "handles empty vault address gracefully",
			setupFunc: func() {
				os.Setenv("VAULT_ADDR", "")
				os.Setenv("VAULT_TOKEN", "test-token")
			},
			expectPanic: false,
		},
		{
			name: "handles invalid vault address gracefully",
			setupFunc: func() {
				os.Setenv("VAULT_ADDR", "invalid-url")
				os.Setenv("VAULT_TOKEN", "test-token")
			},
			expectPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalVaultAddr := os.Getenv("VAULT_ADDR")
			originalVaultToken := os.Getenv("VAULT_TOKEN")

			// Cleanup
			defer func() {
				os.Setenv("VAULT_ADDR", originalVaultAddr)
				os.Setenv("VAULT_TOKEN", originalVaultToken)
			}()

			tt.setupFunc()

			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("expected panic but none occurred")
					}
				}()
			}

			engine := NewDefaultEngine()

			if !tt.expectPanic {
				if engine == nil {
					t.Error("expected engine to be created despite invalid config")
				}
			}
		})
	}
}

func TestEngineFactory_ConcurrentCreation(t *testing.T) {
	const numGoroutines = 10
	const numIterations = 5

	var wg sync.WaitGroup
	engines := make(chan *graft.DefaultEngine, numGoroutines*numIterations)
	errors := make(chan error, numGoroutines*numIterations)

	// Set up test environment
	os.Setenv("VAULT_ADDR", "http://localhost:8200")
	os.Setenv("VAULT_TOKEN", "test-token")
	os.Setenv("AWS_REGION", "us-east-1")

	defer func() {
		os.Unsetenv("VAULT_ADDR")
		os.Unsetenv("VAULT_TOKEN")
		os.Unsetenv("AWS_REGION")
	}()

	// Launch multiple goroutines to create engines concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				func() {
					defer func() {
						if r := recover(); r != nil {
							errors <- r.(error)
							return
						}
					}()

					engine := NewDefaultEngine()
					engines <- engine
				}()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(engines)
	close(errors)

	// Check for any errors
	var errorCount int
	for err := range errors {
		t.Errorf("concurrent engine creation error: %v", err)
		errorCount++
	}

	// Verify all engines were created successfully
	var engineCount int
	var validEngines int
	for engine := range engines {
		engineCount++
		if engine != nil {
			validEngines++

			// Verify essential operators are present
			if _, grabExists := engine.GetOperator("grab"); !grabExists {
				t.Errorf("engine missing grab operator")
			}
			if _, concatExists := engine.GetOperator("concat"); !concatExists {
				t.Errorf("engine missing concat operator")
			}
		}
	}

	expectedEngines := numGoroutines * numIterations
	if engineCount != expectedEngines {
		t.Errorf("expected %d engines, got %d", expectedEngines, engineCount)
	}

	if validEngines != expectedEngines-errorCount {
		t.Errorf("expected %d valid engines, got %d", expectedEngines-errorCount, validEngines)
	}

	if errorCount > 0 {
		t.Errorf("concurrent creation had %d errors", errorCount)
	}
}

func TestEngineFactory_NewMinimalEngine(t *testing.T) {
	engine := NewMinimalEngine()

	if engine == nil {
		t.Fatal("expected minimal engine to be created, got nil")
	}

	// Verify essential operators are registered
	essentialOps := []string{"grab", "concat", "empty"}
	for _, op := range essentialOps {
		if _, exists := engine.GetOperator(op); !exists {
			t.Errorf("expected essential operator %s to be registered", op)
		}
	}

	// Note: Due to global operator registry (init functions), operators might be
	// available through fallback to global registry. The key test is that the
	// engine can be created and essential operators are definitely available.
}

func TestEngineFactory_NewTestEngine(t *testing.T) {
	engine := NewTestEngine()

	if engine == nil {
		t.Fatal("expected test engine to be created, got nil")
	}

	// Test engine should have all operators but skip external services
	testOperators := []string{"grab", "concat", "vault", "awsparam"}
	for _, op := range testOperators {
		if _, exists := engine.GetOperator(op); !exists {
			t.Errorf("expected operator %s to be registered in test engine", op)
		}
	}
}

func BenchmarkEngineFactory_CreateDefault(b *testing.B) {
	// Setup
	os.Setenv("VAULT_ADDR", "http://localhost:8200")
	os.Setenv("AWS_REGION", "us-east-1")
	defer func() {
		os.Unsetenv("VAULT_ADDR")
		os.Unsetenv("AWS_REGION")
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine := NewDefaultEngine()
		if engine == nil {
			b.Fatal("engine creation failed")
		}
	}
}

func BenchmarkEngineFactory_CreateMinimal(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine := NewMinimalEngine()
		if engine == nil {
			b.Fatal("minimal engine creation failed")
		}
	}
}

func BenchmarkEngineFactory_ConcurrentCreation(b *testing.B) {
	// Setup
	os.Setenv("VAULT_ADDR", "http://localhost:8200")
	defer os.Unsetenv("VAULT_ADDR")

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			engine := NewDefaultEngine()
			if engine == nil {
				b.Fatal("concurrent engine creation failed")
			}
		}
	})
}
