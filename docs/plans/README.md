# Engineering plans

**Plans** describe what to build and track delivery phases. They are **not** the source of truth for current behavior.

**As-built behavior** lives in [`docs/features/`](../features/README.md).  
**Authority map / current release:** [`docs/CURRENT.md`](../CURRENT.md).

| | Open plans (`docs/plans/`) | Archived plans (`docs/plans/archive/`) | Feature records |
|--|---------------------------|----------------------------------------|-----------------|
| Question | What should we build next? | Why / how was this delivered? | What exists and must stay true? |
| Lifecycle | `draft` → `active` → `shipped` / `done` | Historical delivery | `current` / `partial` / `deprecated` |
| Agent use | **Implement from these** (with feature records) | Rationale only | Behavior, invariants, routes, code map |

**New plan:** copy [`_template.md`](_template.md), rename, fill in, delete authoring notes. Keep it here while `draft` or `active`. When it ships, set status, add **As-built**, and move the file to [`archive/`](archive/) unless it remains linked from an active version roadmap (see below).

---

## Active

| Plan | Feature record(s) / notes |
|------|---------------------------|
| [operator-embed-ui-mobile-layout.md](operator-embed-ui-mobile-layout.md) | [operator-embed-ui-mobile-layout](../features/operator-embed-ui-mobile-layout.md) (`active` — Phases 1–3 shipped) |

---

## Version 0.4 — assistant turn harness (execution plans)

Linked from [`version-v0.4.md`](../version-v0.4.md). These plans stay in the open tree as delivery history and acceptance references even though most phases are `done`.

| Plan | Feature record(s) |
|------|-------------------|
| [assistant-turn-harness.md](assistant-turn-harness.md) | [operator-assistants](../features/operator-assistants.md), [gateway chat routing pipeline](../features/gateway-chat-routing-pipeline.md), [indexer workspaces](../features/indexer-workspaces.md), [operator conversation history](../features/operator-conversation-history.md), [operator chat UI](../features/operator-chat-ui.md) |
| [assistant-harness-runtime.md](assistant-harness-runtime.md) | [gateway chat routing pipeline](../features/gateway-chat-routing-pipeline.md), [operator assistants](../features/operator-assistants.md) |
| [assistant-harness-settings.md](assistant-harness-settings.md) | [operator assistants](../features/operator-assistants.md) |
| [assistant-harness-retrieval.md](assistant-harness-retrieval.md) | [operator assistants](../features/operator-assistants.md), [gateway chat routing pipeline](../features/gateway-chat-routing-pipeline.md) |
| [assistant-harness-workspace-policy.md](assistant-harness-workspace-policy.md) | [indexer workspaces](../features/indexer-workspaces.md), [gateway chat routing pipeline](../features/gateway-chat-routing-pipeline.md) |
| [assistant-harness-intent.md](assistant-harness-intent.md) | [operator assistants](../features/operator-assistants.md), [gateway chat routing pipeline](../features/gateway-chat-routing-pipeline.md) |
| [assistant-harness-evaluator-escalation.md](assistant-harness-evaluator-escalation.md) | [operator assistants](../features/operator-assistants.md), [gateway chat routing pipeline](../features/gateway-chat-routing-pipeline.md) |
| [assistant-harness-workspace-tools.md](assistant-harness-workspace-tools.md) | [operator assistants](../features/operator-assistants.md), [gateway chat routing pipeline](../features/gateway-chat-routing-pipeline.md), [indexer workspaces](../features/indexer-workspaces.md) |
| [assistant-harness-observability.md](assistant-harness-observability.md) | [operator conversation history](../features/operator-conversation-history.md), [operator chat UI](../features/operator-chat-ui.md), [operator log message registry](../features/operator-log-message-registry.md) |
| [assistant-harness-advanced-modules.md](assistant-harness-advanced-modules.md) | [operator assistants](../features/operator-assistants.md), [gateway chat routing pipeline](../features/gateway-chat-routing-pipeline.md), [operator chat UI](../features/operator-chat-ui.md) |
| [log-conversations.md](log-conversations.md) | [operator-settings-ui](../features/operator-settings-ui.md) — harness timeline extends conversation cards |
| [indexer.md](indexer.md) | [indexer](../features/indexer.md) and related indexer features — v0.4 deferred indexer Phase 7 theme |

---

## Version 0.5 — draft execution plans

Linked from [`version-v0.5.md`](../version-v0.5.md).

| Plan | Summary |
|------|---------|
| [gateway-mcp-tool-backends.md](gateway-mcp-tool-backends.md) | **Shipped** Phases 0–2c — see [delivery record](gateway-mcp-tool-backends-delivery.md); Phases 3–4 deferred |
| [operator-model-assist-tools.md](operator-model-assist-tools.md) | Model-assist API-only tools; hard deny of developer MCP profile |
| [indexer-embedding-model-and-workspace-purge.md](indexer-embedding-model-and-workspace-purge.md) | Operator embedding model selector; workspace delete drops vector collection |
| [operator-workspace-search.md](operator-workspace-search.md) | Direct workspace search API, `/ui/search`, ribbon nav |
| [assistant-harness-workspace-search-quality.md](assistant-harness-workspace-search-quality.md) | Native chat `search` quick wins; staged lexical→semantic pipeline; optional fact store |
| [embedui-settings-card-cleanup.md](embedui-settings-card-cleanup.md) | Settings feed and card component consistency refactor |

---

## Draft — other future work

| Plan | Summary |
|------|---------|
| [indexer-manifest-ingest.md](indexer-manifest-ingest.md) | Manifest-only ingest, line-number snippets (v0.4 optional) |
| [env-precedence-contract.md](env-precedence-contract.md) | Unified env/config precedence |
| [operator-cli.md](operator-cli.md) | `chimeractl` / operator CLI |
| [internal-embedding-provider-exploration.md](internal-embedding-provider-exploration.md) | Exploration — not a product feature |

---

## Archive

**[archive/](archive/)** holds completed plans (infra, v0.1–v0.3 delivery, shipped operator surfaces, and one-off fixes). Indexes below are for discovery; the files themselves are historical.

### With feature records

| Plan | Feature record(s) |
|------|-------------------|
| [chimera-stack-config.md](archive/chimera-stack-config.md) | [chimera-stack-config](../features/chimera-stack-config.md), [product naming](../features/product-naming-contract.md), [locus-desktop-supervisor](../features/locus-desktop-supervisor.md) |
| [indexer-workspaces-sqlite-gateway-api.md](archive/indexer-workspaces-sqlite-gateway-api.md) | [indexer-workspaces](../features/indexer-workspaces.md) |
| [indexer-workspaces-accurate-reporting.md](archive/indexer-workspaces-accurate-reporting.md) | [indexer-workspaces](../features/indexer-workspaces.md), [health/logs](../features/indexer-health-and-operator-logs.md) |
| [indexer-scan-and-fanout-jobs.md](archive/indexer-scan-and-fanout-jobs.md) | [indexer-ingest-pipeline](../features/indexer-ingest-pipeline.md) |
| [indexer-health-and-quiet-logs.md](archive/indexer-health-and-quiet-logs.md) | [indexer-health-and-operator-logs](../features/indexer-health-and-operator-logs.md) |
| [indexer-sync-state-sqlite-and-force-reindex.md](archive/indexer-sync-state-sqlite-and-force-reindex.md) | [indexer-ingest-pipeline](../features/indexer-ingest-pipeline.md) |
| [indexer-memory-usage-analysis.md](archive/indexer-memory-usage-analysis.md) | [indexer](../features/indexer.md) (analysis / mitigations) |
| [operator-chat-ui.md](archive/operator-chat-ui.md) | [operator-chat-ui](../features/operator-chat-ui.md) |
| [operator-conversation-history.md](archive/operator-conversation-history.md) | [operator-conversation-history](../features/operator-conversation-history.md), [session auth](../features/operator-ui-session-auth.md), [operator SQLite](../features/operator-sqlite-store.md) |
| [unified-logs-operator-shell.md](archive/unified-logs-operator-shell.md) | [operator-settings-ui](../features/operator-settings-ui.md), [ribbon](../features/operator-left-navigation-ribbon.md) |
| [embedui-operator-settings-routes.md](archive/embedui-operator-settings-routes.md) | [operator-settings-ui](../features/operator-settings-ui.md), [ribbon](../features/operator-left-navigation-ribbon.md) |
| [embedui-feed-log-service-split.md](archive/embedui-feed-log-service-split.md) | [operator-settings-ui](../features/operator-settings-ui.md) |
| [log-presentation-layer.md](archive/log-presentation-layer.md) | [operator-log-message-registry](../features/operator-log-message-registry.md), [settings UI](../features/operator-settings-ui.md) |
| [operator-message-registry.md](archive/operator-message-registry.md) | [operator-log-message-registry](../features/operator-log-message-registry.md) |
| [virtual-models-operator.md](archive/virtual-models-operator.md) | [operator-assistants](../features/operator-assistants.md), [gateway chat routing pipeline](../features/gateway-chat-routing-pipeline.md) |
| [remove-legacy-gateway-routing.md](archive/remove-legacy-gateway-routing.md) | [operator-assistants](../features/operator-assistants.md), [gateway chat routing pipeline](../features/gateway-chat-routing-pipeline.md) |
| [provider-model-availability.md](archive/provider-model-availability.md) | [operator-provider-model-availability](../features/operator-provider-model-availability.md) |
| [context-window-admission.md](archive/context-window-admission.md) | [context-window-admission](../features/context-window-admission.md) (`partial`) |
| [adminui-filesystem-dev-mode.md](archive/adminui-filesystem-dev-mode.md) | [operator-ui-filesystem-dev-mode](../features/operator-ui-filesystem-dev-mode.md) |
| [harness-eval-stream-and-tools-fixes.md](archive/harness-eval-stream-and-tools-fixes.md) | [gateway chat routing pipeline](../features/gateway-chat-routing-pipeline.md) |

### Platform contracts

| Plan | Feature record(s) |
|------|-------------------|
| [vectorstore-broker-wrapper-hard-cut.md](archive/vectorstore-broker-wrapper-hard-cut.md) | [wrapper binary contract](../features/chimera-wrapper-binary-contract.md), [product naming](../features/product-naming-contract.md), [structured log lines](../features/structured-operator-log-lines.md) |
| [v0-3-naming-migration.md](archive/v0-3-naming-migration.md) | [product naming](../features/product-naming-contract.md) |
| [log-supervisor-normalization-fidelity.md](archive/log-supervisor-normalization-fidelity.md) | [structured log lines](../features/structured-operator-log-lines.md), [indexer health/logs](../features/indexer-health-and-operator-logs.md), [settings UI](../features/operator-settings-ui.md) |
| [locus-desktop-supervisor-contract.md](archive/locus-desktop-supervisor-contract.md) | [locus-desktop-supervisor](../features/locus-desktop-supervisor.md) (`partial`) |

### Plan only (infra / tooling / scaffolding)

| Plan | Notes |
|------|-------|
| [chimera-gateway-refactor.md](archive/chimera-gateway-refactor.md) | v0.3 gateway modularization |
| [chimera-gateway-package-boundaries.md](archive/chimera-gateway-package-boundaries.md) | Package layout |
| [desktop-ui.md](archive/desktop-ui.md) | Locus desktop shell (partially superseded by ribbon + settings) |
| [makefile.md](archive/makefile.md) | Build tooling |
| [upstream-llm-bifrost.md](archive/upstream-llm-bifrost.md) | BiFrost upstream integration |
| Embed UI scaffolding | [embedui-component-system.md](archive/embedui-component-system.md), [embedui-component-gallery.md](archive/embedui-component-gallery.md), [embedui-theme-styleguide.md](archive/embedui-theme-styleguide.md), [embedui-dynamic-provider-cards.md](archive/embedui-dynamic-provider-cards.md), [embedui-event-log-panel.md](archive/embedui-event-log-panel.md), [embedui-logs-workspaces-merge.md](archive/embedui-logs-workspaces-merge.md) |
| Log plumbing | [log-gateway.md](archive/log-gateway.md), [log-bifrost.md](archive/log-bifrost.md), [log-qdrant.md](archive/log-qdrant.md), [log-view-refactor.md](archive/log-view-refactor.md), [log-view-indexer.md](archive/log-view-indexer.md), [logs-ui-page-data-refreshing.md](archive/logs-ui-page-data-refreshing.md), [supervisor-info-log-trim.md](archive/supervisor-info-log-trim.md) |

### Superseded

| Plan | Superseded by |
|------|---------------|
| [logs-ui-maintainability.md](archive/logs-ui-maintainability.md) | [chimera-gateway-refactor.md](archive/chimera-gateway-refactor.md) Phases 5–6; as-built in [operator-settings-ui](../features/operator-settings-ui.md) |
