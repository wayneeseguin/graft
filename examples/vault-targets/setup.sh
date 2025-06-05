#!/bin/bash

# Setup script for vault targets example
# This script demonstrates how to configure multiple vault targets

echo "Setting up vault targets configuration..."

# Production vault configuration
export VAULT_PRODUCTION_ADDR="https://vault.prod.example.com"
export VAULT_PRODUCTION_TOKEN="prod-token-12345"
export VAULT_PRODUCTION_NAMESPACE="production"

# Staging vault configuration  
export VAULT_STAGING_ADDR="https://vault.staging.example.com"
export VAULT_STAGING_TOKEN="staging-token-67890"
export VAULT_STAGING_NAMESPACE="staging"

# Default vault configuration (existing behavior)
export VAULT_ADDR="https://vault.dev.example.com"
export VAULT_TOKEN="dev-token-abcde"

echo "Vault targets configured:"
echo "  Production: $VAULT_PRODUCTION_ADDR"
echo "  Staging:    $VAULT_STAGING_ADDR" 
echo "  Default:    $VAULT_ADDR"

echo ""
echo "You can now use vault operators with targets:"
echo "  (( vault@production \"secret/path:key\" ))"
echo "  (( vault@staging \"secret/path:key\" ))"
echo "  (( vault \"secret/path:key\" ))"

echo ""
echo "To test the configuration:"
echo "  graft merge example.yml"