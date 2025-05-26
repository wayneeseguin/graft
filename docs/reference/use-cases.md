# Spruce Use Cases Guide

This guide helps you find the right Spruce operators and patterns for common configuration management tasks.

## Table of Contents

1. [Managing Secrets](#managing-secrets)
2. [Building Dynamic Configuration](#building-dynamic-configuration)
3. [Multi-Environment Deployments](#multi-environment-deployments)
4. [Kubernetes Manifests](#kubernetes-manifests)
5. [BOSH Deployments](#bosh-deployments)
6. [Configuration Validation](#configuration-validation)
7. [Template Generation](#template-generation)
8. [Data Processing](#data-processing)

---

## Managing Secrets

### Problem: Store secrets securely outside of config files

**Solution: Use Vault integration**
```yaml
database:
  host: db.example.com
  username: dbuser
  password: (( vault "secret/db:password" ))
```

**With fallback for development:**
```yaml
database:
  password: (( vault "secret/db:password" || grab defaults.dev_password ))

defaults:
  dev_password: "changeme"  # Only for local development
```

See: [Vault Integration Guide](../guides/vault-integration.md), [Vault Examples](../../examples/vault/)

### Problem: Rotate secrets without changing config

**Solution: Use vault-try for migration**
```yaml
api_key: (( vault-try "secret/v2/api:key" "secret/v1/api:key" "default-key" ))
```

See: [Vault-try Examples](../../examples/vault-try/)

### Problem: Encode secrets for Kubernetes

**Solution: Base64 encode secrets**
```yaml
apiVersion: v1
kind: Secret
data:
  password: (( base64 (vault "secret/db:password") ))
```

See: [Base64 Examples](../../examples/base64/)

---

## Building Dynamic Configuration

### Problem: Don't repeat yourself (DRY)

**Solution: Use grab to reference values**
```yaml
meta:
  app_name: my-app
  environment: production
  domain: example.com

app:
  name: (( grab meta.app_name ))
  url: (( concat "https://" meta.app_name "." meta.environment "." meta.domain ))
```

See: [Grab Examples](../../examples/grab/)

### Problem: Build configuration from templates

**Solution: Use inject for template inheritance**
```yaml
templates:
  base_service:
    port: 8080
    healthcheck: /health
    timeout: 30

services:
  api:
    _: (( inject templates.base_service ))
    port: 8081  # Override specific value
```

See: [Inject Examples](../../examples/inject/)

### Problem: Conditional configuration

**Solution: Use empty checks and ternary operators**
```yaml
config:
  debug: (( grab environment == "production" ? false : true ))
  log_level: (( empty LOG_LEVEL ? "info" : grab LOG_LEVEL ))
```

See: [Empty Examples](../../examples/empty/), [Ternary Examples](../../examples/ternary/)

---

## Multi-Environment Deployments

### Problem: Environment-specific configuration

**Solution: Use parameters and environment variables**
```yaml
# base.yml
environment: (( param "Please specify environment" ))
config:
  api_endpoint: (( grab endpoints.$ENVIRONMENT ))

endpoints:
  dev: https://api.dev.example.com
  prod: https://api.prod.example.com
```

Usage:
```bash
ENVIRONMENT=prod spruce merge base.yml <(echo "environment: prod")
```

See: [Environment Variables](../concepts/environment-variables.md)

### Problem: Load environment-specific files

**Solution: Use load with dynamic paths**
```yaml
environment: (( grab $ENV || "development" ))
env_config: (( load (concat "config/" environment ".yml") ))

final_config:
  <<: (( grab env_config ))
  <<: (( grab overrides ))
```

See: [Load Examples](../../examples/load/)

---

## Kubernetes Manifests

### Problem: Generate ConfigMaps from files

**Solution: Use file or stringify operators**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  config.yaml: (( file "config/app.yaml" ))
  settings.json: (( stringify app_settings ))

app_settings:
  debug: false
  port: 8080
```

See: [File Examples](../../examples/file/), [Stringify Examples](../../examples/stringify/)

### Problem: Manage multiple similar resources

**Solution: Use array operations and references**
```yaml
services:
  - (( merge on name ))
  - name: web
    replicas: 3
  - name: api
    replicas: 2

deployments:
  - name: web
    image: myapp/web:latest
    replicas: (( grab services.[0].replicas ))
  - name: api
    image: myapp/api:latest
    replicas: (( grab services.[1].replicas ))
```

See: [Array Merging](../concepts/array-merging.md)

---

## BOSH Deployments

### Problem: Allocate static IPs

**Solution: Use static_ips operator**
```yaml
networks:
  - name: private
    static: [10.0.0.10 - 10.0.0.50]

instance_groups:
  - name: database
    instances: 3
    networks:
      - name: private
        static_ips: (( static_ips 0 1 2 ))
```

See: [Static IPs Examples](../../examples/static-ips/)

### Problem: Calculate IP ranges

**Solution: Use ips operator**
```yaml
subnet: "10.0.0.0/24"
services:
  gateway: (( ips subnet 1 ))         # 10.0.0.1
  dns_servers: (( ips subnet 2 2 ))   # [10.0.0.2, 10.0.0.3]
  dhcp_range: (( ips subnet 100 50 )) # 100-149
```

See: [IPs Examples](../../examples/ips/)

---

## Configuration Validation

### Problem: Ensure required values are provided

**Solution: Use param operator**
```yaml
database:
  host: (( param "Database host is required" ))
  port: (( grab $DB_PORT || 5432 ))
  username: (( param "Database username is required" ))
  password: (( param "Database password is required" ))
```

See: [Param Examples](../../examples/params/)

### Problem: Validate configuration completeness

**Solution: Use empty checks**
```yaml
validation:
  has_database: (( ! empty database.host && ! empty database.password ))
  has_api_key: (( ! empty api.key ))
  is_valid: (( validation.has_database && validation.has_api_key ))
```

See: [Null Examples](../../examples/null/)

---

## Template Generation

### Problem: Generate configuration with Spruce operators

**Solution: Use defer operator**
```yaml
# Generate a template that will be processed later
template:
  database:
    host: (( defer (( grab meta.db_host )) ))
    password: (( defer (( vault "secret/db:password" )) ))
```

Output:
```yaml
template:
  database:
    host: (( grab meta.db_host ))
    password: (( vault "secret/db:password" ))
```

See: [Defer Examples](../../examples/defer/), [Meta-Programming Guide](../guides/meta-programming.md)

### Problem: Remove temporary data from output

**Solution: Use prune operator**
```yaml
meta:
  temp_data: (( prune ))
  calculations: (( prune ))
  
app:
  name: my-app
  version: 1.0.0
  # meta fields won't appear in output
```

See: [Prune Examples](../../examples/prune/)

---

## Data Processing

### Problem: Transform arrays

**Solution: Use array operators**
```yaml
# Sort data
users:
  - name: charlie
    age: 30
  - name: alice
    age: 25

sorted_users: (( sort by name ))  # Sorts by name

# Generate combinations
colors: [red, blue]
sizes: [S, M, L]
variants: (( cartesian-product colors sizes ))
# Result: [[red,S], [red,M], [red,L], [blue,S], [blue,M], [blue,L]]
```

See: [Sort Examples](../../examples/sort/), [Cartesian Product Examples](../../examples/cartesian-product/)

### Problem: Perform calculations

**Solution: Use math operators**
```yaml
resources:
  cpu_per_instance: 2
  instances: 5
  total_cpu: (( resources.cpu_per_instance * resources.instances ))
  
  memory_gb: 16
  memory_per_instance: (( calc "floor(memory_gb / instances)" ))
```

See: [Arithmetic Examples](../../examples/arithmetic/), [Calc Examples](../../examples/calc/)

---

## Quick Decision Tree

**"I need to..."**

- **Merge YAML files** → Basic `spruce merge`
- **Keep config DRY** → Use `(( grab ))`
- **Store secrets** → Use `(( vault ))` or `(( awssecret ))`
- **Build strings** → Use `(( concat ))`
- **Check if empty** → Use `(( empty ))`
- **Require input** → Use `(( param ))`
- **Load external files** → Use `(( file ))` or `(( load ))`
- **Generate IPs** → Use `(( static_ips ))` or `(( ips ))`
- **Transform arrays** → Use array operators
- **Calculate values** → Use `(( calc ))` or arithmetic
- **Remove from output** → Use `(( prune ))`
- **Generate templates** → Use `(( defer ))`

## See Also

- [Operator Quick Reference](operator-quick-reference.md)
- [Getting Started Guide](../getting-started.md)
- [Examples Directory](../../examples/)
- [Complete Operator List](../index.md#operators)