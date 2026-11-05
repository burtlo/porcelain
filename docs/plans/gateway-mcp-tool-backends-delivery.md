# Delivery record: Gateway MCP tool backends (Phases 0–2c)

| Field | Value |
|-------|-------|
| **Doc kind** | `working-notes` |
| **Status** | `current` |
| **Plan** | [`gateway-mcp-tool-backends.md`](gateway-mcp-tool-backends.md) (Phases 0–2c **shipped**; 3–4 **deferred**) |
| **As-built** | [`gateway-mcp-tool-backends.md`](../features/gateway-mcp-tool-backends.md) (`partial`) · [`mcp-sidecar-catalog.md`](../reference/mcp-sidecar-catalog.md) |

This document captures **how** Phases 0–2c were executed (implementation train + verification agents) and **Big Lady sign-off** for the shipped slice. It is not a second feature spec—the plan and platform contract remain authoritative for behavior.

---

## Shipped scope (Phases 0–2c)

| Phase | Outcome | Status |
|-------|---------|--------|
| 0 | Platform contract, log slugs, `mcpclient` stdio spike | `done` |
| 1 | `mcpmgr`, `catalog`, composite executor, YAML `mcp`, harness wiring | `done` |
| 2a | Native RAG expansion tools in harness loop | `done` |
| 2b | Blessed `fake_echo` sidecar + catalog doc + YAML test | `done` |
| 2c | SQLite `000012`, `/api/ui/mcp/*`, harness MCP panel | `done` |
| 3 | task-orchestrator workflow MCP | `deferred` |
| 4 | Meta-tools, IDE HTTP facade | `deferred` |

**Gate:** `make precommit` green at end of each implementer/verifier pair and at universe verification.

---

## Implementation train (logical PR order)

The work was delivered bottom-up; this is the recommended merge order for future contributors reviewing history.

1. **Contract + transport** — `internal/mcpclient`, fake stdio server, slug helpers, operator slugs in registry.
2. **Catalog + manager** — `internal/mcptools/catalog`, `internal/mcpmgr`, `chimera.yaml` `mcp` block, env allowlist deny-by-default.
3. **Harness** — `CompositeExecutor`, `NativeExecutor`, `mcp_wiring.go`, dynamic tool declarations, `CloseMCP` on shutdown.
4. **Expansion** — RAG expansion tools via `ExpansionService` when `rag.tooling.enabled`.
5. **Sidecar** — `docs/reference/mcp-sidecar-catalog.md`, `chimera.example.yaml`, blessed YAML integration test.
6. **Operator** — migration `000012_mcp_servers.sql`, `operatorstore/mcp.go`, `runtime/mcp.go`, admin API + harness UI panel.

**Key wiring points (code map):**

- `meta_policy.go` → `wireToolExecutor` / `buildNativeToolExecutor`
- `tools_stage.go` → injected declarations from executor surface (not hardcoded-only when MCP/native expansion on)
- `assistant_chat.go` → `ChatResolved` / effective MCP config per turn

---

## Agent orchestration (planets + astronomers)

Each phase used one **implementer** (planet) and one **verifier** (astronomer). Verifiers could fix gaps and re-run `make precommit`.

| Phase | Implementer | Verifier |
|-------|-------------|----------|
| 0 | Mercury | Tycho Brahe |
| 1 | Venus | Johannes Kepler |
| 2a | Earth | Galileo |
| 2b | Mars | Copernicus |
| 2c | Jupiter | William Herschel |

**Universe verification:** orchestrator re-ran `make precommit` and checked plan phases vs repo layout (`mcpclient`, `mcpmgr`, `catalog`, harness tests, `api/mcp`, migration `000012`).

---

## Delivery checklist (Phases 0–2c)

| Gate | Result |
|------|--------|
| Plan acceptance (per phase sections) | Pass |
| `make precommit` | Pass |
| Platform contract + pipeline known gaps | Updated |
| Operator slugs (`tools.mcp.*`, `tools.policy.denied`) | Registered |
| Feature records (`gateway-mcp-tool-backends`, `operator-assistants`, pipeline) | Synced |
| Gallery / harness preview (`mcp.preview_tools`) | Present |
| Hard cut: SQLite overrides YAML when server rows exist | Yes |
| Phases 3–4 not started | Confirmed deferred |

---

## Known follow-ups (not blocking sign-off)

1. **`tools.policy.denied` on invoke** — disallowed MCP slug may still surface as native “unknown tool” in edge cases; listing is deny-by-default.
2. **No live-LLM CI e2e** — harness MCP round-trip uses stubbed completions; behavior covered by loop + catalog tests.
3. **MCP server create wizard** — use `PUT /api/ui/mcp/servers/{id}` or YAML until a later UX phase.
4. **Phase 3** — task-orchestrator registration recipe + optional fake gate-error test when product wants workflow MCP.
5. **Phase 4** — catalog meta-tools / `POST /mcp` facade — separate plan if pursued.
6. **`operator-model-assist-tools.md`** — separate v0.5 train; must not attach developer MCP profile to model-assist.

---

## Operator quick start

1. Enable `tool_executor` on an assistant harness module; set `config_json.mcp.enabled` when using YAML bindings.
2. Seed `mcp.servers` + `mcp.assistants` in `chimera.yaml` (see `chimera.example.yaml` and `fake_echo` in sidecar catalog), **or** create rows via `/api/ui/mcp/servers` (SQLite then owns servers/bindings).
3. Bind server + allow tools on the assistant harness **Workspace tools** card → **Keep MCP bindings**.
4. Chat on the assistant path; injected tools include native (+ expansion when RAG tooling on) + allowed MCP slugs (e.g. `fake_echo__echo`).

---

## Big Lady sign-off

**Verdict:** Phases **0 through 2c** of [`gateway-mcp-tool-backends`](gateway-mcp-tool-backends.md) are **accepted for merge and operator trial**. The harness MCP universe is coherent: thin gateway, external workflow domain deferred, deny-by-default catalogs, native-first dispatch, precommit as law.

Phases **3–4** remain intentionally off the critical path. When you pick them up, open a new plan slice or reactivate deferred sections—do not expand scope silently in the platform contract.

*Signed off — The Big Lady, orchestrator.*  
*Delivery record created after planet/astronomer agent train; universe `make precommit` verified.*
