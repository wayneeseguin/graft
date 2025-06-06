# Vault Integration Guide

graft integrates with [HashiCorp Vault](https://www.vaultproject.io/) to securely retrieve secrets during YAML processing.

## Prerequisites

- Vault server accessible at `VAULT_ADDR`
- Valid authentication token in `VAULT_TOKEN`
- Appropriate policies to read required secrets

## Basic Usage

The `(( vault ))` operator retrieves secrets from Vault:

```yaml
database:
  username: (( vault "secret/db:username" ))
  password: (( vault "secret/db:password" ))
```

## Vault Path Syntax

```
(( vault "path/to/secret:key" ))
```

- **path/to/secret** - Path to the secret in Vault
- **key** - Specific key within the secret to retrieve

### Examples

```yaml
# Single key from secret
api_key: (( vault "secret/api:key" ))

# Multiple keys from same secret
database:
  host: (( vault "secret/database:host" ))
  port: (( vault "secret/database:port" ))
  username: (( vault "secret/database:username" ))
  password: (( vault "secret/database:password" ))

# Nested paths
production:
  api:
    token: (( vault "secret/production/api:token" ))
    secret: (( vault "secret/production/api:secret" ))
```

## KV Secrets Engine Versions

### KV v1 (Default)

By default, graft uses the KV v1 secrets engine:

```yaml
# Reads from KV v1 path: secret/myapp
password: (( vault "secret/myapp:password" ))
```

### KV v2

To use KV v2, set the `VAULT_VERSION` environment variable:

```bash
export VAULT_VERSION=2
```

With KV v2, graft automatically handles the `/data/` path segment:

```yaml
# Reads from KV v2 path: secret/data/myapp
# But you specify it as:
password: (( vault "secret/myapp:password" ))
```

## Default Values and Multiple Paths

As of v1.31.0, the vault operator supports default values using the logical OR (`||`) syntax. As of v2.0.0, the vault operator has been enhanced to support multiple vault paths natively.

### Basic Default Values

```yaml
# Literal default
username: (( vault "secret/db:username" || "admin" ))

# Reference as default
password: (( vault "secret/db:password" || grab defaults.password ))

# Environment variable as default
api_key: (( vault "secret/api:key" || $DEFAULT_API_KEY ))

# Nil as default for optional values
feature_flag: (( vault "secret/features:new_feature" || nil ))
```

### Multiple Vault Paths

The enhanced vault operator now supports multiple paths natively using different syntaxes:

#### Semicolon-Separated Paths (Recommended)

```yaml
# Try production first, then staging, then use default
password: (( vault "secret/prod:password; secret/staging:password" || "default-password" ))

# With concatenation
meta:
  env: production
  
password: (( vault "secret/" meta.env ":password; secret/shared:password" || "default-password" ))
```

#### Multiple Arguments Syntax

```yaml
# Each argument is a separate path to try
password: (( vault "secret/prod:password" "secret/staging:password" || "default-password" ))

# Last argument as default (without ||)
password: (( vault "secret/prod:password" "secret/staging:password" "default-password" ))
```

#### Legacy vault-try Operator

The `vault-try` operator is still supported for backward compatibility but is now deprecated:

```yaml
# Old syntax (still works but shows deprecation warning)
password: (( vault-try "secret/prod:password" "secret/staging:password" "default-password" ))

# Recommended migration to new syntax
password: (( vault "secret/prod:password; secret/staging:password" || "default-password" ))
```

### Known Limitations

#### Nested Operators in Defaults

Currently, nested operators within the default expression are not supported due to parser limitations:

```yaml
# This will NOT work as expected:
password: (( vault "secret/myapp:password" || grab defaults.password ))
password: (( vault "secret/myapp:password" || concat "prefix-" suffix ))
```

**Recommended Workaround**: Use intermediate variables:

```yaml
# For grab operator:
defaults:
  password: my-secure-default
  
# Create an intermediate variable
default_from_grab: (( grab defaults.password ))
password: (( vault "secret/myapp:password" || default_from_grab ))

# For concat operator:
prefix: "app-"
suffix: "password"

# Create an intermediate variable  
default_from_concat: (( concat prefix suffix ))
password: (( vault "secret/myapp:password" || default_from_concat ))
```

### Chained Vault Lookups

```yaml
# Multiple fallback vaults
password: (( vault "secret/primary:password" || vault "secret/secondary:password" || "default" ))

# For multiple paths, prefer semicolon syntax:
password: (( vault "secret/primary:password; secret/secondary:password" || "default" ))
```

## Sub-Operators (New in v2.1.0)

The vault operator now supports advanced sub-operators for even more powerful path construction:

### Parentheses `()` for Grouping

Use parentheses to control evaluation precedence:

```yaml
# Group expressions for clarity
vault_secret: (( vault ("secret/" env ":password") ))

# Complex grouping with references
vault_secret: (( vault ("secret/" (grab meta.environment) "/db:pass") ))
```

### Bar `|` for Choice/Alternatives

Use the bar operator to try multiple alternatives within vault paths:

```yaml
# Try different key names
database_password: (( vault "secret/db:" ("password" | "pass" | "pwd") ))

# Try different paths
api_key: (( vault ("secret/prod/api:key" | "secret/staging/api:key") ))

# Complex example with both grouping and choice
vault_key: (( vault ( meta.vault_path meta.stub  ":" ("key1" | "key2" ) | meta.exodus_path "subpath:key1") || "default"))
```

### Sub-Operator Precedence

1. **`()`** - Parentheses/grouping (highest)
2. **`|`** - Choice alternatives 
3. **Space** - Concatenation
4. **`||`** - Logical OR defaults (lowest)

### Sub-Operator Examples

```yaml
# Basic choice between key names
database:
  password: (( vault "secret/db:" ("password" | "pass") ))

# Environment-specific with fallback
api_key: (( vault ("secret/" env "/api:key" | "secret/default/api:key") ))

# Migration scenarios
app_secret: (( vault ("new-secrets/app:key" | "old-secrets/app:key" | "legacy/app:key") ))

# Multi-tenant with shared fallback
tenant_secret: (( vault ("secret/tenants/" tenant_id ":api_key" | "secret/shared:default_api_key") ))

# KV version compatibility
secret_value: (( vault ("secret/app:key" | "secret/data/app:key") ))
```

For more details and examples, see the [vault sub-operators documentation](../../examples/vault/sub-operators-README.md).
```

## Environment Variables

### Required Variables

- `VAULT_ADDR` - Vault server address (e.g., `https://vault.example.com:8200`)
- `VAULT_TOKEN` - Authentication token

### Optional Variables

- `VAULT_VERSION` - KV engine version (`1` or `2`, default: `1`)
- `VAULT_SKIP_VERIFY` - Skip TLS certificate verification (not recommended for production)
- `REDACT` - Replace secret values with `REDACTED` in output

## Security Best Practices

### 1. Use REDACT for Development

Prevent accidental credential exposure:

```bash
# Show structure without revealing secrets
REDACT=true graft merge manifest.yml
```

Output:
```yaml
database:
  username: REDACTED
  password: REDACTED
api:
  key: REDACTED
```

### 2. Temporary Files

When you need actual secrets:

```bash
# Generate temporary file with secrets
graft merge manifest.yml > /tmp/manifest-with-secrets.yml

# Use the file
deploy -f /tmp/manifest-with-secrets.yml

# Clean up immediately
rm -f /tmp/manifest-with-secrets.yml
```

### 3. Vault Policies

Create least-privilege policies:

```hcl
# Example Vault policy
path "secret/data/myapp/*" {
  capabilities = ["read"]
}

path "secret/data/shared/database" {
  capabilities = ["read"]
}
```

## Advanced Patterns

### Environment-Specific Secrets

```yaml
# Use environment variable for dynamic paths
environment: (( grab $ENVIRONMENT || "development" ))

database:
  username: (( vault (concat "secret/" environment "/db:username") ))
  password: (( vault (concat "secret/" environment "/db:password") ))

# Usage:
# ENVIRONMENT=production graft merge config.yml
```

### Shared Secrets

```yaml
# Define once, use multiple times
secrets:
  api_base: "secret/api/keys"
  db_base: "secret/database/prod"

services:
  payment:
    key: (( vault (concat secrets.api_base "/payment:key") ))
    secret: (( vault (concat secrets.api_base "/payment:secret") ))
  
  analytics:
    key: (( vault (concat secrets.api_base "/analytics:key") ))
    token: (( vault (concat secrets.api_base "/analytics:token") ))

database:
  primary:
    password: (( vault (concat secrets.db_base "/primary:password") ))
  replica:
    password: (( vault (concat secrets.db_base "/replica:password") ))
```

### Conditional Secrets

```yaml
environment: production

# Only use Vault in production
database:
  password: (( environment == "production" ? (vault "secret/db:password") : "dev-password" ))

# Different paths per environment
api_key: (( vault (concat "secret/" environment "/api:key") ))
```

## Complete Example

```yaml
# config.yml
meta:
  environment: (( grab $ENVIRONMENT || "development" ))
  vault_base: (( concat "secret/" meta.environment ))

app:
  name: my-application
  version: 1.2.3

# Database configuration
database:
  host: (( meta.environment == "production" ? "prod-db.example.com" : "localhost" ))
  port: 5432
  name: myapp
  credentials:
    username: (( vault (concat meta.vault_base "/database:username") || "postgres" ))
    password: (( vault (concat meta.vault_base "/database:password") || "password" ))
  
  # Connection string
  url: (( concat 
    "postgresql://" 
    database.credentials.username ":"
    database.credentials.password "@"
    database.host ":"
    database.port "/"
    database.name
  ))

# External services
services:
  stripe:
    public_key: (( vault (concat meta.vault_base "/stripe:public_key") ))
    secret_key: (( vault (concat meta.vault_base "/stripe:secret_key") ))
  
  sendgrid:
    api_key: (( vault (concat meta.vault_base "/sendgrid:api_key") ))
  
  aws:
    access_key_id: (( vault (concat meta.vault_base "/aws:access_key_id") ))
    secret_access_key: (( vault (concat meta.vault_base "/aws:secret_access_key") ))

# TLS certificates
tls:
  cert: (( vault (concat meta.vault_base "/tls:cert") ))
  key: (( vault (concat meta.vault_base "/tls:key") ))
  ca: (( vault (concat meta.vault_base "/tls:ca") || "" ))

# Feature flags with defaults
features:
  new_payment_flow: (( vault "secret/features:new_payment_flow" || false ))
  beta_ui: (( vault "secret/features:beta_ui" || false ))
```

## Usage Workflow

```bash
# 1. Set up Vault environment
export VAULT_ADDR="https://vault.example.com:8200"
export VAULT_TOKEN="s.abcdef123456"

# 2. Preview without secrets
REDACT=true graft merge config.yml

# 3. Test specific environment
ENVIRONMENT=staging REDACT=true graft merge config.yml

# 4. Generate final configuration
ENVIRONMENT=production graft merge config.yml > final.yml

# 5. Deploy and cleanup
kubectl apply -f final.yml
rm -f final.yml
```

## Troubleshooting

### Common Issues

**Permission Denied:**
```bash
# Check your token capabilities
vault token capabilities secret/myapp
```

**Path Not Found:**
```bash
# Verify the path exists
vault kv get secret/myapp

# For KV v2, check if VAULT_VERSION is set correctly
export VAULT_VERSION=2
```

**Connection Errors:**
```bash
# Verify Vault address
curl -k $VAULT_ADDR/v1/sys/health

# Check token validity
vault token lookup
```

### Debugging

Enable debug mode to see Vault operations:

```bash
graft merge --debug manifest.yml 2>&1 | grep -i vault
```

## vaultinfo Command

Analyze Vault usage in your manifests:

```bash
# List all Vault paths
graft vaultinfo manifest.yml

# Output as YAML
graft vaultinfo --yaml manifest.yml

# Check multiple files
graft vaultinfo base.yml production.yml
```

## Migration from Plain Secrets

Transform hardcoded secrets to Vault references:

Before:
```yaml
database:
  password: "hardcoded-password"
api:
  key: "abcd1234"
```

After:
```yaml
database:
  password: (( vault "secret/database:password" ))
api:
  key: (( vault "secret/api:key" ))
```

Store secrets in Vault:
```bash
vault kv put secret/database password="hardcoded-password"
vault kv put secret/api key="abcd1234"
```

## See Also

- [Vault Operator Reference](../operators/external-data.md#vault)
- [Environment Variables](../concepts/environment-variables.md)
- [AWS Integration](aws-integration.md) - Alternative secret sources
- [Examples](../../examples/vault/) - Vault usage examples