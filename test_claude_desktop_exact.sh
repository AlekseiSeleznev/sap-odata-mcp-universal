#!/bin/bash

echo "=== Testing Claude Desktop Exact Sequence ==="
echo

# Test with trace enabled to capture everything
(
    # Claude Desktop initialize
    echo '{"jsonrpc":"2.0","id":"0adae8a5-34d3-4d7f-b9f3-123456789abc","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"tools":{"listChanged":true},"resources":{"listChanged":true,"subscribe":true},"prompts":{"listChanged":true}},"clientInfo":{"name":"claude-desktop","version":"0.7.26"}}}'
    sleep 0.1
    
    # Initialized notification
    echo '{"jsonrpc":"2.0","method":"initialized","params":{}}'
    sleep 0.1
    
    # List requests with string IDs
    echo '{"jsonrpc":"2.0","id":"list-resources-001","method":"resources/list","params":{}}'
    sleep 0.1
    
    echo '{"jsonrpc":"2.0","id":"list-prompts-002","method":"prompts/list","params":{}}'
    sleep 0.1
    
    echo '{"jsonrpc":"2.0","id":"list-tools-003","method":"tools/list","params":{}}'
    sleep 0.1
    
    # Try calling a tool
    echo '{"jsonrpc":"2.0","id":"call-tool-004","method":"tools/call","params":{"name":"odata_service_info_for_Z001","arguments":{}}}'
    sleep 0.1
    
) | ./sap-odata-mcp-universal --trace-mcp 2>&1

echo
echo "Checking latest trace file..."
TRACE_FILE=$(ls -t /tmp/mcp_trace_*.log | head -1)
echo "Trace saved to: $TRACE_FILE"
echo

# Analyze responses
echo "=== Response Analysis ==="
echo "Initialize response:"
grep -A5 '"id":"0adae8a5' "$TRACE_FILE" | grep TRANSPORT_RAW_OUT | cut -d'"' -f8- | sed 's/\\"/"/g' | python3 -m json.tool 2>/dev/null || echo "Failed to parse"

echo
echo "Checking for errors:"
grep -i error "$TRACE_FILE" | grep -v "has_error.:false" || echo "No errors found"