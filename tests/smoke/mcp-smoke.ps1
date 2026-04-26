param(
    [string]$ServerExe = "",
    [string]$McpUrl = "",
    [string]$HealthUrl = "",
    [string]$Token = ""
)

$ErrorActionPreference = "Stop"

function Get-FreeTcpPort {
    $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
    $listener.Start()
    try {
        return $listener.LocalEndpoint.Port
    }
    finally {
        $listener.Stop()
    }
}

function Invoke-McpRequest {
    param(
        [string]$Url,
        [string]$BearerToken,
        [hashtable]$Body
    )

    $headers = @{
        "Authorization" = "Bearer $BearerToken"
        "Content-Type" = "application/json"
        "Accept" = "application/json"
    }
    $json = $Body | ConvertTo-Json -Depth 20 -Compress
    return Invoke-RestMethod -Method Post -Uri $Url -Headers $headers -Body $json
}

$scriptDir = Split-Path -Parent $PSCommandPath
$projectRoot = Split-Path -Parent (Split-Path -Parent $scriptDir)

if ([string]::IsNullOrWhiteSpace($Token)) {
    if ([string]::IsNullOrWhiteSpace($env:SAP_ODATA_MCP_TOKEN)) {
        $Token = "sap-odata-mcp-validation-token"
    }
    else {
        $Token = $env:SAP_ODATA_MCP_TOKEN
    }
}

$startedProcess = $null
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("sap-odata-mcp-smoke-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tempDir | Out-Null

try {
    if ([string]::IsNullOrWhiteSpace($McpUrl) -or [string]::IsNullOrWhiteSpace($HealthUrl)) {
        $port = Get-FreeTcpPort
        $McpUrl = "http://localhost:$port/mcp"
        $HealthUrl = "http://localhost:$port/health"

        if ([string]::IsNullOrWhiteSpace($ServerExe)) {
            $exeName = if ($IsWindows) { "sap-odata-mcp-universal.exe" } else { "sap-odata-mcp-universal" }
            $ServerExe = Join-Path $tempDir $exeName
            Push-Location $projectRoot
            try {
                & go build -o $ServerExe ./cmd/sap-odata-mcp-universal
            }
            finally {
                Pop-Location
            }
        }

        $stateFile = Join-Path $tempDir "odata_state.json"
        $oldStateFile = $env:ODATA_MCP_STATE_FILE
        $env:ODATA_MCP_STATE_FILE = $stateFile
        try {
            $startedProcess = Start-Process -FilePath $ServerExe `
                -ArgumentList @("--transport", "streamable-http", "--http-addr", "localhost:$port", "--mcp-token", $Token) `
                -WorkingDirectory $projectRoot `
                -PassThru `
                -NoNewWindow
        }
        finally {
            $env:ODATA_MCP_STATE_FILE = $oldStateFile
        }
    }

    $ready = $false
    for ($i = 0; $i -lt 100; $i++) {
        try {
            $health = Invoke-RestMethod -Method Get -Uri $HealthUrl
            if ($health.status -eq "ok") {
                $ready = $true
                break
            }
        }
        catch {
            if ($startedProcess -and $startedProcess.HasExited) {
                throw "MCP server exited before health became ready with code $($startedProcess.ExitCode)"
            }
            Start-Sleep -Milliseconds 100
        }
    }

    if (-not $ready) {
        throw "Timed out waiting for health endpoint $HealthUrl"
    }

    $initialize = Invoke-McpRequest -Url $McpUrl -BearerToken $Token -Body @{
        jsonrpc = "2.0"
        id = 1
        method = "initialize"
        params = @{
            protocolVersion = "2024-11-05"
            capabilities = @{}
            clientInfo = @{
                name = "sap-odata-mcp-pwsh-smoke"
                version = "1.0.0"
            }
        }
    }

    if (-not $initialize.result.protocolVersion) {
        throw "Initialize response did not include protocolVersion"
    }
    if (-not $initialize.result.capabilities.tools) {
        throw "Initialize response did not include tools capability"
    }

    $tools = Invoke-McpRequest -Url $McpUrl -BearerToken $Token -Body @{
        jsonrpc = "2.0"
        id = 2
        method = "tools/list"
        params = @{}
    }

    if ($null -eq $tools.result.tools) {
        throw "tools/list response did not include result.tools"
    }

    Write-Host "PowerShell MCP smoke passed: $McpUrl"
}
finally {
    if ($startedProcess -and -not $startedProcess.HasExited) {
        Stop-Process -Id $startedProcess.Id -Force -ErrorAction SilentlyContinue
        $startedProcess.WaitForExit(5000) | Out-Null
    }
    if (Test-Path $tempDir) {
        Remove-Item -Recurse -Force $tempDir -ErrorAction SilentlyContinue
    }
}
