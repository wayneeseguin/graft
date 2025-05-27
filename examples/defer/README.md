# Defer Operator Examples

The `defer` operator prevents evaluation of expressions, allowing you to create templates that contain Graft operators for later evaluation.

## Examples in this directory:

1. **basic.yml** - Simple defer usage
2. **template-generation.yml** - Creating Graft templates
3. **multi-stage.yml** - Multi-stage processing example
4. **complex-templates.yml** - Advanced template patterns

## Running the examples:

```bash
# Basic defer
graft merge basic.yml

# Template generation
graft merge template-generation.yml > template.yml
# Then later: graft merge template.yml values.yml

# Multi-stage processing
./run-multi-stage.sh
```

## Key Concepts:

- `(( defer ... ))` outputs the expression as-is without evaluation
- Perfect for generating Graft templates programmatically
- Useful for multi-stage configuration pipelines
- Can defer any valid Graft expression