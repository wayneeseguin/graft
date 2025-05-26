# Vault-Try Operator Examples

The `vault-try` operator attempts to retrieve secrets from multiple Vault paths in order, returning the first successful result. This is particularly useful for migration scenarios, multi-tenant configurations, and fallback strategies.

## Files in this directory:

1. **basic.yml** - Basic vault-try usage with fallbacks
2. **migration.yml** - Path migration scenarios
3. **multi-tenant.yml** - Tenant-specific secret retrieval
4. **regional.yml** - Region-based secret paths
5. **versioning.yml** - Version-specific secret management

## Key Differences from `vault`:

- `vault-try` requires at least 2 arguments (paths + default)
- Tries each path in order until one succeeds
- Last argument is always the default value
- More forgiving with malformed paths
- Useful for graceful transitions

## Usage Pattern:

```yaml
# Basic usage
password: (( vault-try "path1" "path2" "default-value" ))

# With more paths
config: (( vault-try "new/path" "old/path" "legacy/path" "fallback" ))

# Dynamic paths
secret: (( vault-try 
  (concat "v2/" env "/secret") 
  (concat "v1/" env "/secret") 
  "default-secret" 
))
```

## Setting up test Vault paths:

```bash
# Create secrets in different paths for testing
vault kv put secret/v1/myapp/db password=oldpass
vault kv put secret/v2/myapp/db password=newpass
vault kv put secret/prod/myapp key=prodkey
vault kv put secret/dev/myapp key=devkey
```

## Running Examples:

```bash
# Test basic fallbacks
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=your-token
spruce merge basic.yml

# Test migration scenarios
spruce merge migration.yml

# Test with environment variables
ENV=prod spruce merge regional.yml
```