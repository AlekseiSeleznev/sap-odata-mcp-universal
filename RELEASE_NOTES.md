# OData MCP Bridge v1.5.0

## 🎉 Release Highlights

This release brings significant improvements for SAP OData service compatibility and introduces support for the modern MCP Streamable HTTP transport protocol.

## ✨ New Features

### SAP GUID Filtering Support
- **Automatic GUID formatting for SAP services**: The bridge now automatically detects SAP OData services and formats GUID filter values correctly
- SAP requires GUID values in filters to use the format `guid'...'` instead of just `'...'`
- Detection works via:
  - URL patterns (containing "sap", "s4hana", etc.)
  - SAP-specific metadata attributes
  - Service type hints
- Example transformation: `ChemicalRevisionUUID eq '06023781-a5b8-1eec-bf88-402534c04747'` → `ChemicalRevisionUUID eq guid'06023781-a5b8-1eec-bf88-402534c04747'`

### Streamable HTTP Transport
- Support for modern MCP protocol version 2024-11-05
- Single `/mcp` endpoint with automatic SSE upgrade
- Bidirectional communication and session management
- Better alignment with Python MCP ecosystem
- Usage: `./sap-odata-mcp-universal --transport streamable-http <service-url>`

## 🐛 Bug Fixes
- Fixed GUID filtering issues for SAP OData services
- Added regression tests to prevent binary ENOENT errors
- Improved metadata parsing for SAP-specific annotations

## 📚 Documentation Updates
- Added comprehensive implementation guide
- Added GitHub Discussions link for community support
- Improved README with v1.5.0 features

## 💾 Binary Downloads

Pre-built binaries are available for:
- **macOS**: Intel (amd64) and Apple Silicon (arm64)
- **Linux**: amd64, arm64, and arm
- **Windows**: amd64 (64-bit), 386 (32-bit), and arm64

## 🔧 Installation

### macOS/Linux
```bash
# Download the appropriate binary for your platform
tar -xzf sap-odata-mcp-universal-v1.5.0-<platform>-<arch>.tar.gz
chmod +x sap-odata-mcp-universal-v1.5.0-<platform>-<arch>
mv sap-odata-mcp-universal-v1.5.0-<platform>-<arch> /usr/local/bin/sap-odata-mcp-universal
```

### Windows
```powershell
# Extract the ZIP file and add to PATH
# Or run directly: .\sap-odata-mcp-universal-v1.5.0-windows-amd64.exe
```

## 🙏 Acknowledgments

Special thanks to Jarl for reporting the SAP GUID filtering issue and helping test the fix!

## 📝 Full Changelog

- a286016 test: Add regression tests to prevent binary ENOENT errors
- 325881a docs: Add v1.5.0 release notes to README
- b082427 feat: Add Streamable HTTP transport support
- 125f0cf docs: Add GitHub Discussions link to README
- 4b6c03b feat: Add comprehensive implementation guide
- [NEW] feat: Add automatic GUID formatting for SAP OData services