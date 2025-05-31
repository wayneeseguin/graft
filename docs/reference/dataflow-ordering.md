# Dataflow Ordering

As of version X.X.X, graft supports configurable ordering of operations in the dataflow output. This can be useful for debugging and understanding the evaluation order of operators.

## CLI Flag

The `--dataflow-order` flag controls how operations are ordered in the internal dataflow representation:

```bash
graft merge --dataflow-order <mode> file1.yml file2.yml
```

### Available Modes

- `alphabetical` (default) - Operations are sorted alphabetically by their path in the YAML structure. This provides deterministic output across runs.
- `insertion` - Operations are listed in the order they were discovered during tree traversal, which generally matches the order they appear in the source files.

## Example

Given this YAML file:

```yaml
domain: (( grab meta.domain || "default" ))
env:    (( grab meta.env || "dev" ))
app:    (( grab meta.app || "myapp" ))
meta:
  env: production
```

With `--dataflow-order alphabetical` (default), the internal processing order would be:
1. `app`
2. `domain`  
3. `env`

With `--dataflow-order insertion`, the internal processing order would be:
1. `domain`
2. `env`
3. `app`

## Important Notes

- The dataflow order **does not affect the final output** - it only changes the internal processing order
- Both modes produce identical results
- The default `alphabetical` mode ensures consistent behavior across different systems
- The `insertion` mode can be helpful when debugging to see operations in a more "natural" order

## API Usage

When using the graft API, you can set the dataflow order through the engine options:

```go
engine, err := graft.NewEngine(
    graft.WithDataflowOrder("insertion"),
)
```

Or directly on an evaluator:

```go
evaluator := &graft.Evaluator{
    Tree:          tree,
    DataflowOrder: "insertion",
}
```