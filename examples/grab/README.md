# Grab Operator Examples

The `grab` operator is the most fundamental operator in Graft. It retrieves values from other parts of your YAML structure.

## Examples in this directory:

1. **basic.yml** - Simple value grabbing
2. **nested.yml** - Grabbing from nested structures
3. **with-env-vars.yml** - Using environment variables in paths
4. **with-defaults.yml** - Using || for fallback values
5. **lists-and-maps.yml** - Grabbing complex data structures

## Running the examples:

```bash
# Basic example
graft merge basic.yml

# With environment variables
ENV=production graft merge with-env-vars.yml

# See evaluation order
graft merge --debug nested.yml
```