package config

import (
	"runtime"

	"github.com/wayneeseguin/graft/pkg/graft"
)

// ToEngineConfig converts the unified Config to graft.EngineConfig
func (c *Config) ToEngineConfig() graft.EngineConfig {
	engineCfg := graft.EngineConfig{
		// Vault configuration
		VaultAddr:     c.Engine.Vault.Address,
		VaultToken:    c.Engine.Vault.Token,
		VaultSkipTLS:  c.Engine.Vault.SkipVerify,
		SkipVault:     c.Engine.Vault.Address == "",
		
		// AWS configuration
		AWSRegion:  c.Engine.AWS.Region,
		AWSProfile: c.Engine.AWS.Profile,
		SkipAWS:    c.Engine.AWS.Region == "",
		
		// Performance configuration
		EnableCaching:  c.Performance.EnableCaching,
		CacheSize:      c.Performance.Cache.ExpressionCacheSize,
		EnableParallel: c.Performance.EnableParallel,
		MaxWorkers:     c.Performance.Concurrency.MaxWorkers,
		
		// Dataflow configuration
		DataflowOrder: c.Engine.DataflowOrder,
		
		// Parser configuration
		UseEnhancedParser: !c.Engine.Parser.StrictYAML,
	}
	
	return engineCfg
}

// ToEngineOptions converts the unified Config to engine creation parameters
func (c *Config) ToEngineOptions() map[string]interface{} {
	options := make(map[string]interface{})
	
	// Vault options
	if c.Engine.Vault.Address != "" {
		options["vault_addr"] = c.Engine.Vault.Address
		options["vault_token"] = c.Engine.Vault.Token
		options["vault_skip_tls"] = c.Engine.Vault.SkipVerify
	}
	
	// AWS options
	if c.Engine.AWS.Region != "" {
		options["aws_region"] = c.Engine.AWS.Region
	}
	if c.Engine.AWS.Profile != "" {
		options["aws_profile"] = c.Engine.AWS.Profile
	}
	
	// Performance options
	options["enable_caching"] = c.Performance.EnableCaching
	options["cache_size"] = c.Performance.Cache.ExpressionCacheSize
	options["enable_parallel"] = c.Performance.EnableParallel
	options["max_workers"] = c.Performance.Concurrency.MaxWorkers
	
	// Dataflow options
	options["dataflow_order"] = c.Engine.DataflowOrder
	
	return options
}

// FromEngineConfig creates a Config from graft.EngineConfig
func FromEngineConfig(engineCfg *graft.EngineConfig) *Config {
	cfg := DefaultConfig()
	
	// Copy Vault configuration
	cfg.Engine.Vault.Address = engineCfg.VaultAddr
	cfg.Engine.Vault.Token = engineCfg.VaultToken
	cfg.Engine.Vault.SkipVerify = engineCfg.VaultSkipTLS
	
	// Copy AWS configuration
	cfg.Engine.AWS.Region = engineCfg.AWSRegion
	cfg.Engine.AWS.Profile = engineCfg.AWSProfile
	
	// Copy performance configuration
	cfg.Performance.EnableCaching = engineCfg.EnableCaching
	cfg.Performance.Cache.ExpressionCacheSize = engineCfg.CacheSize
	cfg.Performance.EnableParallel = engineCfg.EnableParallel
	cfg.Performance.Concurrency.MaxWorkers = engineCfg.MaxWorkers
	
	// Copy dataflow configuration
	cfg.Engine.DataflowOrder = engineCfg.DataflowOrder
	
	// Copy parser configuration
	cfg.Engine.Parser.StrictYAML = !engineCfg.UseEnhancedParser
	
	return cfg
}

// ApplyToEngine applies configuration changes to a running engine
func (c *Config) ApplyToEngine(engine *graft.Engine) error {
	// This would require extending the graft.Engine API to support
	// runtime configuration updates. For now, this is a placeholder.
	
	// Future implementation would:
	// 1. Update cache sizes
	// 2. Adjust worker pools
	// 3. Update timeouts
	// 4. Apply feature flags
	
	return nil
}

// GetFeature returns whether a feature is enabled
func (c *Config) GetFeature(name string) bool {
	if c.Features == nil {
		return false
	}
	return c.Features[name]
}

// SetFeature sets a feature flag
func (c *Config) SetFeature(name string, enabled bool) {
	if c.Features == nil {
		c.Features = make(map[string]bool)
	}
	c.Features[name] = enabled
}

// IsHighPerformanceMode returns true if high performance settings are enabled
func (c *Config) IsHighPerformanceMode() bool {
	return c.Performance.EnableCaching &&
		c.Performance.EnableParallel &&
		c.Performance.Memory.EnablePooling &&
		c.Performance.IO.EnableDeduplication &&
		c.Performance.Memory.MaxHeapSize == 0 // No heap limit
}

// IsLowMemoryMode returns true if low memory settings are enabled
func (c *Config) IsLowMemoryMode() bool {
	return c.Performance.Memory.MaxHeapSize > 0 &&
		c.Performance.Memory.MaxHeapSize < 512*1024*1024 && // Less than 512MB
		!c.Performance.Memory.StringInterning &&
		c.Performance.Cache.ExpressionCacheSize < 1000
}

// GetEffectiveWorkers returns the effective number of workers
func (c *Config) GetEffectiveWorkers() int {
	if !c.Performance.EnableParallel {
		return 1
	}
	if c.Performance.Concurrency.MaxWorkers <= 0 {
		return runtime.NumCPU()
	}
	return c.Performance.Concurrency.MaxWorkers
}