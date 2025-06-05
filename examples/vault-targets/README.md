# Vault Operator with Targets

This example demonstrates how to use the vault operator with different targets to connect to multiple Vault instances.

## Overview

The vault operator now supports target syntax to specify which Vault instance to connect to:

```yaml
# Connect to production vault
production_password: (( vault@production "secret/app:password" ))

# Connect to staging vault  
staging_password: (( vault@staging "secret/app:password" ))

# Use default vault configuration
default_password: (( vault "secret/app:password" ))
```

## Configuration

### Environment Variables

For each target, you need to set environment variables:

```bash
# Production vault
export VAULT_PRODUCTION_ADDR="https://vault.prod.example.com"
export VAULT_PRODUCTION_TOKEN="prod-vault-token"
export VAULT_PRODUCTION_NAMESPACE="production"

# Staging vault
export VAULT_STAGING_ADDR="https://vault.staging.example.com"
export VAULT_STAGING_TOKEN="staging-vault-token"
export VAULT_STAGING_NAMESPACE="staging"

# Default vault (existing behavior)
export VAULT_ADDR="https://vault.default.example.com"
export VAULT_TOKEN="default-vault-token"
```

### Configuration File (Future)

In future versions, targets will be configurable via a configuration file:

```yaml
# ~/.graft/config.yml
targets:
  vault:
    production:
      url: "${VAULT_PROD_ADDR}"
      token: "${VAULT_PROD_TOKEN}"
      namespace: "production"
      skip_verify: false
    staging:
      url: "${VAULT_STAGING_ADDR}"
      token: "${VAULT_STAGING_TOKEN}"
      namespace: "staging"
      skip_verify: true
```

## Usage Examples

### Basic Target Usage

```yaml
# main.yml
name: (( vault@production "secret/app:name" ))
database:
  password: (( vault@production "secret/db:password" ))
  staging_password: (( vault@staging "secret/db:password" ))
```

### With Defaults

```yaml
# Use target with fallback
api_key: (( vault@production "secret/api:key" || "default-key" ))

# Multiple fallbacks
secret: (( vault@production "secret/app:token" || vault@staging "secret/app:token" || "fallback" ))
```

### Complex Expressions

```yaml
# Concatenation with target
connection_string: (( concat "postgresql://user:" (vault@production "secret/db:password") "@" (vault@production "secret/db:host") "/mydb" ))
```

## Features

### Caching

Each target maintains its own cache to avoid conflicts:
- `secret/path` (default vault) and `production@secret/path` are cached separately
- Cache keys include target information: `{target}@{path}`

### Error Handling

Clear error messages when targets are not configured:
```
Error: vault target 'production' configuration not found (expected VAULT_PRODUCTION_ADDR and VAULT_PRODUCTION_TOKEN environment variables)
```

### Backward Compatibility

Existing vault operators without targets continue to work unchanged:
```yaml
# Still works as before
password: (( vault "secret/app:password" ))
```

## Migration Guide

### Step 1: Identify Current Usage

Find all vault operators in your templates:
```bash
grep -r "(( vault " templates/
```

### Step 2: Set Up Target Configuration

Configure environment variables for each target environment.

### Step 3: Update Templates

Replace vault operators with target-specific versions:

```yaml
# Before
password: (( vault "secret/myapp:password" ))

# After
password: (( vault@production "secret/myapp:password" ))
```

### Step 4: Test

Verify that:
1. Target-specific vault calls work correctly
2. Default vault calls still work (backward compatibility)
3. Caching works independently for each target

## Troubleshooting

### Common Issues

1. **Missing configuration**: Ensure `VAULT_{TARGET}_ADDR` and `VAULT_{TARGET}_TOKEN` are set
2. **Wrong target name**: Target names are case-sensitive and must match environment variable suffixes
3. **Cache conflicts**: Different targets use separate caches automatically

### Debug Mode

Enable debug logging to see vault target resolution:
```bash
export GRAFT_DEBUG=1
graft merge template.yml
```

Look for log messages like:
```
vault: using target 'production'
vault: using target-specific client for 'production'
vault: Cache hit for `secret/path` (target: production)
```

## Implementation Notes

- Target extraction is currently a placeholder (returns empty string)
- Full target extraction from parsed expressions will be implemented in the next iteration
- Environment variable configuration is the current approach; configuration file support is planned
- Client pooling ensures efficient reuse of vault connections per target