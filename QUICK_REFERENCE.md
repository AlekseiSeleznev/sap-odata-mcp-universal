# OData MCP Bridge - Quick Reference Guide

## Installation

```bash
# Download latest release
wget https://github.com/your-repo/releases/latest/download/sap-odata-mcp-universal-linux
chmod +x sap-odata-mcp-universal-linux
sudo mv sap-odata-mcp-universal-linux /usr/local/bin/sap-odata-mcp-universal

# Or build from source
git clone <repository>
cd sap-odata-mcp-universal
make build
```

## Basic Usage

```bash
# Anonymous access
./sap-odata-mcp-universal https://services.odata.org/V2/Northwind/Northwind.svc/

# With authentication
./sap-odata-mcp-universal --user admin --password secret https://my-service.com/odata/

# Using environment variables
export ODATA_URL=https://my-service.com/odata/
export ODATA_USERNAME=admin
export ODATA_PASSWORD=secret
./sap-odata-mcp-universal
```

## Common Flags

### Authentication
- `-u, --user` - Username for basic auth
- `-p, --password` - Password for basic auth  
- `--cookie-file` - Path to cookie file
- `--cookie-string` - Cookie string

### Tool Generation
- `--tool-shrink` - Use shortened tool names
- `--tool-prefix` - Custom prefix for tools
- `--tool-postfix` - Custom postfix for tools
- `--no-postfix` - Use prefix instead of postfix

### Filtering
- `--entities` - Filter entities (comma-separated, wildcards supported)
- `--functions` - Filter functions (comma-separated, wildcards supported)
- `--enable` - Enable only specified operation types (C,S,F,G,U,D,A,R)
- `--disable` - Disable specified operation types (C,S,F,G,U,D,A,R)

### Read-Only Modes
- `--read-only, -ro` - Hide all modifying operations
- `--read-only-but-functions, -robf` - Hide create/update/delete but allow functions

### Operation Type Filtering
Operation codes: C=create, S=search, F=filter, G=get, U=update, D=delete, A=actions, R=read (SFG)
```bash
# Examples:
--disable "cud"    # Disable create, update, delete
--enable "r"       # Enable only read operations (search, filter, get)
--disable "a"      # Disable actions/function imports
--enable "gf"      # Enable only get and filter operations
```

### Debugging
- `-v, --verbose` - Verbose output
- `--trace` - Show tools and exit
- `--trace-mcp` - Enable MCP protocol trace logging

### Service Hints
- `--hints-file` - Path to custom hints JSON file
- `--hint` - Direct hint injection from CLI

### Response Options
- `--max-items` - Max items per response (default: 100)
- `--max-response-size` - Max response size (default: 5MB)
- `--response-metadata` - Include __metadata blocks
- `--pagination-hints` - Add pagination information

## Claude Desktop Configuration

Location by platform:
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`

### Minimal Configuration

```json
{
    "mcpServers": {
        "my-odata": {
            "command": "/path/to/sap-odata-mcp-universal",
            "args": ["--service", "https://my-service.com/odata/"]
        }
    }
}
```

### Production Configuration

```json
{
    "mcpServers": {
        "production-api": {
            "command": "/usr/local/bin/sap-odata-mcp-universal",
            "args": [
                "--service", "https://api.company.com/odata/",
                "--read-only",
                "--tool-shrink",
                "--entities", "Products,Orders,Customers",
                "--max-items", "50",
                "--verbose-errors"
            ],
            "env": {
                "ODATA_USERNAME": "api_user",
                "ODATA_PASSWORD": "api_password"
            }
        }
    }
}
```

## Troubleshooting

### Enable Trace Logging
```bash
./sap-odata-mcp-universal --trace-mcp https://my-service.com/odata/
# Check logs in:
# Linux/WSL: /tmp/mcp_trace_*.log
# Windows: %TEMP%\mcp_trace_*.log
```

### Common Issues

1. **Tools not appearing in Claude Desktop**
   - Check service URL is accessible
   - Verify authentication credentials
   - Run with `--verbose` to see errors
   - Check trace logs with `--trace-mcp`

2. **Validation errors in Claude Desktop**
   - Update to latest version
   - Clear Claude Desktop cache and restart
   - Check trace logs for protocol errors

3. **SAP services issues**
   - Use `--legacy-dates` (enabled by default)
   - Check for CSRF token errors
   - Look for service hints in `odata_service_info` tool

## Generated Tools

For each entity set:
- `filter_{entity}` - List/search with OData queries
- `count_{entity}` - Get count with optional filter
- `get_{entity}` - Get single entity by key
- `create_{entity}` - Create new entity (unless read-only)
- `update_{entity}` - Update entity (unless read-only)
- `delete_{entity}` - Delete entity (unless read-only)

For each function import:
- `{function_name}` - Call the function

Service information:
- `odata_service_info` - Get service metadata and hints

## Environment Variables

```bash
ODATA_SERVICE_URL    # Service URL
ODATA_URL           # Alias for SERVICE_URL
ODATA_USERNAME      # Basic auth username
ODATA_USER          # Alias for USERNAME
ODATA_PASSWORD      # Basic auth password
ODATA_PASS          # Alias for PASSWORD
ODATA_COOKIE_FILE   # Path to cookie file
ODATA_COOKIE_STRING # Cookie string
```

## Security Notes

- Never commit credentials to version control
- Use environment variables or secure credential stores
- Enable read-only mode for production services
- Limit entity access with `--entities` flag
- Use HTTPS for all OData services
- HTTP transport has no authentication - use only locally