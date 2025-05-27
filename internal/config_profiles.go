package internal

import (
	"github.com/wayneeseguin/graft/pkg/graft"
)
import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

//go:embed profiles/*.yaml
var profilesFS embed.FS

// ProfileManager manages performance profiles
type ProfileManager struct {
	profiles map[string]*PerformanceConfig
}

// NewProfileManager creates a new profile manager
func NewProfileManager() (*ProfileManager, error) {
	pm := &ProfileManager{
		profiles: make(map[string]*PerformanceConfig),
	}

	// Load embedded profiles
	if err := pm.loadEmbeddedProfiles(); err != nil {
		return nil, err
	}

	return pm, nil
}

// loadEmbeddedProfiles loads profiles from embedded filesystem
func (pm *ProfileManager) loadEmbeddedProfiles() error {
	return fs.WalkDir(profilesFS, "profiles", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		data, err := profilesFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read profile %s: %v", path, err)
		}

		profileName := strings.TrimSuffix(filepath.Base(path), ".yaml")
		profileName = strings.ReplaceAll(profileName, "_", "-")

		config, err := ConfigFromYAML(string(data))
		if err != nil {
			return fmt.Errorf("failed to parse profile %s: %v", profileName, err)
		}

		// Validate the profile
		validator := NewConfigValidator()
		if err := validator.Validate(config); err != nil {
			return fmt.Errorf("profile %s validation failed: %v", profileName, err)
		}

		pm.profiles[profileName] = config
		return nil
	})
}

// GetProfile returns a profile by name
func (pm *ProfileManager) GetProfile(name string) (*PerformanceConfig, error) {
	profile, ok := pm.profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile '%s' not found", name)
	}
	return profile, nil
}

// ListProfiles returns available profile names
func (pm *ProfileManager) ListProfiles() []string {
	names := make([]string, 0, len(pm.profiles))
	for name := range pm.profiles {
		names = append(names, name)
	}
	return names
}

// GetProfileDescription returns a description of what a profile is optimized for
func (pm *ProfileManager) GetProfileDescription(name string) string {
	descriptions := map[string]string{
		"small-docs":       "Optimized for processing many small YAML documents with low latency",
		"large-docs":       "Optimized for processing large YAML documents with high throughput",
		"high-concurrency": "Optimized for high concurrent request processing",
		"low-memory":       "Optimized for environments with memory constraints",
		"default":          "Balanced configuration suitable for most workloads",
	}

	if desc, ok := descriptions[name]; ok {
		return desc
	}
	return "Unknown profile"
}

// RecommendProfile recommends a profile based on workload characteristics
func (pm *ProfileManager) RecommendProfile(avgDocSize int, concurrentRequests int, availableMemoryMB int) string {
	// Simple heuristic-based recommendation
	if availableMemoryMB < 1024 {
		return "low-memory"
	}

	if concurrentRequests > 50 {
		return "high-concurrency"
	}

	if avgDocSize < 10*1024 { // Less than 10KB
		return "small-docs"
	}

	if avgDocSize > 100*1024 { // Greater than 100KB
		return "large-docs"
	}

	return "default"
}

// ApplyProfile applies a profile to the current configuration
func (pm *ProfileManager) ApplyProfile(name string, target *PerformanceConfig) error {
	profile, err := pm.GetProfile(name)
	if err != nil {
		return err
	}

	// Deep copy the profile to the target
	*target = *profile
	return nil
}

// ProfileComparison compares two profiles and returns differences
type ProfileComparison struct {
	Field    string
	Profile1 interface{}
	Profile2 interface{}
}

// CompareProfiles compares two profiles and returns their differences
func (pm *ProfileManager) CompareProfiles(name1, name2 string) ([]ProfileComparison, error) {
	profile1, err := pm.GetProfile(name1)
	if err != nil {
		return nil, err
	}

	profile2, err := pm.GetProfile(name2)
	if err != nil {
		return nil, err
	}

	var differences []ProfileComparison

	// Compare cache settings
	if profile1.Performance.Cache.ExpressionCacheSize != profile2.Performance.Cache.ExpressionCacheSize {
		differences = append(differences, ProfileComparison{
			Field:    "cache.expression_cache_size",
			Profile1: profile1.Performance.Cache.ExpressionCacheSize,
			Profile2: profile2.Performance.Cache.ExpressionCacheSize,
		})
	}

	if profile1.Performance.Cache.OperatorCacheSize != profile2.Performance.Cache.OperatorCacheSize {
		differences = append(differences, ProfileComparison{
			Field:    "cache.operator_cache_size",
			Profile1: profile1.Performance.Cache.OperatorCacheSize,
			Profile2: profile2.Performance.Cache.OperatorCacheSize,
		})
	}

	// Compare concurrency settings
	if profile1.Performance.Concurrency.MaxWorkers != profile2.Performance.Concurrency.MaxWorkers {
		differences = append(differences, ProfileComparison{
			Field:    "concurrency.max_workers",
			Profile1: profile1.Performance.Concurrency.MaxWorkers,
			Profile2: profile2.Performance.Concurrency.MaxWorkers,
		})
	}

	if profile1.Performance.Concurrency.QueueSize != profile2.Performance.Concurrency.QueueSize {
		differences = append(differences, ProfileComparison{
			Field:    "concurrency.queue_size",
			Profile1: profile1.Performance.Concurrency.QueueSize,
			Profile2: profile2.Performance.Concurrency.QueueSize,
		})
	}

	// Compare memory settings
	if profile1.Performance.Memory.MaxHeapMB != profile2.Performance.Memory.MaxHeapMB {
		differences = append(differences, ProfileComparison{
			Field:    "memory.max_heap_mb",
			Profile1: profile1.Performance.Memory.MaxHeapMB,
			Profile2: profile2.Performance.Memory.MaxHeapMB,
		})
	}

	if profile1.Performance.Memory.GCPercent != profile2.Performance.Memory.GCPercent {
		differences = append(differences, ProfileComparison{
			Field:    "memory.gc_percent",
			Profile1: profile1.Performance.Memory.GCPercent,
			Profile2: profile2.Performance.Memory.GCPercent,
		})
	}

	// Add more comparisons as needed...

	return differences, nil
}

// Global profile manager instance
var globalProfileManager *ProfileManager

// InitializeProfiles initializes the global profile manager
func InitializeProfiles() error {
	var err error
	globalProfileManager, err = NewProfileManager()
	return err
}

// GetProfileManager returns the global profile manager
func GetProfileManager() *ProfileManager {
	if globalProfileManager == nil {
		InitializeProfiles()
	}
	return globalProfileManager
}