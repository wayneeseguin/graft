# External Data Sources

These operators load data from external sources like HashiCorp Vault, AWS services, and local files.

## (( vault ))

Usage: `(( vault PATH[:KEY] [|| DEFAULT] ))`

The `(( vault ))` operator retrieves secrets from [HashiCorp Vault](https://www.vaultproject.io/). It connects to Vault using environment variables for authentication and configuration.

### Configuration

Required environment variables:
- `VAULT_ADDR` - Vault server URL
- `VAULT_TOKEN` - Authentication token (or use other auth methods)

Optional:
- `VAULT_SKIP_VERIFY` - Skip TLS verification (not recommended for production)
- `VAULT_NAMESPACE` - Vault namespace (Enterprise feature)

### Examples:

```yaml
# Basic secret retrieval
database:
  password: (( vault "secret/db:password" ))
  # Retrieves the 'password' field from secret/db

# Full path reference
credentials:
  api_key: (( vault "secret/services/api:key" ))

# With default values (v1.31.0+)
config:
  # Use default if secret doesn't exist
  db_password: (( vault "secret/db:password" || "changeme" ))
  
  # Reference as default
  api_key: (( vault "secret/api:key" || grab defaults.api_key ))
  
  # Environment variable as default
  token: (( vault "secret/token:value" || $DEFAULT_TOKEN ))

# Dynamic path construction
meta:
  env: production
  app: myapp

secrets:
  # Constructs: secret/production/myapp/db:password
  db_pass: (( vault (concat "secret/" meta.env "/" meta.app "/db:password") ))

# Multiple fields from same path
database:
  host: db.example.com
  username: (( vault "secret/db:username" ))
  password: (( vault "secret/db:password" ))
  port: (( vault "secret/db:port" || 5432 ))
```

See also: [vault examples](/examples/vault/), [vault integration guide](../guides/vault-integration.md)

## (( vault-try ))

Usage: `(( vault-try PATH1 PATH2 ... DEFAULT ))`

The `(( vault-try ))` operator attempts multiple Vault paths in sequence, using the first successful result. This is useful for migration scenarios or multi-environment setups.

### Examples:

```yaml
# Try multiple paths
database:
  # Try new path structure, fall back to legacy
  password: (( vault-try "secret/v2/db:password" "secret/db:password" "default-pass" ))

# Environment-specific with fallback
meta:
  env: staging

config:
  # Try env-specific, then shared, then default
  api_key: (( vault-try 
    (concat "secret/" meta.env "/api:key")
    "secret/shared/api:key" 
    "dev-api-key" ))

# Migration scenario
old_path: "secret/myapp/prod"
new_path: "secret/data/prod/myapp"

credentials:
  db_user: (( vault-try 
    (concat new_path "/db:username")
    (concat old_path "/db:username")
    "postgres" ))
  db_pass: (( vault-try 
    (concat new_path "/db:password")
    (concat old_path "/db:password")
    "changeme" ))

# Multi-tenant configuration
tenant:
  id: "acme-corp"

secrets:
  # Try tenant-specific, then default tenant, then hardcoded
  api_token: (( vault-try
    (concat "secret/tenants/" tenant.id "/api:token")
    "secret/tenants/default/api:token"
    "no-token" ))
```

See also: [vault-try examples](/examples/vault/vault-try.yml)

## (( awsparam ))

Usage: `(( awsparam PATH [?key=SUBKEY] ))`

The `(( awsparam ))` operator retrieves parameters from [AWS Systems Manager Parameter Store](https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html).

### Configuration

Uses standard AWS SDK authentication:
- Environment variables: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
- IAM instance profile
- AWS credentials file

### Examples:

```yaml
# Simple parameter
database:
  host: (( awsparam "/myapp/prod/db_host" ))

# With JSON extraction
# If parameter contains: {"username": "admin", "password": "secret"}
credentials:
  user: (( awsparam "/myapp/prod/db_creds?key=username" ))
  pass: (( awsparam "/myapp/prod/db_creds?key=password" ))

# Dynamic paths
environment: production
app_name: myapp

config:
  # Constructs: /myapp/production/api_key
  api_key: (( awsparam (concat "/" app_name "/" environment "/api_key") ))

# With references
paths:
  db_config: /myapp/prod/database

database:
  host: (( awsparam (concat paths.db_config "?key=host") ))
  port: (( awsparam (concat paths.db_config "?key=port") ))
```

## (( awssecret ))

Usage: `(( awssecret NAME_OR_ARN [?key=SUBKEY&stage=STAGE&version=VERSION] ))`

The `(( awssecret ))` operator retrieves secrets from [AWS Secrets Manager](https://aws.amazon.com/secrets-manager/).

### Parameters:
- `key` - Extract specific key from JSON secret
- `stage` - Retrieve specific stage (e.g., AWSCURRENT, AWSPREVIOUS)
- `version` - Retrieve specific version ID

### Examples:

```yaml
# Simple secret
api:
  key: (( awssecret "prod/myapp/api_key" ))

# Using ARN
database:
  password: (( awssecret "arn:aws:secretsmanager:us-east-1:123456789:secret:prod/db-AbCdEf" ))

# JSON secret with key extraction
# If secret contains: {"username": "dbuser", "password": "dbpass", "host": "db.example.com"}
database:
  username: (( awssecret "prod/myapp/db?key=username" ))
  password: (( awssecret "prod/myapp/db?key=password" ))
  host: (( awssecret "prod/myapp/db?key=host" ))

# Specific version/stage
config:
  current_key: (( awssecret "prod/api/key?stage=AWSCURRENT" ))
  previous_key: (( awssecret "prod/api/key?stage=AWSPREVIOUS" ))
  specific_version: (( awssecret "prod/api/key?version=abc123def456" ))

# Combined parameters
rotating_secret: (( awssecret "prod/rotating?key=password&stage=AWSPENDING" ))

# Dynamic secret names
meta:
  env: production
  service: api

secrets:
  api_key: (( awssecret (concat meta.env "/" meta.service "/key") ))
```

## (( file ))

Usage: `(( file PATH ))`

The `(( file ))` operator reads the contents of a file and inserts it as a string value. This is useful for embedding certificates, keys, or other text files.

### Examples:

```yaml
# Load certificate
tls:
  certificate: (( file "certs/server.crt" ))
  private_key: (( file "certs/server.key" ))
  ca_bundle: (( file "certs/ca-bundle.crt" ))

# Dynamic file paths
environment: prod
certificates:
  path: (( concat "certs/" environment "-cert.pem" ))
  cert: (( file certificates.path ))

# Load configuration file
app_config: (( file "config/application.conf" ))

# Kubernetes secret from files
apiVersion: v1
kind: Secret
metadata:
  name: app-secrets
type: Opaque
data:
  cert.pem: (( base64 (file "ssl/cert.pem") ))
  key.pem: (( base64 (file "ssl/key.pem") ))

# Multi-line scripts
scripts:
  startup: (( file "scripts/startup.sh" ))
  shutdown: (( file "scripts/shutdown.sh" ))
```

See also: [file examples](/examples/file/)

## (( load ))

Usage: `(( load PATH ))`

The `(( load ))` operator parses a YAML or JSON file and inserts its structure into the current document. Unlike `file`, which treats content as a string, `load` parses the content as structured data.

**Note:** graft operators in loaded files are NOT evaluated. Pre-process files separately if needed.

### Examples:

```yaml
# Load YAML structure
# users.yml contains:
# - name: alice
#   role: admin
# - name: bob
#   role: user

users: (( load "config/users.yml" ))

# Load JSON configuration
# app-config.json contains:
# {
#   "debug": true,
#   "port": 8080,
#   "features": ["auth", "api", "ui"]
# }

app_settings: (( load "config/app-config.json" ))

# Dynamic loading
environment: staging
config_file: (( concat "config/" environment ".yml" ))
environment_config: (( load config_file ))

# Modular configuration
modules:
  database: (( load "modules/database.yml" ))
  cache: (( load "modules/cache.yml" ))
  monitoring: (( load "modules/monitoring.yml" ))

# Load and merge
base_config: (( load "config/base.yml" ))
overrides:
  port: 9090
  debug: false

# Note: Would need separate merge step
final_config:
  <<: (( grab base_config ))
  <<: (( grab overrides ))
```

See also: [load examples](/examples/load/)

## (( nats ))

Usage: `(( nats "STORE_TYPE:PATH" [CONNECTION_CONFIG] ))`

The `(( nats ))` operator retrieves values from [NATS JetStream](https://docs.nats.io/jetstream) Key-Value (KV) stores and Object stores. It intelligently handles different data types and provides seamless integration with graft's YAML processing.

### Store Types:
- `kv:` - Key-Value store for simple key-value pairs
- `obj:` - Object store for files and larger data

### Connection Configuration:
Can be a URL string or configuration map:
```yaml
# Simple URL
value: (( nats "kv:store/key" "nats://localhost:4222" ))

# Configuration map
value: (( nats "kv:store/key" {
  url: "nats://user:pass@localhost:4222",
  timeout: "10s",
  retries: 5,
  tls: true,
  cert_file: "/path/to/cert.pem",
  key_file: "/path/to/key.pem"
} ))
```

### Key-Value Store Examples:

```yaml
# Simple value retrieval
database:
  host: (( nats "kv:config/db_host" ))
  port: (( nats "kv:config/db_port" ))

# With connection URL
api:
  key: (( nats "kv:secrets/api_key" "nats://localhost:4222" ))

# Dynamic path construction
services:
  - name: api
    config: (( nats (concat "kv:services/" name "/config") ))
  - name: worker
    config: (( nats (concat "kv:services/" name "/config") ))

# YAML data in KV (automatically parsed)
# If KV contains: 'foo: bar\nnested:\n  key: value'
config: (( nats "kv:app/config" ))
# Result: {foo: bar, nested: {key: value}}
```

### Object Store Examples:

```yaml
# YAML/JSON files (automatically parsed)
app_config: (( nats "obj:configs/app.yaml" ))
settings: (( nats "obj:configs/settings.json" ))

# Text files (returned as strings)
readme: (( nats "obj:docs/README.md" ))
license: (( nats "obj:docs/LICENSE" ))

# Binary files (automatically base64 encoded)
assets:
  logo: (( nats "obj:assets/logo.png" ))
  favicon: (( nats "obj:assets/favicon.ico" ))

# Multi-environment setup
environments:
  dev:
    config: (( nats "obj:configs/dev.yaml" ))
  staging:
    config: (( nats "obj:configs/staging.yaml" ))
  production:
    config: (( nats "obj:configs/prod.yaml" {
      url: "nats://prod-nats:4222",
      tls: true
    } ))
```

### Content Type Handling:
Object store automatically handles content types:
- `text/yaml`, `application/yaml` - Parsed as YAML structure
- `application/json` - Parsed as JSON (converted to YAML)
- `text/plain`, no content type - Returned as string
- Other types - Base64 encoded

### Advanced Examples:

```yaml
# TLS configuration
secure_data: (( nats "kv:secure/data" {
  url: "nats://secure-nats:4222",
  tls: true,
  cert_file: "/etc/nats/client.crt",
  key_file: "/etc/nats/client.key"
} ))

# Token authentication
protected: (( nats "obj:protected/config.yaml" "nats://mytoken@localhost:4222" ))

# Combining with other operators
database:
  # Fetch config from NATS
  config: (( nats "obj:database/config.yaml" ))
  
  # Override with vault secret
  password: (( vault "secret/db:password" || grab config.password ))
  
  # Use static IPs from NATS network config
  network: (( nats "obj:networks/production.yaml" ))
  static_ips: (( static_ips 0 10 (grab network.subnets) ))
```

### Advanced Configuration:

```yaml
# Enhanced configuration with all available options
reliable_config: (( nats "kv:config/critical" {
  url: "nats://cluster.example.com:4222"
  timeout: "15s"
  retries: 10
  retry_interval: "2s"
  retry_backoff: 2.0
  max_retry_interval: "60s"
  cache_ttl: "10m"
  streaming_threshold: 5242880
  audit_logging: true
} ))

# Full TLS configuration with client certificates
secure_config: (( nats "obj:secrets/app.yaml" {
  url: "tls://secure.nats.example.com:4222"
  timeout: "20s"
  retries: 5
  tls: true
  cert_file: "/etc/ssl/certs/client.crt"
  key_file: "/etc/ssl/private/client.key"
  ca_file: "/etc/ssl/certs/ca.crt"
  insecure_skip_verify: false
} ))
```

### Enhanced Performance Features:
- **Connection Pooling**: Automatic connection reuse with reference counting and 5-minute idle timeout
- **TTL-based Caching**: Configurable cache expiration with thread-safe implementation
- **Streaming Support**: Memory-efficient handling of large objects with configurable thresholds
- **Retry Logic**: Exponential backoff with configurable intervals and maximum retry limits
- **Automatic Cleanup**: Background cleanup of idle connections and expired cache entries

### Security and Observability:
- **Audit Logging**: Optional audit trail for sensitive data access (configurable via `audit_logging`)
- **Metrics Collection**: Built-in metrics for operations, cache performance, and error rates
- **TLS Support**: Full certificate-based authentication with CA validation
- **Memory Safety**: Streaming prevents excessive memory usage for large objects

### Caching:
The NATS operator caches fetched values for the duration of the graft execution to improve performance.

See also: [nats examples](/examples/nats/)

## Common Patterns

### Multi-Source Configuration
```yaml
# Combine multiple external sources
config:
  # From Vault
  database:
    password: (( vault "secret/db:password" ))
  
  # From AWS Parameter Store
  feature_flags: (( awsparam "/myapp/features" ))
  
  # From local file
  ssl_cert: (( file "certs/server.crt" ))
  
  # From loaded config
  defaults: (( load "config/defaults.yml" ))
```

### Environment-Specific Loading
```yaml
environment: (( grab $ENVIRONMENT || "development" ))

# Load environment-specific config
env_config: (( load (concat "config/" environment ".yml") ))

# Try multiple secret sources
database:
  password: (( vault-try 
    (concat "secret/" environment "/db:password")
    "secret/shared/db:password"
    (grab env_config.database.password) ))
```

### Certificate Management
```yaml
# Load certificates from files or vault
tls:
  # From files
  file_based:
    cert: (( file "certs/server.crt" ))
    key: (( file "certs/server.key" ))
  
  # From Vault
  vault_based:
    cert: (( vault "secret/tls:cert" ))
    key: (( vault "secret/tls:key" ))
  
  # From AWS Secrets Manager
  aws_based:
    cert: (( awssecret "prod/tls/cert" ))
    key: (( awssecret "prod/tls/key" ))
```

### Modular Configuration
```yaml
# Main configuration file
name: my-application

# Load modules
database: (( load "modules/database.yml" ))
cache: (( load "modules/cache.yml" ))
logging: (( load "modules/logging.yml" ))

# Override specific values
database:
  host: prod.db.example.com

# Add secrets
database:
  password: (( vault "secret/prod/db:password" ))
```