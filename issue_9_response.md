# Response to Issue #9: OData V4 Query Parameters Not Properly Converted to URL Query String

## Investigation Summary

Thank you for reporting this issue. After thorough investigation of the codebase and recent changes, I've identified that this issue has already been addressed.

## Root Cause

The issue you're experiencing in v1.3.0 was related to improper URL encoding of query parameters. The root cause was that spaces in OData query parameters were being encoded as `+` instead of `%20`, which many OData servers (including SAP CAP) don't accept.

## Solution Status

**This issue has been fixed in v1.4.0** through commit 600e8ca, which implemented:

1. A dedicated `encodeQueryParams` helper function that ensures proper URL encoding
2. Replacement of `+` with `%20` for space characters in query parameters
3. Comprehensive unit tests to prevent regression

## The Fix

The fix ensures that query parameters are properly encoded according to RFC 3986 standards. For example:
- **Before (v1.3.0)**: `/odata/v4/todo/Todos?$filter=done+eq+false` 
- **After (v1.4.0)**: `/odata/v4/todo/Todos?$filter=done%20eq%20false`

## Resolution

To resolve your issue, please **upgrade to v1.4.0**:

```bash
# For Windows
curl -L https://github.com/AlekseiSeleznev/sap-odata-mcp-universal/releases/download/v1.4.0/sap-odata-mcp-universal.exe -o sap-odata-mcp-universal.exe

# For other platforms, check the releases page
```

## Verification

After upgrading, your filter queries should work correctly:
- Input: `{"$filter": "done eq false"}`
- Generated URL: `/odata/v4/todo/Todos?$filter=done%20eq%20false`

## Additional Context

This issue was initially identified by @cpf-hse in PR #8, who experienced similar problems with CAP OData v4 services. The fix has been tested with various OData servers including SAP CAP.

## Proposal to Close

Since this issue has been resolved in v1.4.0 (released after v1.3.0), I propose closing this issue with the following resolution:

**Resolution**: Fixed in v1.4.0
**Action Required**: Users experiencing this issue should upgrade from v1.3.0 to v1.4.0

Please confirm if upgrading to v1.4.0 resolves your issue. If you continue to experience problems after upgrading, please feel free to reopen this issue with additional details.

Thank you for your detailed bug report, which helps improve the OData MCP Bridge for all users!