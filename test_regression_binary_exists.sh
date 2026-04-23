#!/bin/bash

# Regression test to ensure the binary exists and is functional
# This test should be run BEFORE any cleanup and AFTER builds

set -e

echo "========================================"
echo "  Regression Test: Binary Availability"
echo "========================================"
echo

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

BINARY_PATH="./sap-odata-mcp-universal"
TESTS_PASSED=0
TESTS_FAILED=0

# Test 1: Binary exists
echo -n "1. Checking if binary exists... "
if [ -f "$BINARY_PATH" ]; then
    echo -e "${GREEN}✓ PASSED${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAILED${NC} - Binary not found at $BINARY_PATH"
    echo -e "${YELLOW}Run 'go build -o sap-odata-mcp-universal ./cmd/sap-odata-mcp-universal' to build${NC}"
    ((TESTS_FAILED++))
    exit 1
fi

# Test 2: Binary is executable
echo -n "2. Checking if binary is executable... "
if [ -x "$BINARY_PATH" ]; then
    echo -e "${GREEN}✓ PASSED${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAILED${NC} - Binary is not executable"
    echo -e "${YELLOW}Run 'chmod +x $BINARY_PATH' to fix${NC}"
    ((TESTS_FAILED++))
    exit 1
fi

# Test 3: Binary responds to stdio
echo -n "3. Testing stdio transport... "
RESPONSE=$(echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | \
    $BINARY_PATH --service https://services.odata.org/V2/Northwind/Northwind.svc/ 2>/dev/null | \
    head -1 || echo "ERROR")

if echo "$RESPONSE" | grep -q '"result"'; then
    echo -e "${GREEN}✓ PASSED${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAILED${NC} - Binary does not respond correctly to stdio"
    ((TESTS_FAILED++))
fi

# Test 4: Binary file size (sanity check)
echo -n "4. Checking binary size... "
SIZE=$(stat -f%z "$BINARY_PATH" 2>/dev/null || stat -c%s "$BINARY_PATH" 2>/dev/null || echo "0")
MIN_SIZE=$((5 * 1024 * 1024))  # 5MB minimum

if [ "$SIZE" -gt "$MIN_SIZE" ]; then
    SIZE_MB=$((SIZE / 1024 / 1024))
    echo -e "${GREEN}✓ PASSED${NC} (${SIZE_MB}MB)"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAILED${NC} - Binary seems too small (${SIZE} bytes)"
    ((TESTS_FAILED++))
fi

# Test 5: Test integration with Claude Desktop config path
echo -n "5. Checking Claude Desktop integration... "
CLAUDE_CONFIG_PATH="/Users/alice/Library/Application Support/Claude/claude_desktop_config.json"
if [ -f "$CLAUDE_CONFIG_PATH" ]; then
    # Check if our binary path is referenced in config
    if grep -q "sap-odata-mcp-universal/sap-odata-mcp-universal" "$CLAUDE_CONFIG_PATH"; then
        # Extract the path from config
        CONFIG_BINARY=$(grep -o '"/Users/[^"]*sap-odata-mcp-universal"' "$CLAUDE_CONFIG_PATH" | tr -d '"' | head -1)
        if [ -n "$CONFIG_BINARY" ]; then
            if [ -f "$CONFIG_BINARY" ]; then
                echo -e "${GREEN}✓ PASSED${NC} - Binary exists at configured path"
                ((TESTS_PASSED++))
            else
                echo -e "${RED}✗ FAILED${NC} - Binary missing at configured path: $CONFIG_BINARY"
                echo -e "${YELLOW}The Claude config points to a non-existent binary!${NC}"
                ((TESTS_FAILED++))
            fi
        else
            echo -e "${YELLOW}⚠ WARNING${NC} - Could not extract binary path from config"
            ((TESTS_PASSED++))
        fi
    else
        echo -e "${YELLOW}⚠ SKIPPED${NC} - Not configured in Claude Desktop"
        ((TESTS_PASSED++))
    fi
else
    echo -e "${YELLOW}⚠ SKIPPED${NC} - Claude Desktop config not found"
    ((TESTS_PASSED++))
fi

echo
echo "========================================"
echo "           TEST SUMMARY"
echo "========================================"
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}All regression tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}Regression tests failed!${NC}"
    echo -e "${YELLOW}This would have prevented the ENOENT error in production.${NC}"
    exit 1
fi