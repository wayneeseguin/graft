# Join Operator - Map Support

The `join` operator now supports joining map entries in addition to lists and literals.

## How It Works

When joining a map, the operator:
1. Converts each entry to a `key:value` string format
2. Sorts the entries by key for consistent output
3. Joins them using the specified separator

## Syntax

```yaml
result: (( join "separator" map_reference ))
```

## Examples

### Basic Map Join
```yaml
config:
  settings:
    debug: true
    timeout: 30
    retries: 3

# Join with comma separator
settings_string: (( join ", " config.settings ))
# Result: "debug:true, retries:3, timeout:30"
```

### Multiple Maps and Mixed Types
```yaml
# Join multiple maps
all_config: (( join "; " map1 map2 ))

# Mix maps with lists and literals
combined: (( join ", " "start" some_map some_list "end" ))
```

## Use Cases

1. **Environment Variable Strings**: Convert configuration maps to environment variable formats
2. **Configuration Dumps**: Create human-readable strings from configuration maps
3. **Log Output**: Format map data for logging purposes
4. **API Parameters**: Build query strings or parameter lists from maps

## Notes

- Map entries are always sorted alphabetically by key
- The format is always `key:value` with a colon separator
- Maps can be mixed with lists and literals in a single join operation