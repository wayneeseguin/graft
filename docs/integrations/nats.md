# NATS Integration Guide

This guide covers how to use graft with NATS JetStream for configuration management and data storage.

## Overview

The NATS operator allows graft to fetch values from:
- **Key-Value (KV) stores** - For simple configuration values
- **Object stores** - For files, templates, and larger data

## Prerequisites

1. **NATS Server with JetStream enabled**
   ```bash
   # Docker example
   docker run -d --name nats \
     -p 4222:4222 \
     -p 8222:8222 \
     nats:latest \
     -js
   ```

2. **NATS CLI tools** (optional but recommended)
   ```bash
   # Install NATS CLI
   brew install nats-io/nats-tools/nats
   # or
   go install github.com/nats-io/natscli/nats@latest
   ```

## Setting Up NATS

### Create a Key-Value Store

```bash
# Create a KV store for configuration
nats kv add config --replicas 3 --ttl 1h

# Add values
nats kv put config database_host "db.example.com"
nats kv put config database_port "5432"
nats kv put config api_timeout "30s"

# Store YAML data
cat <<EOF | nats kv put config app_settings -
debug: false
features:
  - auth
  - api
  - websocket
limits:
  max_connections: 1000
  request_timeout: 30s
EOF
```

### Create an Object Store

```bash
# Create an object store for files
nats object add assets --replicas 3

# Upload files
nats object put assets config.yaml --file=config.yaml
nats object put assets logo.png --file=logo.png
nats object put assets template.tmpl --file=template.tmpl
```

## Basic Usage

### Simple Configuration

```yaml
# config.yml
application:
  name: myapp
  
  # Fetch from KV store
  database:
    host: (( nats "kv:config/database_host" ))
    port: (( nats "kv:config/database_port" ))
    
  # Fetch from object store
  settings: (( nats "obj:assets/config.yaml" ))
```

### With Connection Configuration

```yaml
# Specify NATS server URL
production:
  api_key: (( nats "kv:secrets/api_key" "nats://prod-nats:4222" ))

# Full configuration
secure:
  data: (( nats "kv:secure/data" {
    url: "nats://secure-nats:4222",
    timeout: "10s",
    retries: 5,
    tls: true
  } ))
```

## Authentication Methods

### Basic Authentication

```yaml
# Username/password in URL
authenticated:
  value: (( nats "kv:protected/value" "nats://user:password@localhost:4222" ))
```

### Token Authentication

```yaml
# Token in URL
token_auth:
  config: (( nats "obj:protected/config.yaml" "nats://mytoken@localhost:4222" ))
```

### TLS Configuration

```yaml
tls_connection:
  secrets: (( nats "kv:encrypted/secrets" {
    url: "nats://secure-nats:4222",
    tls: true,
    cert_file: "/etc/nats/client.crt",
    key_file: "/etc/nats/client.key"
  } ))
```

## Advanced Patterns

### Environment-Specific Configuration

```yaml
# base.yml
meta:
  environment: (( grab $ENVIRONMENT || "development" ))

# NATS servers per environment
nats_config:
  development: "nats://localhost:4222"
  staging: "nats://staging-nats:4222"
  production:
    url: "nats://prod-nats:4222"
    tls: true

# Use environment-specific server
config:
  database: (( nats 
    (concat "obj:configs/" meta.environment "/database.yaml")
    (grab (concat "nats_config." meta.environment)) ))
```

### Dynamic Path Construction

```yaml
# Service-based configuration
services:
  - name: api-gateway
    port: 8080
    config: (( nats (concat "kv:services/" name "/config") ))
    
  - name: auth-service
    port: 8081
    config: (( nats (concat "kv:services/" name "/config") ))
```

### Multi-Tenant Configuration

```yaml
tenants:
  - id: acme-corp
    config: (( nats (concat "obj:tenants/" id "/config.yaml") ))
    
  - id: globex
    config: (( nats (concat "obj:tenants/" id "/config.yaml") ))
```

### Binary Asset Management

```yaml
# Binary files are automatically base64 encoded
web_assets:
  images:
    logo: (( nats "obj:assets/images/logo.png" ))
    favicon: (( nats "obj:assets/images/favicon.ico" ))
    
  # Use with Kubernetes secrets
  kubernetes_secret:
    apiVersion: v1
    kind: Secret
    metadata:
      name: app-assets
    type: Opaque
    data:
      logo.png: (( grab web_assets.images.logo ))
      favicon.ico: (( grab web_assets.images.favicon ))
```

## Best Practices

### 1. Use Namespaces

Organize your KV stores and object buckets:
```bash
# KV stores by purpose
nats kv add config      # General configuration
nats kv add secrets     # Sensitive data
nats kv add features    # Feature flags

# Object stores by type
nats object add configs    # YAML/JSON configs
nats object add templates  # Templates
nats object add assets     # Binary assets
```

### 2. Set TTLs for Dynamic Data

```bash
# Create KV store with TTL
nats kv add cache --ttl 5m

# Values will expire automatically
nats kv put cache session_token "abc123"
```

### 3. Use Replication for High Availability

```bash
# Create replicated stores
nats kv add config --replicas 3
nats object add assets --replicas 3
```

### 4. Monitor Store Usage

```bash
# Check KV store status
nats kv status config

# Check object store status
nats object status assets
```

### 5. Implement Access Controls

Use NATS security features:
- User authentication
- Permissions per store
- TLS encryption

## Troubleshooting

### Connection Issues

1. **Check NATS server is running**
   ```bash
   nats server check
   ```

2. **Verify JetStream is enabled**
   ```bash
   nats server report jetstream
   ```

3. **Test connectivity**
   ```yaml
   test: (( nats "kv:config/test" "nats://localhost:4222" ))
   ```

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `failed to connect to NATS` | Server unreachable | Check URL and network |
| `failed to access KV store` | Store doesn't exist | Create with `nats kv add` |
| `failed to get key` | Key doesn't exist | Add with `nats kv put` |
| `failed to access Object store` | Bucket doesn't exist | Create with `nats object add` |

### Performance Tips

1. **Use caching** - Values are cached per graft execution
2. **Batch operations** - Fetch multiple values in one template
3. **Use appropriate store types** - KV for small values, Object for files
4. **Set connection timeouts** - Prevent hanging on network issues

## Migration from Other Systems

### From File-Based Config

Before:
```yaml
config: (( load "config/production.yml" ))
```

After:
```yaml
config: (( nats "obj:configs/production.yml" ))
```

### From Environment Variables

Before:
```yaml
database:
  host: (( grab $DB_HOST ))
  port: (( grab $DB_PORT ))
```

After:
```yaml
database:
  host: (( nats "kv:config/db_host" ))
  port: (( nats "kv:config/db_port" ))
```

### From Vault

Before:
```yaml
secrets:
  api_key: (( vault "secret/api:key" ))
```

After:
```yaml
secrets:
  api_key: (( nats "kv:secrets/api_key" ))
```

## Security Considerations

1. **Use TLS in production**
2. **Implement proper authentication**
3. **Separate sensitive data into different stores**
4. **Use NATS ACLs to restrict access**
5. **Consider using Vault for highly sensitive data**

## Examples

See the [NATS examples directory](/examples/nats/) for complete working examples.