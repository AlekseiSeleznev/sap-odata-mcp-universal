#!/bin/bash

echo "=== MCP Edge Case Testing ==="
echo "Testing various edge cases that might cause client errors"
echo

# Test missing JSON-RPC field
echo "1. Testing missing JSON-RPC field:"
echo '{"id":1,"method":"initialize"}' | ./sap-odata-mcp-universal 2>&1 | head -1
echo

# Test wrong JSON-RPC version
echo "2. Testing wrong JSON-RPC version:"
echo '{"jsonrpc":"1.0","id":1,"method":"initialize"}' | ./sap-odata-mcp-universal 2>&1 | head -1
echo

# Test with extra fields in params
echo "3. Testing extra fields in params:"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"extra":"field","protocolVersion":"2024-11-05"}}' | ./sap-odata-mcp-universal 2>&1 | head -1
echo

# Test tools/call with various argument types
echo "4. Testing tools/call with null arguments:"
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"odata_service_info_for_Z001","arguments":null}}' | ./sap-odata-mcp-universal 2>&1 | head -1
echo

# Test tools/call with empty arguments object
echo "5. Testing tools/call with empty arguments:"
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"odata_service_info_for_Z001","arguments":{}}}' | ./sap-odata-mcp-universal 2>&1 | head -1
echo

# Test tools/call without arguments key
echo "6. Testing tools/call without arguments key:"
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"odata_service_info_for_Z001"}}' | ./sap-odata-mcp-universal 2>&1 | head -1
echo

# Test notification (should not respond)
echo "7. Testing notification (should be silent):"
echo '{"jsonrpc":"2.0","method":"initialized"}' | ./sap-odata-mcp-universal 2>&1
echo

# Test batch request (not supported in MCP)
echo "8. Testing batch request (should fail):"
echo '[{"jsonrpc":"2.0","id":1,"method":"ping"},{"jsonrpc":"2.0","id":2,"method":"ping"}]' | ./sap-odata-mcp-universal 2>&1 | head -1
echo

# Test malformed JSON
echo "9. Testing malformed JSON:"
echo '{"jsonrpc":"2.0","id":1,"method":"ping"' | ./sap-odata-mcp-universal 2>&1 | head -1
echo

# Test very large ID
echo "10. Testing large ID number:"
echo '{"jsonrpc":"2.0","id":999999999999999999999999999999,"method":"ping"}' | ./sap-odata-mcp-universal 2>&1 | head -1
echo