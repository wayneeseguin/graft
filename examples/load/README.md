# Load Operator Examples

The `load` operator parses external YAML or JSON files and includes their structured data. Unlike `file` which returns raw strings, `load` parses the content.

## Examples in this directory:

1. **basic.yml** - Simple file loading
2. **modular-config.yml** - Building modular configurations
3. **environment-specific.yml** - Loading environment-specific configs
4. **with-arrays.yml** - Loading and merging arrays
5. **dynamic-loading.yml** - Loading files based on variables

## Directory Structure:
```
load/
├── configs/
│   ├── database.yml
│   ├── redis.yml
│   ├── features.json
│   └── environments/
│       ├── dev.yml
│       ├── staging.yml
│       └── prod.yml
└── data/
    ├── users.yml
    └── permissions.json
```

## Running the examples:

```bash
# Basic loading
spruce merge basic.yml

# Modular configuration
spruce merge modular-config.yml

# Environment-specific (set ENV variable)
ENV=production spruce merge environment-specific.yml

# See what files are being loaded
spruce merge --debug dynamic-loading.yml
```

## Key Differences from `file`:
- `load` parses YAML/JSON → returns data structures
- `file` reads raw content → returns strings
- Use `load` for configuration data
- Use `file` for scripts, certificates, templates

## Important Notes:
- Loaded files are NOT evaluated for Spruce operators
- Use relative paths from the main YAML file
- Supports both YAML and JSON files