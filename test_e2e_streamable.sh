#!/bin/bash

# Comprehensive E2E test for Streamable HTTP transport
set -e

echo "=============================================="
echo "  E2E Test Suite for Streamable HTTP Transport"
echo "=============================================="
echo

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run a test
run_test() {
    local test_name="$1"
    local test_cmd="$2"
    
    echo -n "Testing $test_name... "
    if eval "$test_cmd" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PASSED${NC}"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗ FAILED${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Build the binary first
echo "Building OData MCP binary..."
if go build -o sap-odata-mcp-universal ./cmd/sap-odata-mcp-universal; then
    echo -e "${GREEN}Build successful${NC}"
else
    echo -e "${RED}Build failed!${NC}"
    exit 1
fi
echo

# Test 1: Start server with streamable-http transport
echo "=== Test Suite 1: Basic Streamable HTTP ==="
./sap-odata-mcp-universal --transport streamable-http --verbose --service https://services.odata.org/V2/Northwind/Northwind.svc/ > server1.log 2>&1 &
SERVER_PID=$!
sleep 3

# Test health endpoint
run_test "Health endpoint" "curl -s http://localhost:8080/health | grep -q 'streamable-http'"

# Test MCP endpoint with initialize
run_test "MCP initialize" "curl -s -X POST http://localhost:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"2024-11-05\",\"capabilities\":{},\"clientInfo\":{\"name\":\"test\",\"version\":\"1.0\"}}}' \
  | grep -q '\"protocolVersion\":\"2024-11-05\"'"

# Test tools/list
run_test "MCP tools/list" "curl -s -X POST http://localhost:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\",\"params\":{}}' \
  | grep -q 'odata_service_info'"

# Test with SSE accept header
run_test "SSE upgrade" "curl -s -X POST http://localhost:8080/mcp \
  -H 'Content-Type: application/json' \
  -H 'Accept: text/event-stream' \
  -d '{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/list\",\"params\":{}}' \
  --max-time 2 2>/dev/null | grep -q 'filter_.*_for_NorthSvc'"

# Test legacy SSE endpoint
run_test "Legacy SSE endpoint" "curl -s -X POST http://localhost:8080/sse \
  -H 'Content-Type: application/json' \
  -H 'Accept: text/event-stream' \
  -d '{\"jsonrpc\":\"2.0\",\"id\":4,\"method\":\"initialize\",\"params\":{}}' \
  --max-time 2 2>/dev/null | grep -q 'jsonrpc'"

kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true
echo

# Test 2: Test with different port
echo "=== Test Suite 2: Custom Port Configuration ==="
./sap-odata-mcp-universal --transport streamable-http --http-addr localhost:9090 --service https://services.odata.org/V2/Northwind/Northwind.svc/ > server2.log 2>&1 &
SERVER_PID=$!
sleep 3

run_test "Custom port health" "curl -s http://localhost:9090/health | grep -q 'ok'"
run_test "Custom port MCP" "curl -s -X POST http://localhost:9090/mcp \
  -H 'Content-Type: application/json' \
  -d '{\"jsonrpc\":\"2.0\",\"id\":5,\"method\":\"initialize\",\"params\":{}}' \
  | grep -q 'sap-odata-mcp-universal'"

kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true
echo

# Test 3: Compare with legacy SSE transport
echo "=== Test Suite 3: Legacy SSE Transport Comparison ==="
./sap-odata-mcp-universal --transport http --service https://services.odata.org/V2/Northwind/Northwind.svc/ > server3.log 2>&1 &
SERVER_PID=$!
sleep 3

run_test "Legacy SSE health" "curl -s http://localhost:8080/health | grep -q 'ok'"
run_test "Legacy SSE endpoint exists" "curl -s -I http://localhost:8080/sse 2>/dev/null | head -n1 | grep -q 'HTTP'"
run_test "Legacy RPC endpoint" "curl -s -X POST http://localhost:8080/rpc \
  -H 'Content-Type: application/json' \
  -d '{\"jsonrpc\":\"2.0\",\"id\":6,\"method\":\"initialize\",\"params\":{}}' \
  | grep -q 'serverInfo'"

kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true
echo

# Test 4: Test with OData V4 service
echo "=== Test Suite 4: OData V4 Service ==="
./sap-odata-mcp-universal --transport streamable-http --service https://services.odata.org/V4/Northwind/Northwind.svc/ > server4.log 2>&1 &
SERVER_PID=$!
sleep 3

run_test "V4 service initialize" "curl -s -X POST http://localhost:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{\"jsonrpc\":\"2.0\",\"id\":7,\"method\":\"initialize\",\"params\":{}}' \
  | grep -q 'capabilities'"

run_test "V4 service tools" "curl -s -X POST http://localhost:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{\"jsonrpc\":\"2.0\",\"id\":8,\"method\":\"tools/list\",\"params\":{}}' \
  | grep -q 'filter_.*_for_NorthSvc'"

kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true
echo

# Test 5: Test actual tool invocation
echo "=== Test Suite 5: Tool Invocation ==="
./sap-odata-mcp-universal --transport streamable-http --service https://services.odata.org/V2/Northwind/Northwind.svc/ > server5.log 2>&1 &
SERVER_PID=$!
sleep 3

# First initialize
curl -s -X POST http://localhost:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":9,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' > /dev/null

# Test service info tool
run_test "Service info tool" "curl -s -X POST http://localhost:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{\"jsonrpc\":\"2.0\",\"id\":10,\"method\":\"tools/call\",\"params\":{\"name\":\"odata_service_info_for_NorthSvc\",\"arguments\":{}}}' \
  | grep -q 'service_url'"

# Test filter tool
run_test "Filter Categories tool" "curl -s -X POST http://localhost:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{\"jsonrpc\":\"2.0\",\"id\":11,\"method\":\"tools/call\",\"params\":{\"name\":\"filter_Categories_for_NorthSvc\",\"arguments\":{\"\$top\":2}}}' \
  | grep -q 'CategoryName'"

kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true
echo

# Test 6: Security features
echo "=== Test Suite 6: Security Features ==="

# Test non-localhost rejection
set +e  # Allow this to fail
./sap-odata-mcp-universal --transport streamable-http --http-addr 0.0.0.0:8080 --service https://services.odata.org/V2/Northwind/Northwind.svc/ > server6.log 2>&1 &
SERVER_PID=$!
sleep 1

if kill -0 $SERVER_PID 2>/dev/null; then
    echo -e "${RED}✗ FAILED${NC} - Server should not start on 0.0.0.0 without expert flag"
    ((TESTS_FAILED++))
    kill $SERVER_PID 2>/dev/null || true
else
    echo -e "${GREEN}✓ PASSED${NC} - Security check prevented non-localhost binding"
    ((TESTS_PASSED++))
fi
set -e
echo

# Final summary
echo "=============================================="
echo "              TEST SUMMARY"
echo "=============================================="
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}All tests passed successfully!${NC}"
    
    # Clean up test files
    rm -f server*.log
    
    exit 0
else
    echo -e "\n${RED}Some tests failed. Check the logs for details.${NC}"
    echo "Server logs are available in server*.log files"
    exit 1
fi