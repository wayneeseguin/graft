# go-patch Integration

graft can work seamlessly with [go-patch](https://github.com/cppforlife/go-patch) operations files, allowing you to combine the power of both tools.

## Overview

As of v1.11.0, graft supports the `--go-patch` flag which enables mixing:
- Regular YAML files
- graft operators
- go-patch operations files

This integration occurs during the **Merge Phase**, which means you can even use go-patch to insert graft operators into your data structure!

## Basic Usage

Use the `--go-patch` flag when merging files:

```bash
graft merge --go-patch base.yml operations.yml final.yml
```

## How It Works

1. graft detects if a file is in go-patch format (array of operations)
2. Parses the operations defined in the file
3. Executes those operations on the root document
4. Continues with normal graft processing

This allows you to:
- Use upstream BOSH templates and ops-files
- Add custom graft-based templates on top
- Mix and match different approaches

## Complete Example

### Base YAML File

```yaml
# base.yml
key: 1

key2:
  nested:
    super_nested: 2
  other: 3

array: [4,5,6]

items:
- name: item7
- name: item8
- name: item9
```

### go-patch Operations File

```yaml
# operations.yml
- type: replace
  path: /key
  value: 10

- type: replace
  path: /new_key?
  value: 10

- type: replace
  path: /key2/nested/super_nested
  value: 10

- type: replace
  path: /key2/nested?/another_nested/super_nested
  value: 10

- type: replace
  path: /array/0
  value: 10

# You can even insert graft operators!
- type: replace
  path: /graft_array_grab?
  value: (( grab items ))
```

### Final graft Template

```yaml
# final.yml
more_stuff: is here

items:
- (( prepend ))
- add graft stuff in the beginning of the array
```

### Execution and Result

```bash
graft merge --go-patch base.yml operations.yml final.yml
```

Output:
```yaml
array:
- 10
- 5
- 6
items:
- add graft stuff in the beginning of the array
- name: item7
- name: item8
- name: item9
key: 10
key2:
  nested:
    another_nested:
      super_nested: 10
    super_nested: 10
  other: 3
more_stuff: is here
new_key: 10
graft_array_grab:
- add graft stuff in the beginning of the array
- name: item7
- name: item8
- name: item9
```

## go-patch Operations Supported

graft supports all standard go-patch operations:

### Replace
```yaml
- type: replace
  path: /key/subkey
  value: new_value
```

### Add
```yaml
- type: add
  path: /array/-
  value: new_item
```

### Remove
```yaml
- type: remove
  path: /unwanted_key
```

### Test
```yaml
- type: test
  path: /key
  value: expected_value
```

### Move
```yaml
- type: move
  path: /new_location
  from: /old_location
```

### Copy
```yaml
- type: copy
  path: /new_location
  from: /existing_location
```

## Advanced Patterns

### Injecting graft Operators

Use go-patch to insert graft operators dynamically:

```yaml
# ops-file.yml
- type: replace
  path: /database/password?
  value: (( vault "secret/db:password" ))

- type: replace
  path: /api/endpoint?
  value: (( concat "https://" (grab domain) "/api" ))
```

### Conditional Operations

Combine with graft's conditional logic:

```yaml
# base.yml
environment: production
enable_monitoring: true

# ops-file.yml
- type: replace
  path: /monitoring?
  value: (( grab enable_monitoring ? "enabled" : "disabled" ))
```

### Environment-Specific Operations

```yaml
# production-ops.yml
- type: replace
  path: /replicas
  value: 3

- type: replace
  path: /resources/limits/memory
  value: "2Gi"

# development-ops.yml  
- type: replace
  path: /replicas
  value: 1

- type: replace
  path: /resources/limits/memory
  value: "512Mi"
```

Usage:
```bash
# Production
graft merge --go-patch base.yml production-ops.yml

# Development
graft merge --go-patch base.yml development-ops.yml
```

## Best Practices

### 1. Order Matters

Place files in the correct order:
```bash
# Correct order
graft merge --go-patch base.yml ops-files.yml graft-templates.yml

# Not recommended
graft merge --go-patch graft-templates.yml base.yml ops-files.yml
```

### 2. Use Conditional Paths

Use `?` for optional paths in go-patch operations:
```yaml
- type: replace
  path: /optional_key?
  value: some_value
```

### 3. Combine with graft Features

Take advantage of graft's features after go-patch operations:
```yaml
# After go-patch creates the structure, use graft to enhance it
final_config: (( grab base_config ))
enhanced_config:
  - (( inline ))
  - (( grab final_config ))
  - additional_features: true
```

### 4. Test Operations

Use go-patch test operations to validate state:
```yaml
- type: test
  path: /environment
  value: production

- type: replace
  path: /settings
  value: (( grab production_settings ))
```

## Common Use Cases

### BOSH Deployments

```bash
# Apply upstream ops-files and custom graft templates
graft merge --go-patch \
  cf-deployment.yml \
  ops-files/use-postgres.yml \
  ops-files/scale-to-one.yml \
  custom-graft-config.yml \
  --prune meta
```

### Kubernetes Manifests

```bash
# Transform base manifests with ops-files
graft merge --go-patch \
  base-deployment.yml \
  environment-specific.yml \
  secrets-injection.yml
```

### Configuration Management

```bash
# Layer configuration from multiple sources
graft merge --go-patch \
  default-config.yml \
  platform-ops.yml \
  environment-overrides.yml \
  local-customizations.yml
```

## Troubleshooting

### Invalid go-patch Format

Ensure operations files are arrays:
```yaml
# Correct
- type: replace
  path: /key
  value: value

# Incorrect (not an array)
type: replace
path: /key
value: value
```

### Path Errors

Use proper JSON pointer syntax:
```yaml
# Correct
path: /key/subkey/0

# Incorrect
path: key.subkey[0]
```

### Mixed Processing

Remember that go-patch runs during merge phase, graft operators during eval phase:
```yaml
# This works - go-patch inserts the operator, graft evaluates it later
- type: replace
  path: /dynamic_value?
  value: (( grab some.reference ))
```

## See Also

- [go-patch GitHub Repository](https://github.com/cppforlife/go-patch)
- [graft Commands Reference](../reference/commands.md) - Details on --go-patch flag
- [Merging Concepts](../concepts/merging.md) - Understanding merge phases
- [BOSH Integration](bosh.md) - Using with BOSH deployments