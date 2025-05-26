# Negate Operator Examples

The `negate` operator returns the logical NOT of a boolean value. It's the legacy operator for boolean negation (the `!` operator is preferred in the enhanced parser).

## Files in this directory:

1. **basic.yml** - Basic boolean negation
2. **feature-flags.yml** - Using negate with feature flags
3. **inverse-config.yml** - Creating inverse configuration values
4. **with-conditionals.yml** - Combining with conditional logic

## Key Differences from `!` operator:

- `negate` is a function-style operator: `(( negate value ))`
- `!` is a prefix operator: `(( ! value ))`
- Both produce the same result, but `!` is more concise
- `negate` works in both legacy and enhanced parsers
- `!` only works in the enhanced parser

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
```

## Running Examples:

```bash
# Test basic negation
spruce merge basic.yml

# Test with parameters
spruce merge feature-flags.yml <(echo "debug: true")

# Compare with ! operator (enhanced parser)
spruce merge with-conditionals.yml
```