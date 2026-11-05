---
name: harness-plan-delivery
description: >-
  Closes out a assistant harness child plan: acceptance, gallery, feature
  records, hard-cut check, design validation, and make precommit. Use when
  marking a assistant-harness-* plan done, shipping a harness phase train,
  or the user asks for plan delivery / precommit delivery gates.
---

# Harness plan delivery

Orchestrates **plan delivery** for `docs/plans/assistant-harness-*.md`
(and the umbrella when wrapping up a slice). Mid-phase check-ins may skip this;
**marking a child plan `done` requires it**.

## Normative gates

From [`docs/plans/assistant-turn-harness.md`](../../docs/plans/assistant-turn-harness.md) § Delivery gates:

1. `make precommit` must pass
2. Debt owned by this plan cleared or reassigned with **Fix by** in `docs/plans/backlog/harness-ui-design-debt.md`
3. Hard cut — no legacy dual-path for this surface
4. UI plans: gallery contract + [harness-ui-design-validator](../harness-ui-design-validator/SKILL.md)

## Workflow

Copy and complete:

```
Delivery — {plan file}
- [ ] 1. Read plan acceptance + gallery deliverables
- [ ] 2. Verify code matches acceptance (tests / manual notes)
- [ ] 3. Hard-cut scan (no dual-read / compat aliases for this feature)
- [ ] 4. UI? gallery fixtures + part registry + nav ([harness-gallery-fixtures](../harness-gallery-fixtures/SKILL.md))
- [ ] 5. UI? design-validator agent (delivery scope)
- [ ] 6. New log slugs? ([operator-log-slug-registry](../operator-log-slug-registry/SKILL.md))
- [ ] 7. Feature records + As-built ([feature-record-sync](../feature-record-sync/SKILL.md))
- [ ] 8. Plan status → done; umbrella child table status; docs/plans/README if needed
- [ ] 9. make precommit (repo root)
- [ ] 10. Clear or reassign backlog rows owned by this plan
```

### Step details

**Hard-cut scan** — Search changed packages for: `legacy`, dual env keys, “preserve for existing”, fallback to old API paths. Remove or rewrite; greenfield only.

**Gallery** — Inventory in umbrella § Gallery; fixtures use production builders.

**Feature records** — Typical touches: `operator-assistants.md`, `gateway-chat-routing-pipeline.md`, and plan-specific features (RAG, workspaces, chat UI, conversation history). Create new feature record only if behavior is a distinct operator-visible surface.

**Plan front-matter** — Set **Status** `done` (or `shipped` if used); fill **As-built** link(s).

**precommit** — `make precommit` from repo root. On failure: fix and re-run; do not mark done.

## Output

```markdown
## Harness plan delivery — {plan}

**Verdict:** ready | blocked

### Checklist
(completed / remaining)

### make precommit
pass | fail — {summary}

### Feature records / As-built
- …

### Debt
cleared | reassigned: …

### Notes
- …
```

## Related skills

- [harness-ui-design-validator](../harness-ui-design-validator/SKILL.md)
- [harness-gallery-fixtures](../harness-gallery-fixtures/SKILL.md)
- [harness-stage-implementer](../harness-stage-implementer/SKILL.md)
- [operator-log-slug-registry](../operator-log-slug-registry/SKILL.md)
- [feature-record-sync](../feature-record-sync/SKILL.md)
