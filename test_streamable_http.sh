#!/bin/bash

# Test script for Streamable HTTP transport
set -e

echo "=== Testing Streamable HTTP Transport ==="
echo

# Start the server with streamable-http transport
echo "Starting OData MCP with Streamable HTTP transport..."
./sap-odata-mcp-universal --transport streamable-http --verbose --service https://services.odata.org/V2/Northwind/Northwind.svc/ &
SERVER_PID=$!

# Give the server time to start
sleep 2

echo
echo "Testing health endpoint..."
curl -s http://localhost:8080/health | python3 -m json.tool

echo
echo "Testing MCP endpoint with regular POST..."
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"roots":{},"sampling":{}},"clientInfo":{"name":"test-client","version":"1.0.0"}}}' \
  | python3 -m json.tool

echo
echo "Testing MCP endpoint with SSE Accept header..."
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  --max-time 2 2>/dev/null || true

echo
echo "Testing legacy SSE endpoint..."
curl -s -X POST http://localhost:8080/sse \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/list","params":{}}' \
  --max-time 2 2>/dev/null || true

# Clean up
echo
echo "Stopping server..."
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

echo
echo "=== Streamable HTTP Transport Test Complete ==="