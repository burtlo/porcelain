# Plan: Harness workspace search quality and staged retrieval

| Field | Value |
|-------|-------|
| **Doc kind** | `feature-plan` |
| **Owners / areas** | Gateway harness, workspace tools, RAG, operator assistants |
| **Status** | `draft` |
| **Targets** | Gateway v0.5+ (Phase 1–2 shippable without MCP); Phase 3–4 train |
| **Last updated** | 2026-10-01 |
| **Supersedes / superseded by** | Optional follow-on called out in [`gateway-mcp-tool-backends.md`](gateway-mcp-tool-backends.md) (native `search` quality); extends [`assistant-harness-workspace-tools.md`](assistant-harness-workspace-tools.md) (v0.4 native tools, `done`) |
| **As-built** | None — link to [`gateway-chat-routing-pipeline.md`](../features/gateway-chat-routing-pipeline.md) and workspace-tools section of [`operator-assistants.md`](../features/operator-assistants.md) when shipped |

## At a glance

Chat **native `search`** today is a line-level substring walk (see `WorkspaceExecutor.search` in `chimera/chimera-gateway/internal/harness/tools/tools.go`). It is easy to misuse (globs, whole-sentence queries, `path: "/"`) and does not compete with **indexed semantic search** (`/ui/search`, harness RAG). This plan improves **filesystem search** immediately, then optionally adds a **staged retrieval pipeline** (lexical → semantic at stepped confidence → targeted reads) with per-step scores in the turn envelope—and documents an alternative **fact-store** pattern for multi-step tool turns.

| Phase | Outcome | Status |
|-------|---------|--------|
| [Phase 1 — Native search quick wins](#phase-1--native-search-quick-wins) | Predictable grep-like tool + docs; fewer false “searches” | `todo` |
| [Phase 2 — Search orchestration inside tool_executor](#phase-2--search-orchestration-inside-tool_executor) | Gateway picks lexical vs read vs RAG expansion from intent + rules | `todo` |
| [Phase 3 — Staged confidence retrieval](#phase-3--staged-confidence-retrieval) | Semantic passes at multiple thresholds; stop when confidence sufficient | `todo` |
| [Phase 4 — Fact store (optional track)](#phase-4--fact-store-optional-track) | Structured facts with confidence; actions gated on evidence count | `todo` |

**Depends on:** [workspace tools](assistant-harness-workspace-tools.md) (`done`), [retrieval](assistant-harness-retrieval.md), [intent](assistant-harness-intent.md), [operator workspace search](operator-workspace-search.md) (UI RAG API, Phases 1–3 shipped)

**Does not replace:** [gateway MCP tool backends](gateway-mcp-tool-backends.md) — external MCP search sidecars remain optional; native improvements stay in `WorkspaceExecutor` or harness orchestration.

**Delivery gates:** `make precommit` per phase; operator log slugs for new stages via [operator log message registry](../features/operator-log-message-registry.md); update feature record when behavior is normative.

---

## Background

**Problem.** Operators and eval assistants expect “search the repo” to behave like ripgrep plus semantic recall. As-built native `search` only does `strings.Contains` per line, walks a directory tree (or a single file if `path` resolves to a file), caps at 100 hits, and returns `path:line:text`. Models often emit JSON tutorials, wrong paths (`/`), or glob queries (`*.md`). RAG injects snippets on the chat path but does not write files or run deterministic link extraction.

**Related surfaces**

| Surface | Role |
|---------|------|
| Native `search` tool | Synchronous filesystem grep in `tool_executor` loop |
| Harness `retrieval` stage | Vector hits → system message (score floor, top-k) |
| `POST /api/ui/rag/search` | Operator UI semantic search ([operator-workspace-search](operator-workspace-search.md)) |
| RAG expansion tools | Chunk-oriented reads when index + `content_sha256` available |

**Related docs:** [`gateway-rag-ingest-and-retrieval.md`](../features/gateway-rag-ingest-and-retrieval.md), [`gateway-mcp-tool-backends.md`](../features/gateway-mcp-tool-backends.md), [`assistant-harness-internals.md`](../features/assistant-harness-internals.md).

---

## Phase 1 — Native search quick wins

**Goal.** Make native `search` honest and harder to misuse without adding new infrastructure.

**Deliverables**

- **Tool schema copy** — Clarify in OpenAI function descriptions: literal substring only; no globs/regex unless a future `search_mode` is set; `path` is workspace-relative directory or file; default `"."`.
- **Single-file mode** — When `path` resolves to a file, search only that file (already true for `WalkDir` on a file); return clear error if path is missing.
- **Safer defaults** — Reject or normalize `path` values that escape roots (`/`, `..`); map `"."` when empty.
- **Result shaping** — Stable sort by path then line; optional `max_hits` (default 100); header line when truncated (`search truncated at 100 hits`).
- **Empty-result message** — Tool result text: `no lines matched; query must appear verbatim on a line` plus hint to try shorter tokens or `read_file`.
- **Optional `search_file` alias** — Same implementation as `search` with `path` required to be a file (reduces model confusion for “search inside version-v0.5.md”).
- **Optional `extract_markdown_links`** — New native tool: `read_file` + parse `](path)` / relative links; filter by prefix (e.g. `docs/plans/`); return deduped path list (fits “list plans referenced in charter” without fake semantic search).
- **Harness tests** — Unit tests for file-only path, link extraction, empty query, truncation.

**Acceptance**

- Prompt: search `docs` for substring `docs/plans/` returns real line hits from the repo.
- Prompt: extract links with prefix `docs/plans/` from `docs/version-v0.5.md` returns paths without requiring the model to hallucinate a search API.
- Tool declarations visible to broker no longer imply glob semantics.

**Status:** `todo`

---

## Phase 2 — Search orchestration inside tool_executor

**Goal.** Before or alongside the model’s tool choice, the gateway applies **deterministic routing rules** so common tasks do not depend on a small coder model picking the right JSON shape.

**Deliverables**

- **`search_routing` config** on `tool_executor` harness module (JSON): rules keyed by intent tags or regex on user message (e.g. `extract.*link`, `search.*file`, `find.*plan`).
- **Rule actions** (internal, not extra OpenAI tools): `lexical_search`, `read_file`, `extract_markdown_links`, `rag_query` (call existing `RAG().Retrieve` with workspace scope), `list_dir`.
- **Integration with intent stage** — Extend envelope `intent` (or `intent.tags`) with `search_task` enum: `lexical`, `semantic`, `in_file`, `extract_links`, `unknown` (heuristics first; optional LLM assist reuses [intent plan](assistant-harness-intent.md) pattern).
- **Logging** — Slugs e.g. `harness.search.routed`, `harness.search.step_completed` with `strategy`, `hit_count`, `confidence` (when applicable).
- **Fail-open** — If routing cannot run, fall back to today’s model-driven tool loop.

**Acceptance**

- Eval-style prompt “find plan links in `docs/version-v0.5.md` and write `LynnLovesPlans.md`” triggers `extract_markdown_links` or `read_file` + write without relying on `search` with `*.md`.
- Turn logs show routing decision and tool invoked.

**Status:** `todo`

---

## Phase 3 — Staged confidence retrieval

**Goal.** For questions that need **semantic** recall, run a **pipeline** inside the gateway (not in the client model) with **monotonic confidence** after each step and early exit when sufficient.

**Conceptual pipeline**

```text
User query + workspace scope
  → (A) Intent + rules (Phase 2)
  → (B) Lexical pass: native search or ripgrep-equivalent, high precision
  → (C) If confidence < T1: RAG retrieve @ score_threshold S1 (higher)
  → (D) If confidence < T2: RAG @ S2 (lower), optional query rewrite (cheap model or template)
  → (E) If confidence < T3: read_file / expansion tools on top hits
  → (F) Stop: pass evidence to model or synthesize answer
```

**Deliverables**

- **`SearchPipeline` service** in gateway (package TBD: `internal/harness/searchpipe` or extend `internal/rag`): inputs `TurnContext`, query, scope; outputs `[]Evidence` + `PipelineReport`.
- **Confidence model (v1, simple)** — Combine: lexical hit count (capped), max RAG score, count of distinct sources, optional cross-encoder rerank later. Store in turn envelope extension: `retrieval.pipeline_steps[]` with `{stage, hit_count, max_score, confidence, stopped_reason}`.
- **Tunable thresholds** — Per-assistant JSON: `S1 > S2 > S3`, `T1 > T2 > T3`, `max_steps` (default 3). Defaults align with existing RAG floor in gateway config.
- **Reading rules** — Documented table: which stages may call `read_file` / expansion (e.g. only after semantic hit with score ≥ S2).
- **Operator visibility** — Harness summary header (eventually post-loop) includes pipeline report; debug bundle mentions grep slugs.
- **Differentiation from chat RAG** — Pipeline runs on **tool intent** or auto-route (Phase 2), not only on every chat turn (avoids double-RAG unless configured).

**Acceptance**

- Query “supervisor plans” returns lexical hits when `supervisor` appears in `docs/plans`; otherwise semantic step lowers threshold until hits or abstain.
- Envelope or logs show at least two steps with confidence values for a multi-step retrieval turn.

**Status:** `todo`

---

## Phase 4 — Fact store (optional track)

**Goal.** Support multi-step tasks (“collect evidence, then write file”) without trusting the model to remember: **write facts with confidence** during the tool loop and **gate actions** on evidence quality.

**Conceptual model**

```text
Tool results → Fact{ subject, predicate, value, source, confidence, tool_call_id }
  → FactStore (per-turn, in-memory; optional persist to conversation metadata later)
  → Policy: e.g. require ≥ N facts with confidence ≥ C before auto write_file
  → Generator stage uses facts as structured context (not raw tool JSON)
```

**Deliverables**

- **`Fact` and `FactStore`** types on `TurnContext`; append from native/MCP tool results (read_file excerpt, search hit, link extract).
- **Confidence on write** — Heuristic: lexical match = 0.9, RAG score normalized, model paraphrase = 0.5 unless backed by tool output.
- **Action gates** — VM config: `min_facts_for_write`, `min_aggregate_confidence`; when unmet, return operator-visible message instead of executing `write_file` from model JSON alone.
- **Alternative to full pipeline** — Can ship fact gating **without** Phase 3 semantic ladder (smaller increment).

**Acceptance**

- Turn that only hallucinates “file written” without tool success does not pass write gate.
- Turn with `extract_markdown_links` + 5 paths at confidence 1.0 allows `write_file` for list output.

**Status:** `todo`

---

## External landscape (informal survey, 2026-10)

Not endorsements—patterns to borrow or integrate via MCP later.

| Project | Pattern | Relevance to Porcelain |
|---------|---------|------------------------|
| [codetect](https://github.com/brian-lai/codetect) | ripgrep + Ollama embeddings, `hybrid_search_v2`, MCP | Tier B sidecar or reference for Phase 3 hybrid |
| [semcode](https://github.com/GoodbyePlanet/semcode) | Tree-sitter symbols, dense + BM25, RRF in Qdrant | Similar to existing Qdrant stack; multi-repo oriented |
| [codesearch](https://github.com/flupkede/codesearch) | Rust MCP, vector + BM25, AST chunks, serve-mode instructions | Good MCP “when to use search vs grep” copy |
| [VecGrep](https://github.com/VecGrep/VecGrep) | Semantic + graph hybrid, token budgets | Confidence / top-k discipline |
| [grepika](https://github.com/agentika-labs/grepika) | FTS5 + ripgrep + query intent detection | Phase 2 intent routing analogue |
| [Azure AI Search agentic retrieval](https://learn.microsoft.com/en-us/azure/search/agentic-retrieval-overview) | Query planning, parallel subqueries, semantic ranker | Phase 3 enterprise reference architecture |
| [Google agentic RAG (Gemini Enterprise)](https://research.google/blog/unlocking-dependable-responses-with-gemini-enterprise-agent-platforms-agentic-rag/) | “Sufficient context” agent, iterative search | Phase 3 stop rules + abstain |
| [Domino agentic RAG blueprint](https://domino.ai/resources/blueprints/building-agentic-rag-pipeline-with-multi-model-orchestration) | Intent → strategy → rerank → constraint check | Phase 2–3 orchestration |
| [HeshamFS/agentic-rag](https://github.com/HeshamFS/agentic-rag) | CRAG confidence thresholds, Self-RAG, multi-agent | Phase 3–4 confidence + fact gating vocabulary |

**Porcelain-specific stance**

- **Keep** gateway-native tools reserved names; MCP hybrids remain **renamed slugs** ([gateway-mcp-tool-backends](../features/gateway-mcp-tool-backends.md)).
- **Prefer** reusing **existing Qdrant collections** and `RAG().Retrieve` for semantic steps rather than a second index in Phase 3.
- **Optional** blessed MCP: ripgrep-only sidecar for power users who outgrow native lexical search—does not block Phase 1.

---

## Design choices (open)

| Question | Default for Phase 1–2 | Revisit in |
|----------|----------------------|------------|
| Regex in native `search`? | Defer; use MCP ripgrep if needed | Phase 1+ |
| Link extraction as tool vs mode? | Dedicated `extract_markdown_links` | Phase 1 |
| Auto-run pipeline without model tool call? | Phase 2 routing only on strong heuristics | Phase 3 |
| Persist facts across turns? | Per-turn only in Phase 4 | Conversation store later |
| Cross-encoder rerank? | Defer; use RAG score + lexical count first | Phase 3 |

---

## Non-goals

- Replacing the indexer or embedding pipeline ([indexer-embedding-model-and-workspace-purge.md](archive/indexer-embedding-model-and-workspace-purge.md) owns model selection).
- Full-text search engine inside gateway (Elasticsearch, etc.) for v1.
- Porting semcode/codesearch into Go as a hard dependency.
- Operator `/ui/search` UX redesign (covered by [operator-workspace-search.md](operator-workspace-search.md) Phase 4 wizard polish).

---

## References

- Code: `chimera/chimera-gateway/internal/harness/tools/tools.go` (`search`), `internal/rag/`, `internal/harness/tools_stage.go` (tool loop)
- Plans: [assistant-harness-workspace-tools.md](assistant-harness-workspace-tools.md), [gateway-mcp-tool-backends.md](gateway-mcp-tool-backends.md) (Tier A native search note)
- Eval context: [archive/harness-eval-stream-and-tools-fixes.md](archive/harness-eval-stream-and-tools-fixes.md)
