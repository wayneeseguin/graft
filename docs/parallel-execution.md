# Parallel Execution in Spruce

Spruce now supports parallel execution of operators, providing significant performance improvements for large configuration files while maintaining data integrity and backward compatibility.

## Table of Contents
- [Overview](#overview)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [How It Works](#how-it-works)
- [Safe Operators](#safe-operators)
- [Monitoring](#monitoring)
- [Performance](#performance)
- [Troubleshooting](#troubleshooting)
- [API Reference](#api-reference)

## Overview

Parallel execution allows Spruce to evaluate multiple independent operators simultaneously, reducing processing time for large YAML files. This feature is particularly beneficial for:

- Large configuration files with many operators
- Files with network-based operations (vault, AWS)
- Templates with complex dependency chains
- CI/CD pipelines processing multiple manifests

Key features:
- **Automatic dependency analysis** ensures correct execution order
- **Safe by default** with opt-in enablement
- **Zero API changes** - fully backward compatible
- **Production monitoring** with metrics and observability
- **Configurable concurrency** based on workload

## Quick Start

### Basic Usage

Enable parallel execution with a single environment variable:

```bash
# Enable parallel execution
export SPRUCE_PARALLEL=true

# Run spruce as normal
spruce merge base.yml overlay.yml
```

### Recommended Configuration

For optimal performance:

```bash
# Enable parallel execution with 8 workers
export SPRUCE_PARALLEL=true
export SPRUCE_PARALLEL_WORKERS=8

# Enable metrics for monitoring
export SPRUCE_METRICS=true

# Run spruce
spruce merge large-manifest.yml
```

### Verify It's Working

With debug mode enabled, you'll see parallel execution information:

```bash
export DEBUG=1
export SPRUCE_PARALLEL=true

spruce merge base.yml overlay.yml
# Output will include:
# DEBUG> Running 45 operations: 30 in parallel, 15 sequential
# DEBUG> Parallel execution completed in 1.2s (3.5x speedup)
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPRUCE_PARALLEL` | `false` | Enable/disable parallel execution |
| `SPRUCE_PARALLEL_WORKERS` | CPU count | Maximum number of worker threads |
| `SPRUCE_PARALLEL_MIN_OPS` | `10` | Minimum operations required for parallel execution |
| `SPRUCE_PARALLEL_STRATEGY` | `conservative` | Execution strategy: `conservative`, `aggressive`, or `adaptive` |
| `SPRUCE_METRICS` | `false` | Enable metrics collection |
| `SPRUCE_METRICS_PORT` | `9090` | Port for metrics HTTP server |

### Execution Strategies

#### Conservative (Default)
Only operators on the safe list are executed in parallel. This ensures maximum safety.

```bash
export SPRUCE_PARALLEL_STRATEGY=conservative
```

#### Aggressive
All operators except those explicitly blacklisted are executed in parallel. Use with caution.

```bash
export SPRUCE_PARALLEL_STRATEGY=aggressive
```

#### Adaptive (Future)
Dynamically adjusts parallelization based on runtime behavior and metrics.

### Feature Flags

For fine-grained control, use the programmatic API:

```go
import "github.com/geofffranks/spruce"

// Get current features
features := spruce.GetFeatures()

// Enable parallel execution
features.SetParallelEnabled(true)

// Configure workers
features.ParallelMaxWorkers = 16

// Enable metrics
features.EnableMetrics = true
```

## How It Works

### Dependency Analysis

Spruce automatically analyzes operator dependencies to determine which operations can run in parallel:

```yaml
# These can run in parallel (no dependencies)
name: (( grab meta.name ))
version: (( grab meta.version ))

# This must wait for 'name' to complete
full_name: (( concat name "-app" ))

# These form a dependency chain
base_port: 8080
api_port: (( grab base_port ))
admin_port: (( calc base_port + 1000 ))
```

### Execution Phases

1. **Analysis Phase**
   - Parse all operators
   - Build dependency graph
   - Identify parallelizable groups

2. **Execution Phase**
   - Execute independent operations in parallel
   - Wait for dependencies before proceeding
   - Maintain execution order where required

3. **Result Phase**
   - Collect results from all workers
   - Apply changes to the document
   - Report any errors

### Thread Safety

Spruce uses several mechanisms to ensure thread safety:

- **SafeTree**: RWMutex-protected tree for concurrent access
- **COWTree**: Copy-on-Write tree for lock-free reads
- **Atomic Operations**: For metrics and state management
- **Worker Pools**: Controlled concurrency with bounded workers

## Safe Operators

The following operators are verified safe for parallel execution:

### Read-Only Operators
- `grab` - Retrieves values from the tree
- `empty` - Checks if values are empty
- `keys` - Extracts map keys

### Transformation Operators
- `base64` - Base64 encoding
- `base64-decode` - Base64 decoding
- `concat` - String concatenation
- `join` - Array joining
- `sort` - Array sorting
- `stringify` - Convert to string
- `cartesian-product` - Cartesian product of arrays

### I/O Operators (Benefit Most)
- `file` - File reading
- `load` - Load external files
- `vault` - Vault secret retrieval
- `awsparam` - AWS Parameter Store
- `awssecret` - AWS Secrets Manager

### Unsafe Operators
These operators modify global state and run sequentially:
- `static_ips` - IP allocation
- `inject` - Tree injection
- `merge` - Deep merging
- `prune` - Tree pruning

## Monitoring

### Metrics Server

When metrics are enabled, Spruce starts an HTTP server with monitoring endpoints:

```bash
# Enable metrics
export SPRUCE_METRICS=true
export SPRUCE_METRICS_PORT=9090

# Run spruce
spruce merge manifest.yml

# In another terminal:
# Prometheus metrics
curl http://localhost:9090/metrics

# JSON metrics
curl http://localhost:9090/metrics/json

# Health check
curl http://localhost:9090/health
```

### Available Metrics

#### Prometheus Format
```prometheus
# Operations executed
spruce_operations_total{type="parallel"} 150
spruce_operations_total{type="sequential"} 50
spruce_operations_total{type="failed"} 2

# Operation duration percentiles
spruce_operation_duration_seconds{quantile="0.5"} 0.010
spruce_operation_duration_seconds{quantile="0.95"} 0.050
spruce_operation_duration_seconds{quantile="0.99"} 0.100

# Concurrency levels
spruce_concurrency{type="current"} 4
spruce_concurrency{type="max"} 8

# Performance metrics
spruce_speedup 3.5
```

#### JSON Format
```json
{
  "operations": {
    "total": 200,
    "parallel": 150,
    "sequential": 50,
    "failed": 2
  },
  "duration": {
    "total": "2.5s",
    "parallel": "500ms",
    "sequential": "2s"
  },
  "concurrency": {
    "current": 4,
    "max": 8
  },
  "performance": {
    "speedup": 3.5,
    "parallel_ratio": 75.0,
    "failure_rate": 1.0
  }
}
```

### Grafana Integration

Import the Spruce dashboard for real-time monitoring:

1. Add Prometheus data source pointing to `http://localhost:9090`
2. Import dashboard from `examples/grafana-dashboard.json`
3. Monitor execution performance in real-time

## Performance

### Expected Speedups

Performance improvements depend on your workload:

| Workload Type | Operations | Expected Speedup |
|---------------|------------|------------------|
| Small files | <10 ops | No benefit (sequential) |
| Medium files | 10-100 ops | 2-3x |
| Large files | 100-500 ops | 3-5x |
| Very large files | >500 ops | 4-8x |
| Network-heavy | Any size | 5-10x |

### Benchmarking

Run the included benchmarks to test on your hardware:

```bash
# Run all benchmarks
go test -bench=. -benchtime=10s

# Run specific benchmark
go test -bench=BenchmarkParallelExecution -benchtime=10s

# Compare sequential vs parallel
go test -bench='BenchmarkParallelExecution/(Sequential|Parallel)' -benchtime=10s
```

### Optimization Tips

1. **Increase workers for I/O-bound workloads**
   ```bash
   export SPRUCE_PARALLEL_WORKERS=16
   ```

2. **Use aggressive strategy for trusted inputs**
   ```bash
   export SPRUCE_PARALLEL_STRATEGY=aggressive
   ```

3. **Enable batching for similar operations**
   ```bash
   export SPRUCE_BATCH=true
   ```

4. **Monitor and tune based on metrics**
   ```bash
   export SPRUCE_METRICS=true
   curl http://localhost:9090/metrics/json
   ```

## Troubleshooting

### Parallel Execution Not Working

1. **Check if enabled**
   ```bash
   echo $SPRUCE_PARALLEL  # Should be "true"
   ```

2. **Verify minimum operations threshold**
   ```bash
   # Lower threshold for testing
   export SPRUCE_PARALLEL_MIN_OPS=2
   ```

3. **Enable debug logging**
   ```bash
   export DEBUG=1
   spruce merge test.yml
   # Look for: "Parallel execution disabled or too few ops"
   ```

### Performance Issues

1. **Too many workers**
   - Reduce workers to match CPU cores
   ```bash
   export SPRUCE_PARALLEL_WORKERS=4
   ```

2. **Lock contention**
   - Check metrics for high lock wait times
   ```bash
   curl http://localhost:9090/metrics/json | jq .percentiles.lock_wait
   ```

3. **Unsuitable workload**
   - Small files may not benefit
   - Check parallel ratio in metrics

### Race Conditions

If you suspect race conditions:

1. **Run with race detector**
   ```bash
   go test -race ./...
   ```

2. **Use conservative strategy**
   ```bash
   export SPRUCE_PARALLEL_STRATEGY=conservative
   ```

3. **Report issues**
   - File a GitHub issue with reproduction steps
   - Include operator types and error messages

### Debugging

Enable comprehensive debugging:

```bash
# Enable all debugging
export DEBUG=1
export SPRUCE_PARALLEL=true
export SPRUCE_METRICS=true
export SPRUCE_PARALLEL_WORKERS=2  # Low for easier debugging

# Run with verbose output
spruce merge -v manifest.yml

# Check metrics after run
curl http://localhost:9090/metrics/json | jq
```

## API Reference

### Go API

```go
package main

import "github.com/geofffranks/spruce"

// Enable parallel execution globally
spruce.EnableParallelExecution(true)

// Configure parallel execution
config := &spruce.ParallelEvaluatorConfig{
    Enabled:           true,
    MaxWorkers:        8,
    MinOpsForParallel: 5,
    Strategy:          "aggressive",
}
spruce.SetParallelConfig(config)

// Use with evaluator
ev := &spruce.Evaluator{Tree: data}
err := ev.RunPhaseParallel(spruce.EvalPhase)

// Get execution statistics
stats := spruce.ParallelExecutionStats()
fmt.Printf("Speedup: %.2fx\n", stats["speedup"])
```

### Feature Flags API

```go
// Get feature flags
features := spruce.GetFeatures()

// Configure features
features.SetParallelEnabled(true)
features.ParallelMaxWorkers = 16
features.EnableMetrics = true

// Update at runtime
features.Update(map[string]interface{}{
    "parallel_execution": true,
    "parallel_max_workers": 8,
    "enable_metrics": true,
})
```

### Metrics API

```go
// Get metrics collector
metrics := spruce.GetParallelMetrics()

// Get current snapshot
snapshot := metrics.GetSnapshot()
fmt.Printf("Operations: %d (%.1f%% parallel)\n", 
    snapshot.OpsTotal, 
    snapshot.ParallelRatio)

// Start metrics server
server := spruce.NewMetricsServer(metrics, 9090)
server.Start()
defer server.Stop()
```

## Best Practices

1. **Start Conservative**: Begin with default settings and increase parallelism gradually
2. **Monitor Performance**: Use metrics to verify improvements
3. **Test Thoroughly**: Ensure your YAML files work correctly with parallel execution
4. **Profile First**: Identify bottlenecks before optimizing
5. **Document Dependencies**: Make operator dependencies explicit in complex templates

## Migration Guide

### From Sequential to Parallel

1. **Test without changes**
   ```bash
   # Your existing command works unchanged
   spruce merge base.yml overlay.yml
   ```

2. **Enable parallel execution**
   ```bash
   export SPRUCE_PARALLEL=true
   spruce merge base.yml overlay.yml
   ```

3. **Verify output is identical**
   ```bash
   # Compare outputs
   diff output-sequential.yml output-parallel.yml
   ```

4. **Tune for performance**
   ```bash
   export SPRUCE_PARALLEL_WORKERS=8
   export SPRUCE_METRICS=true
   ```

### Gradual Rollout

For production environments:

1. **Stage 1**: Enable for development
   ```bash
   # dev environment only
   export SPRUCE_PARALLEL=true
   ```

2. **Stage 2**: Enable for CI/CD
   ```yaml
   # .gitlab-ci.yml
   variables:
     SPRUCE_PARALLEL: "true"
     SPRUCE_PARALLEL_WORKERS: "4"
   ```

3. **Stage 3**: Production with monitoring
   ```bash
   export SPRUCE_PARALLEL=true
   export SPRUCE_METRICS=true
   export SPRUCE_PARALLEL_STRATEGY=conservative
   ```

## Examples

### Example 1: Large Cloud Foundry Manifest

```bash
# Sequential: ~5 seconds
time spruce merge cf-base.yml cf-prod.yml > manifest.yml

# Parallel: ~1.5 seconds (3.3x speedup)
export SPRUCE_PARALLEL=true
export SPRUCE_PARALLEL_WORKERS=8
time spruce merge cf-base.yml cf-prod.yml > manifest.yml
```

### Example 2: Vault-Heavy Configuration

```bash
# Many vault lookups benefit greatly from parallelization
export SPRUCE_PARALLEL=true
export SPRUCE_PARALLEL_WORKERS=16  # Increase for I/O

spruce merge secrets.yml app.yml
# 10x speedup for 50+ vault operations
```

### Example 3: CI/CD Pipeline

```yaml
# .github/workflows/deploy.yml
- name: Generate manifest
  env:
    SPRUCE_PARALLEL: "true"
    SPRUCE_PARALLEL_WORKERS: "4"
    SPRUCE_METRICS: "true"
  run: |
    spruce merge base.yml ${{ matrix.env }}.yml > manifest.yml
    
    # Print performance metrics
    curl -s http://localhost:9090/metrics/json | \
      jq '.performance.speedup'
```

## Conclusion

Parallel execution in Spruce provides significant performance improvements while maintaining safety and backward compatibility. Start with conservative settings, monitor performance, and gradually increase parallelism based on your workload characteristics.

For questions or issues, please visit the [GitHub repository](https://github.com/geofffranks/spruce/issues).