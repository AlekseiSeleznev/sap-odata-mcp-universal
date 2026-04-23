# Windows Trace Log Location Guide

## Where to Find MCP Trace Logs on Windows

The trace logs are saved in the Windows temp directory with the pattern `mcp_trace_*.log`.

### 1. Default Locations

The logs will be in one of these locations:

```
C:\Users\%USERNAME%\AppData\Local\Temp\mcp_trace_*.log
```

Or simply:
```
%TEMP%\mcp_trace_*.log
```

### 2. How to Find Your Trace Logs

#### Method 1: Using Windows Explorer
1. Press `Windows + R`
2. Type `%TEMP%` and press Enter
3. Look for files starting with `mcp_trace_` (e.g., `mcp_trace_20250704_000221.log`)

#### Method 2: Using Command Prompt
```cmd
cd %TEMP%
dir mcp_trace_*.log
```

#### Method 3: Using PowerShell
```powershell
Get-ChildItem $env:TEMP -Filter "mcp_trace_*.log" | Sort-Object LastWriteTime -Descending
```

### 3. Viewing the Trace Logs

#### Option 1: Using Notepad++
Open the file in Notepad++ and use the JSON Language mode for better formatting.

#### Option 2: Using PowerShell to Pretty-Print
```powershell
Get-Content "$env:TEMP\mcp_trace_YYYYMMDD_HHMMSS.log" | ConvertFrom-Json | ConvertTo-Json -Depth 10
```

#### Option 3: Using VS Code
Open in VS Code which will automatically format JSON files.

### 4. Enabling Trace Logging

#### For Claude Desktop on Windows

1. Edit your Claude Desktop config file:
   - Location: `%APPDATA%\Claude\claude_desktop_config.json`
   - Or: `C:\Users\%USERNAME%\AppData\Roaming\Claude\claude_desktop_config.json`

2. Add the `--trace-mcp` flag to your command:
   ```json
   {
     "mcpServers": {
       "sap-odata-mcp-universal": {
         "command": "C:\\path\\to\\sap-odata-mcp-universal.exe --trace-mcp"
       }
     }
   }
   ```

#### For Testing
```cmd
sap-odata-mcp-universal.exe --trace-mcp
```

### 5. Quick Test Script for Windows

Create `test_trace.bat`:
```batch
@echo off
echo Testing MCP with trace enabled...
echo.

REM Start the server with trace
echo {"jsonrpc":"2.0","id":1,"method":"initialize","params":{}} | sap-odata-mcp-universal.exe --trace-mcp

echo.
echo Trace file location:
dir %TEMP%\mcp_trace_*.log /B /O-D | head -1

echo.
echo To view the trace:
echo notepad %TEMP%\mcp_trace_[filename].log
```

### 6. Cleaning Up Old Trace Files

To remove old trace files:
```cmd
del %TEMP%\mcp_trace_*.log
```

Or keep only the most recent:
```powershell
Get-ChildItem $env:TEMP -Filter "mcp_trace_*.log" | 
    Sort-Object LastWriteTime -Descending | 
    Select-Object -Skip 5 | 
    Remove-Item
```

### 7. Troubleshooting

If you can't find the trace files:

1. **Check if tracing is enabled**: Make sure you're using `--trace-mcp` flag
2. **Check permissions**: Ensure the process can write to %TEMP%
3. **Look for errors**: Run with `--verbose` to see if there are errors creating the trace file
4. **Alternative location**: The app might use a different temp directory. Check:
   - `C:\Windows\Temp`
   - `C:\ProgramData\Temp`
   - The directory where `sap-odata-mcp-universal.exe` is located

### 8. Example Trace Output

When you open a trace file, you'll see JSON entries like:
```json
{
  "timestamp": "2025-07-04T00:02:21.800766298+01:00",
  "level": "TRACE",
  "message": "Trace logging started",
  "data": {
    "filename": "C:\\Users\\YourName\\AppData\\Local\\Temp\\mcp_trace_20250704_000221.log",
    "pid": 12345,
    "time": "2025-07-04T00:02:21+01:00"
  }
}
```

Each line is a separate JSON object showing:
- What happened (`message`)
- When it happened (`timestamp`)
- Additional details (`data`)
- Severity level (`level`)

This helps debug what Claude Desktop is sending and receiving.