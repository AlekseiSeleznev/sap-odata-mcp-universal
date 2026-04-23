# SAP OData Hierarchical System → Entity → Operation Dashboard Plan

> **Superpowers contract:** This document is the design and execution contract for the next dashboard/backend rework. It defines the target model where one SAP system profile can aggregate multiple OData services and expose them as business entities with hierarchical operations.

**Goal:** Replace the current `one active OData service = one MCP profile` model with a hierarchical model:

- `System` (for example `S4D`, client `100`)
- `Entity` (for example `Materials`)
- `Operations` (`GET`, `LIST`, `POST`, `PATCH`, `DELETE`, `ACTION`)

The key requirement is that one business entity can route different operations to different SAP OData services. Example:

- `Materials.GET` -> `MMIM_MATERIAL_DATA_SRV / MaterialHeaders`
- `Materials.POST` -> `API_PRODUCT_SRV / A_Product`

That lets the dashboard represent business intent instead of raw SAP OData service fragmentation.

**Architecture direction:** Introduce a hierarchical registry, a multi-service system runtime, and business-level MCP tool generation. The dashboard becomes a tree-based configuration UI rather than a flat service list.

**Tech Stack:** Go 1.21+, `net/http`, existing bridge/client packages, JSON persistence, dashboard HTML/CSS/JS, Go tests.

---

## Problem Statement

The current model works like this:

- one saved profile = one OData service root
- one active profile = one active service root
- tools are generated directly from one metadata document

This breaks down for real SAP usage because:

- reading and creating the "same" business object often use different OData services
- users think in terms of `System -> Business entity -> Operation`, not `service root -> entity set`
- the dashboard cannot express "Materials GET comes from service A, Materials POST comes from service B"

As a result, the current dashboard is still too OData-centric and not business-centric enough.

---

## Target Mental Model

### Example hierarchy

```text
S4D / Client 100
  Materials
    GET     -> MMIM_MATERIAL_DATA_SRV.MaterialHeaders
    LIST    -> MMIM_MATERIAL_DATA_SRV.MaterialHeaders
    POST    -> API_PRODUCT_SRV.A_Product
    PATCH   -> API_PRODUCT_SRV.A_Product
    DELETE  -> API_PRODUCT_SRV.A_Product
  Business Partners
    GET     -> API_BUSINESS_PARTNER.A_BusinessPartner
    LIST    -> API_BUSINESS_PARTNER.A_BusinessPartner
    POST    -> API_BUSINESS_PARTNER.A_BusinessPartner
```

### What the user configures

1. A SAP system profile with shared connection/auth parameters.
2. One or more business entities under that system.
3. One or more operation bindings under each entity.

### What MCP exposes

Instead of raw OData-service-specific tools, MCP should expose business-level tools such as:

- `materials_get_for_s4d`
- `materials_list_for_s4d`
- `materials_create_for_s4d`
- `business_partners_get_for_s4d`

Internally, each tool routes to the correct OData service binding.

---

## Architecture Overview

```text
Dashboard
  System tree
    System profile
      Entity profile
        Operation binding

        |
        v

Hierarchical Registry (JSON)
  systems[]
    entities[]
      operations[]

        |
        v

System Runtime
  shared auth/client/session per system
  metadata cache per service root
  operation router

        |
        v

Business-level MCP Tool Layer
  materials_get_for_s4d
  materials_create_for_s4d
  ...

        |
        v

OData clients
  MMIM_MATERIAL_DATA_SRV
  API_PRODUCT_SRV
  API_BUSINESS_PARTNER
  ...
```

---

## Core Data Model

### 1. System profile

Represents one SAP landscape + client + shared credentials.

Suggested shape:

```json
{
  "id": "s4d-100",
  "name": "S4D",
  "client": "100",
  "base_url": "http://s4d.msgplaut.com:8000",
  "username": "ASELEZNEV",
  "password": "...",
  "entities": []
}
```

Semantics:

- one system profile should own auth/session/cookies
- service roots beneath it inherit these credentials
- `sap-client` belongs at system level by default, with optional per-operation override

### 2. Entity profile

Represents one business concept, not one raw OData service.

Suggested shape:

```json
{
  "id": "materials",
  "label": "Materials",
  "description": "Material master operations",
  "operations": []
}
```

Semantics:

- stable business name shown in dashboard and MCP
- may aggregate multiple OData services
- becomes the parent node for operations

### 3. Operation binding

Represents one business operation mapped to one concrete OData binding.

Suggested shape:

```json
{
  "id": "materials-create",
  "verb": "create",
  "service_root": "/sap/opu/odata/sap/API_PRODUCT_SRV/",
  "entity_set": "A_Product",
  "mode": "generated",
  "enabled": true,
  "read_only": false,
  "payload_profile": "material-create-basic"
}
```

Important fields:

- `verb`: `get`, `list`, `create`, `update`, `delete`, `action`
- `service_root`: concrete service under the system
- `entity_set`: concrete OData entity set
- `mode`:
  - `generated` = generic bridge-generated passthrough
  - `composite` = custom multi-step or deep-insert flow
- `payload_profile`: optional preset for UI/forms/validation

---

## Runtime / Backend Design

### 1. Replace flat registry with hierarchical registry

Current registry stores flat connections only.

Target registry must store:

- systems
- entity nodes under systems
- operation bindings under entities
- active system
- optionally enabled entity set / tool visibility state

New persistence file should be versioned:

```json
{
  "schema_version": 2,
  "active_system": "s4d-100",
  "systems": [...]
}
```

### 2. Introduce system runtime instead of single active service

Current bridge runtime assumes one active OData service.

Target runtime should:

- keep one shared auth/session context per active system
- lazily create OData clients per `service_root`
- cache metadata per service root
- expose an operation router keyed by `system + entity + verb`

### 3. Multi-service metadata cache

For one active system:

- `MMIM_MATERIAL_DATA_SRV` metadata
- `API_PRODUCT_SRV` metadata
- other service metadata

should coexist in memory at the same time.

Required cache key:

- `system_id + service_root`

### 4. Business-level MCP tool generation

The MCP layer should no longer publish only raw OData entity names.

Instead:

- one bound operation => one MCP tool
- name generated from system + entity + verb

Examples:

- `materials_get_for_s4d`
- `materials_list_for_s4d`
- `materials_create_for_s4d`

Tool descriptions should still include the underlying mapping:

- system
- service root
- entity set
- access mode

### 5. Operation execution modes

#### Generated mode

For straightforward CRUD/list/get operations:

- reuse existing bridge handlers
- route to the bound `service_root` and `entity_set`

#### Composite mode

For SAP scenarios like deep insert or post-create follow-up:

- allow custom operation pipelines
- example: `Materials.POST`
  - POST `A_Product`
  - optionally POST or verify description
  - normalize response into one business result

This is the right place for cases where NiFi currently does a CSRF dance + custom HTTP chain.

### 6. Shared auth and CSRF strategy

CSRF and cookies should become system-runtime responsibilities:

- credentials stored once at system level
- client/cache reused across operation bindings
- modifying operations fetch CSRF automatically through the bound service root

Important nuance:

- CSRF token is service-specific, not just system-specific
- so runtime should cache tokens per `system + service_root`

---

## Dashboard / UX Design

### 1. Replace flat list with hierarchical tree

Current left panel: flat service profiles.

Target left panel:

```text
S4D / 100
  Materials
    GET
    LIST
    POST
  Business Partners
    GET
    POST
```

Interaction:

- expand/collapse system nodes
- expand/collapse entity nodes
- select an operation node to edit its binding

### 2. Two-pane editor model

Recommended layout:

- left pane: hierarchy tree
- right pane: detail editor for selected node

Detail editor changes depending on selection:

- system selected -> system form
- entity selected -> entity form
- operation selected -> operation binding form

### 3. Create flow

Recommended wizard flow:

1. Create system
   - system name
   - client
   - base host
   - login/password
2. Add entity
   - business label
   - slug/tool prefix
3. Add operation
   - verb
   - service root
   - entity set
   - generated/composite mode
   - read/write semantics

### 4. Operation binding editor

Each operation should expose:

- operation verb
- service picker or service URL
- entity set picker
- capability preview from metadata
- payload strategy:
  - generic passthrough
  - preset payload profile
  - composite flow

### 5. Metadata-assisted UX

When the user enters a service root:

- dashboard should fetch metadata
- show available entity sets
- indicate if the entity set supports create/update/delete/search
- help bind the right entity set to the desired business operation

This reduces manual errors and makes the hierarchy manageable.

### 6. Recommended visual hierarchy

Top node:

- `S4D`
- small badge: `client 100`
- status: connected / disconnected

Second level:

- `Materials`
- badge: entity

Third level:

- `GET`
- `LIST`
- `POST`
- `PATCH`
- `DELETE`
- each line shows bound service + entity set in muted text

---

## Example for the current SAP case

### System

```text
S4D / client 100
```

### Entity

```text
Materials
```

### Operations

```text
GET   -> MMIM_MATERIAL_DATA_SRV.MaterialHeaders
LIST  -> MMIM_MATERIAL_DATA_SRV.MaterialHeaders
POST  -> API_PRODUCT_SRV.A_Product
PATCH -> API_PRODUCT_SRV.A_Product
```

Optional composite enhancement:

```text
POST -> API_PRODUCT_SRV.A_Product
      payload profile: material-basic-with-description
      deep insert: to_Description
```

This is the business-centric representation the user actually expects.

---

## Migration Strategy

### Phase 1: Data model and registry

Introduce the hierarchical registry without removing the flat one immediately.

Add migration:

- old flat connection -> new system with one synthetic entity and one synthetic operation set

This keeps backward compatibility while the dashboard transitions.

### Phase 2: Runtime and routing

Add multi-service runtime and business-level operation router.

Keep the old single-service bridge available behind a compatibility layer.

### Phase 3: New dashboard tree

Switch dashboard UI from flat list to hierarchical editor.

### Phase 4: Composite operations

Add first-class composite mode for cases like:

- product creation with description
- custom SAP create flows
- post-create verification

---

## Risks and Decisions

### Decision 1: `System` must own credentials

Reason:

- auth belongs to the SAP landscape/client, not to every operation binding

### Decision 2: `Entity` must be business-level, not OData-level

Reason:

- users think in "Materials", "Customers", "Orders"
- raw service names are implementation details

### Decision 3: `Operation` must bind to one concrete service root

Reason:

- execution must still resolve to one exact OData backend

### Decision 4: composite flows are necessary

Reason:

- some SAP business operations cannot be represented well by one generic create tool schema
- deep insert and multi-step creation need explicit support

### Risk 1: Tool-name stability

Moving from raw OData names to business names changes MCP tool names.

Mitigation:

- keep compatibility aliases for one release
- optionally expose both raw and business naming during transition

### Risk 2: Metadata explosion for many services

One active system may load many metadata docs.

Mitigation:

- lazy-load per bound operation/service
- cache per service root
- do not eagerly fetch everything on startup

### Risk 3: UI complexity

The tree editor is more complex than the current connection card.

Mitigation:

- introduce it in layers: systems first, then entities, then operation bindings

---

## Execution Tasks

### Task 1: Define the hierarchical registry

**Files:**
- Create: `internal/dashboard/hierarchy.go`
- Create: `internal/dashboard/migration.go`
- Modify: `internal/dashboard/registry.go`

- [ ] Add `SystemProfile`, `EntityProfile`, `OperationBinding` structs
- [ ] Add schema-versioned persistence format
- [ ] Add migration from flat connections to hierarchical systems
- [ ] Add tests for load/save/migrate flows

### Task 2: Add multi-service runtime

**Files:**
- Create: `internal/runtime/system_runtime.go`
- Modify: `internal/bridge/bridge.go`
- Modify: `internal/client/*` as needed

- [ ] Introduce per-system runtime with shared auth
- [ ] Add per-service metadata cache
- [ ] Add per-service CSRF/cache state
- [ ] Add operation routing by business binding

### Task 3: Generate business-level MCP tools

**Files:**
- Modify: `internal/bridge/generators.go`
- Modify: `internal/bridge/handlers.go`
- Create: `internal/bridge/business_bindings.go`

- [ ] Generate tools from entity-operation bindings instead of one raw metadata doc
- [ ] Include underlying service/entity-set mapping in descriptions
- [ ] Keep optional compatibility aliases for legacy raw tool names

### Task 4: Support composite operations

**Files:**
- Create: `internal/bridge/composite_handlers.go`
- Create: `internal/models/payload_profiles.go`

- [ ] Add `generated` and `composite` operation modes
- [ ] Implement first composite profile for `Materials.POST`
- [ ] Support deep-insert payloads like `to_Description`

### Task 5: Replace the dashboard UI with hierarchy editor

**Files:**
- Modify: `internal/dashboard/content.go`
- Modify: `internal/dashboard/server.go`
- Modify: `internal/dashboard/provider.go`
- Modify: `internal/dashboard/server_test.go`

- [ ] Replace flat system list with tree view
- [ ] Add editor panels for system/entity/operation nodes
- [ ] Add metadata-assisted service/entity-set binding UI
- [ ] Keep RU/EN support and the current visual language

### Task 6: Documentation and migration guidance

**Files:**
- Modify: `README.md`
- Modify: `internal/dashboard/content.go`
- Create: `docs/009-hierarchical-system-entity-model.md`

- [ ] Document the new hierarchy model
- [ ] Explain migration from flat service profiles
- [ ] Add examples for `Materials GET via MMIM_MATERIAL_DATA_SRV` and `Materials POST via API_PRODUCT_SRV`

---

## Approval Gate

This design should be approved before implementation starts because it changes:

- persistence model
- dashboard mental model
- MCP tool naming strategy
- runtime architecture

Once approved, implementation should proceed in phases, not as one large cutover.
