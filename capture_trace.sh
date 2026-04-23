#!/bin/bash

echo "=== MCP Trace Capture Tool ==="
echo
echo "This will capture a trace of MCP communication during startup"
echo

# Run the server with tracing enabled
echo "Starting server with trace enabled..."
echo

# Send a few test commands and capture the trace
(
    sleep 1
    echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"trace-test","version":"1.0"}}}'
    sleep 0.5
    echo '{"jsonrpc":"2.0","method":"initialized"}'
    sleep 0.5
    echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
    sleep 0.5
    echo '{"jsonrpc":"2.0","id":3,"method":"resources/list","params":{}}'
    sleep 0.5
    echo '{"jsonrpc":"2.0","id":4,"method":"prompts/list","params":{}}'
    sleep 0.5
) | ./sap-odata-mcp-universal --trace-mcp 2>&1 | grep -E "(TRACE|ERROR)" &

# Wait for background process
PID=$!
sleep 3
kill $PID 2>/dev/null

echo
echo "Trace capture complete. Looking for trace file..."
echo

# Find and display the trace file
TRACE_FILE=$(ls -t /tmp/mcp_trace_*.log 2>/dev/null | head -1)

if [ -f "$TRACE_FILE" ]; then
    echo "Found trace file: $TRACE_FILE"
    echo
    echo "=== Trace Contents ==="
    echo
    cat "$TRACE_FILE" | python3 -m json.tool 2>/dev/null || cat "$TRACE_FILE"
    echo
    echo "=== Analysis ==="
    echo
    echo "Total lines in trace: $(wc -l < "$TRACE_FILE")"
    echo "Request messages: $(grep -c '"REQUEST"' "$TRACE_FILE" 2>/dev/null || echo 0)"
    echo "Response messages: $(grep -c '"RESPONSE"' "$TRACE_FILE" 2>/dev/null || echo 0)"
    echo "Errors: $(grep -c '"ERROR"' "$TRACE_FILE" 2>/dev/null || echo 0)"
    echo
    echo "Trace file saved as: $TRACE_FILE"
else
    echo "No trace file found. The server may have failed to start."
fi