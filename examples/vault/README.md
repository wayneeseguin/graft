# Vault Operator Examples

The `vault` operator integrates with HashiCorp Vault to securely retrieve secrets. These examples demonstrate various patterns for production secret management.

## Examples in this directory:

1. **basic.yml** - Simple secret retrieval
2. **with-defaults.yml** - Using || operator for fallbacks
3. **database-secrets.yml** - Database credential management
4. **kubernetes-integration.yml** - Vault + Kubernetes patterns
5. **multi-environment.yml** - Environment-specific paths
6. **vault-try.yml** - Using vault-try for multiple paths

## Prerequisites:

```bash
# Start a dev Vault server (for testing)
vault server -dev -dev-root-token-id="root"

# In another terminal, set up environment
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN="root"

# Add some test secrets
vault kv put secret/database username=dbuser password=dbpass host=localhost
vault kv put secret/api key=my-api-key endpoint=https://api.example.com
vault kv put secret/app/production db_password=prod-pass api_key=prod-key
```

## Running the examples:

```bash
# Basic vault operations
graft merge basic.yml

# With defaults (some secrets might not exist)
graft merge with-defaults.yml

# Database secrets
graft merge database-secrets.yml

# Skip vault (see the placeholders)
GRAFT_SKIP_VAULT=1 graft merge basic.yml
```

## Important Notes:

- Requires `VAULT_ADDR` and `VAULT_TOKEN` environment variables
- Use `GRAFT_SKIP_VAULT=1` to skip vault lookups (returns "REDACTED")
- Vault paths use format: `path/to/secret:key`
- KV v2 secrets engine paths need `data/` after mount point
- See [Vault documentation](../doc/pulling-creds-from-vault.md) for setup details