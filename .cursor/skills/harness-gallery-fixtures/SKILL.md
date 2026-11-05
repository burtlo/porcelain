---
name: harness-gallery-fixtures
description: >-
  Adds multi-state /ui/settings/gallery fixtures for assistant harness UI
  using production card builders. Use when implementing harness settings,
  workspace policy, retrieval config, observability, or any UI-bearing
  assistant-harness plan that requires gallery samples.
---

# Harness gallery fixtures

Extends **`GET /ui/settings/gallery`** so reviewers can compare harness
configuration states without live SQLite or chat.

## Contract

Umbrella: [`docs/plans/assistant-turn-harness.md`](../../docs/plans/assistant-turn-harness.md) § Gallery.

Rules:

1. Same production render modules as `/ui/settings` (via `gallery-card-fixtures.js`)
2. ≥2 meaningful configuration states, labeled, preferably side-by-side
3. Stable `id` anchor + top-nav link when adding a subsection
4. `data-ui-part` + `card-parts-registry.md` + `styleguide-part-slugs` caption
5. No hand-copied card HTML that can drift from builders

## Workflow

1. Read the child plan’s gallery inventory row.
2. Extend fixture data in `embedui/gallery/gallery-card-fixtures.js` (`adminStateCache`, `assistantDetails`, workspace caches, etc.).
3. Add mount point(s) in `embedui/settings/gallery.html` with short `styleguide-sub` labels.
4. Call production builders (`buildAssistantCardHtml`, workspace card builders, conversation/chat helpers when added).
5. Register new parts in `embedui/settings/card-parts-registry.md`.
6. Update nav links in `gallery.html` if a new subsection id is introduced.
7. Spot-check: with `CHIMERA_ADMINUI_ROOT` pointing at embed, refresh `/ui/settings/gallery`; optional `?parts=1`.

## File anchors

| Concern | Path |
|---------|------|
| Page | `…/embedui/settings/gallery.html` |
| Fixtures | `…/embedui/gallery/gallery-card-fixtures.js` |
| README | `…/embedui/gallery/README.md` |
| Parts | `…/embedui/settings/card-parts-registry.md` |
| VM card builder | `…/settings/render/cards/adminAssistants.js` |

## Patterns

**Second VM harness state** — add another `assistants[]` summary + matching `assistantDetails[id].harness_modules` (or module config), second `#gallery-fixture-…` mount, `setHtml` in `renderFixtures`.

**Workspace policy states** — multiple managed-workspace fixture objects (sensitivity / allow_cloud / file_action_policy) with separate mounts.

**Open the section under test** — set `assistantUi[id].sectionOpen.harness = true` (or relevant section) so demos show expanded controls.

## After fixtures

Run [harness-ui-design-validator](../harness-ui-design-validator/SKILL.md) on the UI phase.

## Anti-patterns

- Gallery-only markup duplicating `sum-asst-section` structure
- Single state when the plan lists multiple
- Forgetting part registry / slug caption
- Live-only controls with no gallery mount
