# Base64-Decode Operator Examples

The `base64-decode` operator decodes base64-encoded strings back to their original form. It's useful for handling encoded secrets, certificates, and data that needs to be decoded at merge time.

## Files in this directory:

1. **basic.yml** - Basic base64 decoding examples
2. **secrets.yml** - Decoding secrets and credentials
3. **certificates.yml** - Decoding certificates and keys
4. **data-processing.yml** - Processing encoded data

## Key Features:

- Decodes base64-encoded strings
- Works with literals and references
- Handles multi-line encoded content
- Useful for secrets management
- Preserves original formatting

## Usage Pattern:

```yaml
# Decode a literal
decoded: (( base64-decode "SGVsbG8gV29ybGQ=" ))  # "Hello World"

# Decode from reference
encoded_data: "SGVsbG8gV29ybGQ="
decoded_data: (( base64-decode encoded_data ))

# Decode and use
password: (( base64-decode encoded_password ))
```

## Running Examples:

```bash
# Basic decoding
spruce merge basic.yml

# Decode secrets
spruce merge secrets.yml

# Process certificates
spruce merge certificates.yml
```

## Common Use Cases:

- Decoding secrets from external sources
- Processing base64-encoded configuration
- Handling encoded certificates
- Decoding data from APIs
- Converting encoded environment variables