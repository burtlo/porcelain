# Project agent skills

Skills under `.cursor/skills/` guide harness and operator delivery work.

| Skill | Use for |
|-------|---------|
| [harness-plan-delivery](harness-plan-delivery/SKILL.md) | Mark a `assistant-harness-*` plan done (`make precommit`, hard cut, docs) |
| [harness-stage-implementer](harness-stage-implementer/SKILL.md) | Add/extend `internal/harness` stages |
| [harness-gallery-fixtures](harness-gallery-fixtures/SKILL.md) | Multi-state `/ui/settings/gallery` samples |
| [harness-ui-design-validator](harness-ui-design-validator/SKILL.md) | UI look-and-feel + gallery review |
| [operator-log-slug-registry](operator-log-slug-registry/SKILL.md) | `messages.yaml` slugs + contract regenerate |
| [feature-record-sync](feature-record-sync/SKILL.md) | Update `docs/features` + plan As-built |

**Typical loop:** stage-implementer → (UI) gallery-fixtures → design-validator → (logs) slug-registry → plan-delivery → feature-record-sync.

Umbrella: [`docs/plans/assistant-turn-harness.md`](../../docs/plans/assistant-turn-harness.md).
