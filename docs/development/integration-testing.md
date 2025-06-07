# Integration Testing

This document describes how to run integration tests for graft's external operators (Vault and NATS).

## Overview

The integration tests verify that graft correctly interacts with external services like HashiCorp Vault and NATS. These tests use Docker containers to spin up real instances of the services, populate them with test data, and then run graft commands to verify the operators work correctly.

## Prerequisites

- Docker must be installed and running
- The `graft` binary must be built (`make build`)
- Ports 8200 (Vault) and 4222 (NATS) must be available
- For NATS tests: `nats` CLI tool (will be auto-installed if missing)

## Running Tests

### Run All Integration Tests

```bash
make integration
```

### Run Specific Tests

```bash
# Run only Vault tests
scripts/integration --no-nats

# Run only NATS tests  
scripts/integration --no-vault

# Run with verbose output
scripts/integration -v

# Keep containers running after tests (for debugging)
scripts/integration --keep
```

## Test Coverage

### Vault Operator Tests

1. **Basic secret retrieval** - Fetches username/password from Vault
2. **Secret with specific key** - Uses `:key` syntax to get specific fields
3. **Multiple vault references** - Tests multiple operators in one document
4. **Vault with defaults** - Tests `||` operator for fallback values
5. **Invalid path handling** - Verifies error handling for missing secrets
6. **Target-specific vault** - Tests `@target` syntax for multi-instance support
7. **Vault in complex merge** - Tests vault operators across merged documents

### NATS Operator Tests

1. **Basic KV retrieval** - Fetches values from JetStream KV store
2. **Object store YAML retrieval** - Fetches and parses YAML from object store
3. **Multiple NATS references** - Tests multiple operators in one document
4. **Invalid key handling** - Verifies error handling for missing keys
5. **Target-specific NATS** - Tests `@target` syntax for multi-instance support
6. **NATS with timeout** - Tests custom timeout configuration

## Test Data

### Vault Test Data

The integration tests create the following secrets in Vault:

- `secret/test/credentials` - Contains username, password, api_key
- `secret/test/database` - Contains host, port, name
- `secret/test/features` - Contains enabled (bool), max_users (int), tier (string)

### NATS Test Data

The integration tests create:

**KV Store (`test-bucket`)**:
- `config.host` = "api.example.com"
- `config.port` = "8080"
- `config.timeout` = "30"
- `features.auth` = "enabled"
- `features.cache` = "true"

**Object Store (`test-objects`)**:
- `config.yml` - A YAML file with version, app, and env fields

## Environment Variables

The integration tests use these environment variables:

### Vault
- `VAULT_ADDR` - Vault server address (default: http://localhost:8200)
- `VAULT_TOKEN` - Vault authentication token
- `VAULT_<TARGET>_ADDR` - Target-specific Vault address
- `VAULT_<TARGET>_TOKEN` - Target-specific Vault token

### NATS
- `NATS_URL` - NATS server URL (default: nats://localhost:4222)
- `NATS_TIMEOUT` - Connection timeout
- `NATS_<TARGET>_URL` - Target-specific NATS URL

## Output Format

The tests use TAP (Test Anything Protocol) format for output:

```
1..7 # Vault operator tests
ok 1 - Vault basic secret retrieval
ok 2 - Retrieved correct username
ok 3 - Retrieved correct password
...
```

## Troubleshooting

### Port Already in Use

If you see "Port 8200 is already in use" or similar:
- Check for running Vault/NATS instances: `docker ps`
- Stop conflicting services or use different ports

### Docker Not Found

Ensure Docker is installed and the Docker daemon is running:
```bash
docker --version
docker ps
```

### Tests Fail to Connect

If tests fail with connection errors:
1. Check Docker is running
2. Verify no firewall is blocking localhost connections
3. Run with `-v` flag for verbose output
4. Use `--keep` to inspect running containers

### Debugging Failed Tests

To debug failing tests:
1. Run with verbose mode: `scripts/integration -v`
2. Keep containers running: `scripts/integration --keep`
3. Inspect container logs: `docker logs graft-test-vault-<PID>`
4. Manually test with graft: `VAULT_ADDR=http://localhost:8200 VAULT_TOKEN=root-token-for-testing ./graft merge test.yml`