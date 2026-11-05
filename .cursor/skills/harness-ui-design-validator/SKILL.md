---
name: harness-ui-design-validator
description: >-
  Validates assistant harness UI and gallery fixtures against Chimera
  operator embed UI design patterns. Use after completing a harness plan phase,
  when delivering a harness plan (with make precommit), or when the user asks
  to design-validate harness work.
---

# Harness UI design validator

Run after each **UI-bearing** phase of `docs/plans/assistant-harness-*.md`,
and again at **plan delivery** (when marking the child plan shipped / `done`).

Backend-only phases (e.g. runtime with no controls) may skip visual checks but
still run `make precommit` at plan delivery.

## Hard-cut policy (normative)

- **No legacy support** — do not keep old endpoints, config keys, env aliases,
  dual-read paths, or “preserve pre-feature behavior for existing installs.”
  Assume **no existing users**; update call sites to the new path only.
- **Phase vs plan** — mid-plan phases may leave known gaps if a **later phase
  of the same train** will fix them; log those gaps. At **plan delivery**,
  acceptance must hold and debt owned by that plan must be fixed (or explicitly
  reassigned to a named later child plan).
- **`make precommit`** — required validation at **plan delivery**. Optional but
  encouraged after substantial phases. Failures must be fixed before the plan
  is marked done.

## When invoked

1. Identify scope: **phase review** or **plan delivery**.
2. Read the plan acceptance + gallery deliverables.
3. Diff / read changed embed UI, gallery, CSS, part-registry, and related API/store code.
4. Compare against [reference.md](reference.md).
5. For **plan delivery**, run `make precommit` from the repo root and include the result.
6. Produce the report. Fix must-fix items for this scope; log only per the table below.

## Validation checklist

### Plan / acceptance
- [ ] Phase or plan deliverables present (API, store, UI, gallery as listed)
- [ ] Gallery multi-state fixtures exist when the plan requires them
- [ ] Nav anchor + `styleguide-part-slugs` updated on `/ui/settings/gallery`
- [ ] `data-ui-part` slugs registered in `card-parts-registry.md`
- [ ] **Plan delivery:** `make precommit` passes
- [ ] **Plan delivery:** no leftover legacy/dual-path code for this feature surface

### Look and feel
- [ ] Reuses `sum-asst-section` / `sum-asst-section__hdr*` / `sum-router-toggle` (or documented sibling)
- [ ] Typography uses existing classes (`sg-op-card-note`, `muted`, `sum-section-label`)
- [ ] Toggles match routing/tool-router patterns
- [ ] Section open/edit/save/cancel mirrors other VM sections
- [ ] Stays within `theme-tokens.css` / `settings.css`
- [ ] Gallery fixtures call production builders (no duplicated card HTML)

### Accessibility / structure
- [ ] Controls labeled; toggles show On/Off (or equivalent) state text
- [ ] `details`/`summary` keyboard-usable like sibling sections
- [ ] Disabled-until-ready controls visibly disabled with short reason

## Output format

```markdown
## Harness UI validation — {plan} {Phase n | delivery}

**Verdict:** pass | pass-with-log | fail

### make precommit
- skipped (phase) | pass | fail (summary)

### Must fix
- …

### Fixed in this pass
- …

### Logged for later
Append to `docs/plans/backlog/harness-ui-design-debt.md`:
| Date | Plan/phase | Issue | Why deferred | Fix by |
|------|------------|-------|--------------|--------|

### Notes
- …
```

## Fix vs log

| Severity | Phase review | Plan delivery |
|----------|--------------|---------------|
| Breaks this phase acceptance, wrong wire-up, missing required gallery, broken primary a11y | **Fix now** | **Fix now** |
| Style drift / missing part slug | Fix if small; else **log** with Fix-by phase | **Fix now** (unless Fix-by is another named child plan) |
| Blocked on unshipped sibling plan | **Log** with child-plan link | **Log** only if that sibling is the Fix-by owner |
| Nice-to-have polish | **Log** | **Log** or fix |
| `make precommit` failure | Encourage fix; may log if later phase owns it | **Fix now** — plan cannot be `done` |

## How to run as a separate agent

Launch a `generalPurpose` Task agent with this skill and:

- Plan path + phase **or** “plan delivery”
- Changed UI/gallery/API files
- Instruction: follow this skill; run `make precommit` on delivery; fix or append backlog; return the report

## Additional resources

- [reference.md](reference.md) — class names, file anchors, anti-patterns
- Umbrella: `docs/plans/assistant-turn-harness.md` (gallery + delivery gates)
- Sibling skills: [harness-plan-delivery](../harness-plan-delivery/SKILL.md), [harness-gallery-fixtures](../harness-gallery-fixtures/SKILL.md), [feature-record-sync](../feature-record-sync/SKILL.md)
- Index: [README.md](../README.md)
