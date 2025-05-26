# Base64 Encoding/Decoding Examples

The `base64` and `base64-decode` operators are essential for handling credentials, certificates, and other sensitive data that needs encoding.

## Examples in this directory:

1. **basic.yml** - Simple encoding and decoding
2. **credentials.yml** - Encoding authentication credentials
3. **certificates.yml** - Working with certificates and keys
4. **kubernetes-secrets.yml** - Creating Kubernetes secrets
5. **with-vault.yml** - Combining with vault for secure workflows

## Running the examples:

```bash
# Basic encoding/decoding
spruce merge basic.yml

# Credentials example
USERNAME=admin PASSWORD=secret123 spruce merge credentials.yml

# Kubernetes secrets
spruce merge kubernetes-secrets.yml

# With vault integration
VAULT_ADDR=http://localhost:8200 spruce merge with-vault.yml
```

## Common Use Cases:

- Basic auth headers: `(( base64 (concat username ":" password) ))`
- Kubernetes secrets: Require base64 encoded values
- Certificate storage: PEM certificates often need base64 encoding
- API keys: Often transmitted in base64 format