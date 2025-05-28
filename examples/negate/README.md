# Negate Operator Examples

The `negate` operator returns the logical NOT of a boolean value. For more concise syntax, the `!` operator is also available.

## Files in this directory:

1. **basic.yml** - Basic boolean negation
2. **feature-flags.yml** - Using negate with feature flags
3. **inverse-config.yml** - Creating inverse configuration values
4. **with-conditionals.yml** - Combining with conditional logic

## Key Differences between `negate` and `!`:

- `negate` is a function-style operator: `(( negate value ))`
- `!` is a prefix operator: `(( ! value ))`
- Both produce the same result, but `!` is more concise
- Both operators are fully supported

## Usage Patterns:

```yaml
# Basic negation
debug_mode: true
production_mode: (( negate debug_mode ))  # false

# With references
config:
  disabled: false
  enabled: (( negate config.disabled ))  # true

# Feature toggling
features:
  legacy_ui: true
  modern_ui: (( negate features.legacy_ui ))  # false

# Using ! operator (more concise)
settings:
  verbose: true
  quiet: (( ! settings.verbose ))  # false
```

## Running Examples:

```bash
# Test basic negation
graft merge basic.yml

# Test with parameters
graft merge feature-flags.yml <(echo "debug: true")

# Test conditional logic
graft merge with-conditionals.yml
```