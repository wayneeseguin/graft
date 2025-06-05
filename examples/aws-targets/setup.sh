#!/bin/bash

# Setup script for AWS targets example
# This script demonstrates how to configure multiple AWS targets

echo "Setting up AWS targets configuration..."

# Production AWS configuration
export AWS_PRODUCTION_REGION="us-east-1"
export AWS_PRODUCTION_PROFILE="production"
export AWS_PRODUCTION_ROLE="arn:aws:iam::123456789012:role/GraftRole"
export AWS_PRODUCTION_ACCESS_KEY_ID="AKIAPRODUCTION123"
export AWS_PRODUCTION_SECRET_ACCESS_KEY="production-secret-key"
export AWS_PRODUCTION_MAX_RETRIES="5"
export AWS_PRODUCTION_HTTP_TIMEOUT="60s"
export AWS_PRODUCTION_CACHE_TTL="15m"
export AWS_PRODUCTION_AUDIT_LOGGING="true"
export AWS_PRODUCTION_ASSUME_ROLE_DURATION="2h"
export AWS_PRODUCTION_EXTERNAL_ID="external-id-prod-12345"
export AWS_PRODUCTION_SESSION_NAME="graft-production"

# Staging AWS configuration  
export AWS_STAGING_REGION="us-west-2"
export AWS_STAGING_PROFILE="staging"
export AWS_STAGING_ACCESS_KEY_ID="AKIASTAGING456"
export AWS_STAGING_SECRET_ACCESS_KEY="staging-secret-key"
export AWS_STAGING_MAX_RETRIES="3"
export AWS_STAGING_HTTP_TIMEOUT="30s"
export AWS_STAGING_CACHE_TTL="5m"
export AWS_STAGING_AUDIT_LOGGING="false"

# Development AWS configuration (LocalStack)
export AWS_DEV_REGION="us-east-1"
export AWS_DEV_ACCESS_KEY_ID="test"
export AWS_DEV_SECRET_ACCESS_KEY="test"
export AWS_DEV_ENDPOINT="http://localhost:4566"
export AWS_DEV_DISABLE_SSL="true"
export AWS_DEV_S3_FORCE_PATH_STYLE="true"
export AWS_DEV_CACHE_TTL="1m"

# Partner AWS account (cross-account access)
export AWS_PARTNER_REGION="us-east-1"
export AWS_PARTNER_ROLE="arn:aws:iam::987654321098:role/PartnerGraftRole"
export AWS_PARTNER_EXTERNAL_ID="partner-external-id-67890"
export AWS_PARTNER_SESSION_NAME="graft-partner"
export AWS_PARTNER_ASSUME_ROLE_DURATION="1h"

# Default AWS configuration (existing behavior)
export AWS_REGION="us-east-1"
export AWS_PROFILE="default"

echo "AWS targets configured:"
echo "  Production: $AWS_PRODUCTION_REGION (Account: 123456789012, Audit: enabled)"
echo "  Staging:    $AWS_STAGING_REGION (Profile: $AWS_STAGING_PROFILE, Audit: disabled)"
echo "  Development: $AWS_DEV_REGION (LocalStack: $AWS_DEV_ENDPOINT)"
echo "  Partner:    $AWS_PARTNER_REGION (Cross-account role assumption)"
echo "  Default:    $AWS_REGION (Profile: $AWS_PROFILE)"

echo ""
echo "You can now use AWS operators with targets:"
echo "  (( awsparam@production \"/app/database/password\" ))"
echo "  (( awssecret@staging \"database-credentials\" ))"
echo "  (( awsparam@dev \"/localstack/test/param\" ))"
echo "  (( awsparam \"/app/dev/password\" ))"

echo ""
echo "Cross-account examples:"
echo "  (( awsparam@partner \"/shared/api/key\" ))"
echo "  (( awssecret@partner \"shared-secret\" ))"

echo ""
echo "Secrets Manager with versioning:"
echo "  (( awssecret@production \"api-key\" ))"
echo "  (( awssecret@production \"api-key?version=1\" ))"
echo "  (( awssecret@production \"api-key?stage=AWSPENDING\" ))"

echo ""
echo "Parameter Store hierarchical access:"
echo "  (( awsparam@production \"/app/prod/database/host\" ))"
echo "  (( awsparam@production \"/app/prod/database/port\" ))"
echo "  (( awsparam@production \"/app/prod/api/rate_limit\" ))"

echo ""
echo "To test the configuration:"
echo "  graft merge example.yml"

echo ""
echo "For LocalStack testing, start LocalStack first:"
echo "  docker run --rm -p 4566:4566 localstack/localstack"