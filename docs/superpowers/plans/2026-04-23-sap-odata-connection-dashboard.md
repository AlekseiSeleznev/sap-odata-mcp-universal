# SAP OData Connection Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current status-oriented dashboard with a postgres-universal style SAP OData connection manager that supports multiple saved systems, login/password authentication, RU/EN UI switching, and detailed bilingual documentation.

**Architecture:** Introduce a persistent OData connection registry and let dashboard actions connect, edit, remove, and switch the active SAP OData system. The bridge remains the MCP engine, but dashboard actions reconfigure the active client, metadata, and generated tools dynamically so the MCP server follows the selected connection.

**Tech Stack:** Go 1.21+, `net/http`, embedded HTML/CSS/JS, JSON file persistence, existing OData client/bridge, Go tests.

---

### Task 1: Capture postgres-universal dashboard parity

**Files:**
- Create: `docs/superpowers/plans/2026-04-23-sap-odata-connection-dashboard.md`
- Reference: `/home/as/Документы/AI_PROJECTS/postgres-mcp-universal/gateway/gateway/web_ui_content.py`
- Reference: `/home/as/Документы/AI_PROJECTS/postgres-mcp-universal/gateway/gateway/web_ui.py`

- [ ] **Step 1: Keep the plan in-repo**

Confirm this plan exists and is used as the implementation contract for the rework.

- [ ] **Step 2: Match the postgres-universal dashboard shell**

Mirror the visual structure of postgres-universal:
- same header layout
- same RU/EN switcher placement
- same docs and refresh button placement
- same two-column card layout
- same form and modal styling

- [ ] **Step 3: Map PostgreSQL concepts to SAP OData**

Translate the UI semantics:
- databases list → SAP OData systems list
- connection string fields → OData connection fields
- allow writes toggle → read/write vs read-only mode for the connection

### Task 2: Add persistent SAP OData connection registry

**Files:**
- Create: `internal/dashboard/registry.go`
- Create: `internal/dashboard/service.go`
- Test: `internal/dashboard/server_test.go`

- [ ] **Step 1: Define persistent connection model**

Create a registry model storing:
- `name`
- `system_name`
- `service_url`
- `username`
- `password`
- `access_mode`
- `connected`

- [ ] **Step 2: Add JSON persistence**

Persist registry state to a JSON file with:
- active/default connection name
- connection list without transient runtime-only fields where appropriate

- [ ] **Step 3: Add safe redaction helpers**

Provide helpers so dashboard responses never echo the password while still showing a useful redacted URL or summary.

- [ ] **Step 4: Validate registry behavior**

Run: `go test ./internal/dashboard -v`
Expected: PASS with coverage for save/load/list/add/remove/switch flows.

### Task 3: Make the bridge switch active SAP OData connection dynamically

**Files:**
- Modify: `internal/bridge/bridge.go`
- Modify: `cmd/odata-mcp/main.go`
- Modify: `internal/mcp/server.go` (only if needed)
- Test: `internal/dashboard/server_test.go`

- [ ] **Step 1: Allow HTTP startup without preconfigured service URL**

Adjust startup so HTTP dashboard mode can launch even when no initial OData service URL is supplied. Keep stdio mode strict unless a service is configured.

- [ ] **Step 2: Add bridge reconfiguration method**

Add a method that:
- applies new config/auth
- fetches metadata from the chosen SAP OData service
- clears old MCP tools
- regenerates tools for the active connection

- [ ] **Step 3: Preserve backward compatibility**

If the user still starts the binary with a single `service-url`, keep the old one-service flow working.

- [ ] **Step 4: Verify dynamic switching**

Run: `go test ./...`
Expected: PASS

### Task 4: Replace dashboard routes/UI/docs with postgres-style connection management

**Files:**
- Modify: `internal/dashboard/server.go`
- Modify: `internal/dashboard/assets.go`
- Modify: `internal/dashboard/server_test.go`
- Modify: `internal/dashboard/static/index.html`
- Modify: `internal/dashboard/static/app.js`
- Modify: `internal/dashboard/static/style.css`

- [ ] **Step 1: Replace the current dashboard API contract**

Expose dashboard APIs for:
- list systems
- connect/add system
- edit system
- delete/disconnect system
- switch default system

- [ ] **Step 2: Match postgres-universal layout and interaction model**

Keep the same:
- header structure
- cards
- list item layout
- buttons
- toasts
- confirmation modal
- edit modal

- [ ] **Step 3: Add RU/EN localization**

Localize:
- dashboard labels
- button text
- validation messages
- success/error toasts
- docs page content

- [ ] **Step 4: Keep docs detailed in both languages**

Create a detailed `/dashboard/docs` page in Russian and English that explains:
- what the gateway does
- how to add SAP OData systems
- how active connection switching works
- how read-only mode affects MCP operations
- how to connect from Codex and other MCP clients
- all dashboard/API endpoints

### Task 5: Update README and validate full flow

**Files:**
- Modify: `README.md`
- Test: `internal/dashboard/server_test.go`

- [ ] **Step 1: Update README**

Document the new dashboard as a connection manager rather than a metadata/status viewer.

- [ ] **Step 2: Add tests for RU/EN rendering and connection APIs**

Cover:
- dashboard page rendering
- docs rendering in both languages
- connect/edit/delete/switch APIs
- safe handling of passwords in HTML and JSON responses

- [ ] **Step 3: Run full validation**

Run: `go test ./...`
Expected: PASS
