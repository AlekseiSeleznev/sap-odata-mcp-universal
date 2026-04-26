import json
import os
import urllib.request

from mcp_eval import task


MCP_URL = os.environ.get("MCP_URL", "http://localhost:18080/mcp")
HEALTH_URL = os.environ.get("HEALTH_URL", "http://localhost:18080/health")
TOKEN = os.environ.get("SAP_ODATA_MCP_TOKEN", "sap-odata-mcp-validation-token")
INTEGRATION_URL = os.environ.get("SAP_ODATA_INTEGRATION_URL", "")
ALLOW_WRITES = os.environ.get("SAP_ODATA_MCP_ALLOW_WRITES", "").lower() == "true"


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
    with urllib.request.urlopen(req, timeout=20) as response:
        raw = response.read().decode("utf-8")
        return response.status, json.loads(raw) if raw else {}


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
    return payload


@task("Integration profile is explicitly gated by SAP_ODATA_INTEGRATION_URL.")
async def test_integration_profile_is_gated(session):
    _touch_trace(session)
    if not INTEGRATION_URL:
        return
    assert INTEGRATION_URL.startswith("http://") or INTEGRATION_URL.startswith("https://")


@task("Read-only integration server starts and exposes tools when fixtures are configured.")
async def test_integration_tools_visible_readonly(session):
    _touch_trace(session)
    if not INTEGRATION_URL:
        return

    status, health = _request_json(HEALTH_URL)
    assert status == 200, health
    assert health.get("status") == "ok", health

    initialized = _mcp(
        "initialize",
        {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "mcp-eval-integration", "version": "1.0.0"},
        },
    )
    assert initialized.get("result", {}).get("serverInfo", {}).get("name") == "sap-odata-mcp-universal"

    listed = _mcp("tools/list", {}, request_id=2)
    tools = listed.get("result", {}).get("tools")
    assert isinstance(tools, list), listed
    assert tools, "integration profile should expose at least service_info/read tools"

    if not ALLOW_WRITES:
        names = [tool.get("name", "").lower() for tool in tools]
        write_markers = ["create", "update", "delete", "patch", "post", "put"]
        assert not any(any(marker in name for marker in write_markers) for name in names), names
