# File Operator Examples

The `file` operator reads the contents of a file and inserts it as a string value. Perfect for including scripts, certificates, or configuration snippets.

## Examples in this directory:

1. **basic.yml** - Simple file inclusion
2. **dynamic-paths.yml** - Building file paths dynamically
3. **with-base-path.yml** - Using relative paths with a base directory
4. **scripts/** - Sample files to be included
5. **configs/** - Sample configuration files

## Setup:

First, create the sample files:
```bash
./setup.sh
```

## Running the examples:

```bash
# Basic file inclusion
graft merge basic.yml

# Dynamic file paths
ENV=production graft merge dynamic-paths.yml

# With base path
graft merge with-base-path.yml --file-base-path ./configs
```

## Important Notes:

- File contents are included as-is (string value)
- Use `(( load ))` if you need to parse YAML/JSON files
- Paths can be absolute or relative
- The `--file-base-path` flag changes the base directory for relative paths