#!/bin/bash

echo "=== Simple MCP Compliance Test ==="
echo

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test counter
PASSED=0
FAILED=0

# Test function
test_mcp() {
    local test_name="$1"
    local request="$2"
    local expected_pattern="$3"
    
    echo -n "Testing: $test_name... "
    
    response=$(echo "$request" | ./sap-odata-mcp-universal 2>&1)
    
    if echo "$response" | grep -q "$expected_pattern"; then
        echo -e "${GREEN}PASS${NC}"
        ((PASSED++))
    else
        echo -e "${RED}FAIL${NC}"
        echo "  Request: $request"
        echo "  Response: $response"
        echo "  Expected pattern: $expected_pattern"
        ((FAILED++))
    fi
}

# Run tests
test_mcp "JSON-RPC version" \
    '{"jsonrpc":"2.0","id":1,"method":"initialize"}' \
    '"jsonrpc":"2.0"'

test_mcp "Initialize method" \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' \
    '"protocolVersion"'

test_mcp "Server info in response" \
    '{"jsonrpc":"2.0","id":1,"method":"initialize"}' \
    '"serverInfo"'

test_mcp "Tools capability" \
    '{"jsonrpc":"2.0","id":1,"method":"initialize"}' \
    '"tools":'

test_mcp "Tools list method" \
    '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
    '"tools":\['

test_mcp "Resources list method" \
    '{"jsonrpc":"2.0","id":3,"method":"resources/list"}' \
    '"resources":\['

test_mcp "Prompts list method" \
    '{"jsonrpc":"2.0","id":4,"method":"prompts/list"}' \
    '"prompts":\['

test_mcp "Error for unknown method" \
    '{"jsonrpc":"2.0","id":5,"method":"unknown/method"}' \
    '"error"'

test_mcp "Error code -32601 for unknown method" \
    '{"jsonrpc":"2.0","id":5,"method":"unknown/method"}' \
    '"code":-32601'

test_mcp "Ping method" \
    '{"jsonrpc":"2.0","id":6,"method":"ping"}' \
    '"result":{}'

test_mcp "ID preservation (number)" \
    '{"jsonrpc":"2.0","id":123,"method":"ping"}' \
    '"id":123'

test_mcp "ID preservation (string)" \
    '{"jsonrpc":"2.0","id":"test-id","method":"ping"}' \
    '"id":"test-id"'

test_mcp "ID preservation (null)" \
    '{"jsonrpc":"2.0","id":null,"method":"ping"}' \
    '"id":null'

test_mcp "Tools have required fields" \
    '{"jsonrpc":"2.0","id":7,"method":"tools/list"}' \
    '"name"'

# Summary
echo
echo "=== Test Summary ==="
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "\n${GREEN}✅ All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}❌ Some tests failed${NC}"
    exit 1
fi