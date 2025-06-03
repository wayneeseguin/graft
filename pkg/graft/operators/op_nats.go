package operators

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/geofffranks/yaml"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/goutils/tree"
	"github.com/wayneeseguin/graft/pkg/graft"
)

// natsConnection holds a shared NATS connection
var natsConnection *nats.Conn

// natsJetStream holds a shared JetStream context
var natsJetStream jetstream.JetStream

// natsKVCache caches values from NATS KV stores
var natsKVCache = make(map[string]interface{})

// natsObjCache caches values from NATS Object stores
var natsObjCache = make(map[string]interface{})

// SkipNats toggles whether NatsOperator will attempt to connect to NATS
// When true will always return "REDACTED"
var SkipNats bool

// NatsOperator provides the (( nats "store_type:path" )) operator
// It will fetch values from NATS JetStream KV or Object stores
type NatsOperator struct{}

// natsConfig holds connection configuration
type natsConfig struct {
	URL      string
	Timeout  time.Duration
	Retries  int
	TLS      bool
	CertFile string
	KeyFile  string
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
	config := &natsConfig{
		URL:     nats.DefaultURL,
		Timeout: 5 * time.Second,
		Retries: 3,
		TLS:     false,
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
		default:
			return nil, fmt.Errorf("second argument must be URL string or configuration map")
		}
	}
	
	return config, nil
}

// connectToNats establishes or reuses a NATS connection
func connectToNats(config *natsConfig) error {
	if natsConnection != nil && natsConnection.IsConnected() {
		return nil
	}
	
	opts := []nats.Option{
		nats.Timeout(config.Timeout),
		nats.MaxReconnects(config.Retries),
	}
	
	if config.TLS {
		// TODO: Add TLS configuration in Phase 3
		opts = append(opts, nats.Secure())
	}
	
	var err error
	natsConnection, err = nats.Connect(config.URL, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %v", err)
	}
	
	// Create JetStream context
	natsJetStream, err = jetstream.New(natsConnection)
	if err != nil {
		natsConnection.Close()
		natsConnection = nil
		return fmt.Errorf("failed to create JetStream context: %v", err)
	}
	
	return nil
}

// fetchFromKV retrieves a value from a NATS KV store
func fetchFromKV(storePath string) (interface{}, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("kv:%s", storePath)
	if val, ok := natsKVCache[cacheKey]; ok {
		return val, nil
	}
	
	// Parse store name and key
	parts := strings.SplitN(storePath, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid KV path format, expected 'store/key'")
	}
	storeName, key := parts[0], parts[1]
	
	// Get KV store
	kv, err := natsJetStream.KeyValue(context.Background(), storeName)
	if err != nil {
		return nil, fmt.Errorf("failed to access KV store '%s': %v", storeName, err)
	}
	
	// Get the entry
	entry, err := kv.Get(context.Background(), key)
	if err != nil {
		return nil, fmt.Errorf("failed to get key '%s' from store '%s': %v", key, storeName, err)
	}
	
	// Determine the value type and process accordingly
	value := entry.Value()
	
	// Try to parse as YAML first (could be embedded YAML in KV)
	var result interface{}
	err = yaml.Unmarshal(value, &result)
	if err != nil {
		// If YAML parsing fails, treat as string
		result = string(value)
	}
	
	// Cache the result
	natsKVCache[cacheKey] = result
	
	return result, nil
}

// fetchFromObject retrieves a value from a NATS Object store
func fetchFromObject(storePath string) (interface{}, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("obj:%s", storePath)
	if val, ok := natsObjCache[cacheKey]; ok {
		return val, nil
	}
	
	// Parse bucket name and object name
	parts := strings.SplitN(storePath, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid Object path format, expected 'bucket/object'")
	}
	bucketName, objectName := parts[0], parts[1]
	
	// Get Object store
	obj, err := natsJetStream.ObjectStore(context.Background(), bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to access Object store '%s': %v", bucketName, err)
	}
	
	// Get the object info first to check content type
	info, err := obj.GetInfo(context.Background(), objectName)
	if err != nil {
		return nil, fmt.Errorf("failed to get object info for '%s' from bucket '%s': %v", objectName, bucketName, err)
	}
	
	// Get the object data
	data, err := obj.GetBytes(context.Background(), objectName)
	if err != nil {
		return nil, fmt.Errorf("failed to get object '%s' from bucket '%s': %v", objectName, bucketName, err)
	}
	
	// Process based on content type from headers
	var result interface{}
	contentType := ""
	if info.Headers != nil {
		contentType = info.Headers.Get("Content-Type")
	}
	
	switch contentType {
	case "text/yaml", "text/x-yaml", "application/x-yaml", "application/yaml":
		// Parse as YAML
		err = yaml.Unmarshal(data, &result)
		if err != nil {
			return nil, fmt.Errorf("failed to parse YAML from object '%s': %v", objectName, err)
		}
	case "application/json", "text/json":
		// Parse as JSON (YAML parser handles JSON too)
		err = yaml.Unmarshal(data, &result)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON from object '%s': %v", objectName, err)
		}
	case "text/plain", "":
		// Return as string if text or no content type
		result = string(data)
	default:
		// For any other content type, base64 encode
		result = base64.StdEncoding.EncodeToString(data)
	}
	
	// Cache the result
	natsObjCache[cacheKey] = result
	
	return result, nil
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
	
	// Connect to NATS
	err = connectToNats(config)
	if err != nil {
		return nil, err
	}
	
	// Fetch the value based on store type
	var value interface{}
	switch storeType {
	case "kv":
		value, err = fetchFromKV(storePath)
	case "obj":
		value, err = fetchFromObject(storePath)
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
	natsKVCache = make(map[string]interface{})
	natsObjCache = make(map[string]interface{})
}

func init() {
	RegisterOp("nats", NatsOperator{})
}