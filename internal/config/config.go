// Package config provides a unified configuration system for graft
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete graft configuration
type Config struct {
	// Engine configuration
	Engine EngineConfig `yaml:"engine" json:"engine"`

	// Performance configuration
	Performance PerformanceConfig `yaml:"performance" json:"performance"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging" json:"logging"`

	// Feature flags
	Features map[string]bool `yaml:"features" json:"features"`

	// Metadata
	Version string `yaml:"version" json:"version"`
	Profile string `yaml:"profile" json:"profile"`
}

// EngineConfig contains core engine settings
type EngineConfig struct {
	// Vault configuration
	Vault VaultConfig `yaml:"vault" json:"vault"`

	// AWS configuration
	AWS AWSConfig `yaml:"aws" json:"aws"`

	// Parser configuration
	Parser ParserConfig `yaml:"parser" json:"parser"`

	// Dataflow configuration
	DataflowOrder string `yaml:"dataflow_order" json:"dataflow_order" default:"breadth-first"`

	// Output configuration
	OutputFormat string `yaml:"output_format" json:"output_format" default:"yaml"`
	ColorOutput  bool   `yaml:"color_output" json:"color_output" default:"true"`

	// Security configuration
	StrictMode bool `yaml:"strict_mode" json:"strict_mode" default:"false"`
}

// VaultConfig contains HashiCorp Vault settings
type VaultConfig struct {
	Address    string `yaml:"address" json:"address" env:"VAULT_ADDR"`
	Token      string `yaml:"token" json:"token" env:"VAULT_TOKEN"`
	SkipVerify bool   `yaml:"skip_verify" json:"skip_verify" env:"VAULT_SKIP_VERIFY"`
	Namespace  string `yaml:"namespace" json:"namespace" env:"VAULT_NAMESPACE"`
	Timeout    string `yaml:"timeout" json:"timeout" default:"30s"`
}

// AWSConfig contains AWS settings
type AWSConfig struct {
	Region          string `yaml:"region" json:"region" env:"AWS_REGION"`
	Profile         string `yaml:"profile" json:"profile" env:"AWS_PROFILE"`
	AccessKeyID     string `yaml:"access_key_id" json:"access_key_id" env:"AWS_ACCESS_KEY_ID"`
	SecretAccessKey string `yaml:"secret_access_key" json:"secret_access_key" env:"AWS_SECRET_ACCESS_KEY"`
	SessionToken    string `yaml:"session_token" json:"session_token" env:"AWS_SESSION_TOKEN"`
	Endpoint        string `yaml:"endpoint" json:"endpoint" env:"AWS_ENDPOINT"`
}

// ParserConfig contains parser settings
type ParserConfig struct {
	StrictYAML      bool `yaml:"strict_yaml" json:"strict_yaml" default:"false"`
	PreserveTags    bool `yaml:"preserve_tags" json:"preserve_tags" default:"true"`
	MaxDocumentSize int  `yaml:"max_document_size" json:"max_document_size" default:"10485760"` // 10MB
}

// PerformanceConfig contains performance tuning settings
type PerformanceConfig struct {
	// Basic performance settings
	EnableCaching  bool `yaml:"enable_caching" json:"enable_caching" default:"true"`
	EnableParallel bool `yaml:"enable_parallel" json:"enable_parallel" default:"true"`

	// Cache configuration
	Cache CacheConfig `yaml:"cache" json:"cache"`

	// Concurrency configuration
	Concurrency ConcurrencyConfig `yaml:"concurrency" json:"concurrency"`

	// Memory configuration
	Memory MemoryConfig `yaml:"memory" json:"memory"`

	// I/O configuration
	IO IOConfig `yaml:"io" json:"io"`
}

// CacheConfig contains cache-related settings
type CacheConfig struct {
	ExpressionCacheSize int  `yaml:"expression_cache_size" json:"expression_cache_size" default:"10000"`
	OperatorCacheSize   int  `yaml:"operator_cache_size" json:"operator_cache_size" default:"5000"`
	FileCacheSize       int  `yaml:"file_cache_size" json:"file_cache_size" default:"100"`
	TTL                 time.Duration `yaml:"ttl" json:"ttl" default:"5m"`
	EnableWarmup        bool `yaml:"enable_warmup" json:"enable_warmup" default:"false"`
}

// ConcurrencyConfig contains concurrency settings
type ConcurrencyConfig struct {
	MaxWorkers      int `yaml:"max_workers" json:"max_workers" default:"0"` // 0 = auto
	QueueSize       int `yaml:"queue_size" json:"queue_size" default:"1000"`
	BatchSize       int `yaml:"batch_size" json:"batch_size" default:"10"`
	EnableAdaptive  bool `yaml:"enable_adaptive" json:"enable_adaptive" default:"true"`
}

// MemoryConfig contains memory management settings
type MemoryConfig struct {
	MaxHeapSize     int64 `yaml:"max_heap_size" json:"max_heap_size" default:"0"` // 0 = unlimited
	GCPercent       int   `yaml:"gc_percent" json:"gc_percent" default:"100"`
	EnablePooling   bool  `yaml:"enable_pooling" json:"enable_pooling" default:"true"`
	StringInterning bool  `yaml:"string_interning" json:"string_interning" default:"false"`
}

// IOConfig contains I/O settings
type IOConfig struct {
	ConnectionTimeout   time.Duration `yaml:"connection_timeout" json:"connection_timeout" default:"30s"`
	RequestTimeout      time.Duration `yaml:"request_timeout" json:"request_timeout" default:"60s"`
	MaxRetries          int           `yaml:"max_retries" json:"max_retries" default:"3"`
	EnableDeduplication bool          `yaml:"enable_deduplication" json:"enable_deduplication" default:"true"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level       string `yaml:"level" json:"level" default:"info" env:"GRAFT_LOG_LEVEL"`
	Format      string `yaml:"format" json:"format" default:"text"`
	Output      string `yaml:"output" json:"output" default:"stderr"`
	EnableColor bool   `yaml:"enable_color" json:"enable_color" default:"true"`
}

// Manager manages configuration loading, validation, and hot-reloading
type Manager struct {
	config       *Config
	configPath   string
	mu           sync.RWMutex
	changeHooks  []func(*Config)
	stopWatcher  chan struct{}
	watcherDone  chan struct{}
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{
		config:      DefaultConfig(),
		changeHooks: make([]func(*Config), 0),
		stopWatcher: make(chan struct{}),
		watcherDone: make(chan struct{}),
	}
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Engine: EngineConfig{
			DataflowOrder: "breadth-first",
			OutputFormat:  "yaml",
			ColorOutput:   true,
			StrictMode:    false,
			Parser: ParserConfig{
				StrictYAML:      false,
				PreserveTags:    true,
				MaxDocumentSize: 10 * 1024 * 1024, // 10MB
			},
		},
		Performance: PerformanceConfig{
			EnableCaching:  true,
			EnableParallel: true,
			Cache: CacheConfig{
				ExpressionCacheSize: 10000,
				OperatorCacheSize:   5000,
				FileCacheSize:       100,
				TTL:                 5 * time.Minute,
				EnableWarmup:        false,
			},
			Concurrency: ConcurrencyConfig{
				MaxWorkers:     0, // auto-detect
				QueueSize:      1000,
				BatchSize:      10,
				EnableAdaptive: true,
			},
			Memory: MemoryConfig{
				MaxHeapSize:     0, // unlimited
				GCPercent:       100,
				EnablePooling:   true,
				StringInterning: false,
			},
			IO: IOConfig{
				ConnectionTimeout:   30 * time.Second,
				RequestTimeout:      60 * time.Second,
				MaxRetries:          3,
				EnableDeduplication: true,
			},
		},
		Logging: LoggingConfig{
			Level:       "info",
			Format:      "text",
			Output:      "stderr",
			EnableColor: true,
		},
		Features: make(map[string]bool),
		Version:  "1.0",
		Profile:  "default",
	}
}

// Load loads configuration from a file
func (m *Manager) Load(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Expand path
	expandedPath, err := expandPath(path)
	if err != nil {
		return fmt.Errorf("expanding config path: %w", err)
	}

	// Read file
	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return fmt.Errorf("reading config file: %w", err)
	}

	// Parse configuration
	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("parsing config file: %w", err)
	}

	// Apply environment overrides
	if err := applyEnvOverrides(config); err != nil {
		return fmt.Errorf("applying environment overrides: %w", err)
	}

	// Validate configuration
	if err := Validate(config); err != nil {
		return fmt.Errorf("validating configuration: %w", err)
	}

	// Update configuration
	m.config = config
	m.configPath = expandedPath

	// Notify change hooks
	m.notifyChangeHooks(config)

	return nil
}

// LoadProfile loads a named configuration profile
func (m *Manager) LoadProfile(profileName string) error {
	profilePath := filepath.Join(getProfilesDir(), profileName+".yaml")
	if err := m.Load(profilePath); err != nil {
		return fmt.Errorf("loading profile %s: %w", profileName, err)
	}
	
	m.mu.Lock()
	m.config.Profile = profileName
	m.mu.Unlock()
	
	return nil
}

// Get returns the current configuration
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy to prevent mutations
	configCopy := *m.config
	return &configCopy
}

// Update updates the configuration and notifies hooks
func (m *Manager) Update(updateFunc func(*Config)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a copy for update
	configCopy := *m.config
	updateFunc(&configCopy)

	// Validate the updated configuration
	if err := Validate(&configCopy); err != nil {
		return fmt.Errorf("validating updated configuration: %w", err)
	}

	// Apply the update
	m.config = &configCopy

	// Notify change hooks
	m.notifyChangeHooks(&configCopy)

	return nil
}

// OnChange registers a callback for configuration changes
func (m *Manager) OnChange(hook func(*Config)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.changeHooks = append(m.changeHooks, hook)
}

// Watch starts watching the configuration file for changes
func (m *Manager) Watch() error {
	m.mu.RLock()
	configPath := m.configPath
	m.mu.RUnlock()

	if configPath == "" {
		return fmt.Errorf("no configuration file loaded")
	}

	// Implementation would use fsnotify or similar
	// This is a placeholder
	go func() {
		// Watch logic here
		close(m.watcherDone)
	}()

	return nil
}

// StopWatch stops watching the configuration file
func (m *Manager) StopWatch() {
	close(m.stopWatcher)
	<-m.watcherDone
}

// notifyChangeHooks calls all registered change hooks
func (m *Manager) notifyChangeHooks(config *Config) {
	for _, hook := range m.changeHooks {
		// Call hooks in goroutines to prevent blocking
		go hook(config)
	}
}

// expandPath expands ~ and environment variables in paths
func expandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Expand ~ to home directory
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}

	// Expand environment variables
	path = os.ExpandEnv(path)

	return path, nil
}

// getProfilesDir returns the directory containing configuration profiles
func getProfilesDir() string {
	// First check if we're in development (internal/profiles exists)
	if _, err := os.Stat("internal/profiles"); err == nil {
		return "internal/profiles"
	}

	// Otherwise use system location
	return "/etc/graft/profiles"
}

// applyEnvOverrides applies environment variable overrides to the configuration
func applyEnvOverrides(config *Config) error {
	// This would use reflection to find struct tags with env:"VAR_NAME"
	// and override values from environment
	// Placeholder for now
	return nil
}