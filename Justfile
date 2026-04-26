set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

conformance := "/home/as/Документы/AI_PROJECTS/modelcontextprotocol-conformance/dist/index.js"
inspector := "/home/as/Документы/AI_PROJECTS/modelcontextprotocol-inspector/cli/build/index.js"
mcpeval_dir := "/home/as/Документы/AI_PROJECTS/lastmile-ai-mcp-eval"
pwsh := "/home/as/Документы/AI_PROJECTS/PowerShell-PowerShell/runtime-7.6.1-linux-x64/pwsh"
validation_token := "sap-odata-mcp-validation-token"
default_validation_port := "18080"
default_mcp_url := "http://localhost:18080/mcp?token=sap-odata-mcp-validation-token"
default_health_url := "http://localhost:18080/health"

default:
    @echo "Available: test, health, mcp-init, mcp-tools-list, mcp-conformance, mcp-inspector-tools, mcp-eval, mcp-eval-integration, pwsh-version, pwsh-smoke, smoke"

test:
    go test ./...

health:
    curl -fsS "${HEALTH_URL:-{{default_health_url}}}"

mcp-init:
    #!/usr/bin/env bash
    set -euo pipefail
    token="${SAP_ODATA_MCP_TOKEN:-{{validation_token}}}"
    port="${SAP_ODATA_MCP_PORT:-{{default_validation_port}}}"
    url="${MCP_URL:-http://localhost:${port}/mcp?token=${token}}"
    node "{{conformance}}" server --url "$url" --scenario server-initialize --output-dir "${MCP_CONFORMANCE_RESULTS:-/tmp/sap-odata-mcp-conformance}"

mcp-tools-list:
    #!/usr/bin/env bash
    set -euo pipefail
    token="${SAP_ODATA_MCP_TOKEN:-{{validation_token}}}"
    port="${SAP_ODATA_MCP_PORT:-{{default_validation_port}}}"
    url="${MCP_URL:-http://localhost:${port}/mcp?token=${token}}"
    node "{{conformance}}" server --url "$url" --scenario tools-list --output-dir "${MCP_CONFORMANCE_RESULTS:-/tmp/sap-odata-mcp-conformance}"

mcp-conformance: mcp-init mcp-tools-list

mcp-inspector-tools:
    #!/usr/bin/env bash
    set -euo pipefail
    token="${SAP_ODATA_MCP_TOKEN:-{{validation_token}}}"
    port="${SAP_ODATA_MCP_PORT:-{{default_validation_port}}}"
    url="${MCP_URL:-http://localhost:${port}/mcp}"
    node "{{inspector}}" --transport http --header "Authorization: Bearer ${token}" --method tools/list "$url"

mcp-eval path="tests/mcp-eval":
    #!/usr/bin/env bash
    set -euo pipefail
    project_dir="$PWD"
    tmpdir="$(mktemp -d)"
    cleanup() {
      if [ -n "${server_pid:-}" ]; then
        kill "$server_pid" >/dev/null 2>&1 || true
        wait "$server_pid" >/dev/null 2>&1 || true
      fi
      rm -rf "$tmpdir"
    }
    trap cleanup EXIT

    token="${SAP_ODATA_MCP_TOKEN:-{{validation_token}}}"
    port="${SAP_ODATA_MCP_PORT:-}"
    if [ -z "$port" ]; then
      port="$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()')"
    fi

    go build -o "$tmpdir/sap-odata-mcp-universal" ./cmd/sap-odata-mcp-universal
    ODATA_MCP_STATE_FILE="$tmpdir/odata_state.json" "$tmpdir/sap-odata-mcp-universal" \
      --transport streamable-http \
      --http-addr "localhost:${port}" \
      --mcp-token "$token" >"$tmpdir/server.log" 2>&1 &
    server_pid="$!"

    for _ in $(seq 1 100); do
      if curl -fsS "http://localhost:${port}/health" >/dev/null 2>&1; then
        break
      fi
      if ! kill -0 "$server_pid" >/dev/null 2>&1; then
        cat "$tmpdir/server.log" >&2
        exit 1
      fi
      sleep 0.1
    done
    curl -fsS "http://localhost:${port}/health" >/dev/null

    mkdir -p "$project_dir/test-reports"
    cd "{{mcpeval_dir}}"
    MCP_URL="http://localhost:${port}/mcp" \
      HEALTH_URL="http://localhost:${port}/health" \
      DASHBOARD_URL="http://localhost:${port}/dashboard" \
      SAP_ODATA_MCP_TOKEN="$token" \
      uv run mcp-eval run "$project_dir/{{path}}" --json "$project_dir/test-reports/mcp-eval-safe.json"

mcp-eval-integration path="tests/mcp-eval-integration":
    #!/usr/bin/env bash
    set -euo pipefail
    if [ -z "${SAP_ODATA_INTEGRATION_URL:-}" ]; then
      echo "Skipping integration eval: SAP_ODATA_INTEGRATION_URL is not set."
      exit 0
    fi

    project_dir="$PWD"
    tmpdir="$(mktemp -d)"
    cleanup() {
      if [ -n "${server_pid:-}" ]; then
        kill "$server_pid" >/dev/null 2>&1 || true
        wait "$server_pid" >/dev/null 2>&1 || true
      fi
      rm -rf "$tmpdir"
    }
    trap cleanup EXIT

    token="${SAP_ODATA_MCP_TOKEN:-{{validation_token}}}"
    port="${SAP_ODATA_MCP_PORT:-}"
    if [ -z "$port" ]; then
      port="$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()')"
    fi

    go build -o "$tmpdir/sap-odata-mcp-universal" ./cmd/sap-odata-mcp-universal
    args=(
      --transport streamable-http
      --http-addr "localhost:${port}"
      --mcp-token "$token"
      --service "$SAP_ODATA_INTEGRATION_URL"
      --read-only
    )
    if [ -n "${SAP_ODATA_USERNAME:-${ODATA_USERNAME:-}}" ]; then
      args+=(--user "${SAP_ODATA_USERNAME:-${ODATA_USERNAME:-}}")
    fi
    if [ -n "${SAP_ODATA_PASSWORD:-${ODATA_PASSWORD:-}}" ]; then
      args+=(--password "${SAP_ODATA_PASSWORD:-${ODATA_PASSWORD:-}}")
    fi
    ODATA_MCP_STATE_FILE="$tmpdir/odata_state.json" "$tmpdir/sap-odata-mcp-universal" "${args[@]}" >"$tmpdir/server.log" 2>&1 &
    server_pid="$!"

    for _ in $(seq 1 150); do
      if curl -fsS "http://localhost:${port}/health" >/dev/null 2>&1; then
        break
      fi
      if ! kill -0 "$server_pid" >/dev/null 2>&1; then
        cat "$tmpdir/server.log" >&2
        exit 1
      fi
      sleep 0.2
    done
    curl -fsS "http://localhost:${port}/health" >/dev/null

    mkdir -p "$project_dir/test-reports"
    cd "{{mcpeval_dir}}"
    MCP_URL="http://localhost:${port}/mcp" \
      HEALTH_URL="http://localhost:${port}/health" \
      DASHBOARD_URL="http://localhost:${port}/dashboard" \
      SAP_ODATA_MCP_TOKEN="$token" \
      SAP_ODATA_MCP_ALLOW_WRITES="${SAP_ODATA_MCP_ALLOW_WRITES:-false}" \
      uv run mcp-eval run "$project_dir/{{path}}" --json "$project_dir/test-reports/mcp-eval-integration.json"

pwsh-version:
    @"{{pwsh}}" -NoLogo -NoProfile -Command '$PSVersionTable.PSVersion.ToString()'

pwsh-smoke:
    @"{{pwsh}}" -NoLogo -NoProfile -File tests/smoke/mcp-smoke.ps1

smoke:
    #!/usr/bin/env bash
    set -euo pipefail
    project_dir="$PWD"
    tmpdir="$(mktemp -d)"
    cleanup() {
      if [ -n "${server_pid:-}" ]; then
        kill "$server_pid" >/dev/null 2>&1 || true
        wait "$server_pid" >/dev/null 2>&1 || true
      fi
      rm -rf "$tmpdir"
    }
    trap cleanup EXIT

    token="${SAP_ODATA_MCP_TOKEN:-{{validation_token}}}"
    port="${SAP_ODATA_MCP_PORT:-}"
    if [ -z "$port" ]; then
      port="$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()')"
    fi

    go build -o "$tmpdir/sap-odata-mcp-universal" ./cmd/sap-odata-mcp-universal
    ODATA_MCP_STATE_FILE="$tmpdir/odata_state.json" "$tmpdir/sap-odata-mcp-universal" \
      --transport streamable-http \
      --http-addr "localhost:${port}" \
      --mcp-token "$token" >"$tmpdir/server.log" 2>&1 &
    server_pid="$!"

    for _ in $(seq 1 100); do
      if curl -fsS "http://localhost:${port}/health" >/dev/null 2>&1; then
        break
      fi
      if ! kill -0 "$server_pid" >/dev/null 2>&1; then
        cat "$tmpdir/server.log" >&2
        exit 1
      fi
      sleep 0.1
    done
    curl -fsS "http://localhost:${port}/health" >/dev/null

    HEALTH_URL="http://localhost:${port}/health" curl -fsS "http://localhost:${port}/health" >/dev/null
    node "{{conformance}}" server --url "http://localhost:${port}/mcp?token=${token}" --scenario server-initialize --output-dir "${MCP_CONFORMANCE_RESULTS:-/tmp/sap-odata-mcp-conformance}"
    node "{{conformance}}" server --url "http://localhost:${port}/mcp?token=${token}" --scenario tools-list --output-dir "${MCP_CONFORMANCE_RESULTS:-/tmp/sap-odata-mcp-conformance}"
    node "{{inspector}}" --transport http --header "Authorization: Bearer ${token}" --method tools/list "http://localhost:${port}/mcp"
    "{{pwsh}}" -NoLogo -NoProfile -File tests/smoke/mcp-smoke.ps1
    mkdir -p "$project_dir/test-reports"
    cd "{{mcpeval_dir}}"
    MCP_URL="http://localhost:${port}/mcp" \
      HEALTH_URL="http://localhost:${port}/health" \
      DASHBOARD_URL="http://localhost:${port}/dashboard" \
      SAP_ODATA_MCP_TOKEN="$token" \
      uv run mcp-eval run "$project_dir/tests/mcp-eval" --json "$project_dir/test-reports/mcp-eval-safe.json"
