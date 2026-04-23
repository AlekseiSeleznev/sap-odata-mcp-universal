#!/bin/bash

# Enhanced E2E test that includes regression checks
# This ensures the binary exists BEFORE testing, catching deployment issues

set -e

echo "=============================================="
echo "  Enhanced E2E Test Suite with Regression Check"
echo "=============================================="
echo

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# STEP 1: Run regression test FIRST (don't build)
echo "=== Step 1: Pre-flight Binary Check ==="
if ./test_regression_binary_exists.sh; then
    echo -e "${GREEN}Binary regression check passed${NC}"
else
    echo -e "${RED}Binary regression check failed!${NC}"
    echo -e "${YELLOW}This is the same issue that caused the ENOENT error.${NC}"
    echo -e "${YELLOW}Build the binary first: go build -o sap-odata-mcp-universal ./cmd/sap-odata-mcp-universal${NC}"
    exit 1
fi
echo

# STEP 2: Run the full E2E test suite (without building)
echo "=== Step 2: Running E2E Tests ==="
echo "(Using existing binary, not building new one)"
echo

# Run the streamable HTTP tests
./test_streamable_http.sh

echo
echo "=============================================="
echo -e "${GREEN}All tests passed INCLUDING regression checks!${NC}"
echo "=============================================="
echo
echo "This test suite would have caught the missing binary issue"
echo "because it tests the ACTUAL binary that would be used in production,"
echo "not a freshly built one."