# Merging Rules

Merging in graft is designed to be intuitive. Files to merge are listed in-order on the command line. The first file serves as the base, and subsequent files are merged on top, adding new keys and replacing existing ones.

## Order of Operations

graft processes files through multiple phases to handle various operations:

### 1. Build the Root Document

The first file is loaded as the root document. Each subsequent file is merged on top, overwriting, appending, and deleting as specified.

Array operators are evaluated as each new document is merged. This allows greater control over how arrays are merged (append, prepend, insert, merge, replace).

If any `(( prune ))` operators are defined, the objects they apply to are marked for pruning but remain unmodified. No other operators are evaluated at this time.

### 2. Merge Phase

In this phase, `(( inject ))` operators are evaluated to flesh out the root document.

Since `(( inject ))` happens after array operators, you cannot use array operators when overriding arrays provided via `(( inject ))`. Currently, data will be appended to arrays.

### 3. Param Phase

The document is scanned for `(( param ))` operators. If any exist, it means a required property wasn't overridden by later files. graft will print errors for missing parameters and exit.

### 4. Eval Phase

Unless `--skip-eval` is specified, graft scans for operators, generates a dependency graph, and evaluates them in order. Each operator modifies the document. All remaining operators are evaluated at this stage.

### 5. Pruning

Any parts marked for pruning via `(( prune ))` operators or the `--prune` flag are deleted.

### 6. Cherry Picking

If `--cherry-pick` is specified, only the requested data structures are extracted from the document.

### 7. Output

Any errors from the Eval Phase or Pruning/Cherry Picking are displayed. If successful, graft formats the document as YAML and outputs it.

## Array Merging

Arrays require special handling because order matters. graft provides array operators to control merging:

- `(( append ))` - Adds data to the end of the array
- `(( prepend ))` - Inserts data at the beginning
- `(( insert ))` - Inserts data at a specific position
- `(( merge ))` - Merges based on a common key
- `(( inline ))` - Merges based on array indices
- `(( replace ))` - Replaces the entire array
- `(( delete ))` - Deletes specific elements

### Default Array Merging Behavior

Without explicit operators, arrays merge according to:

1. If all elements are objects with a `name` key, `(( merge on name ))` is implied. Elements with matching names are merged; new elements are appended.

2. If merge-by-name isn't possible, an `(( inline ))` merge is performed. With `--fallback-append`, an `(( append ))` merge is used instead.

## Key Concepts

### Map Merging

Maps (objects) are merged recursively:
- New keys are added
- Existing keys have their values replaced or merged (if both are maps)
- Keys can be deleted with `null` values or `(( delete ))` operator

### Operator Evaluation

Operators are evaluated in dependency order:
- References `(( grab ))` are resolved first
- Dependent operators wait for their dependencies
- Circular dependencies cause errors

### Environment Variables

Environment variables can be used in references:
- `$VAR` syntax in grab paths
- `(( grab $ENV.path ))` for dynamic references
- Defaults with `||` operator

## Examples

### Basic Merge

```yaml
# base.yml
name: my-app
port: 8080
features:
  auth: true
  cache: false

# override.yml
port: 9090
features:
  cache: true
  logging: true

# Result: graft merge base.yml override.yml
name: my-app
port: 9090
features:
  auth: true
  cache: true
  logging: true
```

### Array Merging

```yaml
# base.yml
servers:
  - name: web-1
    ip: 10.0.0.1
  - name: web-2
    ip: 10.0.0.2

# add-server.yml
servers:
  - (( append ))
  - name: web-3
    ip: 10.0.0.3

# Result shows web-3 appended
```

### Using Operators

```yaml
# config.yml
domain: example.com
url: (( concat "https://" domain ))
credentials:
  password: (( vault "secret/app:password" ))
```

## Best Practices

1. **Order files from general to specific** - Base configurations first, environment-specific overrides last

2. **Use explicit array operators** - Don't rely on default behavior for critical merges

3. **Validate parameters** - Use `(( param ))` for required values

4. **Organize operators** - Group related operators together for clarity

5. **Test merges** - Use `graft diff` to verify merge results

## See Also

- [Array Merging](array-merging.md) - Detailed array operator documentation
- [Operators Reference](../operators/README.md) - Complete operator list
- [Environment Variables](environment-variables.md) - Using environment variables