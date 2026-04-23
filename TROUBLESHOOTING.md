# MCP Server Troubleshooting Guide

## Common Issues and Solutions

### 1. Server Launch Errors / Validation Errors in Claude Desktop

If you're seeing Zod validation errors or the server isn't showing tools properly:

**Symptoms:**
- Multiple ZodError validation failures
- "Unrecognized key(s) in object" errors
- Type mismatches (expecting string/number but receiving null)
- Server disconnection

**Diagnostics Run:**
```bash
# Test basic server functionality
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./sap-odata-mcp-universal
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | ./sap-odata-mcp-universal
```

**Root Cause:**
The server is functioning correctly, but MCP clients (especially Claude Desktop) have strict schema validation that may reject responses with:
- Extra fields in params
- Null values where not expected
- Missing optional fields

**Solutions:**

1. **Ensure Clean Environment:**
   ```bash
   # Clean build
   go clean -cache
   go build ./cmd/sap-odata-mcp-universal
   ```

2. **Check Service URL:**
   - Ensure your ODATA_URL environment variable is set correctly
   - The service must be accessible and return valid metadata

3. **Verify Authentication:**
   - Check ODATA_USERNAME and ODATA_PASSWORD are set
   - For cookie auth, ensure ODATA_COOKIES is properly formatted

4. **Test with Simple Service:**
   ```bash
   # Test with a simple OData v2 service
   export ODATA_URL="https://services.odata.org/V2/Northwind/Northwind.svc/"
   ./sap-odata-mcp-universal
   ```

### 2. Tools Not Appearing in Client

**Possible Causes:**
1. Metadata parsing failed silently
2. Service URL is incorrect or inaccessible
3. Authentication failed

**Debug Steps:**
```bash
# Run with verbose mode to see what's happening
./sap-odata-mcp-universal --verbose
```

### 3. Specific Client Issues

**Claude Desktop:**
- Restart Claude Desktop after configuration changes
- Check ~/.config/claude/claude_desktop_config.json for proper MCP configuration

**RooCode:**
- Tools may appear briefly then disappear - this is often due to initialization timing
- Try restarting the extension

**GitHub Copilot:**
- Ensure the MCP extension is properly configured
- Check extension logs for errors

### 4. Service-Specific Issues

**Using Service Hints:**

The bridge includes a hint system that provides guidance for known service issues:

```bash
# Check for hints in service info
./sap-odata-mcp-universal https://my-service.com/odata/
# Then call odata_service_info tool to see implementation_hints

# Use custom hints file
./sap-odata-mcp-universal --hints-file my-hints.json https://my-service.com/odata/

# Add quick hint from CLI
./sap-odata-mcp-universal --hint "Check field casing in \$metadata" https://my-service.com/odata/
```

**SAP Services:**
- Many SAP services require CSRF tokens
- Use --legacy-dates flag for date compatibility
- Some services return HTTP 501 for direct entity access - use $expand workaround (see hints)
- Check implementation_hints in service info for specific guidance

**Example for SAP PO Tracking Service:**
- Returns HTTP 501 for direct entity access
- **Workaround**: Use `filter_PurchaseOrderSet` with `$expand=PurchaseOrderItemDetails` instead of `get_PurchaseOrderItem`
- PONumber field expects numeric strings despite type definition
- Use quotes in filters: `$filter=PONumber eq '1234567890'`
- Try with/without leading zeros if queries fail

### 5. Read-Only Mode Issues

If create/update/delete operations are appearing when they shouldn't:
```bash
# Use read-only mode
./sap-odata-mcp-universal --read-only

# Or allow only function imports
./sap-odata-mcp-universal --read-only-but-functions
```

## Debug Commands

```bash
# Check if server is responding
echo '{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}' | ./sap-odata-mcp-universal

# Get detailed service info
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"odata_service_info_for_[ID]","arguments":{"include_metadata":true}}}' | ./sap-odata-mcp-universal

# Test with curl (HTTP transport)
./sap-odata-mcp-universal --transport http --http-addr :8080 &
curl -X POST http://localhost:8080/sse \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
```

## Reporting Issues

When reporting issues, please include:
1. Output of `./sap-odata-mcp-universal --version`
2. The ODATA_URL you're connecting to (sanitized)
3. Output of running with `--verbose` flag
4. The specific MCP client you're using
5. Any error messages from the client logs