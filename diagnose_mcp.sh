#!/bin/bash

echo "=== MCP Server Diagnostic Test ==="
echo

# Test 1: Basic server response
echo "1. Testing server initialization..."
INIT_RESPONSE=$(echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./sap-odata-mcp-universal 2>&1)
echo "Response: $INIT_RESPONSE"
echo

# Test 2: Check tools list
echo "2. Testing tools/list..."
TOOLS_RESPONSE=$(echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | ./sap-odata-mcp-universal 2>&1)
echo "Tools count: $(echo "$TOOLS_RESPONSE" | grep -o '"name"' | wc -l)"
echo

# Test 3: Test calling service info tool
echo "3. Testing service info tool..."
INFO_RESPONSE=$(echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"odata_service_info_for_Z001","arguments":{}}}' | ./sap-odata-mcp-universal 2>&1)
echo "Service info call status: $(echo "$INFO_RESPONSE" | grep -q '"error"' && echo "ERROR" || echo "SUCCESS")"
echo

# Test 4: Check for null values in responses
echo "4. Checking for null values in responses..."
echo "$INIT_RESPONSE" | grep -o 'null' | wc -l | xargs echo "Null count in init response:"
echo "$TOOLS_RESPONSE" | grep -o 'null' | wc -l | xargs echo "Null count in tools response:"
echo

# Test 5: Validate JSON structure
echo "5. Validating JSON structure..."
echo "$INIT_RESPONSE" | python3 -m json.tool > /dev/null 2>&1 && echo "Init response: Valid JSON" || echo "Init response: Invalid JSON"
echo "$TOOLS_RESPONSE" | python3 -m json.tool > /dev/null 2>&1 && echo "Tools response: Valid JSON" || echo "Tools response: Invalid JSON"
echo

# Test 6: Check for empty or missing fields
echo "6. Checking for potential issues..."
echo "$TOOLS_RESPONSE" | grep -E '"":|:\s*,|:\s*}' && echo "Found empty values" || echo "No empty values found"

echo
echo "=== Diagnostic complete ==="