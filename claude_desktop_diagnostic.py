#!/usr/bin/env python3
"""
Claude Desktop specific diagnostic for MCP servers
Checks for common issues that cause validation errors
"""
import json
import subprocess
import sys

def check_server():
    """Run diagnostic checks for Claude Desktop compatibility"""
    print("=== Claude Desktop MCP Diagnostic ===\n")
    
    issues = []
    tools = []
    
    # Test 1: Check tool inputSchema format
    print("1. Checking tool schemas...")
    tools_req = {"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {}}
    proc = subprocess.Popen(['./sap-odata-mcp-universal'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    stdout, _ = proc.communicate(json.dumps(tools_req))
    
    try:
        response = json.loads(stdout)
        tools = response.get('result', {}).get('tools', [])
        
        for tool in tools:
            # Check required fields
            if 'name' not in tool:
                issues.append(f"Tool missing 'name' field: {tool}")
            if 'description' not in tool:
                issues.append(f"Tool missing 'description' field: {tool.get('name', 'unknown')}")
            if 'inputSchema' not in tool:
                issues.append(f"Tool missing 'inputSchema' field: {tool.get('name', 'unknown')}")
            else:
                schema = tool['inputSchema']
                # Check schema structure
                if not isinstance(schema, dict):
                    issues.append(f"Tool {tool['name']} has invalid inputSchema type")
                elif 'type' not in schema:
                    issues.append(f"Tool {tool['name']} inputSchema missing 'type' field")
                elif schema.get('type') != 'object':
                    issues.append(f"Tool {tool['name']} inputSchema type should be 'object'")
                
                # Check for properties
                if 'properties' in schema:
                    props = schema['properties']
                    if not isinstance(props, dict):
                        issues.append(f"Tool {tool['name']} has invalid properties type")
                    else:
                        # Check each property
                        for prop_name, prop_def in props.items():
                            if not isinstance(prop_def, dict):
                                issues.append(f"Tool {tool['name']} property {prop_name} has invalid definition")
                            elif 'type' not in prop_def:
                                issues.append(f"Tool {tool['name']} property {prop_name} missing 'type'")
        
        print(f"   Found {len(tools)} tools")
        
    except Exception as e:
        issues.append(f"Failed to parse tools response: {e}")
    
    # Test 2: Check response content format
    print("\n2. Checking tool response format...")
    if tools:
        first_tool = tools[0]['name']
        call_req = {
            "jsonrpc": "2.0", 
            "id": 2, 
            "method": "tools/call", 
            "params": {"name": first_tool, "arguments": {}}
        }
        
        proc = subprocess.Popen(['./sap-odata-mcp-universal'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        stdout, _ = proc.communicate(json.dumps(call_req))
        
        try:
            response = json.loads(stdout)
            if 'result' in response:
                result = response['result']
                if 'content' not in result:
                    issues.append("Tool response missing 'content' field")
                elif not isinstance(result['content'], list):
                    issues.append("Tool response 'content' should be an array")
                else:
                    for item in result['content']:
                        if 'type' not in item:
                            issues.append("Content item missing 'type' field")
                        elif item['type'] != 'text':
                            issues.append(f"Unknown content type: {item['type']}")
                        if 'text' not in item:
                            issues.append("Content item missing 'text' field")
                        elif not isinstance(item['text'], str):
                            issues.append("Content 'text' field must be a string")
            
            print("   Tool call response validated")
            
        except Exception as e:
            issues.append(f"Failed to parse tool call response: {e}")
    
    # Test 3: Check capability format
    print("\n3. Checking capability declarations...")
    init_req = {"jsonrpc": "2.0", "id": 3, "method": "initialize", "params": {}}
    proc = subprocess.Popen(['./sap-odata-mcp-universal'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    stdout, _ = proc.communicate(json.dumps(init_req))
    
    try:
        response = json.loads(stdout)
        caps = response.get('result', {}).get('capabilities', {})
        
        # Check required capability sections
        required_caps = ['tools', 'resources', 'prompts']
        for cap in required_caps:
            if cap not in caps:
                issues.append(f"Missing capability section: {cap}")
            elif not isinstance(caps[cap], dict):
                issues.append(f"Capability {cap} should be an object")
        
        print("   Capabilities validated")
        
    except Exception as e:
        issues.append(f"Failed to parse initialize response: {e}")
    
    # Summary
    print("\n=== Diagnostic Summary ===")
    if issues:
        print(f"\n❌ Found {len(issues)} potential issues:\n")
        for i, issue in enumerate(issues, 1):
            print(f"   {i}. {issue}")
        print("\nThese issues might cause validation errors in Claude Desktop.")
    else:
        print("\n✅ No issues found! Server appears to be Claude Desktop compatible.")
    
    return len(issues) == 0

if __name__ == "__main__":
    if check_server():
        sys.exit(0)
    else:
        sys.exit(1)