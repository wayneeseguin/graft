package operators

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/geofffranks/yaml"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/wayneeseguin/graft/internal/utils/ansi"
	"github.com/wayneeseguin/graft/internal/utils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// TTL cache structures
type cacheItem struct {
	value     interface{}
	expiresAt time.Time
}

type ttlCache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
}

// Metrics structures for observability
type operationStats struct {
	count         int64
	totalDuration time.Duration
	errors        int64
	cacheHits     int64
	lastAccess    time.Time
}

type operatorMetrics struct {
	mu           sync.RWMutex
	operations   map[string]*operationStats
	startTime    time.Time
}

// Connection pool for NATS connections
type natsConnectionPool struct {
	mu          sync.RWMutex
	connections map[string]*pooledConnection
	stopCleanup chan struct{}
}

type pooledConnection struct {
	conn      *nats.Conn
	js        jetstream.JetStream
	lastUsed  time.Time
	refCount  int
}

var (
	// Global connection pool
	natsPool = &natsConnectionPool{
		connections: make(map[string]*pooledConnection),
		stopCleanup: make(chan struct{}),
	}
	
	// Connection pool settings
	poolMaxIdleTime = 5 * time.Minute
	poolCleanupInterval = 1 * time.Minute
	
	// TTL-based cache for NATS values
	natsCache = &ttlCache{
		items: make(map[string]*cacheItem),
	}
	
	// Cache settings
	defaultCacheTTL = 5 * time.Minute
	cacheCleanupInterval = 1 * time.Minute
	
	// Metrics and observability
	natsMetrics = &operatorMetrics{
		operations:   make(map[string]*operationStats),
		startTime:    time.Now(),
	}
)

// SkipNats toggles whether NatsOperator will attempt to connect to NATS
// When true will always return "REDACTED"
var SkipNats bool

// NatsTarget represents a NATS target configuration
type NatsTarget struct {
	URL                string        `yaml:"url"`
	Timeout            time.Duration `yaml:"timeout"`
	Retries            int           `yaml:"retries"`
	RetryInterval      time.Duration `yaml:"retry_interval"`
	RetryBackoff       float64       `yaml:"retry_backoff"`
	MaxRetryInterval   time.Duration `yaml:"max_retry_interval"`
	TLS                bool          `yaml:"tls"`
	CertFile           string        `yaml:"cert_file"`
	KeyFile            string        `yaml:"key_file"`
	CAFile             string        `yaml:"ca_file"`
	InsecureSkipVerify bool          `yaml:"insecure_skip_verify"`
	CacheTTL           time.Duration `yaml:"cache_ttl"`
	StreamingThreshold int64         `yaml:"streaming_threshold"`
	AuditLogging       bool          `yaml:"audit_logging"`
}

// NatsClientPool manages NATS connections for different targets
type NatsClientPool struct {
	mu          sync.RWMutex
	connections map[string]*pooledConnection
	configs     map[string]*NatsTarget
}

// Global client pool for target-aware NATS connections
var natsTargetPool = &NatsClientPool{
	connections: make(map[string]*pooledConnection),
	configs:     make(map[string]*NatsTarget),
}

// GetConnection returns a NATS connection for the specified target
func (ncp *NatsClientPool) GetConnection(targetName string) (*pooledConnection, error) {
	ncp.mu.RLock()
	if conn, exists := ncp.connections[targetName]; exists {
		conn.refCount++
		conn.lastUsed = time.Now()
		ncp.mu.RUnlock()
		return conn, nil
	}
	ncp.mu.RUnlock()
	
	// Get target configuration
	config, err := ncp.getTargetConfig(targetName)
	if err != nil {
		return nil, fmt.Errorf("NATS target '%s' not found: %v", targetName, err)
	}
	
	// Create NATS configuration from target config
	natsConfig := &natsConfig{
		URL:                config.URL,
		Timeout:            config.Timeout,
		Retries:            config.Retries,
		RetryInterval:      config.RetryInterval,
		RetryBackoff:       config.RetryBackoff,
		MaxRetryInterval:   config.MaxRetryInterval,
		TLS:                config.TLS,
		CertFile:           config.CertFile,
		KeyFile:            config.KeyFile,
		CAFile:             config.CAFile,
		InsecureSkipVerify: config.InsecureSkipVerify,
		CacheTTL:           config.CacheTTL,
		StreamingThreshold: config.StreamingThreshold,
		AuditLogging:       config.AuditLogging,
	}
	
	// Create new connection
	conn, err := createNatsConnectionFromConfig(natsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS connection for target '%s': %v", targetName, err)
	}
	
	pooledConn := &pooledConnection{
		conn:     conn,
		lastUsed: time.Now(),
		refCount: 1,
	}
	
	// Create JetStream context
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context for target '%s': %v", targetName, err)
	}
	pooledConn.js = js
	
	// Store connection for reuse
	ncp.mu.Lock()
	ncp.connections[targetName] = pooledConn
	ncp.configs[targetName] = config
	ncp.mu.Unlock()
	
	return pooledConn, nil
}

// getTargetConfig retrieves target configuration from environment variables
func (ncp *NatsClientPool) getTargetConfig(targetName string) (*NatsTarget, error) {
	// Check if we have cached config
	ncp.mu.RLock()
	if config, exists := ncp.configs[targetName]; exists {
		ncp.mu.RUnlock()
		return config, nil
	}
	ncp.mu.RUnlock()
	
	// Use environment variables with target suffix
	envPrefix := fmt.Sprintf("NATS_%s_", strings.ToUpper(targetName))
	
	// Check if the URL environment variable is set (required for target configurations)
	urlEnvVar := envPrefix + "URL"
	url := os.Getenv(urlEnvVar)
	if url == "" {
		return nil, fmt.Errorf("NATS target '%s' configuration incomplete (expected %s environment variable)", 
			targetName, urlEnvVar)
	}
	
	config := &NatsTarget{
		URL:                url,
		Timeout:            parseDurationOrDefault(getEnvOrDefault(envPrefix+"TIMEOUT", "5s"), 5*time.Second),
		Retries:            parseIntOrDefault(getEnvOrDefault(envPrefix+"RETRIES", "3"), 3),
		RetryInterval:      parseDurationOrDefault(getEnvOrDefault(envPrefix+"RETRY_INTERVAL", "1s"), 1*time.Second),
		RetryBackoff:       parseFloatOrDefault(getEnvOrDefault(envPrefix+"RETRY_BACKOFF", "2.0"), 2.0),
		MaxRetryInterval:   parseDurationOrDefault(getEnvOrDefault(envPrefix+"MAX_RETRY_INTERVAL", "30s"), 30*time.Second),
		TLS:                parseBoolOrDefault(getEnvOrDefault(envPrefix+"TLS", "false"), false),
		CertFile:           getEnvOrDefault(envPrefix+"CERT_FILE", ""),
		KeyFile:            getEnvOrDefault(envPrefix+"KEY_FILE", ""),
		CAFile:             getEnvOrDefault(envPrefix+"CA_FILE", ""),
		InsecureSkipVerify: parseBoolOrDefault(getEnvOrDefault(envPrefix+"INSECURE_SKIP_VERIFY", "false"), false),
		CacheTTL:           parseDurationOrDefault(getEnvOrDefault(envPrefix+"CACHE_TTL", "5m"), 5*time.Minute),
		StreamingThreshold: parseInt64OrDefault(getEnvOrDefault(envPrefix+"STREAMING_THRESHOLD", "10485760"), 10*1024*1024),
		AuditLogging:       parseBoolOrDefault(getEnvOrDefault(envPrefix+"AUDIT_LOGGING", "false"), false),
	}
	
	return config, nil
}

// ReleaseConnection decreases the reference count for a target connection
func (ncp *NatsClientPool) ReleaseConnection(targetName string) {
	ncp.mu.Lock()
	defer ncp.mu.Unlock()
	
	if conn, exists := ncp.connections[targetName]; exists {
		conn.refCount--
		if conn.refCount <= 0 {
			// Connection no longer in use, but keep it cached for reuse
			conn.refCount = 0
		}
	}
}

// NatsOperator provides the (( nats "store_type:path" )) operator
// It will fetch values from NATS JetStream KV or Object stores
type NatsOperator struct{}

// extractTarget extracts target name from operator call (placeholder)
func (n NatsOperator) extractTarget(ev *graft.Evaluator, args []*graft.Expr) string {
	// TODO: Extract target from parsed expression when parser supports it
	// For now, return empty string to use default configuration
	return ""
}

// getCacheKey generates a cache key that includes target information
func (n NatsOperator) getCacheKey(target, storeType, storePath string) string {
	if target == "" {
		return fmt.Sprintf("%s:%s", storeType, storePath)
	}
	return fmt.Sprintf("%s@%s:%s", target, storeType, storePath)
}

// fetchFromKVWithTarget retrieves a value from a NATS KV store with target-aware caching
func (n NatsOperator) fetchFromKVWithTarget(js jetstream.JetStream, storePath string, config *natsConfig, target string) (interface{}, error) {
	startTime := time.Now()
	operationType := "kv"
	
	// Audit logging
	if config.AuditLogging {
		if target != "" {
			DEBUG("AUDIT: Accessing KV store: %s (target: %s)", storePath, target)
		} else {
			DEBUG("AUDIT: Accessing KV store: %s", storePath)
		}
	}
	
	// Check TTL cache first with target-aware key
	cacheKey := n.getCacheKey(target, "kv", storePath)
	if val, ok := natsCache.get(cacheKey); ok {
		duration := time.Since(startTime)
		natsMetrics.recordOperation(operationType, duration, false, true)
		return val, nil
	}
	
	// Use existing fetchFromKV logic but with target-aware caching
	result, err := fetchFromKV(js, storePath, config)
	if err != nil {
		return nil, err
	}
	
	// Cache the result with target-aware key
	natsCache.set(cacheKey, result, config.CacheTTL)
	
	return result, nil
}

// fetchFromObjectWithTarget retrieves a value from a NATS Object store with target-aware caching  
func (n NatsOperator) fetchFromObjectWithTarget(js jetstream.JetStream, storePath string, config *natsConfig, target string) (interface{}, error) {
	startTime := time.Now()
	operationType := "obj"
	
	// Audit logging
	if config.AuditLogging {
		if target != "" {
			DEBUG("AUDIT: Accessing Object store: %s (target: %s)", storePath, target)
		} else {
			DEBUG("AUDIT: Accessing Object store: %s", storePath)
		}
	}
	
	// Check TTL cache first with target-aware key
	cacheKey := n.getCacheKey(target, "obj", storePath)
	if val, ok := natsCache.get(cacheKey); ok {
		duration := time.Since(startTime)
		natsMetrics.recordOperation(operationType, duration, false, true)
		return val, nil
	}
	
	// Use existing fetchFromObject logic but with target-aware caching
	result, err := fetchFromObject(js, storePath, config)
	if err != nil {
		return nil, err
	}
	
	// Cache the result with target-aware key
	natsCache.set(cacheKey, result, config.CacheTTL)
	
	return result, nil
}

// natsConfig holds connection configuration with enhanced retry and TLS options
type natsConfig struct {
	URL                string
	Timeout            time.Duration
	Retries            int
	RetryInterval      time.Duration
	RetryBackoff       float64
	MaxRetryInterval   time.Duration
	TLS                bool
	CertFile           string
	KeyFile            string
	CAFile             string
	InsecureSkipVerify bool
	CacheTTL           time.Duration
	StreamingThreshold int64 // Size threshold for streaming objects (bytes)
	AuditLogging       bool  // Enable audit logging for access
}

// parseNatsPath extracts store type (kv/obj) and path from the argument
func parseNatsPath(path string) (storeType, storePath string, err error) {
	parts := strings.SplitN(path, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid NATS path format, expected 'kv:store/key' or 'obj:bucket/object'")
	}
	
	storeType = strings.ToLower(parts[0])
	if storeType != "kv" && storeType != "obj" {
		return "", "", fmt.Errorf("invalid store type '%s', must be 'kv' or 'obj'", storeType)
	}
	
	storePath = parts[1]
	if storePath == "" {
		return "", "", fmt.Errorf("empty path after store type")
	}
	
	return storeType, storePath, nil
}

// parseNatsConfig extracts configuration from arguments
func parseNatsConfig(ev *graft.Evaluator, args []*graft.Expr) (*natsConfig, error) {
	// Default URL from environment or fallback to NATS default
	defaultURL := os.Getenv("NATS_URL")
	if defaultURL == "" {
		defaultURL = nats.DefaultURL
	}
	
	// Parse environment variables for default configuration
	defaultTimeout := parseDurationOrDefault(os.Getenv("NATS_TIMEOUT"), 5*time.Second)
	defaultRetries := parseIntOrDefault(os.Getenv("NATS_RETRIES"), 3)
	defaultRetryInterval := parseDurationOrDefault(os.Getenv("NATS_RETRY_INTERVAL"), 1*time.Second)
	defaultRetryBackoff := parseFloatOrDefault(os.Getenv("NATS_RETRY_BACKOFF"), 2.0)
	defaultMaxRetryInterval := parseDurationOrDefault(os.Getenv("NATS_MAX_RETRY_INTERVAL"), 30*time.Second)
	defaultTLS := parseBoolOrDefault(os.Getenv("NATS_TLS"), false)
	defaultCacheTTLEnv := parseDurationOrDefault(os.Getenv("NATS_CACHE_TTL"), defaultCacheTTL)
	defaultStreamingThreshold := parseInt64OrDefault(os.Getenv("NATS_STREAMING_THRESHOLD"), 10*1024*1024)
	defaultAuditLogging := parseBoolOrDefault(os.Getenv("NATS_AUDIT_LOGGING"), false)
	
	config := &natsConfig{
		URL:                defaultURL,
		Timeout:            defaultTimeout,
		Retries:            defaultRetries,
		RetryInterval:      defaultRetryInterval,
		RetryBackoff:       defaultRetryBackoff,
		MaxRetryInterval:   defaultMaxRetryInterval,
		TLS:                defaultTLS,
		CertFile:           os.Getenv("NATS_CERT_FILE"),
		KeyFile:            os.Getenv("NATS_KEY_FILE"),
		CAFile:             os.Getenv("NATS_CA_FILE"),
		InsecureSkipVerify: parseBoolOrDefault(os.Getenv("NATS_INSECURE_SKIP_VERIFY"), false),
		CacheTTL:           defaultCacheTTLEnv,
		StreamingThreshold: defaultStreamingThreshold,
		AuditLogging:       defaultAuditLogging,
	}
	
	// If we have a second argument, it could be URL string or config map
	if len(args) > 1 {
		val, err := ResolveOperatorArgument(ev, args[1])
		if err != nil {
			return nil, err
		}
		
		switch v := val.(type) {
		case string:
			// Simple URL string
			config.URL = v
		case map[interface{}]interface{}:
			// Configuration map
			if url, ok := v["url"]; ok {
				if urlStr, ok := url.(string); ok {
					config.URL = urlStr
				}
			}
			if timeout, ok := v["timeout"]; ok {
				if timeoutStr, ok := timeout.(string); ok {
					if d, err := time.ParseDuration(timeoutStr); err == nil {
						config.Timeout = d
					}
				}
			}
			if retries, ok := v["retries"]; ok {
				switch r := retries.(type) {
				case int:
					config.Retries = r
				case float64:
					config.Retries = int(r)
				}
			}
			if tls, ok := v["tls"]; ok {
				if tlsBool, ok := tls.(bool); ok {
					config.TLS = tlsBool
				}
			}
			if cert, ok := v["cert_file"]; ok {
				if certStr, ok := cert.(string); ok {
					config.CertFile = certStr
				}
			}
			if key, ok := v["key_file"]; ok {
				if keyStr, ok := key.(string); ok {
					config.KeyFile = keyStr
				}
			}
			if ca, ok := v["ca_file"]; ok {
				if caStr, ok := ca.(string); ok {
					config.CAFile = caStr
				}
			}
			if insecure, ok := v["insecure_skip_verify"]; ok {
				if insecureBool, ok := insecure.(bool); ok {
					config.InsecureSkipVerify = insecureBool
				}
			}
			if cacheTTL, ok := v["cache_ttl"]; ok {
				if ttlStr, ok := cacheTTL.(string); ok {
					if d, err := time.ParseDuration(ttlStr); err == nil {
						config.CacheTTL = d
					}
				}
			}
			if streamingThreshold, ok := v["streaming_threshold"]; ok {
				switch st := streamingThreshold.(type) {
				case int:
					config.StreamingThreshold = int64(st)
				case int64:
					config.StreamingThreshold = st
				case float64:
					config.StreamingThreshold = int64(st)
				}
			}
			if auditLogging, ok := v["audit_logging"]; ok {
				if auditBool, ok := auditLogging.(bool); ok {
					config.AuditLogging = auditBool
				}
			}
			if retryInterval, ok := v["retry_interval"]; ok {
				if intervalStr, ok := retryInterval.(string); ok {
					if d, err := time.ParseDuration(intervalStr); err == nil {
						config.RetryInterval = d
					}
				}
			}
			if retryBackoff, ok := v["retry_backoff"]; ok {
				switch b := retryBackoff.(type) {
				case float64:
					config.RetryBackoff = b
				case int:
					config.RetryBackoff = float64(b)
				}
			}
			if maxRetryInterval, ok := v["max_retry_interval"]; ok {
				if intervalStr, ok := maxRetryInterval.(string); ok {
					if d, err := time.ParseDuration(intervalStr); err == nil {
						config.MaxRetryInterval = d
					}
				}
			}
		default:
			return nil, fmt.Errorf("second argument must be URL string or configuration map")
		}
	}
	
	return config, nil
}

// init starts the connection pool cleanup goroutine

// cleanupLoop periodically removes idle connections
func (p *natsConnectionPool) cleanupLoop() {
	ticker := time.NewTicker(poolCleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			p.cleanup()
		case <-p.stopCleanup:
			return
		}
	}
}

// cleanup removes idle connections from the pool
func (p *natsConnectionPool) cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	now := time.Now()
	for key, pc := range p.connections {
		if pc.refCount == 0 && now.Sub(pc.lastUsed) > poolMaxIdleTime {
			pc.conn.Close()
			delete(p.connections, key)
			DEBUG("closed idle NATS connection to %s", key)
		}
	}
}

// getConnection retrieves or creates a pooled connection
func (p *natsConnectionPool) getConnection(config *natsConfig) (*pooledConnection, error) {
	key := config.URL
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Check if we have an existing connection
	if pc, ok := p.connections[key]; ok {
		if pc.conn.IsConnected() {
			pc.refCount++
			pc.lastUsed = time.Now()
			return pc, nil
		}
		// Connection is dead, remove it
		delete(p.connections, key)
	}
	
	// Create new connection with retry logic
	conn, js, err := createNatsConnectionWithRetry(config)
	if err != nil {
		return nil, err
	}
	
	pc := &pooledConnection{
		conn:     conn,
		js:       js,
		lastUsed: time.Now(),
		refCount: 1,
	}
	
	p.connections[key] = pc
	DEBUG("created new NATS connection to %s", key)
	
	return pc, nil
}

// releaseConnection decrements the reference count
func (p *natsConnectionPool) releaseConnection(config *natsConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	key := config.URL
	if pc, ok := p.connections[key]; ok {
		pc.refCount--
		pc.lastUsed = time.Now()
	}
}

// createNatsConnectionWithRetry creates a NATS connection with retry logic
func createNatsConnectionWithRetry(config *natsConfig) (*nats.Conn, jetstream.JetStream, error) {
	opts := buildConnectionOptions(config)
	
	var conn *nats.Conn
	var err error
	
	retryInterval := config.RetryInterval
	for attempt := 0; attempt <= config.Retries; attempt++ {
		if attempt > 0 {
			DEBUG("retrying NATS connection (attempt %d/%d) after %v", attempt, config.Retries, retryInterval)
			time.Sleep(retryInterval)
			
			// Apply backoff
			if config.RetryBackoff > 1 {
				retryInterval = time.Duration(float64(retryInterval) * config.RetryBackoff)
				if config.MaxRetryInterval > 0 && retryInterval > config.MaxRetryInterval {
					retryInterval = config.MaxRetryInterval
				}
			}
		}
		
		conn, err = nats.Connect(config.URL, opts...)
		if err == nil {
			break
		}
		
		DEBUG("failed to connect to NATS: %v", err)
	}
	
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to NATS after %d attempts: %v", config.Retries+1, err)
	}
	
	// Create JetStream context
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("failed to create JetStream context: %v", err)
	}
	
	return conn, js, nil
}

// buildConnectionOptions builds NATS connection options with enhanced TLS support
func buildConnectionOptions(config *natsConfig) []nats.Option {
	opts := []nats.Option{
		nats.Timeout(config.Timeout),
		nats.MaxReconnects(config.Retries),
		nats.ReconnectWait(config.RetryInterval),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				DEBUG("NATS disconnected: %v", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			DEBUG("NATS reconnected to %s", nc.ConnectedUrl())
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			DEBUG("NATS error: %v", err)
		}),
	}
	
	// TLS configuration
	if config.TLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify, // #nosec G402 - controlled by user configuration
		}
		
		if config.CertFile != "" && config.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
			if err == nil {
				tlsConfig.Certificates = []tls.Certificate{cert}
			} else {
				DEBUG("failed to load client certificates: %v", err)
			}
		}
		
		if config.CAFile != "" {
			opts = append(opts, nats.RootCAs(config.CAFile))
		}
		
		opts = append(opts, nats.Secure(tlsConfig))
	}
	
	return opts
}

// fetchFromKV retrieves a value from a NATS KV store with retry logic
func fetchFromKV(js jetstream.JetStream, storePath string, config *natsConfig) (interface{}, error) {
	startTime := time.Now()
	operationType := "kv"
	
	// Audit logging
	if config.AuditLogging {
		DEBUG("AUDIT: Accessing KV store: %s", storePath)
	}
	
	// Check TTL cache first  
	cacheKey := fmt.Sprintf("kv:%s", storePath)
	if val, ok := natsCache.get(cacheKey); ok {
		duration := time.Since(startTime)
		natsMetrics.recordOperation(operationType, duration, false, true)
		return val, nil
	}
	
	// Parse store name and key
	parts := strings.SplitN(storePath, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid KV path format, expected 'store/key'")
	}
	storeName, key := parts[0], parts[1]
	
	var result interface{}
	var err error
	
	retryInterval := config.RetryInterval
	for attempt := 0; attempt <= config.Retries; attempt++ {
		if attempt > 0 {
			DEBUG("retrying KV fetch (attempt %d/%d) after %v", attempt, config.Retries, retryInterval)
			time.Sleep(retryInterval)
			
			// Apply backoff
			if config.RetryBackoff > 1 {
				retryInterval = time.Duration(float64(retryInterval) * config.RetryBackoff)
				if config.MaxRetryInterval > 0 && retryInterval > config.MaxRetryInterval {
					retryInterval = config.MaxRetryInterval
				}
			}
		}
		
		// Get KV store
		kv, err := js.KeyValue(context.Background(), storeName)
		if err != nil {
			continue
		}
		
		// Get the entry
		entry, err := kv.Get(context.Background(), key)
		if err != nil {
			continue
		}
		
		// Determine the value type and process accordingly
		value := entry.Value()
		
		// Handle empty values explicitly
		if len(value) == 0 {
			result = ""
		} else {
			// For KV store, check if it looks like YAML/JSON that should be parsed
			valueStr := string(value)
			
			// Try parsing as YAML if it looks like structured data
			// Be conservative to avoid parsing simple strings with colons as YAML
			trimmed := strings.TrimSpace(valueStr)
			looksLikeYAML := false
			
			if trimmed != "" {
				// For KV store, only parse multi-line YAML content
				// Single-line values (even JSON) are preserved as strings
				// This allows storing JSON strings, URLs, and other text with special characters
				if strings.Contains(trimmed, "\n") {
					// Multi-line content is likely YAML, try to parse it
					looksLikeYAML = true
				}
			}
			
			if looksLikeYAML {
				
				// Try to parse as YAML
				var parsed interface{}
				err = yaml.Unmarshal(value, &parsed)
				if err == nil && parsed != nil {
					// Successfully parsed and got non-string result
					if _, isString := parsed.(string); !isString {
						// Convert to ensure map[interface{}]interface{} for compatibility
						result = convertYAMLTypes(parsed)
					} else {
						// Parsed but still a string, keep original
						result = valueStr
					}
				} else {
					// Failed to parse, keep as string
					result = valueStr
				}
			} else {
				// Simple string value
				result = valueStr
			}
		}
		
		// Cache the result with TTL
		natsCache.set(cacheKey, result, config.CacheTTL)
		
		// Audit logging for successful KV access
		if config.AuditLogging {
			DEBUG("AUDIT: Successfully retrieved KV data from %s", storePath)
		}
		
		duration := time.Since(startTime)
		natsMetrics.recordOperation(operationType, duration, false, false)
		return result, nil
	}
	
	// Audit logging for failed KV access
	if config.AuditLogging {
		DEBUG("AUDIT: Failed to retrieve KV data from %s after %d attempts", storePath, config.Retries+1)
	}
	
	duration := time.Since(startTime)
	natsMetrics.recordOperation(operationType, duration, true, false)
	return nil, fmt.Errorf("failed to get key '%s' from store '%s' after %d attempts: %v", key, storeName, config.Retries+1, err)
}

// fetchFromObject retrieves a value from a NATS Object store with retry logic
func fetchFromObject(js jetstream.JetStream, storePath string, config *natsConfig) (interface{}, error) {
	startTime := time.Now()
	operationType := "obj"
	
	// Audit logging
	if config.AuditLogging {
		DEBUG("AUDIT: Accessing Object store: %s", storePath)
	}
	
	// Check TTL cache first
	cacheKey := fmt.Sprintf("obj:%s", storePath)
	if val, ok := natsCache.get(cacheKey); ok {
		duration := time.Since(startTime)
		natsMetrics.recordOperation(operationType, duration, false, true)
		return val, nil
	}
	
	// Parse bucket name and object name
	parts := strings.SplitN(storePath, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid Object path format, expected 'bucket/object'")
	}
	bucketName, objectName := parts[0], parts[1]
	
	var result interface{}
	var err error
	
	retryInterval := config.RetryInterval
	for attempt := 0; attempt <= config.Retries; attempt++ {
		if attempt > 0 {
			DEBUG("retrying Object fetch (attempt %d/%d) after %v", attempt, config.Retries, retryInterval)
			time.Sleep(retryInterval)
			
			// Apply backoff
			if config.RetryBackoff > 1 {
				retryInterval = time.Duration(float64(retryInterval) * config.RetryBackoff)
				if config.MaxRetryInterval > 0 && retryInterval > config.MaxRetryInterval {
					retryInterval = config.MaxRetryInterval
				}
			}
		}
		
		// Get Object store
		obj, err := js.ObjectStore(context.Background(), bucketName)
		if err != nil {
			continue
		}
		
		// Get the object info first to check content type
		info, err := obj.GetInfo(context.Background(), objectName)
		if err != nil {
			continue
		}
		
		// Get the object data using streaming for large objects
		data, err := streamLargeObject(obj, objectName, config.StreamingThreshold)
		if err != nil {
			DEBUG("streaming error for object %s: %v", objectName, err)
			continue
		}
		
		// Process based on content type from headers
		contentType := ""
		if info.Headers != nil {
			contentType = info.Headers.Get("Content-Type")
		}
		
		switch contentType {
		case "text/yaml", "text/x-yaml", "application/x-yaml", "application/yaml":
			// Parse as YAML
			var yamlResult interface{}
			err = yaml.Unmarshal(data, &yamlResult)
			if err != nil {
				return nil, fmt.Errorf("failed to parse YAML from object '%s': %v", objectName, err)
			}
			// Ensure we return map[interface{}]interface{} for compatibility
			result = convertYAMLTypes(yamlResult)
		case "application/json", "text/json":
			// Parse as JSON (YAML parser handles JSON too)
			err = yaml.Unmarshal(data, &result)
			if err != nil {
				return nil, fmt.Errorf("failed to parse JSON from object '%s': %v", objectName, err)
			}
		case "text/plain", "":
			// Check file extension if no content type
			if contentType == "" && (strings.HasSuffix(objectName, ".yaml") || strings.HasSuffix(objectName, ".yml")) {
				// Parse as YAML for .yaml/.yml files
				var yamlResult interface{}
				err = yaml.Unmarshal(data, &yamlResult)
				if err != nil {
					// If parsing fails, return as string
					result = string(data)
				} else {
					// Ensure we return map[interface{}]interface{} for compatibility
					result = convertYAMLTypes(yamlResult)
				}
			} else {
				// Return as string if text or no content type
				result = string(data)
			}
		default:
			// For any other content type, base64 encode
			result = base64.StdEncoding.EncodeToString(data)
		}
		
		// Cache the result with TTL
		natsCache.set(cacheKey, result, config.CacheTTL)
		
		// Audit logging for successful Object access
		if config.AuditLogging {
			DEBUG("AUDIT: Successfully retrieved Object data from %s (content-type: %s)", storePath, contentType)
		}
		
		duration := time.Since(startTime)
		natsMetrics.recordOperation(operationType, duration, false, false)
		return result, nil
	}
	
	// Audit logging for failed Object access
	if config.AuditLogging {
		DEBUG("AUDIT: Failed to retrieve Object data from %s after %d attempts", storePath, config.Retries+1)
	}
	
	duration := time.Since(startTime)
	natsMetrics.recordOperation(operationType, duration, true, false)
	return nil, fmt.Errorf("failed to get object '%s' from bucket '%s' after %d attempts: %v", objectName, bucketName, config.Retries+1, err)
}

// Setup initializes the NATS operator
func (NatsOperator) Setup() error {
	return nil
}

// Phase returns the phase when this operator runs
func (NatsOperator) Phase() graft.OperatorPhase {
	return graft.EvalPhase
}

// Dependencies returns the dependencies for this operator
func (NatsOperator) Dependencies(ev *graft.Evaluator, args []*graft.Expr, locs []*tree.Cursor, auto []*tree.Cursor) []*tree.Cursor {
	return auto
}

// Run executes the NATS operator
func (n NatsOperator) Run(ev *graft.Evaluator, args []*graft.Expr) (*graft.Response, error) {
	DEBUG("running (( nats ... )) operation at $.%s", ev.Here)
	defer DEBUG("done with (( nats ... )) operation at $%s\n", ev.Here)
	
	if SkipNats {
		return &graft.Response{
			Type:  graft.Replace,
			Value: "REDACTED",
		}, nil
	}
	
	// Validate arguments
	if len(args) < 1 {
		return nil, fmt.Errorf("nats operator requires at least one argument")
	}
	
	// Resolve the path argument
	pathVal, err := ResolveOperatorArgument(ev, args[0])
	if err != nil {
		return nil, err
	}
	
	path, ok := pathVal.(string)
	if !ok {
		return nil, ansi.Errorf("@R{first argument to nats operator must be a string}")
	}
	
	// Parse the path to get store type and path
	storeType, storePath, err := parseNatsPath(path)
	if err != nil {
		return nil, err
	}
	
	// Parse configuration
	config, err := parseNatsConfig(ev, args)
	if err != nil {
		return nil, err
	}
	
	// Extract target information (placeholder for now)
	targetName := n.extractTarget(ev, args)
	
	var pc *pooledConnection
	if targetName != "" {
		// Use target-aware client pool
		pc, err = natsTargetPool.GetConnection(targetName)
		if err != nil {
			return nil, err
		}
		defer natsTargetPool.ReleaseConnection(targetName)
	} else {
		// Use default connection pool
		pc, err = natsPool.getConnection(config)
		if err != nil {
			return nil, err
		}
		defer natsPool.releaseConnection(config)
	}
	
	// Fetch the value based on store type
	var value interface{}
	switch storeType {
	case "kv":
		value, err = n.fetchFromKVWithTarget(pc.js, storePath, config, targetName)
	case "obj":
		value, err = n.fetchFromObjectWithTarget(pc.js, storePath, config, targetName)
	}
	
	if err != nil {
		return nil, err
	}
	
	return &graft.Response{
		Type:  graft.Replace,
		Value: value,
	}, nil
}

// ClearNatsCache clears the NATS cache (useful for testing)
func ClearNatsCache() {
	natsCache.clear()
}

// TTL cache methods
func (c *ttlCache) get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	item, exists := c.items[key]
	if !exists {
		return nil, false
	}
	
	if time.Now().After(item.expiresAt) {
		// Item expired, remove it
		c.mu.RUnlock()
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		c.mu.RLock()
		return nil, false
	}
	
	return item.value, true
}

func (c *ttlCache) set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.items[key] = &cacheItem{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

func (c *ttlCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	for key, item := range c.items {
		if now.After(item.expiresAt) {
			delete(c.items, key)
		}
	}
}

func (c *ttlCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*cacheItem)
}

// Metrics methods
func (m *operatorMetrics) recordOperation(operationType string, duration time.Duration, isError bool, isCacheHit bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	stats, exists := m.operations[operationType]
	if !exists {
		stats = &operationStats{}
		m.operations[operationType] = stats
	}
	
	stats.count++
	stats.totalDuration += duration
	stats.lastAccess = time.Now()
	
	if isError {
		stats.errors++
	}
	if isCacheHit {
		stats.cacheHits++
	}
}

func (m *operatorMetrics) getStats() map[string]operationStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string]operationStats)
	for key, stats := range m.operations {
		result[key] = *stats
	}
	return result
}

// GetNatsMetrics returns current NATS operator metrics
func GetNatsMetrics() map[string]interface{} {
	stats := natsMetrics.getStats()
	result := make(map[string]interface{})
	
	for opType, opStats := range stats {
		avgDuration := float64(0)
		if opStats.count > 0 {
			avgDuration = float64(opStats.totalDuration) / float64(opStats.count) / float64(time.Millisecond)
		}
		
		cacheHitRate := float64(0)
		if opStats.count > 0 {
			cacheHitRate = float64(opStats.cacheHits) / float64(opStats.count) * 100
		}
		
		result[opType] = map[string]interface{}{
			"total_operations":    opStats.count,
			"total_errors":        opStats.errors,
			"cache_hits":          opStats.cacheHits,
			"cache_hit_rate_pct":  cacheHitRate,
			"avg_duration_ms":     avgDuration,
			"last_access":         opStats.lastAccess,
		}
	}
	
	natsMetrics.mu.RLock()
	uptime := time.Since(natsMetrics.startTime)
	natsMetrics.mu.RUnlock()
	
	result["operator_uptime"] = uptime.String()
	result["cache_size"] = len(natsCache.items)
	result["pool_connections"] = len(natsPool.connections)
	
	return result
}

// streamLargeObject handles streaming of large objects to reduce memory usage
func streamLargeObject(obj jetstream.ObjectStore, objectName string, maxSize int64) ([]byte, error) {
	// Get object info first to check size
	info, err := obj.GetInfo(context.Background(), objectName)
	if err != nil {
		return nil, err
	}
	
	// Safely compare with bounds checking
	if maxSize < 0 || (maxSize >= 0 && info.Size <= uint64(maxSize)) {
		// Object is small enough, use normal method
		return obj.GetBytes(context.Background(), objectName)
	}
	
	// Object is large, use streaming approach
	reader, err := obj.Get(context.Background(), objectName)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	
	// Use a buffer to read in chunks
	var result []byte
	buffer := make([]byte, 64*1024) // 64KB chunks
	
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			result = append(result, buffer[:n]...)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		
		// Safety check to prevent excessive memory usage
		if int64(len(result)) > maxSize*2 {
			return nil, fmt.Errorf("object too large for processing: %d bytes", len(result))
		}
	}
	
	return result, nil
}

// Helper functions for environment variable parsing

// parseInt64OrDefault parses int64 string or returns default
func parseInt64OrDefault(value string, defaultValue int64) int64 {
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i
	}
	return defaultValue
}

// parseFloatOrDefault parses float64 string or returns default
func parseFloatOrDefault(value string, defaultValue float64) float64 {
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}
	return defaultValue
}

// createNatsConnectionFromConfig creates a NATS connection from target configuration
func createNatsConnectionFromConfig(config *natsConfig) (*nats.Conn, error) {
	opts := buildConnectionOptions(config)
	
	var conn *nats.Conn
	var err error
	
	retryInterval := config.RetryInterval
	for attempt := 0; attempt <= config.Retries; attempt++ {
		if attempt > 0 {
			DEBUG("retrying NATS connection (attempt %d/%d) after %v", attempt, config.Retries, retryInterval)
			time.Sleep(retryInterval)
			
			// Apply backoff
			if config.RetryBackoff > 1 {
				retryInterval = time.Duration(float64(retryInterval) * config.RetryBackoff)
				if config.MaxRetryInterval > 0 && retryInterval > config.MaxRetryInterval {
					retryInterval = config.MaxRetryInterval
				}
			}
		}
		
		conn, err = nats.Connect(config.URL, opts...)
		if err == nil {
			break
		}
		
		DEBUG("failed to connect to NATS: %v", err)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS after %d attempts: %v", config.Retries+1, err)
	}
	
	return conn, nil
}

var natsCacheStopCleanup = make(chan struct{})

func init() {
	go natsPool.cleanupLoop()
	go func() {
		ticker := time.NewTicker(cacheCleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				natsCache.cleanup()
			case <-natsCacheStopCleanup:
				return
			}
		}
	}()
	RegisterOp("nats", NatsOperator{})
}

// ShutdownNatsOperator gracefully shuts down NATS connections and goroutines
func ShutdownNatsOperator() {
	// Stop cleanup goroutines
	close(natsPool.stopCleanup)
	close(natsCacheStopCleanup)
	
	// Close all pooled connections
	natsPool.mu.Lock()
	for _, pc := range natsPool.connections {
		if pc.conn != nil {
			pc.conn.Close()
		}
	}
	natsPool.connections = make(map[string]*pooledConnection)
	natsPool.mu.Unlock()
	
	// Close target pool connections
	natsTargetPool.mu.Lock()
	for _, pc := range natsTargetPool.connections {
		if pc.conn != nil {
			pc.conn.Close()
		}
	}
	natsTargetPool.connections = make(map[string]*pooledConnection)
	natsTargetPool.mu.Unlock()
	
	// Clear cache
	ClearNatsCache()
}

// convertYAMLTypes ensures YAML data uses map[interface{}]interface{} for consistency
func convertYAMLTypes(input interface{}) interface{} {
	switch v := input.(type) {
	case map[string]interface{}:
		result := make(map[interface{}]interface{})
		for k, val := range v {
			result[k] = convertYAMLTypes(val)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = convertYAMLTypes(val)
		}
		return result
	case map[interface{}]interface{}:
		// Already the right type, but check nested values
		result := make(map[interface{}]interface{})
		for k, val := range v {
			result[k] = convertYAMLTypes(val)
		}
		return result
	default:
		return input
	}
}