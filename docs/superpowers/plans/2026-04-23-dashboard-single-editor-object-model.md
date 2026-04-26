# Dashboard Single Editor Object Model Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current multi-card dashboard with a single dynamic editor on the right, move creation actions into the left hierarchy, and rename all user-facing `entity` terminology to `object`.

**Architecture:** Keep the backend API and internal `entity` model unchanged for now, but refactor the frontend renderer in `internal/dashboard/content.go` so the left tree becomes the primary navigation and action surface. The right side becomes one contextual editor that switches between system, object, and operation modes, while system editing also embeds the service catalog because services belong to the selected system.

**Tech Stack:** Go embedded HTML/JS template in `internal/dashboard/content.go`, existing `/api/system/*`, `/api/service/*`, `/api/entity/*`, `/api/operation/*` endpoints, local `go build` smoke validation.

---

### Task 1: Lock The New Interaction Model

**Files:**
- Modify: `internal/dashboard/content.go`
- Test: local `curl` against `/dashboard` and `/health`

- [ ] Replace the left toolbar so only `New system` remains global.
- [ ] Add contextual tree actions:
  `system -> new object / edit / activate / delete`
  `object -> new operation / edit / delete`
  `operation -> edit / delete`
- [ ] Replace user-facing labels `entity` -> `object` in RU and EN translations.

### Task 2: Replace The Right Column With One Dynamic Editor

**Files:**
- Modify: `internal/dashboard/content.go`

- [ ] Remove the static stack of four cards on the right.
- [ ] Add one `editor-pane` card that renders one of:
  `system editor`, `object editor`, `operation editor`, or an empty guidance state.
- [ ] Keep service management inside the system editor because services are scoped to a system.

### Task 3: Rewire Frontend State And Actions

**Files:**
- Modify: `internal/dashboard/content.go`

- [ ] Add a lightweight editor state so create/edit actions know which panel to render.
- [ ] Make tree selection open the corresponding editor instead of only changing hidden form values.
- [ ] Reuse the existing save/delete/discovery endpoints; avoid backend changes in this pass.

### Task 4: Extend Tree-Level Actions

**Files:**
- Modify: `internal/dashboard/content.go`

- [ ] Add `activate system` directly on the system row.
- [ ] Add per-service metadata refresh inside the system editor service list.
- [ ] Keep delete actions explicit and independent from the current form selection.

### Task 5: Update Embedded Documentation And Verify

**Files:**
- Modify: `internal/dashboard/content.go`

- [ ] Update embedded dashboard docs so they describe `System -> Object -> Operation`.
- [ ] Run `gofmt -w internal/dashboard/content.go`.
- [ ] Run `go build -o sap-odata-mcp-universal ./cmd/sap-odata-mcp-universal`.
- [ ] Restart the local server on `localhost:3000`.
- [ ] Verify `/health`, `/dashboard?lang=ru`, `/dashboard?lang=en`, and visible UI markers for the new single-editor layout.
