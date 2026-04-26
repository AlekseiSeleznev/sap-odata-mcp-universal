set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

conformance := "/home/as/Документы/AI_PROJECTS/modelcontextprotocol-conformance/dist/index.js"
inspector := "/home/as/Документы/AI_PROJECTS/modelcontextprotocol-inspector/cli/build/index.js"
mcpeval_dir := "/home/as/Документы/AI_PROJECTS/lastmile-ai-mcp-eval"
pwsh := "/home/as/Документы/AI_PROJECTS/PowerShell-PowerShell/runtime-7.6.1-linux-x64/pwsh"
default_mcp_url := "http://localhost:8080/mcp"
default_health_url := "http://localhost:8080/health"

default:
    @echo "Available: test, health, mcp-init, mcp-tools-list, mcp-conformance, mcp-inspector-tools, mcp-eval, pwsh-version, smoke"

test:
    go test ./...

health:
    curl -fsS "${HEALTH_URL:-{{default_health_url}}}"

mcp-init:
    node "{{conformance}}" server --url "${MCP_URL:-{{default_mcp_url}}}" --scenario server-initialize --output-dir "${MCP_CONFORMANCE_RESULTS:-/tmp/sap-odata-mcp-conformance}"

mcp-tools-list:
    node "{{conformance}}" server --url "${MCP_URL:-{{default_mcp_url}}}" --scenario tools-list --output-dir "${MCP_CONFORMANCE_RESULTS:-/tmp/sap-odata-mcp-conformance}"

mcp-conformance: mcp-init mcp-tools-list

mcp-inspector-tools:
    #!/usr/bin/env bash
    set -euo pipefail
    url="${MCP_URL:-{{default_mcp_url}}}"
    if [ -n "${SAP_ODATA_MCP_TOKEN:-}" ]; then
      node "{{inspector}}" --transport http --header "Authorization: Bearer ${SAP_ODATA_MCP_TOKEN}" --method tools/list "$url"
    else
      node "{{inspector}}" --transport http --method tools/list "$url"
    fi

mcp-eval path="tests/mcp-eval":
    #!/usr/bin/env bash
    set -euo pipefail
    project_dir="$PWD"
    cd "{{mcpeval_dir}}"
    uv run mcp-eval run "$project_dir/{{path}}"

pwsh-version:
    @"{{pwsh}}" -NoLogo -NoProfile -Command '$PSVersionTable.PSVersion.ToString()'

smoke: health mcp-conformance
