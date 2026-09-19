---
id: ASST-001
title: Rename Virtual Models to Assistants (operator-facing)
type: Story
labels: [docs, ui, naming]
app: porcelain
---

# ASST-001 — Rename Virtual Models to Assistants (operator-facing)

## Summary

Rename **Virtual Model(s)** to **Assistant(s)** across operator-facing documentation, the web UI, REST API paths, database schema, and internal code identifiers. Historical delivery plans keep the existing virtual model naming.

## Description

Identifier **ASST-001** tracks local Foundry intake for Chimera porcelain.

Recent virtual-model harness evaluation work established the concept throughout the operator experience. Product naming is shifting to **Assistants** for anything an operator reads in the UI or current documentation.

**In scope**

- Operator web UI copy on `/ui/settings`, `/ui/chat`, gallery/styleguide pages, and related embed UI strings
- Present-facing documentation: `docs/features/`, `docs/README.md`, `docs/chimera.plan.md`, configuration/runbook docs where operators see the term, version docs describing current product behavior
- Operator log message summaries shown in settings UI (`operator_copy.js` and related display strings)
- Feature record rename/update (e.g. `operator-virtual-models.md` → assistant naming) with cross-links fixed in **features** docs only

**Out of scope**

- `docs/plans/**` — historical delivery records keep **virtual model** naming
- Harness eval pack outside the repo (`harness-eval/`) only
- Eval harness pack outside the repo (`harness-eval/`)

## Acceptance criteria

- [ ] All operator-visible UI labels, headings, buttons, toasts, and empty states say **Assistant** / **Assistants** instead of **Virtual model** / **Virtual models** (grammar-correct singular/plural)
- [ ] `docs/features/` records describe **Assistants** as the operator term; links between feature records updated; primary VM feature record renamed or superseded with assistant naming
- [ ] `docs/README.md`, `docs/chimera.plan.md`, and other **current** operator docs (not `docs/plans/`) use **Assistant(s)** consistently
- [ ] `docs/plans/**` files are unchanged (still reference virtual models where historically accurate)
- [ ] REST paths, DB schema objects, and internal code identifiers are renamed to assistant/assistants conventions with migrations or compatibility notes where persisted data or external clients are affected
- [ ] Gallery fixtures render with Assistant labeling; existing tests pass (`make precommit` or equivalent gateway/UI checks)
