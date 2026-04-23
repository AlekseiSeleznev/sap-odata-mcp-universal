#!/bin/bash

echo "Testing AI Foundry Protocol Compatibility"
echo "=========================================="
echo ""

# Test 1: Default protocol version (2024-11-05)
echo "Test 1: Default protocol version"
echo "---------------------------------"
./build/sap-odata-mcp-universal --service "https://services.odata.org/V2/Northwind/Northwind.svc/" <<EOF | grep -A 10 '"protocolVersion"'
{
  "jsonrpc": "2.0",
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "test-client",
      "version": "1.0"
    }
  },
  "id": 1
}
EOF

echo ""
echo "Test 2: AI Foundry protocol version (2025-06-18)"
echo "-------------------------------------------------"
./build/sap-odata-mcp-universal --service "https://services.odata.org/V2/Northwind/Northwind.svc/" --protocol-version "2025-06-18" <<EOF | grep -A 10 '"protocolVersion"'
{
  "jsonrpc": "2.0",
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-06-18",
    "capabilities": {},
    "clientInfo": {
      "name": "openai-mcp",
      "version": "1.0.0"
    }
  },
  "id": 0
}
EOF

echo ""
echo "Test 3: Verify field ordering matches AI Foundry expectations"
echo "-------------------------------------------------------------"
echo "Expected order: capabilities, protocolVersion, serverInfo"
echo ""
./build/sap-odata-mcp-universal --service "https://services.odata.org/V2/Northwind/Northwind.svc/" --protocol-version "2025-06-18" <<EOF | python3 -c "
import sys
import json

# Read input
response = sys.stdin.read()
try:
    data = json.loads(response)
    result = data.get('result', {})

    # Check field order
    fields = list(result.keys())
    expected = ['capabilities', 'protocolVersion', 'serverInfo']

    if fields == expected:
        print('✅ Field ordering is correct!')
        print(f'   Fields: {fields}')
    else:
        print('❌ Field ordering mismatch')
        print(f'   Expected: {expected}')
        print(f'   Got: {fields}')

    # Print protocol version
    print(f'   Protocol Version: {result.get(\"protocolVersion\")}')

except Exception as e:
    print(f'Error parsing response: {e}')
    print(f'Response: {response}')
"
{
  "jsonrpc": "2.0",
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-06-18",
    "capabilities": {},
    "clientInfo": {
      "name": "openai-mcp",
      "version": "1.0.0"
    }
  },
  "id": 0
}
EOF