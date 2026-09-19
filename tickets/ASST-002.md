---
id: ASST-002
title: Rename Virtual Models to Assistants (full-stack)
type: Story
labels: [naming, ui, api, schema, docs, migration]
app: porcelain
---

# ASST-002 — Rename Virtual Models to Assistants (full-stack)

## Summary

Rename the product concept **Virtual Model(s)** to **Assistant(s)** across operator UI, present-tense documentation, REST API paths, SQLite schema, internal Go/JS/TS identifiers, operator log summaries, and related tests. Historical delivery plans stay on the old name.

## Description

Identifier **ASST-002** tracks local Foundry intake for Chimera porcelain (`C:\Users\lynnv\src\porcelain`).

Recent virtual-model harness work established this concept throughout the operator experience. Product naming is shifting to **Assistants** everywhere operators and integrators encounter the term — not only visible copy, but also API routes, persisted schema, and code identifiers.

This is a **cross-cutting rename**, not a documentation-only pass. Expect coordinated changes across migrations, gateway handlers, embed UI modules, operatorcopy/message registry, harness code, feature records, and tests.

### Resolved product decisions (do not re-litigate unless blocked)

| Topic | Decision |
|-------|----------|
| Section titles / nav | Plural **Assistants** for product-area headings |
| `/ui/chat` model selector | Relabel to **Assistant(s)**, not just surrounding copy |
| `docs/version-v0.*` | Leave unchanged as historical snapshots |
| `.cursor/skills` and agent-facing harness skill copy | **Out of scope** for this story |
| Internal embed dev docs (`CARD_INVENTORY`, `card-parts-registry`, JS comments) | Update to Assistant terminology |
| `docs/plans/**` | **Do not modify** — keep virtual model naming where historically accurate |

### In scope

- Operator web UI: `/ui/settings`, `/ui/chat`, gallery/styleguide, embed UI strings and related JS modules
- REST API paths currently under `/api/ui/virtual-models` (and related JSON field names)
- SQLite operator schema: `virtual_models` and related tables/columns; new numbered migration(s) with documented upgrade path for existing installs
- Internal identifiers: Go packages/types (`virtualmodel`, `VirtualModel`, etc.), JS/TS modules and filenames, structured log slug keys where product-facing
- Present-tense operator documentation: `docs/features/`, `docs/README.md`, `docs/chimera.plan.md`, configuration/runbook docs operators read today
- Operator log **summaries** in settings UI (`operator_copy.js`, `internal/operatorcopy/messages.yaml` display strings)
- Feature record rename: `docs/features/operator-virtual-models.md` → `operator-assistants.md` with redirect note in `docs/features/README.md` only
- Gallery fixtures and gateway/UI tests updated for Assistant labeling
- `make precommit` (or equivalent) passes without regression

### Out of scope

- `docs/plans/**` — historical delivery records
- `docs/version-v0.*` — historical release snapshots
- Harness eval pack outside the repo (`harness-eval/`)
- `.cursor/skills/**` and other agent-only instruction files

### Known touchpoints (starting map for research)

- Migrations: `migrations/chimera-gateway/operator/000002_virtual_models.sql`, `000009_virtual_model_harness_modules.sql`
- Backend: `chimera/chimera-gateway/internal/virtualmodel/`, `internal/operatorstore/virtual_models.go`, `internal/operatorapi/virtual_models.go`, `chimera/chimera-gateway/internal/server/adminui/api/virtualmodels/`
- Embed UI: `virtualModels*.js`, `adminVirtualModels.js`, `operator_copy.js`
- Docs: `docs/features/operator-virtual-models.md`
- Operator copy: `internal/operatorcopy/messages.yaml`

### Human decisions likely needed during planning

- Hard-cut API rename vs temporary `/virtual-models` alias routes
- Whether to rename Go directory `virtualmodel` or only types/symbols
- Whether to rename log slug keys or only operator-visible summaries
- Migration strategy for existing desktop installs with populated `virtual_models` rows

## Acceptance criteria

- [ ] **AC-1** All operator-visible UI labels, headings, buttons, toasts, and empty states display **Assistant** or **Assistants** (grammar-correct singular/plural) and never **Virtual model** or **Virtual models**.
- [ ] **AC-2** `docs/features/` records describe **Assistants** as the operator term; cross-links between feature records use Assistant terminology.
- [ ] **AC-3** `docs/README.md`, `docs/chimera.plan.md`, and other present-tense operator documentation (excluding `docs/plans/`) consistently use **Assistant(s)** terminology.
- [ ] **AC-4** No files under `docs/plans/**` are modified; historical plan records retain virtual model naming where historically accurate.
- [ ] **AC-5** REST API paths, database schema objects (SQLite tables, columns, migrations), and internal code identifiers (Go/JS/TS symbols, module and file names, structured log slug keys) are renamed from `virtual-model` / `virtual_model` / `virtualModel` conventions to `assistant` / `assistants` conventions consistently across the codebase, with compatibility shims or migration steps documented where breaking changes affect persisted data or external clients.
- [ ] **AC-6** Gallery fixtures render with Assistant labeling and `make precommit` (or equivalent gateway/UI checks) passes without regression.
- [ ] **AC-7** `operator_copy.js` harness and routing summary strings shown to operators in settings UI use Assistant terminology.
- [ ] **AC-8** Primary virtual-models feature record is renamed to `operator-assistants.md` with a redirect note from the old filename in `docs/features/README.md` only.

## Foundry notes

- **App folder:** `C:\Users\lynnv\src\porcelain`
- **Factory root:** `C:\Users\lynnv\src\kwiktrip\.github-private-eval-foundry-approach`
- **Profile:** `local-markdown`
- **Run mode:** `implementation`
- **Interaction mode:** `drive_to_pr` (AC freeze gates remain human)
- **Risk tier:** `high` (schema + API + cross-cutting rename)
- Read porcelain plan/feature docs before changing operator or harness behavior (`docs/features/README.md`, `docs/plans/` for context only — do not edit plans)
