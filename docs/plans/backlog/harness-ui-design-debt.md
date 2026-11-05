# Harness UI design debt

Issues found by the [harness-ui-design-validator](../../../.cursor/skills/harness-ui-design-validator/SKILL.md) that were deferred.

**Rules:** Mid-phase debt is OK when a later phase/plan will fix it. Debt owned by a child plan must be cleared (or reassigned) before that plan is marked `done`. Plan delivery also requires `make precommit`.

| Date | Plan/phase | Issue | Why deferred | Fix by |
|------|------------|-------|--------------|--------|
| 2026-07-25 | [assistant-harness-retrieval](../assistant-harness-retrieval.md) delivery | Retrieval UI omits `skip_if` (empty query, no workspace, intent tags) | v0.4 shipped core numeric/strategy knobs only; API + store accept full config | Harness settings follow-up (no child plan yet) |
| 2026-07-25 | [assistant-harness-retrieval](../assistant-harness-retrieval.md) delivery | No `summarize_model_id` control when compression is Summarize | Same subset UI; gallery fixture carries model id in JSON only | Harness settings follow-up (no child plan yet) |
| 2026-07-25 | [assistant-harness-workspace-policy](../assistant-harness-workspace-policy.md) delivery | Managed workspace cards show policy editor only while Configure/edit is open (no read-only summary in view mode) | Edit-mode gallery fixtures cover acceptance; live feed matches other managed fields | Workspace card polish backlog |
