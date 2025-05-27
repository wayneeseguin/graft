# Cartesian Product Operator Examples

The `cartesian-product` operator generates all possible combinations from multiple arrays. It's useful for creating test matrices, generating configuration combinations, and exploring parameter spaces.

## Files in this directory:

1. **basic.yml** - Basic cartesian product operations
2. **test-matrix.yml** - Creating test matrices for CI/CD
3. **configurations.yml** - Generating configuration combinations
4. **parameter-space.yml** - Exploring parameter combinations

## Key Features:

- Takes multiple arrays as input
- Returns array of arrays with all combinations
- Order matters: first array varies slowest
- Works with any data types
- Handles empty arrays (results in empty output)

## Usage Pattern:

```yaml
result: (( cartesian-product array1 array2 array3 ))
```

## Example:
```yaml
colors: [red, blue]
sizes: [S, M, L]
combinations: (( cartesian-product colors sizes ))
# Result: [[red, S], [red, M], [red, L], [blue, S], [blue, M], [blue, L]]
```

## Running Examples:

```bash
# Basic combinations
graft merge basic.yml

# Test matrix generation
graft merge test-matrix.yml

# Configuration combinations
graft merge configurations.yml
```

## Common Use Cases:

- Test matrix generation for CI/CD
- Feature flag combinations
- Multi-dimensional configuration
- Parameter space exploration
- Product variant generation