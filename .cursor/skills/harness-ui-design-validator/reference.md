# Harness UI design reference

## Canonical surfaces

| Surface | Path |
|---------|------|
| Settings gallery | `chimera/.../embedui/settings/gallery.html` + `embedui/gallery/*` |
| VM cards | `embedui/settings/render/cards/adminAssistants.js` |
| VM actions | `embedui/settings/handlers/assistantsAdmin.js` |
| Part registry | `embedui/settings/card-parts-registry.md` |
| Tokens / CSS | `embedui/theme-tokens.css`, `embedui/settings.css` |
| Gallery contract | `docs/plans/assistant-turn-harness.md` § Gallery |

## Patterns to copy

### Section shell
```html
<details class="sum-asst-section" data-vm-section="harness" data-ui-part="assistant.harness">
  <summary class="sum-asst-section__hdr">…</summary>
  <div class="sum-asst-section__body">…</div>
</details>
```

### Header toggles
Use `vmSectionHdrToggleHtml` / `sum-router-toggle` + `sum-asst-hdr-toggle-label` / `sum-asst-hdr-toggle-state muted` — same as routing policy and tool router.

### Toolbar
`sum-asst-section__toolbar` with leading + actions; configure/save/cancel via shared edit toolbar icons when editing dense config.

### Gallery
- Mount via `gallery-card-fixtures.js` using production `buildAssistantCardHtml`
- Multiple states = multiple fixture VMs or sequential mounts with labels
- Caption part slugs under the demo (`styleguide-part-slugs`)

## Anti-patterns

- Hand-written gallery HTML that duplicates card markup
- New card wrapper with borders/shadows that siblings do not use
- Instant-on evaluator / stream_policy controls before those plans ship (keep reserved/hidden/disabled) — log with Fix-by evaluator plan
- Enabling tool executor as if shipped when workspace-tools plan is still `todo` — disabled + reason only
- Inventing new toggle widgets when `sum-router-toggle` exists
- Dual-read / legacy aliases for harness APIs or config (“compat for old installs”) — hard cut; greenfield only

## Related shipped UI to compare

Open `/ui/settings/gallery` § Assistants and Workspaces before judging new harness sections.

## Plan delivery

Before marking a harness child plan `done`: run this skill at **delivery** scope and `make precommit` (see umbrella delivery gates).
