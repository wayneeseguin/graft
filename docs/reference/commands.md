# Spruce Command Reference

Spruce provides several commands for working with YAML and JSON files.

## spruce merge

The primary command for merging YAML files together.

### Synopsis

```bash
spruce merge [options] file1.yml file2.yml ... fileN.yml
```

### Description

Merges multiple YAML files together in order, with later files overriding earlier ones. Evaluates Spruce operators to produce the final output.

### Options

- `--skip-eval` - Don't evaluate operators, leave them as-is
- `--prune KEY` - Remove specified keys from final output
- `--cherry-pick KEY` - Only output specified keys
- `--fallback-append` - Use append instead of inline for array merges
- `--multi-doc` - Process multi-document YAML files
- `--go-patch` - Treat the second file as a go-patch
- `-d, --debug` - Enable debug logging
- `--trace` - Enable trace logging (very verbose)
- `-v, --version` - Show version information

### Examples

Basic merge:
```bash
spruce merge base.yml override.yml > result.yml
```

With pruning:
```bash
spruce merge --prune meta --prune tmp base.yml prod.yml
```

Cherry-picking specific keys:
```bash
spruce merge --cherry-pick jobs large-manifest.yml
```

Reading from stdin:
```bash
echo "name: test" | spruce merge - override.yml
```

## spruce diff

Shows the differences between files after merging and evaluation.

### Synopsis

```bash
spruce diff file1.yml file2.yml
```

### Description

Displays a unified diff showing what changes when file2 is merged on top of file1. Useful for reviewing changes before applying them.

### Examples

```bash
spruce diff base.yml changes.yml
```

Comparing with stdin:
```bash
cat current.yml | spruce diff - new.yml
```

## spruce json

Converts YAML to JSON format.

### Synopsis

```bash
spruce json [options] file.yml
```

### Description

Reads YAML input and outputs equivalent JSON. Can also convert the other direction with `--reverse`.

### Options

- `--reverse` - Convert JSON to YAML instead
- `--multi-doc` - Handle multi-document files

### Examples

YAML to JSON:
```bash
spruce json config.yml > config.json
```

JSON to YAML:
```bash
spruce json --reverse config.json > config.yml
```

From stdin:
```bash
kubectl get pods -o yaml | spruce json
```

## spruce fan

Spreads a source file across multiple target documents.

### Synopsis

```bash
spruce fan [options] source.yml target1.yml target2.yml ...
```

### Description

Takes a source file and merges it with each document in the target files independently, outputting a multi-document YAML stream. Each target document gets the source merged into it.

### Options

Same as `spruce merge`:
- `--skip-eval`
- `--prune KEY`
- `--cherry-pick KEY`
- `-d, --debug`
- `--trace`

### Example

source.yml:
```yaml
meta:
  environment: production
  region: us-east-1
```

targets.yml:
```yaml
---
name: web
instances: (( grab meta.environment == "production" ? 3 : 1 ))
---
name: worker  
region: (( grab meta.region ))
```

Usage:
```bash
spruce fan --prune meta source.yml targets.yml
```

Output:
```yaml
---
name: web
instances: 3
---
name: worker
region: us-east-1
```

## spruce vaultinfo

Extracts information about Vault paths used in a manifest.

### Synopsis

```bash
spruce vaultinfo [options] file.yml
```

### Description

Scans files for `(( vault ))` operators and reports on the Vault paths that will be accessed. Useful for understanding Vault dependencies and setting up appropriate policies.

### Options

- `--yaml` - Output results as YAML
- `--json` - Output results as JSON
- `-d, --debug` - Enable debug logging

### Example

```bash
spruce vaultinfo manifest.yml
```

Output:
```
Vault Paths:
  secret/db:password
  secret/api:key
  secret/certificates:cert
```

## Global Options

These options work with all commands:

### Environment Variables

- `SPRUCE_DEBUG` - Set to enable debug mode
- `REDACT` - Prevent specific keys from being shown in output
- `VAULT_ADDR` - Vault server address
- `VAULT_TOKEN` - Vault authentication token
- `VAULT_SKIP_VERIFY` - Skip TLS verification for Vault

### Debugging

Use `-d` or `--debug` for verbose output showing:
- Files being processed
- Operator evaluation order
- Reference resolution
- Error context

Use `--trace` for even more detail including:
- Full stack traces
- Internal data structures
- Step-by-step evaluation

## Exit Codes

- `0` - Success
- `1` - General error
- `2` - Usage error (invalid arguments)
- `3` - Data error (invalid YAML, missing references, etc.)

## Input/Output

### Input Sources

- Files: `spruce merge file1.yml file2.yml`
- Stdin: `spruce merge - file.yml` or `cat file.yml | spruce merge`
- Multiple files: Processed in order given

### Output Format

- Default: YAML to stdout
- JSON: Use `spruce json` command
- Files: Redirect with `> output.yml`

### Multi-Document Support

Use `--multi-doc` flag to process files containing multiple YAML documents separated by `---`.

## Common Patterns

### Development vs Production

```bash
# Development
spruce merge base.yml dev.yml > manifest.yml

# Production  
spruce merge base.yml prod.yml --prune meta > manifest.yml
```

### Pipeline Usage

```bash
# In CI/CD pipeline
spruce merge \
  base.yml \
  environments/${ENVIRONMENT}.yml \
  secrets.yml \
  --prune meta \
  --prune tmp \
  > final-manifest.yml
```

### Debugging Issues

```bash
# See what's happening
spruce merge --debug base.yml override.yml

# Check specific operator
spruce merge manifest.yml 2>&1 | grep -A5 "grab"
```

## See Also

- [Getting Started](../getting-started.md) - Basic usage guide
- [Operators Reference](../operators/README.md) - All available operators
- [Examples](../../examples/README.md) - Practical examples
- [Troubleshooting](troubleshooting.md) - Common issues and solutions