// Package harness runs the assistant chat turn pipeline as ordered stages
// with a shared turn envelope.
//
// As-built behavior and code map: docs/features/assistant-harness-internals.md.
// Generated stage order: docs/generated/harness-stages.md (make harness-docs-generate).
//
// Fail-safe defaults by stage kind (normative for v0.4):
//
//   - Transform (tool router): fail-open — on disable, misconfig, or error, pass
//     the full request body upstream unchanged.
//   - Retrieval (RAG): fail-open — on disable, empty query, or retrieve error,
//     proceed without injected context.
//   - Policy / initial pick: fail-closed — when no upstream model resolves,
//     return 503 to the client.
//   - Fallback proxy: retriable upstream errors advance the chain; non-retriable
//     errors stop the walk per chat.WithAssistantFallback rules.
package harness
