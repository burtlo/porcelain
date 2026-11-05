# Plan: Operator model-assist tools (configuration APIs only)

| Field | Value |
|-------|-------|
| **Doc kind** | `feature-plan` |
| **Owners / areas** | Gateway, operator UI, admin API |
| **Status** | `draft` |
| **Targets** | Gateway v0.5 |
| **Last updated** | See git history |
| **Supersedes / superseded by** | Normative UX in [`version-v0.5.md`](../version-v0.5.md#model-assisted-configuration); **not** [`gateway-mcp-tool-backends.md`](gateway-mcp-tool-backends.md) |
| **As-built** | None |

## At a glance

When an operator uses **model-assisted configuration** from a settings surface, the model may only call **tools that wrap documented `/api/ui/*` and `/v1/*` mutation and read APIs** for that page — never the developer MCP suite (shell, git, browser, arbitrary filesystem).

| Phase | Outcome | Status |
|-------|---------|--------|
| [Phase 1 — Tool surface and enforcement](#phase-1--tool-surface-and-enforcement) | Dedicated assistant or module profile; hard deny of developer MCP allowlists | `todo` |
| [Phase 2 — Confirm-before-apply UX](#phase-2--confirm-before-apply-ux) | Operator confirms each mutation; audit slugs | `todo` |

**Related:** [`version-v0.5.md`](../version-v0.5.md) (model-assisted configuration, desired-state gateway), [`gateway-mcp-tool-backends.md`](gateway-mcp-tool-backends.md) (explicit non-overlap).

---

## Background

v0.5 model-assist is the **interactive** face of operator desired-state recovery. It must use the same authenticated APIs as the settings UI, with **confirm-before-apply** for destructive changes ([`version-v0.5.md`](../version-v0.5.md)).

The harness **developer MCP** plan ([`gateway-mcp-tool-backends.md`](gateway-mcp-tool-backends.md)) targets **software development assistants**. Those allowlists **must not** attach to the model-assist assistant (separate assistant id, module set, or tool profile — enforced in gateway, not documentation alone).

---

## Phase 1 — Tool surface and enforcement

**Goal.** Model-assist runtime cannot invoke developer MCP or workspace shell paths.

**Deliverables**

- Tool definitions (native or thin HTTP wrappers) scoped per **page guide** / settings card
- Gateway enforcement: model-assist principal **rejects** `tool_executor` MCP profiles and reserved developer tool names not in the page allowlist
- Audit slugs: `operator.model_assist.proposed`, `operator.model_assist.applied` ([`version-v0.5.md`](../version-v0.5.md))

**Acceptance**

- Model-assist session cannot call a developer MCP tool even if misconfigured in YAML (fail closed)
- Documented demo path from v0.5 verification table succeeds without filesystem tools

**Status:** `todo`

---

## Phase 2 — Confirm-before-apply UX

**Goal.** Operator reviews proposed API sequence before mutations run.

**Deliverables**

- UI: proposed changes summary; per-mutation or batch confirm
- No autonomous apply without confirmation ([`version-v0.5.md`](../version-v0.5.md) explicitly not this version)

**Acceptance**

- End-to-end assist flow with confirm-before-apply; audit logs present (v0.5 verification row)

**Status:** `todo`

---

## Non-goals

- Shell, git, browser, or harness workspace **write** tools for model-assist
- Reusing [`gateway-mcp-tool-backends.md`](gateway-mcp-tool-backends.md) MCP server rows for model-assist

---

## References

- [`version-v0.5.md`](../version-v0.5.md) — Model-assisted configuration
- [`gateway-mcp-tool-backends.md`](gateway-mcp-tool-backends.md) — Developer MCP (separate persona)
