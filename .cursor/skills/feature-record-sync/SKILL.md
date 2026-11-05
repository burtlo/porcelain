---
name: feature-record-sync
description: >-
  Updates or creates docs/features records after shipping behavior, and links
  As-built from plans. Use when finishing a harness or operator plan, marking
  a plan done, or when invariants/API/code map changed and feature docs must
  match the codebase.
---

# Feature record sync

**Plans** = delivery intent/history. **Feature records** = what must stay true now.  
See [`.cursor/rules/docs-plans-vs-features.mdc`](../../rules/docs-plans-vs-features.mdc).

## When to run

- Child plan delivery ([harness-plan-delivery](../harness-plan-delivery/SKILL.md))
- Behavior shipped that changes invariants, routes, API, or operator UX
- User asks to update feature docs after implementation

## Workflow

1. Identify shipped behavior vs the draft/active plan.
2. Find existing records under `docs/features/` (prefer update over proliferating).
3. If none fit, copy [`docs/features/_template.md`](../../docs/features/_template.md).
4. Update sections that changed:
   - At a glance / operator-visible behavior
   - Invariants and decisions
   - Interfaces (routes, headers, APIs)
   - Code map
   - Verification commands
5. Set **Status** `current` or `partial` accurately.
6. Plan front-matter: **As-built** → feature path(s); plan **Status** → `done` / shipped when delivery complete.
7. Add/adjust row in [`docs/features/README.md`](../../docs/features/README.md) if new record.
8. Optionally move/note plan in [`docs/plans/README.md`](../../docs/plans/README.md) draft → shipped tables.

## Harness train — usual records

| Change | Likely feature record |
|--------|------------------------|
| VM toggles / harness API | `operator-assistants.md` |
| Stage order / chat path | `gateway-chat-routing-pipeline.md` |
| RAG per-assistant / compress | `gateway-rag-ingest-and-retrieval.md` |
| Workspace sensitivity / files | `indexer-workspaces.md` |
| Turn details / history JSON | `operator-conversation-history.md` |
| Chat Turn details UI | `operator-chat-ui.md` |
| New slugs | `operator-log-message-registry.md` (plus YAML) |

Only create a new `docs/features/assistant-turn-harness.md` (or similar) if the umbrella becomes a lasting cross-cutting contract beyond those pages.

## Verify against code

Do **not** trust draft plan text alone — confirm handlers, migrations, and UI exist. Remove doc claims for unshipped modules.

## Output

```markdown
## Feature record sync — {plan or topic}

### Updated
- path — what changed

### Created
- path | none

### Plan As-built
- set / already set

### Gaps left undocumented
- … | none
```

## Anti-patterns

- Editing only the plan and leaving feature records stale
- Documenting future phases as current behavior
- Duplicating full plan prose into the feature record (distill invariants)
