# Feature: Gateway MCP tool backends

| Field | Value |
|-------|-------|
| **Doc kind** | `platform-contract` |
| **Areas** | `chimera-gateway` harness, operator store (Phase 2c), observability |
| **Status** | `partial` — Phases 0–2c shipped; workflow MCP (Phase 3) and scale/facade (Phase 4) deferred |
| **Introduced** | Gateway v0.5 train |
| **Originated from** | [`plans/gateway-mcp-tool-backends.md`](../plans/gateway-mcp-tool-backends.md) |
| **Related features** | [Gateway chat routing pipeline](gateway-chat-routing-pipeline.md), [Operator assistants](operator-assistants.md), [Operator log message registry](operator-log-message-registry.md) |
| **Depends on** | v0.4 `ToolExecutor` / `tool_executor` stage |
| **Last updated** | See git history |

## At a glance

On the assistant path, the harness merges **native workspace tools** (files, search, optional RAG expansion) with **allowlisted MCP tools** from configured servers, via a composite `ToolExecutor` in the `tool_executor` stage. Config comes from `chimera.yaml` until operator SQLite has MCP server rows; then SQLite owns servers and assistant bindings. Operator model-assist **must not** use this profile ([`operator-model-assist-tools` plan](../plans/operator-model-assist-tools.md)). Delivery history: [`gateway-mcp-tool-backends-delivery.md`](../plans/gateway-mcp-tool-backends-delivery.md).

## System behavior and contracts

**Invariants (target)**

- MCP execution stays in **`tool_executor`**; no new harness stage calls upstream LLM providers.
- **Native tool names are reserved**; MCP never overrides `read_file`, `write_file`, `list_dir`, `search`, or expansion tool names once registered.
- **No silent last-wins** when two MCP servers expose the same MCP tool name.
- **Gateway-injected tools do not pass through `tool_router`** (client-tool slimming only).
- **No human-in-the-loop** inside the tool round loop; policy is pre-authorized via assistant profile and per-tool toggles.
- Phase 1: **gateway-owned** stdio/HTTP MCP clients — not chimera-supervisor generic spawn.
- **MCP authorization is deny-by-default:** a server must be **explicitly bound** to an assistant and each MCP tool **explicitly allowlisted**. Missing binding or allow entry ⇒ tool **not listed** and `tools/call` ⇒ `tools.policy.denied`. MCP `annotations` (e.g. read-only hints) are **display and audit only** — they never enable a tool.
- **MCP session:** gateway client **must** complete `initialize` and send `notifications/initialized` before `tools/list` or `tools/call`.

**Reserved native names**

| Name | Dispatch |
|------|----------|
| `read_file` | `WorkspaceExecutor` |
| `write_file` | `WorkspaceExecutor` |
| `list_dir` | `WorkspaceExecutor` |
| `search` | `WorkspaceExecutor` |
| `workspace_context_around` | Native expansion (Phase 2a) |
| `workspace_adjacent_chunks` | Native expansion (Phase 2a) |
| `workspace_read_lines` | Native expansion (Phase 2a) |

MCP tools whose MCP name collides with a reserved native name are exposed under a **renamed OpenAI slug**; native dispatch always wins for the bare native name.

**Slug mapping (normative)**

| Concept | Rule |
|---------|------|
| `toolKey` | Stable internal dispatch key, unique across native + MCP (e.g. `mcp:{serverId}:{mcpToolName}`) |
| OpenAI `function.name` | Derived slug: `[a-zA-Z0-9_]{1,64}` — see algorithm below |
| MCP name collision (two servers) | Distinct slugs; map to `(serverId, mcpToolName)` |
| MCP vs native name collision | Rename MCP slug; native wins dispatch |

**OpenAI `function.name` algorithm (deterministic)**

1. `left = sanitize(serverId)`, `right = sanitize(mcpToolName)` — keep `[A-Za-z0-9_]`, replace other runes with `_`, trim `_`; empty segment ⇒ `srv` / `tool`.
2. `candidate = left + "__" + right`.
3. If `len(candidate) ≤ 64`, use `candidate`.
4. Else `digest = hex(SHA-256(serverId + "\x00" + mcpToolName))[:16]`, `left = truncate(left, 64 - 2 - len(digest))`, return `left + "__" + digest`.
5. Catalog build **must** detect slug collisions after step 4 and append `_2`, `_3`, … deterministically by sorted `(serverId, mcpToolName)` (Phase 1).

Implementation reference: `mcpclient.OpenAIName` in `chimera/chimera-gateway/internal/mcpclient/slug.go`.

**Slug examples**

| serverId | mcpToolName | toolKey | openaiName (model sees) |
|----------|-------------|---------|-------------------------|
| `git_local` | `status` | `mcp:git_local:status` | `git_local__status` |
| `extra` | `read_file` | `mcp:extra:read_file` | `extra__read_file` |
| — | `read_file` (native) | `native:read_file` | `read_file` |
| `a` × 40 + `b` | `tool` | `mcp:…:tool` | `{left_truncated}__{16_hex_digest}` (≤ 64) |

**Catalog limits**

| Topic | Default (until operator overrides) |
|-------|--------------------------------------|
| Max merged tools injected (`N`) | **32** — configurable per assistant/stack in Phase 1+ |
| Over cap behavior | **Fail closed:** catalog `Build` fails; **no** MCP tools inject for that turn; emit `tools.mcp.catalog_truncated` with `cap`, `count`. Native tools still inject if under cap alone. |
| `tools/list` partial failure | Soft-fail per server; other servers’ tools still inject |
| Per `tools/call` deadline | Default **60s**; independent of client HTTP timeout; `ToolResult{IsError: true}` on timeout |
| Max tool **result** bytes returned to model | Default **256 KiB** (truncate with ellipsis + log); max **arguments** JSON per call default **64 KiB** (reject before upstream MCP) |
| Child stderr capture | Bounded buffer (default **32 KiB**); redact known secret patterns before operator log (Phase 1) |

**Catalog scoping (Phase 1 build)**

| Filter | Rule |
|--------|------|
| Assistant binding | Server must be explicitly bound to the assistant; unbound servers are ignored |
| Per-tool allowlist | Each MCP tool must be explicitly enabled; absent entry ⇒ not listed |
| Workspace roots | Sidecar policies (shell cwd, file tools) must stay within configured workspace roots |
| Principal scope | Optional future filter (`operator_session` vs stack token); **Phase 1** uses assistant binding + workspace roots only |

**Composite `ToolExecutor` errors (Phase 1)**

| Case | Model-visible behavior |
|------|------------------------|
| Unknown tool name | Structured `ToolResult{IsError: true}`; native reserved names never route to MCP |
| Policy deny | `tools.policy.denied`; same error envelope |
| MCP transport / protocol | `tools.mcp.error`; turn continues unless manager wedged |
| Per-call timeout | `ToolResult{IsError: true}`; does not fail the HTTP request |
| Oversized arguments | Reject before MCP; structured error (no upstream call) |

**Manager lifecycle**

| State | Meaning |
|-------|---------|
| `stopped` | No client |
| `starting` | Transport connecting; initialize in flight |
| `running` | Handshake complete; `tools/list` / `tools/call` allowed |
| `error` | Last start or call failed; operator-visible when UI exists |

`autoStart` default **off**. Gateway shutdown **must not** leave orphan stdio children.

**Env and secrets**

- MCP child env: **deny-by-default allowlist** per server record.
- Inherited gateway/supervisor env may contain provider keys — allowlist is required before spawning untrusted MCP CLIs.

**Observability (registry slugs)**

| Slug | When |
|------|------|
| `tools.mcp.invoke` | Successful MCP `tools/call` completion (audit) |
| `tools.mcp.error` | Transport, protocol, or MCP error on call |
| `tools.policy.denied` | Call or list blocked by assistant bind / per-tool allowlist |
| `tools.mcp.catalog_truncated` | Merged catalog exceeded `N` (fail-closed path) |

**Decisions (ADR)**

| Topic | Decision |
|-------|----------|
| Manager placement (Phase 1) | Gateway process owns stdio children and HTTP clients |
| Config (Phase 1) | `chimera.yaml` seed; SQLite + UI in Phase 2c |
| Config merge (Phase 2c) | When **any** `mcp_servers` row exists in operator SQLite, **SQLite is source of truth** for `servers[]` and `assistants[]` MCP bindings; YAML MCP server/bind sections are ignored. Global limits (`catalog_max_tools`, call timeout) still come from YAML. With **zero** SQLite server rows, YAML alone applies. |
| Catalog rebuild (Phase 2c) | Per harness turn: `catalog.Build` runs in `wireToolExecutor` using effective MCP config (merged at chat entry via `Runtime.ChatResolved`). Saving MCP settings reloads the gateway MCP manager; the next assistant turn rebuilds the injected tool list. Shared stdio children stay in the gateway MCP manager until shutdown or config reload removes the server. |
| Third-party MCP CLIs | Not `chimera-*` wrappers; wrapper contract optional later |
| Catalog over cap | **Fail closed** at default **N = 32** (not silent MCP drop) |
| Shell MCP sidecars (Phase 2b+) | **Default off**; when enabled: cwd ⊆ workspace roots, **network default off**, per-call **timeout required** |
| mcp-router | Reimplement the minimal manager / catalog / client slice **inside chimera-gateway**; workspace `mcp-router` go-port is a **design reference only** — not a runtime dependency |
| Workflow / task graphs | **Stay external** — **task-orchestrator** (or equivalent) runs as a **standalone MCP server** only; gateway **must not** embed `WorkItem`, role transitions, or workflow domain types (Phase 3 deferred) |
| SSE upstream MCP | Unsupported in v0.5 unless a later spike documents a blessed sidecar |

## Interfaces

| Surface | Detail |
|---------|--------|
| Harness | `tools.ToolExecutor.Invoke`; composite behind `tool_executor` stage (Phase 1) |
| Transport spike | `mcpclient.Client` — stdio line JSON-RPC; `Initialize` + `notifications/initialized`, `tools/list`, `tools/call` |
| Config | YAML `servers[]` (Phase 1); operator SQLite `mcp_servers` + bindings (Phase 2c) |
| Operator API | `GET/PUT /api/ui/mcp/servers`, `GET /api/ui/mcp/servers/{id}/tools`, `GET/PUT /api/ui/assistants/{id}/mcp` |
| HTTP | No public MCP facade in Phase 1 (Phase 4 deferred) |
| Sidecar reference | [`docs/reference/mcp-sidecar-catalog.md`](../reference/mcp-sidecar-catalog.md) (Phase 2b entries) |

## Code map

| Area | Path |
|------|------|
| MCP client (Phase 0 spike) | `chimera/chimera-gateway/internal/mcpclient/` |
| Fake stdio server (tests) | `chimera/chimera-gateway/internal/mcpclient/fake/` |
| Manager (Phase 1) | `chimera/chimera-gateway/internal/mcpmgr/` |
| Catalog (Phase 1) | `chimera/chimera-gateway/internal/mcptools/catalog/` |
| MCP adapter (Phase 1) | `chimera/chimera-gateway/internal/harness/tools/composite.go`, `internal/harness/mcp_wiring.go` |
| Operator MCP store (Phase 2c) | `chimera/chimera-gateway/internal/operatorstore/mcp.go`, migration `migrations/chimera-gateway/operator/000012_mcp_servers.sql` |
| Operator MCP API (Phase 2c) | `chimera/chimera-gateway/internal/server/adminui/api/mcp/` |
| Runtime MCP merge (Phase 2c) | `chimera/chimera-gateway/internal/server/runtime/mcp.go` |
| Harness MCP settings UI (Phase 2c) | `embedui/settings/render/cards/adminAssistants.js` (tool_executor section) |
| Native tools | `chimera/chimera-gateway/internal/harness/tools/tools.go`, `native_executor.go`, `native_tools.go` (harness wiring) |
| RAG expansion (Phase 2a) | `internal/rag/expansion.go` (`ExpansionService`, `WorkspaceToolDefinitions` / `WorkspaceOpenAIFunctionTools`); injected when `rag.enabled` and `rag.tooling.enabled`; harness test `internal/harness/expansion_tool_loop_test.go` |
| Tool loop | `chimera/chimera-gateway/internal/harness/tools_stage.go` |
| Blessed sidecar (Phase 2b) | [`docs/reference/mcp-sidecar-catalog.md`](../reference/mcp-sidecar-catalog.md); example seed [`config/chimera.example.yaml`](../../config/chimera.example.yaml) (`mcp.servers.fake_echo`); subprocess integration `internal/harness/mcp_blessed_yaml_test.go` |

## Out of scope / known gaps

- MCP server **create** wizard in UI (use API or seed SQLite); YAML remains valid until first SQLite server row.
- Gateway-injected tools skip `tool_router`; cap enforced at catalog build (see [gateway chat routing pipeline](gateway-chat-routing-pipeline.md)).
- Resources/prompts aggregation not in Phase 1–2.
