# Version 0.4 - Assistant turn harness

| Field | Value |
|-------|-------|
| **Doc kind** | `version-roadmap` |
| **Owners / areas** | Gateway runtime, operator SQLite, chat path, RAG, workspaces, operator embed UI (settings, chat, logs) |
| **Status** | `draft` |
| **Targets** | Gateway v0.4 |
| **Last updated** | See git history |
| **Supersedes / superseded by** | Builds on [`version-v0.3.md`](version-v0.3.md); supersedes prior v0.4 ensemble-first framing |

## At a glance

**v0.4** builds the **assistant turn harness**: when an operator picks a **named assistant** in chat (single-model picker UX — one visible model choice, even when the gateway uses multiple upstream models internally), the gateway runs a configurable multi-stage workflow inside a **single client request** — intent signals, retrieval planning, primary model execution, optional evaluation and escalation, then one reply. Each assistant toggles harness modules on or off and holds per-module settings. Stages are visible in conversation logs and chat turn details.

Supporting work in the same train: **workspace policy** (sensitivity, cloud eligibility, file permissions), **per-assistant retrieval**, and **gateway-native workspace file tools** (read/write within permission). MCP tool backends, peer routing, and broad settings-search themes move to **v0.5** or **deferred** unless they unblock harness delivery.

| Focus | Outcome | Status |
|-------|---------|--------|
| [Turn harness (execution plan)](#turn-harness-execution-plan) | Composable per-assistant stages, turn envelope, module toggles, observability | `done` |
| [Client contract](#client-contract) | One `model` string per turn; gateway owns orchestration; no Continue dependency | `done` |
| [Per-assistant retrieval](#per-assistant-retrieval) | VM-scoped top_k, thresholds, skip rules on project/flavor/conversation scope | `done` |
| [Workspace policy](#workspace-policy) | Sensitivity, cloud rules, file action policy on workspace rows | `done` |
| [Gateway workspace tools](#gateway-workspace-tools) | Read/write files under workspace roots inside one chat turn | `done` |
| [Evaluator and escalation](#evaluator-and-escalation) | Unified module: single-pass first, multi-draft ensemble later, human escalation target | `done` |
| [Harness observability](#harness-observability) | Stage timeline in settings conversation cards and chat turn details | `done` |
| [Deferred to v0.5+](#deferred-to-v05) | MCP tools, peer backends, desired-state gateway, app-wide search, indexer Phase 7 | `deferred` |

**Execution plan:** [`plans/assistant-turn-harness.md`](plans/assistant-turn-harness.md) (umbrella index) — child plans `assistant-harness-*.md` cover runtime, settings, retrieval, workspace policy, intent, evaluator/escalation, tools, observability, and advanced modules.

---

## What this version is

**v0.4** is the **orchestration depth** milestone: the gateway stops being a thin router with global RAG and becomes a **turn harness** configurable per assistant. This supersedes the earlier v0.4 plan that treated **two-phase ensemble**, **`//deep` triggers**, and **external human escalation** as separate top-level features. Those capabilities are now **modules inside the harness** (evaluator `multi_draft` mode, escalation `human` target, VM-level depth triggers).

**Normative principles for v0.4**

1. **Assistant = harness profile** — module toggles and config in operator SQLite.
2. **Single HTTP request per user turn** — internal multi-stage loop with bounded `max_tool_rounds`, `max_escalation_rounds`, `max_upstream_calls_per_turn`, and wall-clock timeout.
3. **Meta-policy before LLM classifiers** — deterministic privacy and eligibility from workspace policy; classifiers may use LLM when enabled but cannot override workspace sensitivity.
4. **Ship thin, iterate stages** — Phase 1–4 deliver a working harness with parity behavior plus per-assistant retrieval; later phases add evaluator, tools, ensemble mode, and human escalation.
5. **Chimera chat is the agent surface** — file operations run in-gateway under workspace permission, not via VS Code Continue or external IDE agents.
6. **Explicit assistant id** — clients send a named VM id (e.g. `Research-1.0`); no reserved aliases. Single-model picker UX; internal fallback/evaluator/escalation is invisible to the client.

**Companion docs:** [`chimera.plan.md`](chimera.plan.md), [`design.md`](design.md), [`configuration.md`](configuration.md), [`plans/assistant-turn-harness.md`](plans/assistant-turn-harness.md), [`features/operator-assistants.md`](features/operator-assistants.md), [`features/gateway-chat-routing-pipeline.md`](features/gateway-chat-routing-pipeline.md), [`version-v0.5.md`](version-v0.5.md).

---

## Turn harness (execution plan)

**Goal:** Replace the fixed five-step assistant chat pipeline with a **composable harness**: stage registry, shared **turn envelope** JSON, per-assistant module toggles, and structured observability — while preserving existing fallback, tool router, and RAG behavior in Phase 1.

**Scope**

* **Stage contract** — Input/output: proxied chat JSON + `TurnEnvelope` + `TurnContext` (tenant, VM id, conversation id, scope, catalog snapshot). Stages register in explicit order; fail-safe defaults per kind ([`gateway-chat-routing-pipeline.md`](features/gateway-chat-routing-pipeline.md) extensibility rules).
* **Turn envelope** — Stable `schema_version: 1` artifact through all stages; redacted logging and `conversation_turns.harness_summary_json` persistence.
* **Modules** — Meta-policy, intent, resource planner, retrieval, tool router, tool executor, evidence aggregate, primary, evaluator, escalation, response builder, telemetry. See module table in [`plans/assistant-turn-harness.md`](plans/assistant-turn-harness.md).
* **Settings UI** — Harness section on assistant cards: enable/disable modules, per-module config (models, thresholds, evaluator mode).
* **Gallery / reference UI** — Every UI-bearing child plan ships multi-state fixtures on `/ui/settings/gallery` (see [gallery contract](plans/assistant-turn-harness.md#gallery--reference-ui-contract-normative)); reviewers must not need live CRUD to compare configuration states.
* **Evaluate API** — Dry-run pre-primary stages without upstream completion (extends today’s routing evaluate pattern).

**Phases (summary)** — see child plans under [`plans/assistant-turn-harness.md`](plans/assistant-turn-harness.md):

| Child plan | Theme |
|------------|--------|
| [runtime](plans/assistant-harness-runtime.md) | Harness runtime; turn envelope schema |
| [settings](plans/assistant-harness-settings.md) | VM module toggles (SQLite + API + settings UI) |
| [retrieval](plans/assistant-harness-retrieval.md) | Per-VM retrieval + evidence compression |
| [workspace policy](plans/assistant-harness-workspace-policy.md) | Workspace policy + meta-policy stage |
| [intent](plans/assistant-harness-intent.md) | Heuristic intent (+ optional LLM classifier) |
| [evaluator + escalation](plans/assistant-harness-evaluator-escalation.md) | Evaluator `single_pass`; escalation v1 |
| [workspace tools](plans/assistant-harness-workspace-tools.md) | Gateway workspace tools |
| [observability](plans/assistant-harness-observability.md) | Conversation views + chat turn details |
| [advanced modules](plans/assistant-harness-advanced-modules.md) | Evaluator `multi_draft`; human escalation |

**Acceptance**

* Child plans [runtime](plans/assistant-harness-runtime.md) through [retrieval](plans/assistant-harness-retrieval.md) marked done when parity chat works with per-assistant retrieval toggles and stage logs visible.
* UI-bearing phases include gallery fixtures per [gallery contract](plans/assistant-turn-harness.md#gallery--reference-ui-contract-normative).
* Each child plan delivery: `make precommit` passes; design-validator run for UI plans; no legacy dual-path (see [delivery gates](plans/assistant-turn-harness.md#delivery-gates-normative)).
* Feature records updated for [`gateway-chat-routing-pipeline.md`](features/gateway-chat-routing-pipeline.md) and [`operator-assistants.md`](features/operator-assistants.md) when Phase 1 ships.

**Status:** `done`

---

## Client contract

**Goal:** Operators and integrations send **one model string** per chat turn; the gateway performs all orchestration invisibly.

**Scope**

* **Assistant id** — Primary path: `body.model` resolves through the assistant registry to an **explicit** operator-named id (e.g. `Research-1.0`). No reserved aliases.
* **Direct upstream** — `provider/model` ids still proxy without harness (escape hatch).
* **Single request** — Client does not run tool loops or multi-hop agent logic; workspace scope via `X-Chimera-Project` + `X-Chimera-Flavor-Id` (gateway derives workspace policy from DB lookup). `X-Chimera-Workspace-Id` is conversation-history metadata only.
* **Response metadata** — `X-Chimera-Upstream-Model`, `X-Chimera-RAG-Hits`, optional `X-Chimera-Harness-Summary` (redacted envelope).
* **Streaming** — Flexible per VM: evaluator `stream_policy` (`immediate`, `gate_on_evaluator`, `buffer_until_complete`) may defer client streaming until internal stages complete. See [evaluator plan](plans/assistant-harness-evaluator-escalation.md).

**Acceptance**

* Chimera `/ui/chat` completes turns without external agent tooling.
* Documented contract in [`configuration.md`](configuration.md): model string, headers, harness summary header.

**Status:** `done` — wire contract documented in [`configuration.md`](configuration.md) (model string, scope headers, response metadata, streaming policy, Chimera chat behavior).

---

## Per-assistant retrieval

**Goal:** Retrieval knobs live on the assistant and apply to the request’s **tenant + project + flavor + conversation** scope — not only gateway-global defaults.

**Scope**

* Config per VM: `top_k`, `score_floor`, `max_context_chars`, `compress_strategy` (`none`, `truncate`, `summarize`), `summarize_model_id`, `skip_if` conditions.
* Pluggable `EvidenceCompressor` interface; summarize ships in v0.4 with fail-open to truncate.
* Harness retrieval stage replaces hard-coded global-only path for VM chat.
* Evidence written to turn envelope and injected context block.
* Global `rag.enabled` still gates whether retrieval can run at all.

**Acceptance**

* Two VMs with different `top_k` on the same workspace produce different hit counts and envelope retrieval fields.
* Disabled retrieval module on VM → no inject even when global RAG is on.

**Status:** `done`

---

## Workspace policy

**Goal:** Workspaces carry **sensitivity** and **permission** metadata so meta-policy can enforce cloud and file eligibility before any LLM classifier runs.

**Scope**

* New workspace fields: `sensitivity` (`public` \| `internal` \| `private`), `allow_cloud`, `allow_cloud_summary_only`, `file_action_policy` (`none` \| `read` \| `read_write`).
* Settings UI on workspace cards; migration in operator SQLite.
* Meta-policy harness stage sets `TurnEnvelope.scope` and vetoes disallowed routes/tools.
* Scope from `X-Chimera-Project` + `X-Chimera-Flavor-Id`; gateway looks up workspace row (lowest id wins if ambiguous). See [workspace policy plan](plans/assistant-harness-workspace-policy.md).

**Acceptance**

* `private` workspace with `allow_cloud: false` prevents cloud upstream models from receiving full retrieval context.
* `file_action_policy: none` blocks file tools even if primary model requests them.

**Status:** `done`

---

## Gateway workspace tools

**Goal:** Models act on indexed workspace files **inside the gateway** during one chat turn — read, list, search, write within configured roots and permission.

**Scope**

* `ToolExecutor` interface; native implementations first (MCP adapters in v0.5).
* Path resolution relative to workspace `paths[]`; reject escapes outside roots.
* Internal `max_tool_rounds` loop in primary stage before client response.
* VM module toggle `tool_executor`.
* Gateway-injected tool declarations only when module enabled; client `tools` ignored on VM path.
* Atomic writes; model-facing paths relative, not absolute host paths.

**Acceptance**

* `read_write` workspace: write tool creates/updates file under watched root in one HTTP turn.
* `read`-only: write attempts fail with policy error in envelope and logs.

**Status:** `done`

---

## Evaluator and escalation

**Goal:** One **evaluator module** with multiple implementation modes; **escalation module** decides re-retrieve, chain walk, ensemble, or human handoff.

**Scope**

* **v0.4 MVP:** `evaluator.mode: single_pass` — small model returns confidence and issues; feeds escalation.
* **Escalation v1:** `re_retrieve`, `fallback_chain`; bounded rounds.
* **Advanced modules shipped:** [advanced modules plan](plans/assistant-harness-advanced-modules.md) — `evaluator.mode: multi_draft` and human escalation.
* Flexible `stream_policy` per VM (see [evaluator plan](plans/assistant-harness-evaluator-escalation.md)).

**Acceptance**

* Single-pass evaluator populates `TurnEnvelope.evaluation` and can trigger escalation v1.
* Multi-draft mode runs parallel available fallback drafts, synthesizes one answer, evaluates it, and logs draft/synthesis phases. Human escalation emits configured named surfaces, a privacy disclosure, and a paste-back delimiter after internal remediation exhausts.
* Human escalation path documented end-to-end when advanced modules Phase 2 ships.

**Status:** `done`

---

## Harness observability

**Goal:** Operators see **what the harness did** on each turn — in settings conversation cards and in chat UI — without reading raw gateway logs.

**Scope**

* Slugs: `harness.stage.started`, `harness.stage.completed`, `harness.escalation.*` with `assistant_id`, `turn_index`, `stage`, `duration_ms`.
* Extend conversation card derive ([`log-conversations.md`](plans/archive/log-conversations.md)) to group harness lines per turn.
* Chat UI: collapsible **Turn details** on assistant messages from `harness_summary_json`.
* VM scoped log panel filters harness events.

**Acceptance**

* Turn with retrieval + evaluator shows ordered stage pills in settings conversation expanded view.
* History reload shows turn details without log replay.

**Status:** `done`

---

## Deferred to v0.5+

The following appeared in prior v0.4 drafts; they are **not** core to the harness train unless a dependency forces earlier delivery.

| Theme | Disposition | Notes |
|-------|-------------|-------|
| **Gateway MCP / MCP tool router** | v0.5 | [`plans/gateway-mcp-tool-backends.md`](plans/gateway-mcp-tool-backends.md) — v0.4 defines `ToolExecutor`; MCP registers as adapter |
| **Peer backends** | v0.5 or later | Cross-host upstream routing |
| **Operator desired-state / model-assisted config** | v0.5 | [`version-v0.5.md`](version-v0.5.md) |
| **Settings and application search** | v0.5 | [`plans/embedui-settings-card-cleanup.md`](plans/archive/embedui-settings-card-cleanup.md) |
| **Indexer Phase 7 (model-assisted strategy)** | v0.5 | [`plans/indexer.md`](plans/archive/indexer.md) |
| **Indexer manifest ingest (line metadata)** | Parallel / optional | [`plans/indexer-manifest-ingest.md`](plans/indexer-manifest-ingest.md) — benefits RAG citations; not harness blocker |
| **Workspace embedding scope unions** | Parallel / optional | Base + flavor union retrieval — align with per-assistant retrieval when implemented |
| **Configuration in desktop UI (YAML parity)** | v0.5 | Harness settings ship on VM cards in v0.4 |
| **Env precedence contract** | Draft plan | [`plans/env-precedence-contract.md`](plans/env-precedence-contract.md) |

---

## Explicitly not this version

* **No MCP tool execution** as the primary tool path — gateway-native workspace tools only.
* **No Continue / Cline / external IDE agent** requirement for file operations.
* **No gateway queue** or priority scheduler (v0.8 theme).
* **No TLS/mTLS hardening** requirement (v0.7 theme).
* **No default gateway-on-gateway** peer chaining.

---

## Verification

| Area | Quick check |
|------|-------------|
| Harness runtime | Existing VM chat parity after stage refactor; `harness.stage.*` in logs |
| VM module toggles | Disable retrieval on one VM; other VM still retrieves on same scope |
| Per-VM retrieval | Different `top_k` → different hits and envelope fields |
| Workspace policy | `allow_cloud: false` blocks cloud route with private workspace |
| Gateway file tools | Write under `read_write` root in one turn; denied on `read`-only |
| Evaluator single pass | Confidence in envelope; escalation triggers on threshold |
| Multi-draft mode | N drafts + synthesize + one client answer when VM mode enabled |
| Human escalation | Privacy line + paste-back merge when Phase 12 shipped |
| Observability | Stage timeline in conversation card; turn details in chat history |
| Client contract | One model string; harness invisible except optional summary header |

---

## See also

* [`plans/assistant-turn-harness.md`](plans/assistant-turn-harness.md) — umbrella index and resolved decisions
* Child plans: `plans/assistant-harness-*.md`
* [`version-v0.3.md`](version-v0.3.md) — previous version (assistants, onboarding)
* [`version-v0.5.md`](version-v0.5.md) — next version (MCP, desired-state gateway, deferred themes)
* [`design.md`](design.md) — north-star architecture (update non-goals when harness ships)
* [`chimera.plan.md`](chimera.plan.md) — product requirements (deterministic-routing note superseded for v0.4+ orchestration depth)
* [`plans/README.md`](plans/README.md) — engineering plan index
