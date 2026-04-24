# Dashboard Review Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the dashboard review findings for sap-odata-mcp-universal: real HTTP auth enforcement, safer runtime behavior, better dashboard UX, and consistent object/service management.

**Architecture:** Keep the existing hierarchical System -> Object -> Operation model. Add transport-level token enforcement for MCP/dashboard API, expose password presence without leaking passwords, make system test cover all configured services, and remove stale frontend service-editor paths so edit happens through modal windows while the right pane remains creation-only.

**Tech Stack:** Go HTTP transports, dashboard provider/runtime, embedded HTML/CSS/JS in `internal/dashboard/content.go`, Go tests and local curl smoke tests.

---

### Task 1: Enforce HTTP token on protected endpoints
- [x] Add request token extraction and validation helpers in `internal/transport/http/security.go`.
- [x] Pass the actual token from `cmd/sap-odata-mcp-universal/main.go` into streamable HTTP and SSE transports.
- [x] Protect `/mcp`, `/rpc`, `/sse`, and `/api/*`; keep `/health` public.
- [x] Add unit tests for Authorization, X-MCP-Token, query token, and rejection.

### Task 2: Fix dashboard API contracts
- [x] Add `has_password` to dashboard system DTO.
- [x] Replace single-service connection test result with system-level result containing per-service statuses.
- [x] Return HTTP 400 for provider mutation results where `ok=false`.
- [x] Update fake provider/tests for the new contract.

### Task 3: Make runtime metadata fetches safer
- [x] Change runtime discovery and apply calls to pass request context.
- [x] Avoid holding the runtime mutex while SAP metadata is fetched over network.
- [x] Keep access-mode changes as cheap in-memory updates, without runtime rebuild.

### Task 4: Clean dashboard frontend behavior
- [x] Add dashboard token UX and send Authorization headers from `api()`.
- [x] Show password placeholder based on `has_password`, never on the omitted password field.
- [x] Remove stale right-pane service editor functions and service cards.
- [x] Keep service management in the system modal and creation-only editor on the right.
- [x] Keep toasts centered and object terminology consistent in RU/EN.

### Task 5: Verify end-to-end
- [x] Run `gofmt`.
- [x] Run targeted dashboard/transport tests.
- [x] Run `go test ./...`.
- [x] Start a local token-protected server and verify `/health`, unauthorized `/api/systems`, authorized `/api/systems`, unauthorized `/mcp`, authorized `/mcp`, and dashboard HTML.
