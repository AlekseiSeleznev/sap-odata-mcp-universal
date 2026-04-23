#!/bin/bash

echo "=== Testing Error Response Formats ==="
echo

# Test 1: Invalid method (should return error)
echo "1. Testing invalid method error:"
echo '{"jsonrpc":"2.0","id":1,"method":"invalid/method"}' | ./sap-odata-mcp-universal 2>&1 | python3 -m json.tool

echo
echo "2. Testing with null ID:"
echo '{"jsonrpc":"2.0","id":null,"method":"invalid/method"}' | ./sap-odata-mcp-universal 2>&1 | python3 -m json.tool

echo
echo "3. Testing with string ID:"
echo '{"jsonrpc":"2.0","id":"test-123","method":"invalid/method"}' | ./sap-odata-mcp-universal 2>&1 | python3 -m json.tool

echo
echo "4. Testing parse error (malformed params):"
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":"not-an-object"}' | ./sap-odata-mcp-universal 2>&1 | python3 -m json.tool

echo
echo "5. Testing missing jsonrpc:"
echo '{"id":1,"method":"ping"}' | ./sap-odata-mcp-universal 2>&1 | python3 -m json.tool