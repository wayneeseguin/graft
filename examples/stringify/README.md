# Stringify Operator Examples

The `stringify` operator converts data structures (maps, arrays, etc.) into properly formatted YAML strings. This is particularly useful for embedding configuration as strings in other systems like Kubernetes ConfigMaps.

## Files in this directory:

1. **basic.yml** - Basic stringification of different data types
2. **kubernetes-configmap.yml** - Creating Kubernetes ConfigMaps
3. **nested-configs.yml** - Stringifying complex nested structures
4. **multi-document.yml** - Working with multiple configurations

## Key Features:

- Converts any data structure to a YAML-formatted string
- Preserves structure and formatting
- Handles nested maps and arrays
- Useful for embedding YAML within YAML
- Maintains proper indentation automatically

## Common Use Cases:

```yaml
# Embed configuration as a string
config_string: (( stringify config_data ))

# Create Kubernetes ConfigMap
data:
  application.yml: (( stringify app_config ))

# Store structured data as text
metadata: (( stringify complex_structure ))
```

## Running Examples:

```bash
# Basic stringification
graft merge basic.yml

# Generate Kubernetes manifests
graft merge kubernetes-configmap.yml > configmap.yaml
kubectl apply -f configmap.yaml

# Complex nested configs
graft merge nested-configs.yml
```

## Notes:

- The output is always valid YAML
- Large structures may produce long strings
- Useful for generating configuration files dynamically
- Works well with the `file` operator for comparison