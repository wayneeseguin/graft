#!/bin/bash

# Test script to verify all targets are accessible
# This helps validate target configuration before running graft

set -e

echo "Target Connectivity Test"
echo "========================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track failures
FAILURES=0

# Test function
test_target() {
    local service=$1
    local target=$2
    local test_command=$3
    
    echo -n "Testing $service target '$target'... "
    
    if eval "$test_command" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ OK${NC}"
        return 0
    else
        echo -e "${RED}✗ FAILED${NC}"
        FAILURES=$((FAILURES + 1))
        return 1
    fi
}

# Test Vault targets
echo "Vault Targets:"
echo "--------------"

if [ -n "$VAULT_PRODUCTION_ADDR" ]; then
    test_target "Vault" "production" \
        "curl -s -f -H 'X-Vault-Token: $VAULT_PRODUCTION_TOKEN' \
         $VAULT_PRODUCTION_ADDR/v1/sys/health"
fi

if [ -n "$VAULT_STAGING_ADDR" ]; then
    test_target "Vault" "staging" \
        "curl -s -f -H 'X-Vault-Token: $VAULT_STAGING_TOKEN' \
         $VAULT_STAGING_ADDR/v1/sys/health"
fi

if [ -n "$VAULT_DEV_ADDR" ]; then
    test_target "Vault" "dev" \
        "curl -s -f -H 'X-Vault-Token: $VAULT_DEV_TOKEN' \
         $VAULT_DEV_ADDR/v1/sys/health"
fi

echo ""

# Test AWS targets
echo "AWS Targets:"
echo "------------"

if [ -n "$AWS_PRODUCTION_REGION" ] || [ -n "$AWS_PRODUCTION_PROFILE" ]; then
    AWS_ARGS=""
    [ -n "$AWS_PRODUCTION_REGION" ] && AWS_ARGS="$AWS_ARGS --region $AWS_PRODUCTION_REGION"
    [ -n "$AWS_PRODUCTION_PROFILE" ] && AWS_ARGS="$AWS_ARGS --profile $AWS_PRODUCTION_PROFILE"
    
    test_target "AWS" "production" \
        "aws sts get-caller-identity $AWS_ARGS"
fi

if [ -n "$AWS_STAGING_REGION" ] || [ -n "$AWS_STAGING_PROFILE" ]; then
    AWS_ARGS=""
    [ -n "$AWS_STAGING_REGION" ] && AWS_ARGS="$AWS_ARGS --region $AWS_STAGING_REGION"
    [ -n "$AWS_STAGING_PROFILE" ] && AWS_ARGS="$AWS_ARGS --profile $AWS_STAGING_PROFILE"
    
    test_target "AWS" "staging" \
        "aws sts get-caller-identity $AWS_ARGS"
fi

if [ -n "$AWS_DEV_ENDPOINT" ]; then
    test_target "AWS" "dev (LocalStack)" \
        "aws --endpoint-url $AWS_DEV_ENDPOINT s3 ls"
fi

echo ""

# Test NATS targets
echo "NATS Targets:"
echo "-------------"

# Check if nats CLI is available
if command -v nats &> /dev/null; then
    if [ -n "$NATS_PRODUCTION_URL" ]; then
        test_target "NATS" "production" \
            "nats --server=$NATS_PRODUCTION_URL server check connection"
    fi
    
    if [ -n "$NATS_STAGING_URL" ]; then
        test_target "NATS" "staging" \
            "nats --server=$NATS_STAGING_URL server check connection"
    fi
    
    if [ -n "$NATS_DEV_URL" ]; then
        test_target "NATS" "dev" \
            "nats --server=$NATS_DEV_URL server check connection"
    fi
else
    echo -e "${YELLOW}⚠ NATS CLI not found, skipping NATS tests${NC}"
fi

echo ""

# Test graft with targets
echo "Graft Target Tests:"
echo "------------------"

# Create a test template
cat > /tmp/graft-target-test.yml <<EOF
test:
  vault_prod: (( vault@production "secret/test:value" || "no-production-vault" ))
  vault_staging: (( vault@staging "secret/test:value" || "no-staging-vault" ))
  aws_param_prod: (( awsparam@production "/test/param" || "no-production-aws" ))
  aws_param_staging: (( awsparam@staging "/test/param" || "no-staging-aws" ))
  nats_kv_prod: (( nats@production "kv:test/value" || "no-production-nats" ))
  nats_kv_staging: (( nats@staging "kv:test/value" || "no-staging-nats" ))
EOF

echo -n "Testing graft with target template... "
if graft merge /tmp/graft-target-test.yml > /tmp/graft-target-test-output.yml 2>/dev/null; then
    echo -e "${GREEN}✓ OK${NC}"
    echo ""
    echo "Output:"
    cat /tmp/graft-target-test-output.yml | sed 's/^/  /'
else
    echo -e "${RED}✗ FAILED${NC}"
    FAILURES=$((FAILURES + 1))
fi

# Clean up
rm -f /tmp/graft-target-test.yml /tmp/graft-target-test-output.yml

echo ""
echo "========================"
if [ $FAILURES -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}$FAILURES test(s) failed${NC}"
    echo ""
    echo "Troubleshooting tips:"
    echo "- Check environment variables are set correctly"
    echo "- Verify network connectivity to target services"
    echo "- Ensure authentication credentials are valid"
    echo "- Run with GRAFT_DEBUG=1 for more details"
    exit 1
fi