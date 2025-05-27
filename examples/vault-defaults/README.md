# Vault Defaults Example

This example demonstrates how to use default values with the vault operator in Graft.

## Files

- `base.yml` - Base configuration with vault lookups and defaults
- `defaults.yml` - Default values that can be referenced
- `result-with-vault.yml` - Expected output when vault is available
- `result-with-defaults.yml` - Expected output when vault is not available

## Usage

### When Vault is Available

```bash
# Assuming vault has the secrets set up
graft merge defaults.yml base.yml > result.yml
```

### When Vault is Not Available (using defaults)

```bash
# With VAULT_SKIP_VERIFY=1, defaults will be used
VAULT_SKIP_VERIFY=1 graft merge defaults.yml base.yml > result.yml
```

## Features Demonstrated

1. **Literal Defaults**: Simple string defaults
2. **Reference Defaults**: Using values from elsewhere in the YAML
3. **Environment Variable Defaults**: Using `$ENV_VAR` as fallback
4. **Nil Defaults**: Explicitly setting nil/null
5. **vault-try Operator**: Trying multiple paths before defaulting
6. **Workarounds**: Using intermediate variables for complex expressions