# BOSH Integration

graft was originally designed for BOSH deployment manifests and provides excellent integration with BOSH workflows and Cloud Config.

## Overview

graft helps with BOSH in several key areas:
- **Manifest Generation**: Template BOSH deployment manifests with dynamic values
- **Cloud Config Integration**: Work with BOSH Cloud Config for network and VM configuration
- **Static IP Calculation**: Generate static IPs for instance groups
- **Multi-Environment Deployments**: Manage different environments with shared templates

## Basic BOSH Workflow

### 1. Template Structure

Organize your BOSH templates:

```
deployment/
├── base.yml              # Base deployment manifest
├── environments/
│   ├── dev.yml          # Development overrides
│   ├── staging.yml      # Staging overrides  
│   └── production.yml   # Production overrides
├── ops-files/           # go-patch operations
│   ├── scale-up.yml
│   └── enable-https.yml
└── secrets.yml          # Secret values (for dev)
```

### 2. Base Manifest Template

```yaml
# base.yml
meta:
  environment: (( grab $ENVIRONMENT || "development" ))
  deployment_name: (( concat "myapp-" meta.environment ))

name: (( grab meta.deployment_name ))

releases:
- name: myapp
  version: (( grab $RELEASE_VERSION || "latest" ))

stemcells:
- alias: default
  os: ubuntu-xenial
  version: (( grab $STEMCELL_VERSION || "latest" ))

instance_groups:
- name: web
  instances: (( grab meta.environment == "production" ? 3 : 1 ))
  azs: [z1, z2, z3]
  vm_type: (( grab meta.environment == "production" ? "large" : "small" ))
  stemcell: default
  networks:
  - name: default
  jobs:
  - name: webapp
    release: myapp
    properties:
      domain: (( grab meta.domain ))
      database:
        host: (( grab database.host ))
        port: (( grab database.port || 5432 ))
```

### 3. Environment-Specific Configuration

```yaml
# environments/production.yml
meta:
  domain: myapp.example.com

database:
  host: prod-db.example.com
  
# High availability in production
instance_groups:
- name: web
  instances: 5
  vm_type: large
```

```yaml
# environments/development.yml
meta:
  domain: myapp.dev.example.com

database:
  host: localhost

# Single instance for development
instance_groups:
- name: web
  instances: 1
  vm_type: small
```

### 4. Generate and Deploy

```bash
# Development
ENVIRONMENT=development graft merge \
  base.yml \
  environments/development.yml \
  secrets.yml \
  --prune meta \
  > manifest.yml

bosh deploy manifest.yml

# Production  
ENVIRONMENT=production graft merge \
  base.yml \
  environments/production.yml \
  --prune meta \
  > manifest.yml

bosh deploy manifest.yml
```

## Cloud Config Integration

### The Challenge

With BOSH Cloud Config, network definitions moved out of deployment manifests. However, the `(( static_ips ))` operator still needs network information to calculate IP addresses.

### The Solution

Download Cloud Config and merge it temporarily:

```yaml
# base.yml
instance_groups:
- name: database
  instances: 1
  networks:
  - name: private
    static_ips: (( static_ips(0) ))

# Clean up cloud-config data after processing
azs:           (( prune ))
compilation:   (( prune ))  
disk_types:    (( prune ))
networks:      (( prune ))
vm_extensions: (( prune ))
vm_types:      (( prune ))
```

### Workflow with Cloud Config

```bash
# 1. Download current cloud config
bosh cloud-config > cloud-config.yml

# 2. Merge with cloud config first, then environment config
graft merge \
  base.yml \
  cloud-config.yml \
  environments/production.yml \
  --prune meta \
  > manifest.yml

# 3. Deploy
bosh deploy manifest.yml

# 4. Clean up
rm cloud-config.yml manifest.yml
```

### Automated Script

```bash
#!/bin/bash
# deploy.sh

ENVIRONMENT=${1:-development}

echo "Deploying to $ENVIRONMENT..."

# Download cloud config
bosh cloud-config > cloud-config.yml

# Generate manifest
graft merge \
  base.yml \
  cloud-config.yml \
  "environments/${ENVIRONMENT}.yml" \
  --prune meta \
  > manifest.yml

# Deploy
bosh deploy manifest.yml

# Cleanup
rm cloud-config.yml manifest.yml

echo "Deployment complete!"
```

## Static IP Management

### Basic Static IPs

```yaml
instance_groups:
- name: database
  instances: 3
  networks:
  - name: private
    # Allocate first 3 available IPs
    static_ips: (( static_ips(0, 1, 2) ))

- name: load-balancer  
  instances: 2
  networks:
  - name: public
    # Start from IP index 10
    static_ips: (( static_ips(10, 11) ))
```

### Multi-AZ Static IPs

```yaml
# Network definition in Cloud Config
networks:
- name: private
  type: manual
  subnets:
  - range: 10.0.1.0/24
    gateway: 10.0.1.1
    az: z1
    static: [10.0.1.10-10.0.1.50]
  - range: 10.0.2.0/24  
    gateway: 10.0.2.1
    az: z2
    static: [10.0.2.10-10.0.2.50]

# Instance group spanning AZs
instance_groups:
- name: database
  instances: 2
  azs: [z1, z2]
  networks:
  - name: private
    static_ips: (( static_ips(0, 1) ))
    
# Results in:
# - Instance 0 in z1: 10.0.1.10
# - Instance 1 in z2: 10.0.2.10
```

### Dynamic Static IP Allocation

```yaml
meta:
  base_ip_index: (( grab $IP_START_INDEX || 0 ))

instance_groups:
- name: web
  instances: (( grab meta.web_instances ))
  networks:
  - name: private
    static_ips: (( static_ips(meta.base_ip_index, meta.base_ip_index + meta.web_instances - 1) ))
```

## Advanced BOSH Patterns

### Conditional Instance Groups

```yaml
meta:
  enable_monitoring: (( grab $ENABLE_MONITORING || false ))
  enable_logging: (( grab $ENABLE_LOGGING || false ))

instance_groups:
- name: web
  instances: 3
  # ... basic config

# Conditional monitoring
- (( meta.enable_monitoring ? "present" : "prune" ))
- name: monitoring
  instances: 1
  jobs:
  - name: prometheus
    properties:
      targets: (( grab instance_groups.web.networks.0.static_ips ))

# Conditional logging
- (( meta.enable_logging ? "present" : "prune" ))
- name: logging
  instances: 1
  jobs:
  - name: fluentd
```

### Shared Properties

```yaml
meta:
  shared_properties:
    log_level: info
    timeout: 30
    retries: 3

instance_groups:
- name: web
  jobs:
  - name: webapp
    properties:
      # Merge shared properties
      - (( inline ))
      - (( grab meta.shared_properties ))
      - port: 8080
        workers: 4

- name: worker
  jobs:
  - name: background-worker
    properties:
      # Reuse shared properties
      - (( inline ))
      - (( grab meta.shared_properties ))
      - concurrency: 10
```

### Release Version Management

```yaml
meta:
  versions:
    myapp: (( grab $MYAPP_VERSION || "1.2.3" ))
    postgres: (( grab $POSTGRES_VERSION || "42" ))
    nginx: (( grab $NGINX_VERSION || "1.18.0" ))

releases:
- name: myapp
  version: (( grab meta.versions.myapp ))
- name: postgres  
  version: (( grab meta.versions.postgres ))
- name: nginx
  version: (( grab meta.versions.nginx ))

# Validate versions in CI
validation:
  required_versions:
    myapp: (( grab meta.versions.myapp != "latest" ? "ok" : "error" ))
```

## Integration with Operations Files

Combine graft templates with BOSH ops-files:

```bash
# Use graft to generate base, then apply ops-files
graft merge base.yml environments/prod.yml --prune meta > base-manifest.yml

bosh deploy base-manifest.yml \
  -o ops-files/scale-web.yml \
  -o ops-files/enable-ssl.yml \
  -v web_instances=5
```

Or use graft's go-patch integration:

```bash
graft merge --go-patch \
  base.yml \
  environments/prod.yml \
  ops-files/scale-web.yml \
  ops-files/enable-ssl.yml \
  --prune meta
```

## CI/CD Integration

### GitLab CI Example

```yaml
# .gitlab-ci.yml
deploy:
  stage: deploy
  script:
    - export ENVIRONMENT=${CI_ENVIRONMENT_NAME}
    - export RELEASE_VERSION=${CI_COMMIT_TAG:-latest}
    
    # Generate manifest
    - bosh cloud-config > cloud-config.yml
    - graft merge base.yml cloud-config.yml environments/${ENVIRONMENT}.yml --prune meta > manifest.yml
    
    # Deploy
    - bosh deploy manifest.yml --non-interactive
    
  environment:
    name: ${CI_ENVIRONMENT_NAME}
  only:
    - main
    - tags
```

### GitHub Actions Example

```yaml
# .github/workflows/deploy.yml
name: Deploy to BOSH

on:
  push:
    branches: [main]
    tags: ['v*']

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    
    - name: Setup BOSH CLI
      run: |
        wget https://github.com/cloudfoundry/bosh-cli/releases/download/v6.4.1/bosh-cli-6.4.1-linux-amd64
        sudo mv bosh-cli-* /usr/local/bin/bosh
        chmod +x /usr/local/bin/bosh
    
    - name: Setup graft
      run: |
        wget https://github.com/wayneeseguin/graft/releases/download/v2.1.0/graft-linux-amd64
        sudo mv graft-* /usr/local/bin/graft
        chmod +x /usr/local/bin/graft
    
    - name: Generate and Deploy
      env:
        BOSH_ENVIRONMENT: ${{ secrets.BOSH_ENVIRONMENT }}
        BOSH_CLIENT: ${{ secrets.BOSH_CLIENT }}
        BOSH_CLIENT_SECRET: ${{ secrets.BOSH_CLIENT_SECRET }}
        ENVIRONMENT: production
      run: |
        bosh cloud-config > cloud-config.yml
        graft merge base.yml cloud-config.yml environments/production.yml --prune meta > manifest.yml
        bosh deploy manifest.yml --non-interactive
```

## Best Practices

### 1. Environment Separation

Keep environment-specific values in separate files:
```
environments/
├── shared.yml       # Common to all environments
├── development.yml  # Dev-specific
├── staging.yml      # Staging-specific
└── production.yml   # Prod-specific
```

### 2. Secret Management

Use appropriate secret management for each environment:

```yaml
# Development - local secrets file
database:
  password: dev-password

# Production - Vault integration
database:
  password: (( vault "secret/prod/db:password" ))

# Or CredHub integration
database:
  password: ((!prod-db-password))
```

### 3. Validation

Add validation to catch configuration errors:

```yaml
meta:
  environment: (( grab $ENVIRONMENT ))

# Validate required environment
validation:
  environment_check: (( 
    grab meta.environment == "development" || 
    grab meta.environment == "staging" || 
    grab meta.environment == "production" 
    ? "ok" : error "Invalid environment: " meta.environment 
  ))
```

### 4. Documentation

Document your templates:

```yaml
# Required Environment Variables:
# - ENVIRONMENT: Target environment (development|staging|production)
# - RELEASE_VERSION: Release version to deploy (default: latest)
# - STEMCELL_VERSION: Stemcell version (default: latest)
#
# Optional:
# - WEB_INSTANCES: Number of web instances (default: 1 for dev, 3 for prod)
# - ENABLE_MONITORING: Enable monitoring instance group (default: false)

meta:
  environment: (( grab $ENVIRONMENT ))
  # ... rest of template
```

## Troubleshooting

### Static IPs Not Working

Ensure Cloud Config is merged before other files:
```bash
# Correct order
graft merge base.yml cloud-config.yml environment.yml

# Wrong order  
graft merge cloud-config.yml base.yml environment.yml
```

### AZ Mismatch

Verify instance group AZs match network AZs:
```yaml
# In cloud-config
networks:
- name: private
  subnets:
  - az: z1
  - az: z2

# In manifest - should match
instance_groups:
- name: web
  azs: [z1, z2]  # Must be subset of network AZs
```

### Properties Not Merging

Use proper merge operators for complex properties:
```yaml
# Wrong - replaces entire properties
properties:
  database:
    host: new-host

# Right - merges with existing properties  
properties:
  database:
    host: new-host
```

## See Also

- [BOSH Documentation](https://bosh.io/docs/)
- [Static IPs Operator](../operators/utility-metadata.md#static_ips) - Detailed static IP documentation
- [go-patch Integration](go-patch.md) - Using ops-files with graft
- [CredHub Integration](credhub.md) - Managing secrets with CredHub
- [Examples Directory](../../examples/) - BOSH-specific examples