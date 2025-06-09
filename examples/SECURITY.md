# Security Notes for Examples

This directory contains examples for learning and demonstration purposes.
When using these examples in production environments,
please consider the following security best practices:

## General Security Considerations

1. **Sensitive Data Storage**
   - Never store passwords, API keys, or secrets in ConfigMaps
   - Use Kubernetes Secrets or external secret management solutions
   - Consider using tools like Sealed Secrets, External Secrets Operator, or Vault

2. **Container Security**
   - Always use specific image tags, never `:latest`
   - Run containers as non-root users
   - Use read-only root filesystems where possible
   - Drop all capabilities and add only what's needed
   - Set proper security contexts

3. **Resource Management**
   - Always set resource requests and limits
   - Use PodDisruptionBudgets for availability
   - Implement proper health checks

## Example-Specific Notes

### stringify/kubernetes-configmap.yml
This example has been modified to remove sensitive data from ConfigMaps.
In the original example, sensitive values like `JWT_SECRET` and database credentials were stored in ConfigMaps.
These should always be stored in Secrets.

### vault/kubernetes-integration.yml
This is a simplified example. For production use, see `kubernetes-integration-secure.yml` which includes:
- Proper security contexts
- Resource limits
- Network policies
- Non-root user execution
- Read-only root filesystem
- Specific image tags

## Trivy Scan Results

To check for security issues in the examples, run:
```bash
make trivy
```

Some examples may show warnings for demonstration purposes, but production deployments should aim for zero security findings.
