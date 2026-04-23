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
