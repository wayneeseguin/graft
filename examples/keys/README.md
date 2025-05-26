# Keys Operator Examples

The `keys` operator extracts all the keys from a map/hash and returns them as a sorted array. Very useful for dynamic configurations.

## Examples in this directory:

1. **basic.yml** - Simple key extraction
2. **dynamic-iteration.yml** - Using keys for dynamic processing
3. **with-nested-grab.yml** - Using keys with nested expressions
4. **validation.yml** - Using keys for configuration validation

## Running the examples:

```bash
# Basic example
spruce merge basic.yml

# Dynamic iteration
spruce merge dynamic-iteration.yml

# With nested expressions
spruce merge with-nested-grab.yml
```

## Common Use Cases:

- Getting all environment names from a config
- Listing all available services
- Building dynamic menus or lists
- Validating configuration completeness