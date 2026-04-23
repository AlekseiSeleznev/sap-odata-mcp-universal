# AI Foundry Compatibility Guide

## Overview

The OData MCP Bridge now supports compatibility with AI Foundry's MCP client, which uses protocol version `2025-06-18`. This guide explains how to configure the bridge for use with AI Foundry.

## Quick Start

To run the OData MCP Bridge with AI Foundry compatibility:

```bash
./sap-odata-mcp-universal --service <your-odata-service-url> --protocol-version "2025-06-18"
```

## Problem Background

AI Foundry's MCP client sends initialization requests with:
- Protocol version: `2025-06-18`
- Client info: `openai-mcp` version `1.0.0`

The default OData MCP Bridge uses protocol version `2024-11-05`, which caused compatibility issues.

## Solution

### 1. Protocol Version Override

A new command-line parameter `--protocol-version` allows you to override the MCP protocol version:

```bash
# For AI Foundry compatibility
./sap-odata-mcp-universal --service https://your-service.com/odata --protocol-version "2025-06-18"

# Default (Claude, etc.)
./sap-odata-mcp-universal --service https://your-service.com/odata
```

### 2. Response Field Ordering

The bridge now returns fields in the exact order expected by AI Foundry:
1. `capabilities`
2. `protocolVersion`
3. `serverInfo`

This ensures compatibility with clients that may be sensitive to field ordering.

## Configuration Examples

### Basic AI Foundry Setup

```bash
./sap-odata-mcp-universal \
  --service "https://api.example.com/odata" \
  --protocol-version "2025-06-18" \
  --user "your-username" \
  --password "your-password"
```

### With Additional Options

```bash
./sap-odata-mcp-universal \
  --service "https://api.example.com/odata" \
  --protocol-version "2025-06-18" \
  --user "your-username" \
  --password "your-password" \
  --tool-shrink \
  --max-items 100 \
  --verbose
```

### Environment Variable Support

You can also set the protocol version via environment variable:

```bash
export ODATA_PROTOCOL_VERSION="2025-06-18"
./sap-odata-mcp-universal --service "https://api.example.com/odata"
```

## Testing Protocol Compatibility

To verify the protocol version is correctly set:

```bash
# Send a test initialization request
echo '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"openai-mcp","version":"1.0.0"}},"id":0}' | \
  ./sap-odata-mcp-universal --service "https://services.odata.org/V2/Northwind/Northwind.svc/" --protocol-version "2025-06-18"
```

Expected response structure:
```json
{
  "jsonrpc": "2.0",
  "id": 0,
  "result": {
    "capabilities": {
      "prompts": {
        "listChanged": false
      },
      "resources": {
        "listChanged": false,
        "subscribe": false
      },
      "tools": {
        "listChanged": true
      }
    },
    "protocolVersion": "2025-06-18",
    "serverInfo": {
      "name": "sap-odata-mcp-universal",
      "version": "1.0.0"
    }
  }
}
```

## Troubleshooting

### Protocol Version Mismatch

If you see errors about protocol version mismatch:
1. Ensure you're using `--protocol-version "2025-06-18"` for AI Foundry
2. Check that the client is sending the expected version in its initialize request

### Field Ordering Issues

The bridge automatically handles field ordering. If you still experience issues:
1. Update to the latest version of the OData MCP Bridge
2. Enable verbose mode (`--verbose`) to see detailed protocol exchanges

### Connection Issues

For AI Foundry specific connection issues:
1. Verify your service URL is accessible
2. Check authentication credentials
3. Use `--verbose` flag to see detailed error messages

## Version Compatibility Matrix

| Client | Protocol Version | Command Line Option |
|--------|-----------------|-------------------|
| AI Foundry | 2025-06-18 | `--protocol-version "2025-06-18"` |
| Claude | 2024-11-05 | (default, no option needed) |
| Custom | Any | `--protocol-version "YOUR-VERSION"` |

## Support

For issues specific to AI Foundry integration:
1. Check this guide first
2. Enable verbose logging with `--verbose`
3. Report issues at: https://github.com/AlekseiSeleznev/sap-odata-mcp-universal/issues