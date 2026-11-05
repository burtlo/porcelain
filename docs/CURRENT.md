# Current documentation authority

Short map for humans and agents: **what is authoritative now**, what is historical, and where to pick up work.

| Question | Read |
|----------|------|
| What shipped and must stay true? | [`features/README.md`](features/README.md) |
| What can I implement next? | [`plans/README.md`](plans/README.md) — **Active**, **Version 0.5**, and **Draft** only |
| How do I install / run / configure? | [Runbooks in this folder](README.md#runbooks--install-run-configure) |
| Which release train are we on? | **v0.4.0** (assistants + turn harness; see below) |
| Why was something built this way? | [`plans/archive/`](plans/archive/README.md) (delivery history) |
| Product vision / normative requirements | [`chimera.plan.md`](chimera.plan.md) (vision; verify against features before implementing) |
| Stable integration notes | [`reference/`](reference/) |
| Ports, binaries, health URLs (curated) | [`reference/service-map.md`](reference/service-map.md) — verify defaults in code |
| Codegen / generated artifacts | [`generated/README.md`](generated/README.md) and paths under `docs/generated/` |

## Release line

**Current development line: v0.4.x** on `main` — **assistant turn harness** shipped; product rename **virtual models → assistants** landed in gateway APIs and operator SQLite.

| Doc | Role | Status |
|-----|------|--------|
| [`version-v0.1.md`](version-v0.1.md) | Shipped train | `shipped` |
| [`version-v0.1.1.md`](version-v0.1.1.md) | Shipped train | `shipped` |
| [`version-v0.2.md`](version-v0.2.md) | Shipped train | `shipped` |
| [`version-v0.3.md`](version-v0.3.md) | Shipped train (setup wizard themes may continue in settings) | `shipped` |
| [`version-v0.4.md`](version-v0.4.md) | Harness + workspace policy train | `active` — execution themes `done`; roadmap status `draft` until release sign-off |
| [`version-v0.5.md`](version-v0.5.md) | Desired-state gateway, search, settings cleanup | `draft` |

Version docs are **release-train roadmaps**, not the day-to-day work queue. Prefer an **Active**, **Version 0.5**, or **Draft** plan (or a feature record for follow-up) when starting implementation. Execution plans linked from a version doc may remain in `docs/plans/` even when phases are `done`.

## Document kinds

| Kind | Location | Lifecycle |
|------|----------|-----------|
| **Feature record** | `docs/features/` | As-built behavior; `current` / `partial` / `active` / `deprecated` |
| **Plan (open)** | `docs/plans/*.md` | `draft`, `active`, or version-linked execution history |
| **Plan (archive)** | `docs/plans/archive/` | Older shipped work — do not treat as a to-do list |
| **Version roadmap** | `docs/version-v0.*.md` | Release themes and acceptance narrative |
| **Runbook** | `docs/*.md` (install, config, …) | Operator how-to |
| **Reference** | `docs/reference/` | Stable external/integration knowledge |
| **Service map** | `docs/reference/service-map.md` | Hand-maintained binary/port/health index (not config schema) |
| **Generated** | `docs/generated/` | Curated codegen output index; see [`generated/README.md`](generated/README.md) |

## Agent rules (summary)

1. Implement from a **feature record** when one exists; use archived plans only for rationale.
2. Treat **draft** plans as intent — verify the codebase before assuming behavior exists.
3. After shipping, update or create the feature record; move the plan to `plans/archive/` when it is no longer linked from an active version roadmap.
4. Do not invent work from archived plans unless a version doc or open plan still points at them.

See also [`.cursor/rules/docs-plans-vs-features.mdc`](../.cursor/rules/docs-plans-vs-features.mdc).
