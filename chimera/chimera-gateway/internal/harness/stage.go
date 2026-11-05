package harness

import (
	"context"
	"encoding/json"
	"errors"
)

// Body is the proxied OpenAI chat completion JSON body passed between harness stages.
type Body map[string]json.RawMessage

// ErrTurnComplete signals the harness finished and wrote the HTTP response.
var ErrTurnComplete = errors.New("harness: turn complete")

// AbortError stops the harness and returns a gateway JSON error to the client.
type AbortError struct {
	Status int
	Body   map[string]any
}

func (e *AbortError) Error() string {
	return "harness: abort"
}

// Stage is one ordered step in the assistant turn harness.
type Stage interface {
	// Name is a stable stage id for logs (e.g. "tool_router").
	Name() string
	// Module is the harness module id (e.g. "tool_router", "retrieval").
	Module() string
	Run(ctx context.Context, tc *TurnContext, env *TurnEnvelope, body Body) error
}
