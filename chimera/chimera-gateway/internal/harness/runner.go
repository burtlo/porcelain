package harness

import (
	"context"
	"errors"
	"time"

	"github.com/lynn/porcelain/internal/naming"
)

// Runner executes registered harness stages in order.
type Runner struct {
	stages []Stage
}

// NewRunner returns a runner for the given ordered stages.
func NewRunner(stages ...Stage) *Runner {
	return &Runner{stages: stages}
}

// StageMeta is the stable stage id and harness module id for logs and docs.
type StageMeta struct {
	Name   string
	Module string
}

// Stages returns ordered stage metadata (skips nil entries).
func (r *Runner) Stages() []StageMeta {
	if r == nil {
		return nil
	}
	out := make([]StageMeta, 0, len(r.stages))
	for _, stage := range r.stages {
		if stage == nil {
			continue
		}
		out = append(out, StageMeta{Name: stage.Name(), Module: stage.Module()})
	}
	return out
}

// Run executes all stages. Returns nil when the turn completes successfully or
// after ErrTurnComplete from the terminal fallback stage.
func (r *Runner) Run(ctx context.Context, tc *TurnContext, body Body) error {
	if r == nil || tc == nil {
		return nil
	}
	env := newEnvelope(tc)
	tc.Envelope = env
	for _, stage := range r.stages {
		if stage == nil {
			continue
		}
		logStageStarted(tc, stage, env)
		start := time.Now()
		err := stage.Run(ctx, tc, env, body)
		outcome := stageOutcome(err)
		if err == nil {
			RecordStageCompletion(env, stage.Name())
		}
		logStageCompleted(tc, stage, env, start, outcome)
		if err == nil {
			continue
		}
		var abort *AbortError
		if errors.As(err, &abort) {
			return abort
		}
		if errors.Is(err, ErrTurnComplete) {
			return nil
		}
		return err
	}
	return nil
}

func stageOutcome(err error) string {
	if err == nil {
		return "ok"
	}
	var abort *AbortError
	if errors.As(err, &abort) {
		return "abort"
	}
	if errors.Is(err, ErrTurnComplete) {
		return "complete"
	}
	return "error"
}

func logStageStarted(tc *TurnContext, stage Stage, env *TurnEnvelope) {
	if tc == nil || tc.RouteLog == nil || stage == nil {
		return
	}
	tc.RouteLog.Debug("harness stage started",
		"msg", naming.MsgHarnessStageStarted,
		"stage", stage.Name(),
		"module", stage.Module(),
		"assistant_id", env.AssistantID,
		"turn_index", env.TurnIndex,
		"timeline_kind", naming.TimelineKindBroker,
	)
}

func logStageCompleted(tc *TurnContext, stage Stage, env *TurnEnvelope, start time.Time, outcome string) {
	if tc == nil || tc.RouteLog == nil || stage == nil {
		return
	}
	tc.RouteLog.Debug("harness stage completed",
		"msg", naming.MsgHarnessStageCompleted,
		"stage", stage.Name(),
		"module", stage.Module(),
		"assistant_id", env.AssistantID,
		"turn_index", env.TurnIndex,
		"duration_ms", time.Since(start).Milliseconds(),
		"outcome", outcome,
		"timeline_kind", naming.TimelineKindBroker,
	)
}

// DefaultRunner returns the v0.4 parity pipeline matching the pre-refactor
// handleAssistantChat order.
func DefaultRunner() *Runner {
	return NewRunner(
		StackResolveStage{},
		MetaPolicyStage{},
		IntentStage{},
		ToolRouterStage{},
		ToolExecutorStage{},
		RetrievalStage{},
		RequestWitnessStage{},
		InitialPickStage{},
		FallbackProxyStage{},
	)
}

// PrePrimaryRunner executes only deterministic, non-proxy stages for operator
// dry-runs. It intentionally excludes transforms and retrieval injection.
func PrePrimaryRunner() *Runner {
	return NewRunner(
		StackResolveStage{},
		MetaPolicyStage{},
		IntentStage{},
	)
}
