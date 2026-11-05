package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/assistant"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/chat"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/corpusstale"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness/tools"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/rag"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/testsupport"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/vectorstore"
	"github.com/lynn/porcelain/chimera/internal/config"
	"github.com/lynn/porcelain/internal/naming"
)

// Integration: native RAG expansion tool inside the bounded harness tool loop.
func TestRunWorkspaceToolLoop_ExpansionContextAround(t *testing.T) {
	root := t.TempDir()
	store, err := operatorstore.Open(filepath.Join(t.TempDir(), "operator.sqlite"), testsupport.GatewayOperatorMigrationsDir(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	workspace, err := store.CreateWorkspace(context.Background(), "tenant-a", "project-a", "flavor-a", []string{root}, operatorstore.WorkspacePolicy{
		FileActionPolicy: "read",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = workspace

	coords := vectorstore.Coords{TenantID: "tenant-a", ProjectID: "project-a", FlavorID: "flavor-a"}
	coll := vectorstore.CollectionName(coords)
	pid := vectorstore.PointID(coords, "doc.txt", 0)
	ragStore := newExpansionTestStore(t)
	if err := ragStore.Upsert(context.Background(), coll, []vectorstore.Point{{
		ID: pid, Vector: make([]float32, 8),
		Payload: vectorstore.Payload{TenantID: "tenant-a", ProjectID: "project-a", FlavorID: "flavor-a", Source: "doc.txt", Text: "indexed line one\nindexed line two"},
	}}); err != nil {
		t.Fatal(err)
	}
	rows := []operatorstore.CorpusSegmentRow{{
		SegmentID: pid, TenantID: "tenant-a", ProjectID: "project-a", FlavorID: "flavor-a",
		Source: "doc.txt", ContentSHA256: "sha256:doc", ChunkIndex: 0, ChunkCount: 1,
		StartLine: 1, EndLine: 2, VectorPointID: pid,
	}}
	if err := store.ReplaceCorpusSegmentsForSource(context.Background(), "tenant-a", "project-a", "flavor-a", "doc.txt", rows); err != nil {
		t.Fatal(err)
	}
	ragSvc, err := rag.New(rag.Options{Store: ragStore, Embedder: expansionTestEmbedder{dim: 8}, EmbeddingDim: 8})
	if err != nil {
		t.Fatal(err)
	}

	var upstreamCalls atomic.Int32
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch upstreamCalls.Add(1) {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{{
					"message": map[string]any{
						"role": "assistant",
						"tool_calls": []map[string]any{{
							"id": "call_exp_1", "type": "function",
							"function": map[string]string{
								"name":      "workspace_context_around",
								"arguments": `{"source":"doc.txt","line":1,"before_lines":0,"after_lines":1,"content_sha256":"sha256:doc"}`,
							},
						}},
					},
				}},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{{
					"message": map[string]string{"role": "assistant", "content": "expansion turn complete"},
				}},
			})
		}
	}))
	defer up.Close()

	tc := &TurnContext{
		W:              httptest.NewRecorder(),
		InitialModel:   "Dev-1.0",
		Timeout:        time.Minute,
		TenantID:       "tenant-a",
		ProjectID:      "project-a",
		FlavorID:       "flavor-a",
		RAG:            ragSvc,
		CorpusStale:    corpusstale.NewStore(),
		OperatorStore:  store,
		WorkspaceRoots: []string{root},
		Resolved: &config.Resolved{
			UpstreamBaseURL: up.URL,
			RAG:             config.RAG{Enabled: true, ToolingEnabled: true},
		},
		Stack: VMStack{
			VM: &assistant.Resolved{
				ModelID: "Dev-1.0",
				HarnessModules: map[string]assistant.HarnessModule{
					operatorstore.HarnessModuleToolExecutor: {Enabled: true, ConfigJSON: `{"max_tool_rounds":3}`},
				},
			},
		},
	}
	tc.ToolExecutor = buildNativeToolExecutor(tc, tc.WorkspaceRoots, tools.PolicyRead)

	msgs, _ := json.Marshal([]map[string]string{{"role": "user", "content": "expand context"}})
	body := Body{"model": json.RawMessage(`"Dev-1.0"`), "messages": msgs}
	env := &TurnEnvelope{SchemaVersion: 1, AssistantID: "Dev-1.0"}
	var logBuf bytes.Buffer
	tc.RouteLog = slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	tc.RequestID = "req-expansion"
	tc.ConversationID = "conv-expansion"
	tc.TurnIndex = 1
	rec := tc.W.(*httptest.ResponseRecorder)

	runWorkspaceToolLoop(context.Background(), tc, env, body, &chat.ProxyOpts{})
	logs := logBuf.String()
	for _, slug := range []string{
		naming.MsgHarnessToolLoopStarted,
		naming.MsgHarnessToolModelRound,
		naming.MsgHarnessToolInvokeStarted,
		naming.MsgHarnessToolInvokeCompleted,
		naming.MsgHarnessToolLoopFinished,
	} {
		if !strings.Contains(logs, slug) {
			t.Fatalf("missing log slug %s in:\n%s", slug, logs)
		}
	}
	if !strings.Contains(logs, `"tool_backend":"expansion"`) {
		t.Fatalf("missing expansion backend in logs:\n%s", logs)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "expansion turn complete") {
		t.Fatalf("final body=%s", rec.Body.String())
	}
	var messages []map[string]json.RawMessage
	if err := json.Unmarshal(body["messages"], &messages); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, msg := range messages {
		if string(msg["role"]) != `"tool"` {
			continue
		}
		if strings.Contains(string(msg["content"]), "indexed line one") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing expansion text in tool messages: %v", messages)
	}
	if upstreamCalls.Load() != 2 {
		t.Fatalf("upstream calls=%d", upstreamCalls.Load())
	}
}

type expansionTestStore struct {
	pts map[string][]vectorstore.Point
}

func newExpansionTestStore(t *testing.T) *expansionTestStore {
	t.Helper()
	return &expansionTestStore{pts: map[string][]vectorstore.Point{}}
}

func (s *expansionTestStore) EnsureCollection(context.Context, string, int) error { return nil }
func (s *expansionTestStore) Upsert(_ context.Context, c string, pts []vectorstore.Point) error {
	s.pts[c] = append(s.pts[c], pts...)
	return nil
}
func (s *expansionTestStore) Search(context.Context, string, []float32, int, float32, *vectorstore.Coords) ([]vectorstore.Hit, error) {
	return nil, nil
}
func (s *expansionTestStore) Health(context.Context) error { return nil }
func (s *expansionTestStore) Stats(_ context.Context, c string) (vectorstore.Stats, error) {
	return vectorstore.Stats{Collection: c, Points: int64(len(s.pts[c]))}, nil
}
func (s *expansionTestStore) DeleteBySource(context.Context, string, string) error { return nil }
func (s *expansionTestStore) DeleteCollection(context.Context, string) error       { return nil }
func (s *expansionTestStore) ScrollPoints(context.Context, string, *vectorstore.Coords, int, string) (vectorstore.ScrollBatch, error) {
	return vectorstore.ScrollBatch{}, nil
}
func (s *expansionTestStore) GetPoints(_ context.Context, c string, ids []string) ([]vectorstore.PointPayload, error) {
	byID := map[string]vectorstore.Point{}
	for _, p := range s.pts[c] {
		byID[p.ID] = p
	}
	var out []vectorstore.PointPayload
	for _, id := range ids {
		if p, ok := byID[id]; ok {
			out = append(out, vectorstore.PointPayload{ID: p.ID, Payload: p.Payload})
		}
	}
	return out, nil
}

type expansionTestEmbedder struct{ dim int }

func (e expansionTestEmbedder) EmbedBatch(context.Context, []string) ([][]float32, error) {
	return nil, nil
}
func (e expansionTestEmbedder) EmbedOne(context.Context, string) ([]float32, error) {
	return make([]float32, e.dim), nil
}
func (e expansionTestEmbedder) Model() string { return "test" }
