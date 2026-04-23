#!/bin/bash

echo "=== MCP Inspector Setup Guide ==="
echo
echo "To use the official MCP Inspector with sap-odata-mcp-universal:"
echo
echo "1. Install and run the inspector:"
echo "   npx @modelcontextprotocol/inspector ./sap-odata-mcp-universal"
echo
echo "2. This will open a web UI at http://localhost:5173"
echo
echo "3. In the UI, you can:"
echo "   - See all available tools"
echo "   - Test tool calls interactively"
echo "   - View request/response logs"
echo "   - Debug any issues"
echo
echo "Alternative: Use CLI mode for automated testing:"
echo "   npx @modelcontextprotocol/inspector --cli ./sap-odata-mcp-universal --method tools/list"
echo
echo "=== Quick Local Test ==="
echo "Running a quick local protocol test..."
echo

# Quick protocol test
echo '{"jsonrpc":"2.0","id":"test","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"clientInfo":{"name":"inspector","version":"1.0.0"}}}' | ./sap-odata-mcp-universal 2>&1 | python3 -c "
import sys
import json
try:
    response = json.loads(sys.stdin.read())
    print('✅ Valid JSON response')
    if 'error' in response:
        print(f'❌ Error: {response[\"error\"]}')
    elif 'result' in response:
        result = response['result']
        print(f'✅ Protocol version: {result.get(\"protocolVersion\")}')
        print(f'✅ Server: {result.get(\"serverInfo\", {}).get(\"name\")} v{result.get(\"serverInfo\", {}).get(\"version\")}')
        caps = result.get('capabilities', {})
        print(f'✅ Capabilities: tools={\"tools\" in caps}, resources={\"resources\" in caps}, prompts={\"prompts\" in caps}')
except Exception as e:
    print(f'❌ Failed to parse response: {e}')
"