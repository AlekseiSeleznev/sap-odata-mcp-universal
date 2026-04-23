#!/usr/bin/env python3
"""
Test MCP client simulator to diagnose validation errors
"""
import json
import subprocess
import sys

def send_request(request):
    """Send a request to the MCP server and return the response"""
    try:
        proc = subprocess.Popen(
            ['./sap-odata-mcp-universal'],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        stdout, stderr = proc.communicate(json.dumps(request))
        
        if stderr:
            print(f"STDERR: {stderr}", file=sys.stderr)
            
        return json.loads(stdout) if stdout else None
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        return None

def test_edge_cases():
    """Test various edge cases that might cause validation errors"""
    tests = [
        # Test 1: Standard initialize
        {
            "name": "Standard initialize",
            "request": {
                "jsonrpc": "2.0",
                "id": 1,
                "method": "initialize",
                "params": {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {},
                    "clientInfo": {"name": "test", "version": "1.0"}
                }
            }
        },
        # Test 2: Initialize with missing params
        {
            "name": "Initialize without params",
            "request": {
                "jsonrpc": "2.0",
                "id": 2,
                "method": "initialize"
            }
        },
        # Test 3: Tools/list with extra params
        {
            "name": "Tools/list with cursor",
            "request": {
                "jsonrpc": "2.0",
                "id": 3,
                "method": "tools/list",
                "params": {"cursor": None}
            }
        },
        # Test 4: Tools/call with missing arguments
        {
            "name": "Tools/call without arguments",
            "request": {
                "jsonrpc": "2.0",
                "id": 4,
                "method": "tools/call",
                "params": {
                    "name": "odata_service_info_for_Z001"
                }
            }
        },
        # Test 5: Tools/call with null arguments
        {
            "name": "Tools/call with null arguments",
            "request": {
                "jsonrpc": "2.0",
                "id": 5,
                "method": "tools/call",
                "params": {
                    "name": "odata_service_info_for_Z001",
                    "arguments": None
                }
            }
        }
    ]
    
    for test in tests:
        print(f"\n=== {test['name']} ===")
        response = send_request(test['request'])
        
        if response:
            if 'error' in response:
                print(f"ERROR: {response['error']}")
            else:
                print(f"SUCCESS: Got result with keys: {list(response.get('result', {}).keys())}")
        else:
            print("FAILED: No response")

if __name__ == "__main__":
    print("MCP Server Edge Case Testing")
    print("============================")
    test_edge_cases()