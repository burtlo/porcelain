package operatorapi

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAssistantListResponse_Marshal_UsesAssistantsKey(t *testing.T) {
	body, err := json.Marshal(AssistantListResponse{
		Assistants: []AssistantSummary{{ID: 1, ModelID: "Test-1.0.0", Name: "Test", Version: "1.0.0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	if !strings.Contains(raw, `"assistants"`) {
		t.Fatalf("expected assistants key: %s", raw)
	}
	if strings.Contains(raw, `"virtual_models"`) {
		t.Fatalf("unexpected virtual_models key: %s", raw)
	}
}

func TestGatewayState_Marshal_UsesAssistantIDKey(t *testing.T) {
	body, err := json.Marshal(GatewayState{
		Semver:      "0.1.0",
		AssistantID: "Chimera-0.1.0",
		Assistants:  []AssistantSummary{{ID: 1, ModelID: "Chimera-0.1.0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	if !strings.Contains(raw, `"assistant_id":"Chimera-0.1.0"`) {
		t.Fatalf("expected assistant_id: %s", raw)
	}
	if !strings.Contains(raw, `"assistants"`) {
		t.Fatalf("expected assistants array: %s", raw)
	}
	if strings.Contains(raw, `"virtual_model_id"`) {
		t.Fatalf("unexpected virtual_model_id key: %s", raw)
	}
}

func TestAssistantCreateRequest_Unmarshal_AcceptsModelID(t *testing.T) {
	var req AssistantCreateRequest
	if err := json.Unmarshal([]byte(`{"name":"Demo","version":"1.0.0","model_id":"Demo-1.0.0"}`), &req); err != nil {
		t.Fatal(err)
	}
	if req.ModelID != "Demo-1.0.0" || req.Name != "Demo" {
		t.Fatalf("req=%+v", req)
	}
}

func TestAssistantFallbackSaveRequest_Unmarshal_AcceptsFallbackChain(t *testing.T) {
	var req AssistantFallbackSaveRequest
	if err := json.Unmarshal([]byte(`{"fallback_chain":["groq/a","groq/b"]}`), &req); err != nil {
		t.Fatal(err)
	}
	if len(req.FallbackChain) != 2 || req.FallbackChain[0] != "groq/a" {
		t.Fatalf("chain=%+v", req.FallbackChain)
	}
}

func TestAssistantDetail_Marshal_UsesExpectedFieldNames(t *testing.T) {
	body, err := json.Marshal(AssistantDetail{
		AssistantSummary: AssistantSummary{ID: 2, ModelID: "X-1.0.0"},
		FallbackChain:    []string{"groq/x"},
		CreatedAt:        "2026-01-01T00:00:00Z",
		UpdatedAt:        "2026-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := string(body)
	for _, key := range []string{`"id":2`, `"model_id":"X-1.0.0"`, `"fallback_chain":["groq/x"]`} {
		if !strings.Contains(raw, key) {
			t.Fatalf("missing %s in %s", key, raw)
		}
	}
}
