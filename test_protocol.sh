#!/bin/bash

echo "Testing Protocol Version Support"
echo "================================="

# Test with AI Foundry protocol version
echo ""
echo "Testing with --protocol-version 2025-06-18:"
echo '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"openai-mcp","version":"1.0.0"}},"id":0}' | \
  ./build/sap-odata-mcp-universal --service "https://services.odata.org/V2/Northwind/Northwind.svc/" --protocol-version "2025-06-18" 2>/dev/null | \
  python3 -m json.tool | head -20

echo ""
echo "Checking field order and protocol version in response..."