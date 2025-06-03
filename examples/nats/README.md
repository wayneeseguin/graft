# NATS Operator Examples

The NATS operator allows graft to fetch values from NATS JetStream Key-Value (KV) stores and Object stores.

## Prerequisites

1. A running NATS server with JetStream enabled
2. Create KV stores and/or Object stores with test data

## Basic Usage

### Key-Value Store

```yaml
# Fetch a simple value from KV store
database_url: (( nats "kv:config/database_url" ))

# With connection URL
api_key: (( nats "kv:secrets/api_key" "nats://localhost:4222" ))

# With full configuration
service_token: (( nats "kv:auth/token" {
  url: "nats://prod-nats:4222",
  timeout: "10s",
  retries: 5,
  tls: true
} ))
```

### Object Store

```yaml
# Fetch YAML object (automatically parsed)
app_config: (( nats "obj:configs/app.yaml" ))

# Fetch text file
readme: (( nats "obj:docs/README.md" ))

# Fetch binary file (automatically base64 encoded)
logo: (( nats "obj:assets/logo.png" ))
```

## Setting Up Test Data

### Create a KV Store

```bash
# Using NATS CLI
nats kv add config
nats kv put config database_url "postgres://localhost:5432/mydb"
nats kv put config api_endpoint "https://api.example.com"
```

### Create an Object Store

```bash
# Using NATS CLI
nats object add configs
nats object put configs app.yaml --file=app.yaml
nats object put configs logo.png --file=logo.png
```

## Running Examples

```bash
# Basic KV example
graft merge basic.yml

# Multi-environment configuration
graft merge environment-config.yml

# Binary asset handling
graft merge binary-data.yml
```

## Advanced Features

### Dynamic Path Construction

```yaml
services:
  - name: api
    config: (( nats (( concat "kv:services/" name "/config" )) ))
  - name: worker
    config: (( nats (( concat "kv:services/" name "/config" )) ))
```

### Error Handling with Defaults

```yaml
# Use logical OR for fallback values
database_url: (( nats "kv:config/db_url" || "postgres://localhost:5432/default" ))
```

### Caching

The NATS operator caches fetched values to improve performance. The cache is maintained for the duration of the graft execution.