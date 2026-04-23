#\!/bin/bash
echo '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"capabilities": {}}}'  < /dev/null |  ./sap-odata-mcp-universal --tool-shrink https://services.odata.org/V2/Northwind/Northwind.svc/ 2>/dev/null
echo
echo '{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": {"name": "odata_service_info_for_NorthSvc", "arguments": {}}}' | ./sap-odata-mcp-universal --tool-shrink https://services.odata.org/V2/Northwind/Northwind.svc/ 2>/dev/null
