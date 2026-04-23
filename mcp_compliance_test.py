#!/usr/bin/env python3
"""
MCP Protocol Compliance Tester
Tests MCP server implementation against protocol specification
"""
import json
import subprocess
import sys
from typing import Dict, Any, Optional, Tuple

class MCPComplianceTester:
    def __init__(self, server_command: str):
        self.server_command = server_command
        self.test_results = []
        
    def send_request(self, request: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Send a JSON-RPC request to the server"""
        try:
            proc = subprocess.Popen(
                self.server_command.split(),
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True
            )
            stdout, stderr = proc.communicate(json.dumps(request))
            
            if stdout.strip():
                return json.loads(stdout)
            return None
        except Exception as e:
            print(f"Error sending request: {e}", file=sys.stderr)
            return None
    
    def test(self, name: str, request: Dict[str, Any], 
             validator: callable, expected_error: bool = False) -> bool:
        """Run a single test"""
        print(f"Testing: {name}...", end=" ")
        response = self.send_request(request)
        
        if response is None:
            print("FAIL - No response")
            self.test_results.append((name, False, "No response received"))
            return False
        
        if expected_error and 'error' not in response:
            print("FAIL - Expected error but got success")
            self.test_results.append((name, False, "Expected error response"))
            return False
            
        if not expected_error and 'error' in response:
            print(f"FAIL - Unexpected error: {response['error']}")
            self.test_results.append((name, False, f"Unexpected error: {response['error']}"))
            return False
        
        try:
            if validator(response):
                print("PASS")
                self.test_results.append((name, True, ""))
                return True
            else:
                print("FAIL - Validation failed")
                self.test_results.append((name, False, "Response validation failed"))
                return False
        except Exception as e:
            print(f"FAIL - {e}")
            self.test_results.append((name, False, str(e)))
            return False
    
    def run_compliance_tests(self):
        """Run all compliance tests"""
        print("=== MCP Protocol Compliance Test Suite ===\n")
        
        # Test 1: Basic JSON-RPC structure
        self.test(
            "Valid JSON-RPC request",
            {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}},
            lambda r: r.get('jsonrpc') == '2.0' and 'id' in r and 'result' in r
        )
        
        # Test 2: Missing jsonrpc field
        self.test(
            "Missing jsonrpc field",
            {"id": 2, "method": "initialize", "params": {}},
            lambda r: r.get('error', {}).get('code') == -32600,
            expected_error=True
        )
        
        # Test 3: Invalid jsonrpc version
        self.test(
            "Invalid jsonrpc version",
            {"jsonrpc": "1.0", "id": 3, "method": "initialize", "params": {}},
            lambda r: r.get('error', {}).get('code') == -32600,
            expected_error=True
        )
        
        # Test 4: Method not found
        self.test(
            "Method not found",
            {"jsonrpc": "2.0", "id": 4, "method": "invalid/method", "params": {}},
            lambda r: r.get('error', {}).get('code') == -32601,
            expected_error=True
        )
        
        # Test 5: Initialize response structure
        self.test(
            "Initialize response structure",
            {"jsonrpc": "2.0", "id": 5, "method": "initialize", "params": {
                "protocolVersion": "2024-11-05",
                "capabilities": {},
                "clientInfo": {"name": "test", "version": "1.0"}
            }},
            lambda r: all([
                'protocolVersion' in r.get('result', {}),
                'capabilities' in r.get('result', {}),
                'serverInfo' in r.get('result', {}),
                'name' in r.get('result', {}).get('serverInfo', {}),
                'version' in r.get('result', {}).get('serverInfo', {})
            ])
        )
        
        # Test 6: Capabilities structure
        self.test(
            "Capabilities structure",
            {"jsonrpc": "2.0", "id": 6, "method": "initialize", "params": {}},
            lambda r: all([
                'tools' in r.get('result', {}).get('capabilities', {}),
                'resources' in r.get('result', {}).get('capabilities', {}),
                'prompts' in r.get('result', {}).get('capabilities', {})
            ])
        )
        
        # Test 7: Tools list structure
        self.test(
            "Tools list structure",
            {"jsonrpc": "2.0", "id": 7, "method": "tools/list", "params": {}},
            lambda r: isinstance(r.get('result', {}).get('tools'), list)
        )
        
        # Test 8: Tool structure validation
        tools_response = self.send_request({"jsonrpc": "2.0", "id": 8, "method": "tools/list", "params": {}})
        if tools_response and 'result' in tools_response:
            tools = tools_response['result'].get('tools', [])
            if tools:
                self.test(
                    "Tool structure",
                    {"jsonrpc": "2.0", "id": 8, "method": "tools/list", "params": {}},
                    lambda r: all([
                        all(['name' in tool and 'description' in tool and 'inputSchema' in tool 
                             for tool in r.get('result', {}).get('tools', [])])
                    ])
                )
        
        # Test 9: Resources list structure
        self.test(
            "Resources list structure",
            {"jsonrpc": "2.0", "id": 9, "method": "resources/list", "params": {}},
            lambda r: isinstance(r.get('result', {}).get('resources'), list)
        )
        
        # Test 10: Prompts list structure
        self.test(
            "Prompts list structure",
            {"jsonrpc": "2.0", "id": 10, "method": "prompts/list", "params": {}},
            lambda r: isinstance(r.get('result', {}).get('prompts'), list)
        )
        
        # Test 11: Notification handling (no response expected)
        print("Testing: Notification handling...", end=" ")
        response = self.send_request({"jsonrpc": "2.0", "method": "initialized"})
        if response is None:
            print("PASS")
            self.test_results.append(("Notification handling", True, ""))
        else:
            print("FAIL - Got response for notification")
            self.test_results.append(("Notification handling", False, "Should not respond to notifications"))
        
        # Test 12: ID preservation
        test_ids = [1, "string-id", None, 0, -1]
        for test_id in test_ids:
            self.test(
                f"ID preservation ({test_id})",
                {"jsonrpc": "2.0", "id": test_id, "method": "ping", "params": {}},
                lambda r, tid=test_id: r.get('id') == tid
            )
        
        # Test 13: Tools/call structure
        if tools and len(tools) > 0:
            first_tool = tools[0]['name']
            self.test(
                "Tools/call response structure",
                {"jsonrpc": "2.0", "id": 13, "method": "tools/call", 
                 "params": {"name": first_tool, "arguments": {}}},
                lambda r: 'content' in r.get('result', {}) and 
                         isinstance(r['result']['content'], list)
            )
        
        # Print summary
        print("\n=== Test Summary ===")
        passed = sum(1 for _, success, _ in self.test_results if success)
        total = len(self.test_results)
        print(f"Passed: {passed}/{total}")
        
        if passed < total:
            print("\nFailed tests:")
            for name, success, reason in self.test_results:
                if not success:
                    print(f"  - {name}: {reason}")
        
        return passed == total

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python mcp_compliance_test.py <server-command>")
        print("Example: python mcp_compliance_test.py ./sap-odata-mcp-universal")
        sys.exit(1)
    
    server_command = " ".join(sys.argv[1:])
    tester = MCPComplianceTester(server_command)
    
    if tester.run_compliance_tests():
        print("\n✅ All tests passed! Server is MCP compliant.")
        sys.exit(0)
    else:
        print("\n❌ Some tests failed. Server needs fixes.")
        sys.exit(1)