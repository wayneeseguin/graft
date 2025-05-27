# Array Merging

Arrays in YAML can contain simple types (strings, numbers) or complex types (maps/objects). graft provides powerful operators to control how arrays are merged.

## Array Types

### Simple Arrays
Arrays of strings, numbers, or other simple types:

```yaml
ips:
  - 192.168.1.1
  - 192.168.1.2
  
tags:
  - production
  - web
  - frontend
```

### Arrays of Maps
Arrays containing objects, typically with an identifier field:

```yaml
jobs:
  - name: web
    instances: 3
    vm_type: small
  - name: db
    instances: 1
    vm_type: large
```

## Array Operators

graft provides operators to modify arrays during merging:

### Universal Operators

These work with both simple arrays and arrays of maps:

#### `(( append ))`
Adds entries to the end of the array:

```yaml
# base.yml
tags:
  - production
  - web

# override.yml
tags:
  - (( append ))
  - frontend
  - api

# Result:
tags:
  - production
  - web
  - frontend
  - api
```

#### `(( prepend ))`
Adds entries to the beginning of the array:

```yaml
# base.yml
servers:
  - web-1
  - web-2

# override.yml
servers:
  - (( prepend ))
  - lb-1
  - lb-2

# Result:
servers:
  - lb-1
  - lb-2
  - web-1
  - web-2
```

#### `(( replace ))`
Completely replaces the existing array:

```yaml
# base.yml
environments:
  - dev
  - staging
  - prod

# override.yml
environments:
  - (( replace ))
  - development
  - production

# Result:
environments:
  - development
  - production
```

#### `(( inline ))`
Merges arrays by position (index):

```yaml
# base.yml
settings:
  - timeout: 30
    retries: 3
  - timeout: 60
    retries: 5

# override.yml
settings:
  - (( inline ))
  - retries: 5     # Merges with first element
  - timeout: 120   # Merges with second element

# Result:
settings:
  - timeout: 30
    retries: 5
  - timeout: 120
    retries: 5
```

### Map-Based Operators

These operators work with arrays of maps using identifiers:

#### `(( merge ))`
Merges array entries by matching identifier (default: `name`):

```yaml
# base.yml
jobs:
  - name: web
    instances: 1
    vm_type: small
  - name: db
    instances: 1
    vm_type: small

# override.yml
jobs:
  - (( merge ))  # Optional - this is the default
  - name: web
    instances: 3
    vm_type: large
  - name: worker
    instances: 2
    vm_type: medium

# Result:
jobs:
  - name: web
    instances: 3
    vm_type: large
  - name: db
    instances: 1
    vm_type: small
  - name: worker
    instances: 2
    vm_type: medium
```

#### `(( merge on <key> ))`
Merges using a custom identifier key:

```yaml
# base.yml
components:
  - id: auth-service
    version: 1.0
    enabled: true
  - id: user-service
    version: 1.0
    enabled: true

# override.yml
components:
  - (( merge on id ))
  - id: auth-service
    version: 2.0
  - id: cache-service
    version: 1.0
    enabled: true

# Result:
components:
  - id: auth-service
    version: 2.0
    enabled: true
  - id: user-service
    version: 1.0
    enabled: true
  - id: cache-service
    version: 1.0
    enabled: true
```

#### `(( insert ))`
Inserts entries at specific positions:

**For arrays of maps:**
```yaml
# base.yml
pipeline:
  - name: build
    script: make
  - name: test
    script: make test
  - name: deploy
    script: make deploy

# Insert after a named entry
pipeline:
  - (( insert after "build" ))
  - name: lint
    script: make lint

# Result: lint inserted between build and test
```

**For simple arrays:**
```yaml
# base.yml
sequence:
  - first
  - second
  - third

# Insert by index
sequence:
  - (( insert after 0 ))
  - one-and-half

# Result: [first, one-and-half, second, third]
```

#### `(( delete ))`
Removes specific entries:

**For arrays of maps:**
```yaml
# base.yml
services:
  - name: web
    port: 80
  - name: api
    port: 8080
  - name: metrics
    port: 9090

# override.yml
services:
  - (( delete "metrics" ))

# Result: metrics service removed
```

**For simple arrays:**
```yaml
# base.yml
ports:
  - 80
  - 443
  - 8080
  - 9090

# Delete by index
ports:
  - (( delete 2 ))  # Removes 8080 (0-indexed)

# Result: [80, 443, 9090]
```

## Advanced Usage

### Multiple Operators
You can combine operators in a single merge:

```yaml
items:
  - (( prepend ))
  - new-first
  - new-second
  
  - (( append ))
  - new-last
  
  - (( insert after "existing-item" ))
  - new-middle
  
  - (( delete "unwanted-item" ))
```

### Default Behavior

Without explicit operators:
1. If all array elements have a `name` field, graft uses `(( merge on name ))`
2. Otherwise, graft uses `(( inline ))` merge
3. With `--fallback-append` flag, graft uses `(( append ))` instead of `(( inline ))`

### Identifier Keys

Common identifier keys (in order of precedence):
- `name` (default)
- `key`
- `id`

## Best Practices

1. **Use explicit operators** - Don't rely on default behavior for clarity
2. **Choose appropriate identifiers** - Use meaningful keys for merging
3. **Be careful with inline** - Position-based merging is fragile
4. **Document complex merges** - Add comments explaining the merge strategy
5. **Test array merges** - Verify results with `graft diff`

## Common Patterns

### Adding to BOSH Job Specs
```yaml
# Add environment variables to existing job
jobs:
  - name: web
    properties:
      env:
        - (( append ))
        - FEATURE_FLAG=true
        - DEBUG_MODE=false
```

### Overriding Specific Array Elements
```yaml
# Update specific security group rules
security_groups:
  - (( merge on name ))
  - name: web
    rules:
      - (( replace ))  # Replace all rules
      - protocol: tcp
        ports: 443
        source: 0.0.0.0/0
```

### Conditional Array Building
```yaml
base_services:
  - name: web
  - name: api

services:
  - (( inline ))
  - (( grab base_services.[0] ))
  - (( grab base_services.[1] ))
  - (( append ))
  - name: (( concat "worker-" meta.env ))
```

## See Also

- [Operators Reference](../operators/array-operations.md) - Detailed operator documentation
- [Merging Rules](merging.md) - General merging behavior
- [Examples](../../examples/README.md) - Practical examples