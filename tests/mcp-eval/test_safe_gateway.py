import json
import os
import urllib.error
import urllib.request
from urllib.parse import urlsplit, urlunsplit

from mcp_eval import task


MCP_URL = os.environ.get("MCP_URL", "http://localhost:18080/mcp")
HEALTH_URL = os.environ.get("HEALTH_URL", "http://localhost:18080/health")
DASHBOARD_URL = os.environ.get("DASHBOARD_URL", "http://localhost:18080/dashboard")
TOKEN = os.environ.get("SAP_ODATA_MCP_TOKEN", "sap-odata-mcp-validation-token")


def _touch_trace(session) -> None:
    trace_file = getattr(session, "trace_file", None)
    if trace_file:
        open(trace_file, "a", encoding="utf-8").close()


def _request_json(url: str, method: str = "GET", body: dict | None = None) -> tuple[int, object]:
    data = None
    headers = {"Accept": "application/json"}
    if body is not None:
        data = json.dumps(body).encode("utf-8")
        headers["Content-Type"] = "application/json"
    if TOKEN:
        headers["Authorization"] = f"Bearer {TOKEN}"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=10) as response:
            raw = response.read().decode("utf-8")
            return response.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8")
        try:
            payload = json.loads(raw)
        except json.JSONDecodeError:
            payload = {"error": raw.strip()}
        return exc.code, payload


def _mcp(method: str, params: dict | None = None, request_id: int = 1) -> dict:
    status, payload = _request_json(
        MCP_URL,
        method="POST",
        body={
            "jsonrpc": "2.0",
            "id": request_id,
            "method": method,
            "params": params or {},
        },
    )
    assert status == 200, payload
    assert isinstance(payload, dict), payload
    assert payload.get("jsonrpc") == "2.0", payload
    return payload


def _dashboard_api_url(path: str) -> str:
    parsed = urlsplit(DASHBOARD_URL)
    return urlunsplit((parsed.scheme, parsed.netloc, path, "", ""))


@task("MCP gateway health endpoint is reachable in the safe dashboard-first profile.")
async def test_gateway_health(session):
    _touch_trace(session)
    status, payload = _request_json(HEALTH_URL)
    assert status == 200, payload
    assert isinstance(payload, dict), payload
    assert payload.get("status") == "ok", payload
    assert payload.get("transport") == "streamable-http", payload


@task("MCP initialize succeeds without a real SAP backend.")
async def test_mcp_server_available(session):
    _touch_trace(session)
    payload = _mcp(
        "initialize",
        {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "mcp-eval-safe", "version": "1.0.0"},
        },
    )
    result = payload.get("result", {})
    assert result.get("protocolVersion"), payload
    assert "tools" in result.get("capabilities", {}), payload
    assert result.get("serverInfo", {}).get("name") == "sap-odata-mcp-universal", payload


@task("Agent-facing tools/list is visible and honest in safe mode.")
async def test_agent_sees_tools_list(session):
    _touch_trace(session)
    payload = _mcp("tools/list", {}, request_id=2)
    tools = payload.get("result", {}).get("tools")
    assert isinstance(tools, list), payload
    for tool in tools:
        assert "name" in tool, tool
        assert "inputSchema" in tool, tool


@task("Safe mode must not invent SAP systems, hosts, secrets, or backend inventory.")
async def test_safe_profile_does_not_fabricate_backends_or_secrets(session):
    _touch_trace(session)
    status, systems = _request_json(_dashboard_api_url("/api/systems"))
    assert status == 200, systems
    assert systems == [], systems
    serialized = json.dumps(systems, ensure_ascii=False).lower()
    forbidden = ["password", "secret", "sap_sessionid", "mysapsso2", "postgres://", "ssh://", "nifi"]
    assert not any(item in serialized for item in forbidden), serialized


@task("Unavailable backend/tool errors are explicit and not fabricated as successful results.")
async def test_missing_backend_reports_error_without_fabrication(session):
    _touch_trace(session)
    payload = _mcp(
        "tools/call",
        {"name": "definitely_missing_backend_tool", "arguments": {}},
        request_id=3,
    )
    assert "error" in payload, payload
    error = payload["error"]
    assert error.get("code") == -32602, payload
    assert "Tool not found" in json.dumps(error), payload
    assert "content" not in payload.get("result", {}), payload


@task("Discovery flow fails closed before action when no SAP system/service exists.")
async def test_discovery_flow_required_before_backend_action(session):
    _touch_trace(session)
    status, payload = _request_json(
        _dashboard_api_url("/api/service/discover") + "?system_id=missing&service_id=missing"
    )
    assert status >= 400, payload
    text = json.dumps(payload, ensure_ascii=False).lower()
    assert "not found" in text or "required" in text, payload


@task("Write-capable operations are absent in safe mode without configured backend.")
async def test_no_destructive_tools_in_safe_profile(session):
    _touch_trace(session)
    payload = _mcp("tools/list", {}, request_id=4)
    tools = payload.get("result", {}).get("tools", [])
    names = [tool.get("name", "").lower() for tool in tools]
    destructive_markers = ["create", "update", "delete", "patch", "post", "put", "drop", "write"]
    assert not any(any(marker in name for marker in destructive_markers) for name in names), names
