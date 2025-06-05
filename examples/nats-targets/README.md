# NATS Operator with Targets

This example demonstrates how to use the NATS operator with different targets to connect to multiple NATS instances.

## Overview

The NATS operator now supports target syntax to specify which NATS instance to connect to:

```yaml
# Connect to production NATS
production_config: (( nats@production "kv:config/app" ))

# Connect to staging NATS  
staging_config: (( nats@staging "kv:config/app" ))

# Use default NATS configuration
default_config: (( nats "kv:config/app" ))
```

## Configuration

### Environment Variables

For each target, you need to set environment variables:

```bash
# Production NATS
export NATS_PRODUCTION_URL="nats://nats.prod.example.com:4222"
export NATS_PRODUCTION_TIMEOUT="10s"
export NATS_PRODUCTION_RETRIES="5"
export NATS_PRODUCTION_TLS="true"
export NATS_PRODUCTION_CERT_FILE="/etc/ssl/certs/nats-prod.crt"
export NATS_PRODUCTION_KEY_FILE="/etc/ssl/private/nats-prod.key"
export NATS_PRODUCTION_CA_FILE="/etc/ssl/certs/nats-ca.crt"
export NATS_PRODUCTION_CACHE_TTL="15m"
export NATS_PRODUCTION_AUDIT_LOGGING="true"

# Staging NATS
export NATS_STAGING_URL="nats://nats.staging.example.com:4222"
export NATS_STAGING_TIMEOUT="5s"
export NATS_STAGING_RETRIES="3"
export NATS_STAGING_TLS="false"
export NATS_STAGING_CACHE_TTL="5m"

# Default NATS (existing behavior)
export NATS_URL="nats://nats.default.example.com:4222"
```

### Configuration File (Future)

In future versions, targets will be configurable via a configuration file:

```yaml
# ~/.graft/config.yml
targets:
  nats:
    production:
      url: "${NATS_PROD_URL}"
      timeout: "10s"
      retries: 5
      tls: true
      cert_file: "/etc/ssl/certs/nats-prod.crt"
      key_file: "/etc/ssl/private/nats-prod.key"
      ca_file: "/etc/ssl/certs/nats-ca.crt"
      cache_ttl: "15m"
      audit_logging: true
    staging:
      url: "${NATS_STAGING_URL}"
      timeout: "5s"
      retries: 3
      tls: false
      cache_ttl: "5m"
```

## Usage Examples

### Basic Target Usage

```yaml
# main.yml
app_name: (( nats@production "kv:config/name" ))
database:
  config: (( nats@production "kv:db/config" ))
  staging_config: (( nats@staging "kv:db/config" ))
```

### Object Store Access

```yaml
# Fetch binary data from object store
certificate: (( nats@production "obj:certificates/server.crt" ))
large_config: (( nats@production "obj:configs/app.yml" ))
```

### With Defaults

```yaml
# Use target with fallback
api_key: (( nats@production "kv:secrets/api_key" || "default-key" ))

# Multiple fallbacks across targets
secret: (( nats@production "kv:secrets/token" || nats@staging "kv:secrets/token" || "fallback" ))
```

### Complex Expressions

```yaml
# Concatenation with target
connection_string: (( concat "nats://" (nats@production "kv:nats/user") ":" (nats@production "kv:nats/password") "@" (nats@production "kv:nats/host") ":4222" ))
```

## Features

### Store Types

- **KV Store**: `kv:store_name/key`
  - Retrieves values from NATS JetStream KV stores
  - Supports YAML parsing for structured data
  - Falls back to string for non-YAML content

- **Object Store**: `obj:bucket_name/object_name`
  - Retrieves objects from NATS JetStream Object stores
  - Content-type aware processing:
    - YAML/JSON: Parsed as structured data
    - Text: Returned as string
    - Binary: Base64 encoded

### Caching

Each target maintains its own cache to avoid conflicts:
- `kv:config/app` (default NATS) and `production@kv:config/app` are cached separately
- Cache keys include target information: `{target}@{store_type}:{path}`

### Advanced Features

- **Retry Logic**: Configurable retry attempts with exponential backoff
- **TLS Support**: Full TLS configuration including client certificates
- **Connection Pooling**: Efficient reuse of NATS connections per target
- **Streaming**: Large objects are streamed to reduce memory usage
- **Audit Logging**: Optional audit logging for compliance requirements
- **Metrics**: Built-in metrics collection for observability

### Error Handling

Clear error messages when targets are not configured:
```
Error: NATS target 'production' configuration incomplete (expected NATS_PRODUCTION_URL environment variable)
```

### Backward Compatibility

Existing NATS operators without targets continue to work unchanged:
```yaml
# Still works as before
config: (( nats "kv:config/app" ))
```

## Migration Guide

### Step 1: Identify Current Usage

Find all NATS operators in your templates:
```bash
grep -r "(( nats " templates/
```

### Step 2: Set Up Target Configuration

Configure environment variables for each target environment.

### Step 3: Update Templates

Replace NATS operators with target-specific versions:

```yaml
# Before
config: (( nats "kv:myapp/config" ))

# After
config: (( nats@production "kv:myapp/config" ))
```

### Step 4: Test

Verify that:
1. Target-specific NATS calls work correctly
2. Default NATS calls still work (backward compatibility)
3. Caching works independently for each target
4. TLS and authentication work correctly

## Troubleshooting

### Common Issues

1. **Missing configuration**: Ensure `NATS_{TARGET}_URL` is set
2. **Wrong target name**: Target names are case-sensitive and must match environment variable suffixes
3. **Cache conflicts**: Different targets use separate caches automatically
4. **TLS errors**: Verify certificate paths and permissions
5. **Connection timeouts**: Adjust timeout and retry settings

### Debug Mode

Enable debug logging to see NATS target resolution:
```bash
export GRAFT_DEBUG=1
graft merge template.yml
```

Look for log messages like:
```
nats: using target 'production'
nats: using target-specific client for 'production'
nats: Cache hit for `kv:config/app` (target: production)
AUDIT: Accessing KV store: config/app (target: production)
```

### Performance Tuning

- **Cache TTL**: Adjust `NATS_{TARGET}_CACHE_TTL` based on data freshness requirements
- **Connection Pooling**: Connections are automatically pooled and reused
- **Streaming Threshold**: Set `NATS_{TARGET}_STREAMING_THRESHOLD` for large object handling
- **Retry Settings**: Configure retries and backoff for unreliable networks

## Implementation Notes

- Target extraction is currently a placeholder (returns empty string)
- Full target extraction from parsed expressions will be implemented in the next iteration
- Environment variable configuration is the current approach; configuration file support is planned
- Client pooling ensures efficient reuse of NATS connections per target
- Metrics collection provides visibility into NATS operator performance