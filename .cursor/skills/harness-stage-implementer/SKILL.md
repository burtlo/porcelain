---
name: harness-stage-implementer
description: >-
  Implements or extends chimera-gateway assistant turn harness stages in
  internal/harness. Use when adding retrieval, intent, meta-policy, evaluator,
  escalation, tool executor, or other harness stages; when wiring VM module
  toggles into stages; or when editing TurnEnvelope / DefaultStages.
---

# Harness stage implementer

Patterns for `chimera/chimera-gateway/internal/harness/` stages used by the
v0.4 assistant turn harness.

## Before coding

1. Read the child plan + [umbrella](../../docs/plans/assistant-turn-harness.md) module table.
2. Read [gateway-chat-routing-pipeline.md](../../docs/features/gateway-chat-routing-pipeline.md) for stage order invariants.
3. Verify current registry/stages in code (draft plans may lag).

## Stage contract

```go
type Stage interface {
  Name() string   // stable log id, e.g. "retrieval"
  Module() string // harness module id, e.g. "retrieval"
  Run(ctx context.Context, tc *TurnContext, env *TurnEnvelope, body Body) error
}
```

- **Fail-open** for transform/retrieval-style stages unless the plan says otherwise (log + continue).
- **AbortError** for client-visible hard stops (policy veto, misconfig).
- **ErrTurnComplete** only from the terminal primary/fallback stage after writing the HTTP response.
- Do **not** call upstream outside established chat/fallback helpers unless designing a new stage type documented in the pipeline feature.

## Wiring checklist

- [ ] Implement `Stage` in `stages.go` or a focused file under `internal/harness/`
- [ ] Register in `DefaultStages` / `NewRunner(...)` order (see `runner.go`)
- [ ] Gate on `tc.Stack.VM.HarnessEnabled(operatorstore.HarnessModule…)` when the module is toggleable
- [ ] Respect global prerequisites (e.g. retrieval still needs `tc.Resolved.RAG.Enabled` + `tc.RAG`)
- [ ] Mutate `TurnEnvelope` fields for the module; keep redaction rules (`redact.go`)
- [ ] Prefer existing `harness.stage.started` / `.completed` from the runner — add **extra** slugs only for meaningful events (`harness.escalation.*`, compress fallback, etc.)
- [ ] New slugs → [operator-log-slug-registry](../operator-log-slug-registry/SKILL.md)
- [ ] Config on VM → `operatorstore` harness module `config_json` + compile into `assistant.Resolved` / `VMStack` as needed
- [ ] Tests in `harness/*_test.go` (parity / skip / fail-open)
- [ ] Hard cut — no legacy parallel path beside the new stage

## Module ids (store constants)

`retrieval` · `intent` · `evaluator` · `escalation` · `tool_executor`  
(+ always-on concepts: `meta_policy`, `primary`, `tool_router` — see umbrella)

## Envelope

Normative sketch in the umbrella plan. Lock field names when touching `envelope.go`; bump or document `schema_version` if breaking.

## After implementation

- Update pipeline / VM feature records when behavior ships ([feature-record-sync](../feature-record-sync/SKILL.md))
- UI toggles/config → settings card + [harness-gallery-fixtures](../harness-gallery-fixtures/SKILL.md)
- Plan delivery → [harness-plan-delivery](../harness-plan-delivery/SKILL.md)

## Anti-patterns

- Orchestrating multi-stage loops inside `assistant_chat.go` instead of registered stages
- Streaming policy forks scattered outside evaluator plan decisions
- Enabling `tool_executor` behavior before workspace-tools plan ships
- Silent skip without a debug log reason when a module is off

## References

- Code: `internal/harness/{stage,runner,stages,context,envelope,redact}.go`
- Store modules: `internal/operatorstore/assistant_harness.go`
- Runtime compile: `internal/assistant/registry.go`
