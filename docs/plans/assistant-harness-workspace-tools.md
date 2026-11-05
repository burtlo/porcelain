# Plan: Virtual model harness — gateway workspace tools

| Field | Value |
|-------|-------|
| **Doc kind** | `feature-plan` |
| **Owners / areas** | Gateway harness, workspace paths, chat primary stage |
| **Status** | `done` |
| **Targets** | Gateway v0.4 |
| **Last updated** | See git history |
| **Supersedes / superseded by** | Child of [`assistant-turn-harness.md`](assistant-turn-harness.md) |
| **As-built** | [`gateway-chat-routing-pipeline.md`](../features/gateway-chat-routing-pipeline.md), [`operator-assistants.md`](../features/operator-assistants.md), [`indexer-workspaces.md`](../features/indexer-workspaces.md) |

## At a glance

Primary model can read and write files within workspace permission inside one chat turn. Tool declarations are **gateway-injected only** when the tool executor module is enabled.

| Phase | Outcome | Status |
|-------|---------|--------|
| [Phase 1 — Gateway workspace tools](#phase-1--gateway-workspace-tools) | Native file tools under workspace roots in one HTTP turn | `done` |

**Depends on:** [runtime](assistant-harness-runtime.md), [settings](assistant-harness-settings.md), [workspace policy](assistant-harness-workspace-policy.md)

**Delivery gates:** [umbrella](assistant-turn-harness.md#delivery-gates-normative) — `make precommit` at plan done; hard cut (no legacy paths); phase debt OK if Fix-by is set.

---

## Background

Chimera chat does not send client tool declarations today ([`operator-chat-ui.md`](../features/operator-chat-ui.md)). v0.4 adds gateway-native workspace tools executed inside the harness primary stage. **Normative decision:** when `tool_executor` is enabled, the gateway injects fixed tool schemas and **ignores** any `tools` array in the client request body on the VM harness path.

**Related docs:** [`assistant-turn-harness.md`](assistant-turn-harness.md), [`indexer-workspaces.md`](../features/indexer-workspaces.md).

---

## Phase 1 — Gateway workspace tools

**Goal.** Primary model can read and write files within workspace permission inside one chat turn.

**Deliverables**

- `ToolExecutor` interface in `internal/harness/tools/`: `Invoke(ctx, ToolCall) (ToolResult, error)`.
- Native implementations: `read_file`, `write_file`, `list_dir`, `search` — paths resolved relative to chosen workspace roots; reject paths outside roots.
- When `tool_executor` module enabled: **inject** fixed OpenAI-format tool declarations into proxied body; **strip/ignore** client `tools` on VM harness path.
- Permission gate uses `file_action_policy` from [workspace policy plan](assistant-harness-workspace-policy.md); atomic writes; no absolute host paths in model-facing tool results (relative to root).
- Internal tool round loop in primary stage: `max_tool_rounds` (default 5) before returning to client.
- VM module toggle `tool_executor`.
- **Out of scope:** MCP adapters (v0.5), browser tools, sandboxed code execution, merge with client or VM custom tool schemas.
- **Gallery** ([umbrella contract](assistant-turn-harness.md#gallery--reference-ui-contract-normative)): tool executor off vs on on the VM harness fixture; pair with workspace policy fixtures that show `read` vs `read_write` permission framing (cross-link or adjacent mounts).

**Acceptance**

- Chat with `read_write` workspace: model requests write → file appears under watched root; turn completes in one HTTP request.
- `read`-only workspace: write tool returns policy error visible in envelope `execution.tool_calls`.
- Client sends `tools` in body with tool_executor on → upstream request contains gateway tools only.
- Gallery shows tool-executor off/on and clear pairing with workspace file-policy states.

**Status:** `done`

---

## References

- Code: `internal/server/virtualmodel_chat.go`, `internal/operatorstore/`
- Umbrella: [`assistant-turn-harness.md`](assistant-turn-harness.md)
