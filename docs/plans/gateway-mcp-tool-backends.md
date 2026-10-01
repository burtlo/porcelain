# Plan: Gateway MCP tool backends and developer tool suite

| Field | Value |
|-------|-------|
| **Doc kind** | `feature-plan` |
| **Owners / areas** | Gateway harness, supervisor, operator assistants, observability |
| **Status** | `shipped` (Phases 0–2c); Phases 3–4 `deferred` |
| **Targets** | Gateway v0.5 (MCP client + one stdio path; full developer suite optional) |
| **Last updated** | See git history |
| **Supersedes / superseded by** | Extends [`assistant-harness-workspace-tools.md`](assistant-harness-workspace-tools.md) (v0.4 native tools); deferred item in [`version-v0.4.md`](../version-v0.4.md#deferred-to-v05) |
| **As-built** | [`gateway-mcp-tool-backends.md`](../features/gateway-mcp-tool-backends.md) (`partial`); delivery record [`gateway-mcp-tool-backends-delivery.md`](gateway-mcp-tool-backends-delivery.md) |

## At a glance

Assistants on the turn harness can call **external Model Context Protocol (MCP) servers** for software development work (git, shell, browser, and similar) while **native workspace file tools** stay in the gateway. The gateway **does not** embed workflow or task-graph domain logic; it **manages connections**, **merges tool catalogs** with explicit slug mapping, and **executes** calls through the existing `ToolExecutor` hook in the `tool_executor` harness stage.

**Operator model-assisted configuration** (v0.5 desired-state train) is **out of scope** for this plan. It uses documented operator APIs only — see [`operator-model-assist-tools.md`](operator-model-assist-tools.md). Developer MCP allowlists **must not** attach to the model-assist assistant.

| Phase | Outcome | Status |
|-------|---------|--------|
| [Phase 0 — Contract and design](#phase-0--contract-and-design) | Platform contract + transport spike; harness rules frozen | `done` |
| [Phase 1 — Manager, catalog, composite executor](#phase-1--manager-catalog-composite-executor) | Gateway-owned stdio + HTTP upstream; YAML seed config; end-to-end `tools/call` on assistant path | `done` |
| [Phase 2a — Native RAG expansion tools](#phase-2a--native-rag-expansion-tools) | Expansion tools in harness loop (no MCP required) | `done` |
| [Phase 2b — One blessed MCP sidecar](#phase-2b--one-blessed-mcp-sidecar) | Documented YAML + harness call to one sidecar (e.g. git read-only or no-network echo) | `done` |
| [Phase 2c — Operator MCP settings](#phase-2c--operator-mcp-settings) | SQLite server rows + minimal UI (per-tool toggles, assistant bind) | `done` |
| [Phase 3 — Workflow MCP (task-orchestrator)](#phase-3--workflow-mcp-task-orchestrator) | Registration recipe; optional in-process fake for gate errors | `deferred` |
| [Phase 4 — Scale and IDE ingress](#phase-4--scale-and-ide-ingress) | Large-catalog meta-tools; HTTP MCP facade for IDEs | `deferred` |

**Depends on:** [assistant-turn-harness](assistant-turn-harness.md), [assistant-harness-workspace-tools](assistant-harness-workspace-tools.md) (`done`), [gateway chat routing pipeline](../features/gateway-chat-routing-pipeline.md)

**Design references (patterns only, not vendored code):** workspace `mcp-router` (manager / catalog / aggregator split), workspace `task-orchestrator` (tool definition + server-enforced workflow gates)

**Delivery gates:** `make precommit` when plan phases ship; update [gateway chat routing pipeline](../features/gateway-chat-routing-pipeline.md) and platform-contract [`gateway-mcp-tool-backends.md`](../features/gateway-mcp-tool-backends.md); operator log slugs via [operator log message registry](../features/operator-log-message-registry.md).

**Do not implement Phase 2b–2c or deferred phases until Phase 1 is green** with one **stdio** fake server on the real assistant path.

**Plan review:** [Grok 4.6 review](ddd00087-6554-42bb-9b27-bfa5c92428bd) (2026); prior exploration: [porcelain](8a8ba991-cbb4-4c40-beff-5eb1642cc585), [task-orchestrator](5f6fb2f4-3ffd-4741-9e04-68097ee783bf), [mcp-router](2854a2a3-c932-4a78-a7d1-bca204f3abf9).

### Reference project assessment

These repositories are design inputs and possible standalone MCP servers, not Phase 1 Go dependencies.

| Repository | Useful pattern | Current Go-port limit | Porcelain use |
|------------|----------------|-----------------------|---------------|
| `mcp-router/go-port` | Separate transport, server manager, merged catalog, collision mapping, and MCP aggregation | Packages are under Go `internal/`; its docs leave native SSE and meta-tools for later. It is an MCP-facing gateway, not a reusable client library. | Reproduce the small manager/catalog/client slice needed by the assistant harness. Porcelain may connect to a separately run router later as a remote MCP server. Do not import or vendor the app. |
| `task-orchestrator/go-port` | Tool definition interface, application services behind tools, and server-enforced workflow rules | Its `STATUS.md` reports only `manage_items` body-ported; 12 tools are stubs, HTTP transport is unimplemented, and actor verification is stubbed. | Treat task-orchestrator as an independent MCP server when ready. Do not port its workflow domain into the gateway or count the current Go port as a production-ready backend. |

The initial review was limited to source and documentation; these ports have not been validated as runtime dependencies. Recheck their status before a later integration decision.

---

## Background

v0.4 shipped a **gateway-native** workspace tool loop: `read_file`, `write_file`, `list_dir`, and substring `search` under workspace roots, gated by `file_action_policy`, executed inside the harness **`tool_executor`** stage via `internal/harness/tools.ToolExecutor`. Client `tools` on the assistant path are **replaced** by gateway-injected schemas when the module is enabled.

v0.5 charter ([`version-v0.5.md`](../version-v0.5.md)) adds **MCP tool backends** on top of that foundation. Product requirements in [`chimera.plan.md`](../chimera.plan.md) state **MCP boundaries**: MCP is not the model router; tool loops may delegate to MCP servers; routing stays gateway-controlled.

Today there is **no MCP client** in `chimera-gateway`. BiFrost MCP on the broker is **broker-side** (log normalization only). Indexed workspace expansion tools exist at **`GET /v1/rag/tools`** but are **not** in the harness tool loop.

This plan delivers a **minimal MCP client** for the harness first, then optional developer capabilities. It does **not** absorb workflow graphs, desktop MCP hub UI, or operator model-assist tooling.

**Related docs:** [`assistant-harness-internals.md`](../features/assistant-harness-internals.md), [`operator-assistants.md`](../features/operator-assistants.md), [`configuration.md`](../configuration.md), [`supervisor.md`](../supervisor.md).

---

## Architecture (normative target)

Three internal responsibilities—mirroring proven MCP hub design, implemented **in or beside** chimera-gateway, not as a desktop Electron product:

| Layer | Responsibility | Must not |
|-------|----------------|----------|
| **Server manager** | Desired-state MCP server records; start/stop; stdio child and streamable HTTP upstream clients; health; teardown on disable or gateway shutdown | Expose the full MCP protocol to chat models directly |
| **Tool catalog** | Merge `tools/list` from running servers; filter by assistant/workspace scope; per-tool enable flags; **slug mapping**; produce OpenAI-format schemas for harness injection | Spawn or own upstream transports (Phase 1: gateway-owned processes) |
| **Composite `ToolExecutor`** | `Invoke(ctx, ToolCall)` dispatches to native executor and/or MCP backend; validation and structured errors to the model | Call upstream LLM providers |

```text
Assistant POST /v1/chat/completions
  → harness: tool_router (client tools only; overwritten when tool_executor on)
       → tool_executor
            → Composite ToolExecutor (native first, then MCP by toolKey)
                 → WorkspaceExecutor (+ native expansion tools)
                 → MCP catalog → manager → MCP processes (gateway-spawned stdio / HTTP clients)
```

**Harness rules (non-negotiable):**

1. MCP execution stays in **`tool_executor`** / internal tool round loop—**after** `tool_router`, **before** `fallback_proxy` upstream completion.
2. **No new harness stage** that calls upstream outside existing proxy and `WithAssistantFallback` patterns ([gateway chat routing pipeline](../features/gateway-chat-routing-pipeline.md)).
3. When `tool_executor` is on, injected tools include **native + allowed MCP** tools per assistant config (same v0.4 “gateway-injected only” rule for the assistant path).
4. **Direct upstream** path (`provider/model` in body) continues to **bypass harness** unless a separate plan extends behavior there.
5. **Gateway-injected tool lists do not pass through `tool_router`.** Until a later plan re-runs slim after injection, enforce a **hard cap** on merged tool count (configurable N). When over cap: fail closed or drop MCP tools with operator slug `tools.mcp.catalog_truncated` — **pick one in the platform contract and document in the pipeline feature**.
6. Mid-turn MCP **must not** block on operator UI confirmation. The tool loop is **one HTTP request** with bounded `max_tool_rounds`. Destructive capability is **pre-authorized** via assistant MCP profile, per-tool enable flags, and policy defaults. Confirmations apply to **enabling** tools in settings (and to operator model-assist APIs), not to each `tools/call`.
7. Per MCP `tools/call`: deadline independent of client HTTP timeout; on timeout, return `ToolResult{IsError: true}`; do not fail the whole assistant request unless the manager is wedged.
8. Composite dispatch: **native first** (reserved names), then MCP by `toolKey`; unknown name → structured error (not workspace `default`).
9. MCP tools are **deny-by-default** for each assistant. A server must be explicitly bound and each exposed MCP tool explicitly enabled; missing or unknown risk annotations never grant access. MCP annotations may inform operator display, but are not an authorization source.

The existing `tool_router` remains an optional transform for client-declared tools on assistants that use that client-tool path. MCP-focused assistants can leave it disabled; MCP catalog selection and authorization are controlled by the assistant's MCP profile, and gateway-injected schemas bypass `tool_router` either way. This does not change the current `tool_executor` contract that replaces client declarations on its assistant path.

**Slug and collision policy:**

- Internal dispatch uses a stable **`toolKey`** (unique across native + MCP).
- **OpenAI `function.name`** is a derived slug: `[a-zA-Z0-9_]{1,64}`, collision-safe. Contract documents mapping `openaiName ↔ toolKey ↔ (serverId, mcpToolName)`.
- **Native workspace tool names are reserved** (`read_file`, `write_file`, `list_dir`, `search`, and expansion tool names once shipped). MCP tools that collide are renamed; they **never** overlay native dispatch.
- Operator UI may show `serverId / mcpToolName`; the model only sees the slug.

**Scoping:** Catalog build filters servers and tools by **assistant binding**, **workspace roots**, and optional **principal scope** (operator session vs stack token — defined in Phase 0 contract).

**Known gap (v0.5 Phase 1–2):** Unbounded MCP merge blows context on the first upstream call. Catalog cap is required before production assistants enable large sidecar sets. Phase 4 meta-tools are **deferred**; do not ship unbounded merge to production assistants.

---

## Tool personas (normative)

| Persona | Harness / assistant | Allowed tools |
|---------|---------------------|---------------|
| **Developer assistant** | Operator-defined assistant with `tool_executor` + MCP profile | Native files/search/expansion; configured MCP (git, shell, browser, workflow) per allowlist |

**Developer assistant runtime policy:** Operator enables servers/tools in settings; **no human-in-the-loop per `tools/call`**. Destructive MCP tools default **off**. Shell: cwd ⊆ workspace roots, **network default off**, **timeout required**. Env for child processes: **deny-by-default allowlist** (supervisor/gateway inherit can leak API keys without this).

| Persona | Scope |
|---------|--------|
| **Operator model-assist** | **Out of scope for this plan.** See [`operator-model-assist-tools.md`](operator-model-assist-tools.md). This plan’s MCP allowlists **must not** be attachable to the model-assist assistant (hard deny: separate assistant / module / tool profile). |

---

## Core suite taxonomy

### Tier A — Native (`WorkspaceExecutor` and extensions)

| Tool / capability | Notes |
|-----------------|-------|
| `read_file`, `write_file`, `list_dir`, `search` | Shipped v0.4; policy via `file_action_policy` |
| RAG expansion (`workspace_context_around`, `workspace_adjacent_chunks`, `workspace_read_lines`) | Phase 2a — native executor + `ExpansionService`; schemas at `/v1/rag/tools` |

Native `search` quality improvements are tracked in [`assistant-harness-workspace-search-quality.md`](assistant-harness-workspace-search-quality.md) — **not** required for MCP acceptance.

### Tier B — Standard MCP sidecars (Phase 2b+)

| Domain | Typical provider | Operator controls |
|--------|------------------|-------------------|
| Git | Dedicated git MCP or `gh`-aware server | Default **off**; read-only profile when enabled |
| Shell | Sandboxed command MCP | Default **off**; cwd allowlist, timeout, no secret env |
| Web / browser | Browser or CDP MCP | Default **off**; explicit opt-in per assistant |

Blessed sidecars are **documented commands** (YAML seed). Optional later **`chimera-*` wrappers** if env translation and structured `*line` logs are required. **Third-party MCP CLIs do not use** [wrapper binary contract](../features/chimera-wrapper-binary-contract.md) in v0.5 Phase 1–2.

### Tier C — Domain MCP (connect, do not embed)

| Server | Role |
|--------|------|
| **task-orchestrator** | Standalone MCP — **Phase 3 deferred** |
| **Project-local MCP** | Repo-local stdio via manager config |
| **Foundry / craft** | Out of scope unless registered as MCP |

Gateway **must not** reimplement task-orchestrator workflow domain in Go.

### Tier D — IDE direct path

Continue / Cursor may keep **local MCP** when chat uses **direct** `provider/model`. Gateway-native MCP is for the **assistant harness path** first. Optional HTTP facade is Phase 4 / separate plan.

---

## Phase 0 — Contract and design

**Goal.** Reviewers and implementers share one written contract and a **transport spike** before production MCP code spreads.

**Deliverables**

- Platform-contract feature record: **`docs/features/gateway-mcp-tool-backends.md`** (`Doc kind: platform-contract`) covering:
  - Manager lifecycle (start/stop/disabled; `autoStart` default off; gateway shutdown kills stdio children — no orphans)
  - Catalog merge, soft-fail per server on `tools/list`, slug mapping, reserved native names, **max catalog size**, `tools.mcp.catalog_truncated` behavior
  - Composite `ToolExecutor` error envelope; argument/result size caps
  - Per-call timeout; stderr bounded buffer + log redaction
  - Env allowlist for MCP children (deny-by-default; secrets story for git/`gh`)
  - Config: YAML seed (Phase 1); SQLite + UI (Phase 2c) as eventual operator surface
  - Audit slugs: `tools.mcp.invoke`, `tools.mcp.error`, `tools.policy.denied`, `tools.mcp.catalog_truncated` (finalize in registry)
- **Transport spike:** stdio line-framed JSON-RPC against the Phase 1 fake server (tests in `internal/mcpmgr` or equivalent). **SSE upstream unsupported** in v0.5 unless spike proves a blessed sidecar requires it — blessed list limited to stdio + streamable HTTP that pass the spike.
- **Manager placement (normative for Phase 1):** **Gateway-owned** stdio children and HTTP clients. **Do not** spawn arbitrary MCP via chimera-supervisor until a documented supervisor “sidecar job” type exists.
- ADR subsection: workflow domain stays external (task-orchestrator as MCP only)
- Reference doc stub: **`docs/reference/mcp-sidecar-catalog.md`**
- [`gateway-chat-routing-pipeline.md`](../features/gateway-chat-routing-pipeline.md): note gateway-injected tools skip `tool_router`; MCP catalog cap documented under known gaps / out of scope until implemented

**Acceptance**

- Feature record `draft` or `partial` exists; linked from this plan front matter
- Contract includes **tables** for slug examples, reserved names, and 64-char edge cases (not prose only)
- Open questions below have **closed defaults** recorded in the feature record

**Status:** `done`

---

## Phase 1 — Manager, catalog, composite executor

**Goal.** An assistant with `tool_executor` enabled completes a turn that invokes a tool on a **real MCP server** (stdio or streamable HTTP), with merged schemas injected into the proxied upstream body.

**Deliverables**

- Go packages (names illustrative): `internal/mcpmgr`, `internal/mcptools/catalog`, `internal/harness/tools/composite.go`, `internal/harness/mcp_wiring.go`
- **Manager:** `Start`/`Stop`; states `stopped | starting | running | error`; honor `disabled`; **`autoStart` off by default**
- **Stdio:** line-framed JSON-RPC; drain stderr (bounded buffer); kill on context cancel / assistant disable / gateway shutdown
- **HTTP:** streamable JSON-RPC per request; document TLS and redirect policy (default: no arbitrary redirect to untrusted hosts)
- **MCP session:** perform `initialize` and `notifications/initialized` per transport requirements before listing or calling tools; negotiate a supported protocol version rather than assuming one.
- **Catalog:** `Build(ctx, scope)` → OpenAI tools + route table; soft-fail per server; `toolPermissions` (`false` disables)
- **Authorization:** explicit assistant binding and per-tool allowlist; absent permission is disabled. Preserve MCP annotations when available for display and audit, but enforce configured permissions independently.
- **Composite executor:** native-first dispatch; MCP `tools/call` by `toolKey`
- **Result conversion:** convert MCP text and structured results into the existing `ToolResult` shape deterministically; cap returned bytes and represent MCP `isError` as a tool error without losing the turn.
- **Config v1:** **`chimera.yaml` / example seed required** for tests and first run. Operator SQLite **not** in Phase 1 (in-memory or YAML only)
- **Harness wiring:** factory with `WorkspaceExecutor`; union `OpenAITools()` when MCP enabled on assistant
- **Observability:** register and emit `tools.mcp.invoke` / `tools.mcp.error` (or Phase 1 placeholders in `messages.yaml`)
- **Contract tests:** fake MCP; collision; disabled tool; list soft-fail; timeout; orphan-process test on shutdown
- **Gallery fixture (minimal):** MCP module + one fake tool name visible in harness preview

**Acceptance**

- Integration test: fake MCP → assistant chat → tool round-trip → turn completes within `max_tool_rounds`
- MCP tool whose MCP name collides with `read_file` is exposed under a **non-native** slug; `read_file` still hits `WorkspaceExecutor`
- Two servers, same MCP tool name → two distinct OpenAI slugs; routing correct
- All injected `function.name` values ≤ 64 chars and match `[a-zA-Z0-9_]`
- `tools/list` failure on one of two servers: other server’s tools still inject
- Per-call deadline exceeded → error envelope; turn still completes
- Disabled server or tool → not listed / structured policy error
- Server or tool without an explicit assistant allow entry → not listed; MCP annotations alone cannot enable it
- MCP server must complete initialization before catalog or invocation; returned `isError` and oversized results become bounded tool results
- `make precommit` green

**Status:** `done`

---

## Phase 2a — Native RAG expansion tools

**Goal.** Primary model can use indexed expansion tools inside the harness loop **without MCP**.

**Deliverables**

- Wire `workspace_context_around`, `workspace_adjacent_chunks`, `workspace_read_lines` into `WorkspaceExecutor` (or composite native path) calling `ExpansionService`
- Schemas aligned with `/v1/rag/tools`
- Harness test with fixture workspace

**Acceptance**

- Expansion tool returns indexed context in harness integration test
- Reserved native names documented in platform contract
- **Does not depend on Phase 2b MCP**

**Status:** `done`

---

## Phase 2b — One blessed MCP sidecar

**Goal.** One documented, supervised MCP path beyond the fake server (e.g. git read-only or no-network echo server).

**Deliverables**

- **`docs/reference/mcp-sidecar-catalog.md`:** one blessed entry (command, args, env allowlist, security notes)
- Example snippet in [`chimera.example.yaml`](../chimera.example.yaml)
- Harness or integration test: YAML-defined server → one successful `tools/call`
- Policy defaults: git, shell, browser **all off** for new assistants until operator enables

**Acceptance**

- Operator (or test harness) can enable the blessed server via **YAML** and reach it from the tool loop
- Shell and browser remain off by default in config templates

**Status:** `done`

---

## Phase 2c — Operator MCP settings

**Goal.** Operators manage MCP servers without editing YAML.

**Deliverables**

- Operator SQLite rows for MCP servers; minimal UI: list servers, enable/disable, per-tool toggles, bind to assistant
- Server health visible when `error` or empty catalog (copy + log slugs)
- Workspace / assistant switch: document when catalog rebuilds and shared-server lifetime rules

**Acceptance**

- Add server via UI → harness lists tools on bound assistant
- Disable shell MCP for assistant → tool absent from catalog
- Feature record updated with as-built config fields and code map

**Status:** `done`

---

## Phase 3 — Workflow MCP (task-orchestrator)

**Goal.** Optional registration of external workflow MCP — **not** a v0.5 blocker.

**Deliverables (when promoted from deferred)**

- Registration recipe in sidecar catalog
- Integration doc: workflow MCP vs harness memory vs Foundry
- **No Go port** of task-orchestrator domain
- Tests: **in-process fake** returning structured gate errors (e.g. `missingNotes`) — not required Kotlin CI container

**Acceptance**

- Documented steps to register task-orchestrator; optional fake test for gate error shape
- No `WorkItem` / `RoleTransition` types in gateway domain packages

**Status:** `deferred`

---

## Phase 4 — Scale and IDE ingress

**Goal.** Large catalogs and optional IDE-facing MCP HTTP — **post-v0.5**.

**Deliverables (when promoted)**

- Meta-tools (`tool_discovery` / `tool_execute`) when catalog > N
- Optional stateless `POST /mcp` facade with hashed bearer tokens — may warrant separate plan `gateway-mcp-http-facade.md`

**Acceptance**

- Catalog > N does not inject full tool list when meta-tool mode enabled

**Status:** `deferred`

---

## Non-goals

- Reimplement **mcp-router** desktop app inside Porcelain
- Use **broker BiFrost MCP** as gateway tool execution
- **MCP-driven model routing** or mid-inference provider switching
- **Operator model-assisted configuration** (page guides, confirm-before-apply, `/api/ui/*` tools) — **[`operator-model-assist-tools.md`](operator-model-assist-tools.md)**
- Attaching developer MCP allowlists to the model-assist assistant
- **Spawning MCP via chimera-supervisor** in Phase 1
- **mcp-router go-port** as a runtime dependency
- Applying **wrapper binary contract** to third-party MCP CLIs in Phase 1–2
- **Full MCP resources/prompts** aggregation in Phase 1–2
- **task-orchestrator** as default for all assistants (Phase 3 deferred)
- Unbounded merged tool catalogs on production assistants without cap or meta-tools

---

## Open questions

Record decisions in the platform-contract feature record as they close.

1. **Manager placement:** **Gateway-owned processes in Phase 1.** Supervisor sidecar job type — revisit later.
2. **Config source of truth:** **YAML seed Phase 1**; SQLite + UI **Phase 2c**.
3. **Collision UX:** Internal `toolKey` + 64-char OpenAI slug; UI shows server + MCP name.
4. **Shell trust:** cwd ⊆ workspace roots; network off unless operator enables; **timeout required**.
5. **mcp-router:** Reimplement minimal slice in gateway; go-port as design reference only.
6. **Catalog over cap:** **Fail closed** at default **N = 32**; emit `tools.mcp.catalog_truncated` — closed in Phase 0 feature record.

---

## References

### Porcelain code (current)

- `chimera/chimera-gateway/internal/harness/tools/tools.go` — `ToolExecutor`, `WorkspaceExecutor`
- `chimera/chimera-gateway/internal/harness/tools_stage.go` — tool round loop
- `chimera/chimera-gateway/internal/harness/stages.go` — `ToolRouterStage`, `ToolExecutorStage`
- `chimera/chimera-gateway/internal/rag/expansion.go` — `WorkspaceToolDefinitions()`
- `chimera/chimera-gateway/internal/transform/toolrouter.go` — confidence tool router (client tools; bypassed when tool_executor replaces body tools)

### Porcelain docs

- [`version-v0.4.md`](../version-v0.4.md), [`version-v0.5.md`](../version-v0.5.md), [`chimera.plan.md`](../chimera.plan.md)
- [`assistant-harness-workspace-tools.md`](assistant-harness-workspace-tools.md)
- [`operator-model-assist-tools.md`](operator-model-assist-tools.md)

### External design references

- **mcp-router:** manager / catalog / aggregator; collision via explicit namespacing; project scoping
- **task-orchestrator:** `ToolDefinition` + adapter; server is the product
