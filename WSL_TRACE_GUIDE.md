# Finding MCP Trace Logs in WSL

## Accessing Windows Temp Directory from WSL

Since you're running in WSL, the trace logs are saved in the Linux temp directory, but you can also access Windows directories.

### 1. Linux Temp Directory (WSL)
If you're running sap-odata-mcp-universal from WSL:
```bash
ls -la /tmp/mcp_trace_*.log
```

### 2. Windows Temp Directory from WSL
To access Windows temp directory from WSL:
```bash
# Your Windows user temp directory
ls -la /mnt/c/Users/$USER/AppData/Local/Temp/mcp_trace_*.log

# Or if you know your Windows username
ls -la /mnt/c/Users/[YourWindowsUsername]/AppData/Local/Temp/mcp_trace_*.log

# Quick check
ls -la /mnt/c/Users/*/AppData/Local/Temp/mcp_trace_*.log
```

### 3. Finding the Latest Trace File

#### From WSL temp:
```bash
ls -t /tmp/mcp_trace_*.log | head -1
```

#### From Windows temp via WSL:
```bash
find /mnt/c/Users -name "mcp_trace_*.log" 2>/dev/null | xargs ls -t | head -1
```

### 4. Viewing Trace Logs

#### View in WSL:
```bash
# Pretty print JSON
cat /tmp/mcp_trace_*.log | jq '.'

# Or without jq
cat /tmp/mcp_trace_*.log | python3 -m json.tool

# View raw
cat /tmp/mcp_trace_*.log
```

#### Copy to Windows Desktop:
```bash
cp /tmp/mcp_trace_*.log /mnt/c/Users/$USER/Desktop/
```

### 5. Real-time Trace Monitoring
```bash
# Watch the trace file as it's written
tail -f /tmp/mcp_trace_*.log | jq '.'
```

### 6. Quick Commands

#### Find all trace files:
```bash
# In WSL temp
find /tmp -name "mcp_trace_*.log" -type f 2>/dev/null

# In Windows temp from WSL
find /mnt/c/Users -path "*/AppData/Local/Temp/mcp_trace_*.log" 2>/dev/null
```

#### Get trace file location after running:
```bash
./sap-odata-mcp-universal --trace-mcp 2>&1 | grep "TRACE.*Output file"
```

### 7. Example: Running with Trace and Finding Log
```bash
# Run with trace
echo '{"jsonrpc":"2.0","id":1,"method":"ping"}' | ./sap-odata-mcp-universal --trace-mcp

# Find the trace file
TRACE_FILE=$(ls -t /tmp/mcp_trace_*.log | head -1)
echo "Trace file: $TRACE_FILE"

# View it
cat "$TRACE_FILE" | jq '.'
```

### 8. For Claude Desktop on Windows
If Claude Desktop is running on Windows but calling the WSL binary:

1. The trace will be in WSL's /tmp directory
2. You can copy it to Windows:
   ```bash
   cp /tmp/mcp_trace_*.log /mnt/c/Users/$USER/Desktop/mcp_trace.log
   ```

3. Or view directly:
   ```bash
   code /tmp/mcp_trace_*.log  # Opens in VS Code
   ```