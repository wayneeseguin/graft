#!/bin/bash

# Setup script for NATS targets example
# This script demonstrates how to configure multiple NATS targets

echo "Setting up NATS targets configuration..."

# Production NATS configuration
export NATS_PRODUCTION_URL="nats://nats.prod.example.com:4222"
export NATS_PRODUCTION_TIMEOUT="10s"
export NATS_PRODUCTION_RETRIES="5"
export NATS_PRODUCTION_TLS="true"
export NATS_PRODUCTION_CERT_FILE="/etc/ssl/certs/nats-prod.crt"
export NATS_PRODUCTION_KEY_FILE="/etc/ssl/private/nats-prod.key"
export NATS_PRODUCTION_CA_FILE="/etc/ssl/certs/nats-ca.crt"
export NATS_PRODUCTION_CACHE_TTL="15m"
export NATS_PRODUCTION_AUDIT_LOGGING="true"
export NATS_PRODUCTION_STREAMING_THRESHOLD="52428800"  # 50MB

# Staging NATS configuration  
export NATS_STAGING_URL="nats://nats.staging.example.com:4222"
export NATS_STAGING_TIMEOUT="5s"
export NATS_STAGING_RETRIES="3"
export NATS_STAGING_TLS="false"
export NATS_STAGING_CACHE_TTL="5m"
export NATS_STAGING_AUDIT_LOGGING="false"
export NATS_STAGING_STREAMING_THRESHOLD="10485760"  # 10MB

# Default NATS configuration (existing behavior)
export NATS_URL="nats://nats.dev.example.com:4222"

echo "NATS targets configured:"
echo "  Production: $NATS_PRODUCTION_URL (TLS: enabled, Audit: enabled)"
echo "  Staging:    $NATS_STAGING_URL (TLS: disabled, Audit: disabled)" 
echo "  Default:    $NATS_URL"

echo ""
echo "You can now use NATS operators with targets:"
echo "  (( nats@production \"kv:config/app\" ))"
echo "  (( nats@staging \"kv:config/app\" ))"
echo "  (( nats \"kv:config/app\" ))"

echo ""
echo "Object store examples:"
echo "  (( nats@production \"obj:certificates/server.crt\" ))"
echo "  (( nats@staging \"obj:configs/large-config.yml\" ))"

echo ""
echo "To test the configuration:"
echo "  graft merge example.yml"