package config

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed profiles/*.yaml
var profilesFS embed.FS

// ProfileManager manages configuration profiles
type ProfileManager struct {
	manager *Manager
}

// NewProfileManager creates a new profile manager
func NewProfileManager(manager *Manager) *ProfileManager {
	return &ProfileManager{
		manager: manager,
	}
}

// ListProfiles returns all available profile names
func (pm *ProfileManager) ListProfiles() ([]string, error) {
	entries, err := profilesFS.ReadDir("profiles")
	if err != nil {
		return nil, fmt.Errorf("reading profiles directory: %w", err)
	}

	var profiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			profileName := strings.TrimSuffix(entry.Name(), ".yaml")
			profiles = append(profiles, profileName)
		}
	}

	return profiles, nil
}

// LoadProfile loads a profile by name
func (pm *ProfileManager) LoadProfile(profileName string) (*Config, error) {
	profilePath := filepath.Join("profiles", profileName+".yaml")

	data, err := profilesFS.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("reading profile %s: %w", profileName, err)
	}

	// Start with default config
	config := DefaultConfig()

	// Parse the profile configuration
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("parsing profile %s: %w", profileName, err)
	}

	// Set the profile name
	config.Profile = profileName

	// Validate the configuration
	if err := Validate(config); err != nil {
		return nil, fmt.Errorf("validating profile %s: %w", profileName, err)
	}

	return config, nil
}

// ApplyProfile applies a named profile to the current configuration
func (pm *ProfileManager) ApplyProfile(profileName string) error {
	profile, err := pm.LoadProfile(profileName)
	if err != nil {
		return err
	}

	// Merge with current configuration
	current := pm.manager.Get()
	merged := MergeConfigs(current, profile)

	// Update configuration
	return pm.manager.Update(func(cfg *Config) {
		*cfg = *merged
	})
}

// CompareProfiles compares two profiles and returns differences
func (pm *ProfileManager) CompareProfiles(profile1, profile2 string) (map[string]interface{}, error) {
	cfg1, err := pm.LoadProfile(profile1)
	if err != nil {
		return nil, fmt.Errorf("loading profile %s: %w", profile1, err)
	}

	cfg2, err := pm.LoadProfile(profile2)
	if err != nil {
		return nil, fmt.Errorf("loading profile %s: %w", profile2, err)
	}

	// Compare configurations
	differences := make(map[string]interface{})

	// Compare performance settings
	if cfg1.Performance.Cache.ExpressionCacheSize != cfg2.Performance.Cache.ExpressionCacheSize {
		differences["performance.cache.expression_cache_size"] = map[string]int{
			profile1: cfg1.Performance.Cache.ExpressionCacheSize,
			profile2: cfg2.Performance.Cache.ExpressionCacheSize,
		}
	}

	if cfg1.Performance.Concurrency.MaxWorkers != cfg2.Performance.Concurrency.MaxWorkers {
		differences["performance.concurrency.max_workers"] = map[string]int{
			profile1: cfg1.Performance.Concurrency.MaxWorkers,
			profile2: cfg2.Performance.Concurrency.MaxWorkers,
		}
	}

	if cfg1.Performance.Memory.MaxHeapSize != cfg2.Performance.Memory.MaxHeapSize {
		differences["performance.memory.max_heap_size"] = map[string]int64{
			profile1: cfg1.Performance.Memory.MaxHeapSize,
			profile2: cfg2.Performance.Memory.MaxHeapSize,
		}
	}

	// Add more comparisons as needed

	return differences, nil
}

// RecommendProfile recommends a profile based on input characteristics
func (pm *ProfileManager) RecommendProfile(characteristics ProfileCharacteristics) (string, error) {
	profiles, err := pm.ListProfiles()
	if err != nil {
		return "", err
	}

	// Score each profile
	bestProfile := "default"
	bestScore := 0

	for _, profile := range profiles {
		score := pm.scoreProfile(profile, characteristics)
		if score > bestScore {
			bestScore = score
			bestProfile = profile
		}
	}

	return bestProfile, nil
}

// ProfileCharacteristics describes the expected workload characteristics
type ProfileCharacteristics struct {
	DocumentSize     DocumentSize     `yaml:"document_size"`
	DocumentCount    DocumentCount    `yaml:"document_count"`
	ConcurrencyLevel ConcurrencyLevel `yaml:"concurrency_level"`
	MemoryBudget     MemoryBudget     `yaml:"memory_budget"`
	LatencyPriority  LatencyPriority  `yaml:"latency_priority"`
}

type DocumentSize string

const (
	DocumentSizeSmall  DocumentSize = "small"  // < 1KB
	DocumentSizeMedium DocumentSize = "medium" // 1KB - 100KB
	DocumentSizeLarge  DocumentSize = "large"  // > 100KB
)

type DocumentCount string

const (
	DocumentCountFew  DocumentCount = "few"  // < 10
	DocumentCountMany DocumentCount = "many" // 10 - 100
	DocumentCountMass DocumentCount = "mass" // > 100
)

type ConcurrencyLevel string

const (
	ConcurrencyLevelLow    ConcurrencyLevel = "low"    // Single threaded
	ConcurrencyLevelMedium ConcurrencyLevel = "medium" // CPU count
	ConcurrencyLevelHigh   ConcurrencyLevel = "high"   // > CPU count
)

type MemoryBudget string

const (
	MemoryBudgetLow    MemoryBudget = "low"    // < 512MB
	MemoryBudgetMedium MemoryBudget = "medium" // 512MB - 2GB
	MemoryBudgetHigh   MemoryBudget = "high"   // > 2GB
)

type LatencyPriority string

const (
	LatencyPriorityLow    LatencyPriority = "low"    // Throughput over latency
	LatencyPriorityMedium LatencyPriority = "medium" // Balanced
	LatencyPriorityHigh   LatencyPriority = "high"   // Latency over throughput
)

// scoreProfile scores how well a profile matches the characteristics
func (pm *ProfileManager) scoreProfile(profileName string, characteristics ProfileCharacteristics) int {
	score := 0

	switch profileName {
	case "small_docs":
		if characteristics.DocumentSize == DocumentSizeSmall {
			score += 3
		}
		if characteristics.DocumentCount == DocumentCountMany || characteristics.DocumentCount == DocumentCountMass {
			score += 2
		}

	case "large_docs":
		if characteristics.DocumentSize == DocumentSizeLarge {
			score += 3
		}
		if characteristics.MemoryBudget == MemoryBudgetHigh {
			score += 2
		}

	case "high_concurrency":
		if characteristics.ConcurrencyLevel == ConcurrencyLevelHigh {
			score += 3
		}
		if characteristics.DocumentCount == DocumentCountMass {
			score += 2
		}

	case "low_memory":
		if characteristics.MemoryBudget == MemoryBudgetLow {
			score += 3
		}
		if characteristics.DocumentSize == DocumentSizeSmall {
			score += 1
		}

	case "default":
		// Default gets a base score
		score = 1
	}

	return score
}

// GetCurrentProfile returns the name of the currently active profile
func (pm *ProfileManager) GetCurrentProfile() string {
	return pm.manager.Get().Profile
}

// CreateCustomProfile creates a custom profile based on current configuration
func (pm *ProfileManager) CreateCustomProfile(name string) (*Config, error) {
	current := pm.manager.Get()

	// Create a new configuration based on current
	custom := *current
	custom.Profile = name
	custom.Version = "custom"

	return &custom, nil
}

// Default profiles
func GetDefaultProfiles() map[string]*Config {
	return map[string]*Config{
		"default": {
			Version: "1.0",
			Profile: "default",
			Engine: EngineConfig{
				DataflowOrder: "breadth-first",
				OutputFormat:  "yaml",
				ColorOutput:   true,
				Parser: ParserConfig{
					StrictYAML:      false,
					PreserveTags:    true,
					MaxDocumentSize: 10 * 1024 * 1024,
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
				},
				Concurrency: ConcurrencyConfig{
					MaxWorkers:     0, // auto
					QueueSize:      1000,
					BatchSize:      10,
					EnableAdaptive: true,
				},
				Memory: MemoryConfig{
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
		},

		"small_docs": {
			Version: "1.0",
			Profile: "small_docs",
			Engine: EngineConfig{
				DataflowOrder: "breadth-first",
				OutputFormat:  "yaml",
				ColorOutput:   true,
				Parser: ParserConfig{
					StrictYAML:      false,
					PreserveTags:    true,
					MaxDocumentSize: 1024 * 1024, // 1MB
				},
			},
			Performance: PerformanceConfig{
				EnableCaching:  true,
				EnableParallel: true,
				Cache: CacheConfig{
					ExpressionCacheSize: 50000, // Larger cache for many small docs
					OperatorCacheSize:   25000,
					FileCacheSize:       1000,
					TTL:                 10 * time.Minute,
					EnableWarmup:        true,
				},
				Concurrency: ConcurrencyConfig{
					MaxWorkers:     0,    // auto
					QueueSize:      5000, // Larger queue
					BatchSize:      50,   // Larger batches
					EnableAdaptive: true,
				},
				Memory: MemoryConfig{
					GCPercent:       200, // Less frequent GC
					EnablePooling:   true,
					StringInterning: true, // Enable for small repeated strings
				},
			},
		},

		"large_docs": {
			Version: "1.0",
			Profile: "large_docs",
			Performance: PerformanceConfig{
				EnableCaching:  true,
				EnableParallel: false, // Single threaded for large docs
				Cache: CacheConfig{
					ExpressionCacheSize: 1000, // Smaller cache
					OperatorCacheSize:   500,
					FileCacheSize:       10,
					TTL:                 2 * time.Minute,
				},
				Concurrency: ConcurrencyConfig{
					MaxWorkers:     1,   // Single worker
					QueueSize:      100, // Smaller queue
					BatchSize:      1,   // Process one at a time
					EnableAdaptive: false,
				},
				Memory: MemoryConfig{
					GCPercent:       50, // More frequent GC
					EnablePooling:   true,
					StringInterning: false,
				},
			},
		},

		"high_concurrency": {
			Version: "1.0",
			Profile: "high_concurrency",
			Performance: PerformanceConfig{
				EnableCaching:  true,
				EnableParallel: true,
				Cache: CacheConfig{
					ExpressionCacheSize: 100000, // Very large cache
					OperatorCacheSize:   50000,
					FileCacheSize:       5000,
					TTL:                 15 * time.Minute,
					EnableWarmup:        true,
				},
				Concurrency: ConcurrencyConfig{
					MaxWorkers:     0,     // Use all CPUs
					QueueSize:      10000, // Large queue
					BatchSize:      100,   // Large batches
					EnableAdaptive: true,
				},
				Memory: MemoryConfig{
					GCPercent:       300, // Much less frequent GC
					EnablePooling:   true,
					StringInterning: true,
				},
			},
		},

		"low_memory": {
			Version: "1.0",
			Profile: "low_memory",
			Performance: PerformanceConfig{
				EnableCaching:  false, // Disable caching
				EnableParallel: false, // Single threaded
				Cache: CacheConfig{
					ExpressionCacheSize: 100,
					OperatorCacheSize:   50,
					FileCacheSize:       5,
					TTL:                 30 * time.Second,
				},
				Concurrency: ConcurrencyConfig{
					MaxWorkers:     1,
					QueueSize:      10,
					BatchSize:      1,
					EnableAdaptive: false,
				},
				Memory: MemoryConfig{
					MaxHeapSize:     256 * 1024 * 1024, // 256MB limit
					GCPercent:       25,                // Very frequent GC
					EnablePooling:   false,             // Disable pooling
					StringInterning: false,
				},
			},
		},
	}
}
