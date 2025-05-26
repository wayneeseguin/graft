# Defer Operator Examples

The `defer` operator prevents evaluation of expressions, allowing you to create templates that contain Spruce operators for later evaluation.

## Examples in this directory:

1. **basic.yml** - Simple defer usage
2. **template-generation.yml** - Creating Spruce templates
3. **multi-stage.yml** - Multi-stage processing example
4. **complex-templates.yml** - Advanced template patterns

## Running the examples:

```bash
# Basic defer
spruce merge basic.yml

# Template generation
spruce merge template-generation.yml > template.yml
# Then later: spruce merge template.yml values.yml

# Multi-stage processing
./run-multi-stage.sh
```

## Key Concepts:

- `(( defer ... ))` outputs the expression as-is without evaluation
- Perfect for generating Spruce templates programmatically
- Useful for multi-stage configuration pipelines
- Can defer any valid Spruce expression