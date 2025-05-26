# Concat Operator Examples

The `concat` operator joins multiple strings together. It's one of the most frequently used operators for building dynamic strings.

## Examples in this directory:

1. **basic.yml** - Simple string concatenation
2. **with-grabs.yml** - Concatenating grabbed values
3. **building-urls.yml** - Building URLs and connection strings
4. **nested-concat.yml** - Using concat within other operators
5. **multi-line.yml** - Building multi-line strings

## Running the examples:

```bash
# Basic concatenation
spruce merge basic.yml

# URL building example
ENV=production spruce merge building-urls.yml

# See how nested concat works
spruce merge --debug nested-concat.yml
```

## Common Patterns:

- Building URLs: `(( concat "https://" (grab host) ":" (grab port) ))`
- Creating identifiers: `(( concat (grab prefix) "-" (grab name) "-" (grab suffix) ))`
- Joining paths: `(( concat (grab base_path) "/" (grab filename) ))`