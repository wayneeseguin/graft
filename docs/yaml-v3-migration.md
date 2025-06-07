# YAML v3 Migration Guide

This document describes the migration from `gopkg.in/yaml.v2` to `gopkg.in/yaml.v3` in the graft codebase.

## Overview

Graft has been migrated to use yaml.v3 for improved YAML 1.2 compliance, better JSON compatibility, and enhanced error handling.

## Key Changes

### 1. Import Updates

All core graft files now use yaml.v3:
- `pkg/graft/document.go`
- `pkg/graft/engine.go`
- `pkg/graft/expr_evaluation.go`
- `pkg/graft/operators/operator_helpers.go`

### 2. Map Type Handling

yaml.v3 returns `map[string]interface{}` instead of `map[interface{}]interface{}`. A conversion helper function has been added to maintain backward compatibility:

```go
// convertStringMapToInterfaceMap in engine.go
// Recursively converts map[string]interface{} to map[interface{}]interface{}
```

### 3. YAML 1.1 Boolean Compatibility

To maintain backward compatibility with existing configurations, YAML 1.1 boolean strings are automatically converted:
- `yes`, `Yes`, `YES`, `on`, `On`, `ON` → `true`
- `no`, `No`, `NO`, `off`, `Off`, `OFF` → `false`

This ensures that existing YAML files using these values continue to work as expected.

## Behavioral Changes

### Boolean Values
- YAML 1.2 strictly uses `true`/`false` for booleans
- YAML 1.1 values (`yes/no`, `on/off`) are treated as strings by yaml.v3 unless explicitly converted
- The conversion helper ensures these work as booleans in graft

### JSON Compatibility
- yaml.v3 output can be directly marshaled to JSON without additional conversion
- This improves interoperability with JSON-based tools

## Testing

Comprehensive tests have been added:
- `yaml_migration_baseline_test.go` - Documents yaml.v2 behavior
- `yaml_v2_v3_compatibility_test.go` - Compares v2 vs v3 behavior

## Known Issues

1. Some test files use `geofffranks/yaml` which may show different output formatting
2. The yaml.v2 dependency remains for compatibility testing purposes

## Migration Benefits

1. **Better Standards Compliance**: yaml.v3 follows YAML 1.2 specification more closely
2. **Improved JSON Compatibility**: Direct JSON marshaling without conversion
3. **Enhanced Error Messages**: Better error reporting for malformed YAML
4. **Performance**: More efficient map handling with string keys

## Backwards Compatibility

The migration maintains full backward compatibility through:
- Automatic YAML 1.1 boolean conversion
- Map type conversion for internal consistency
- Preservation of existing merge and evaluation behavior