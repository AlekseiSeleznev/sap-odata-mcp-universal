#!/bin/bash

echo "=== Claude Desktop MCP Trace Instructions ==="
echo
echo "To capture a trace of what Claude Desktop is sending:"
echo
echo "1. First, make sure Claude Desktop is NOT running"
echo
echo "2. Update your Claude Desktop config to use the trace-enabled version:"
echo "   Edit ~/.config/claude/claude_desktop_config.json"
echo "   Change the command to include --trace-mcp flag:"
echo
echo '   "command": "/path/to/sap-odata-mcp-universal --trace-mcp"'
echo
echo "3. Start Claude Desktop"
echo
echo "4. Wait a few seconds for it to connect"
echo
echo "5. Check the trace file:"
echo "   ls -la /tmp/mcp_trace_*.log"
echo
echo "6. View the trace:"
echo "   cat /tmp/mcp_trace_*.log | python3 -m json.tool"
echo
echo "The trace will show exactly what Claude Desktop is sending"
echo "and help identify any validation issues."
echo
echo "=== Quick Test ==="
echo "You can also run this to simulate Claude Desktop startup:"
echo

cat > test_claude_startup.sh << 'EOF'
#!/bin/bash
# Simulate Claude Desktop startup sequence
(
    # Claude Desktop usually sends these in sequence
    echo '{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"tools":{"listChanged":true},"resources":{"listChanged":true,"subscribe":true},"prompts":{"listChanged":true}},"clientInfo":{"name":"claude-desktop","version":"0.7.26"}}}'
    sleep 0.1
    echo '{"jsonrpc":"2.0","method":"initialized","params":{}}'
    sleep 0.1
    echo '{"jsonrpc":"2.0","id":"2","method":"resources/list","params":{}}'
    sleep 0.1
    echo '{"jsonrpc":"2.0","id":"3","method":"prompts/list","params":{}}'
    sleep 0.1
    echo '{"jsonrpc":"2.0","id":"4","method":"tools/list","params":{}}'
) | ./sap-odata-mcp-universal --trace-mcp 2>&1
EOF

chmod +x test_claude_startup.sh
echo "./test_claude_startup.sh"