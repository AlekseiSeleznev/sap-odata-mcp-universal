#!/bin/bash

# Test MCP server startup
echo "Testing MCP server startup..."

# Create test input for initialize
cat > test_init.json << 'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"tools":{},"resources":{},"prompts":{}},"clientInfo":{"name":"test-client","version":"1.0.0"}}}
EOF

# Start server and send initialize request
echo "Sending initialize request..."
cat test_init.json | ./sap-odata-mcp-universal 2>&1 | jq '.' || echo "Failed to parse JSON response"

# Clean up
rm -f test_init.json