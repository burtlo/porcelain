package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lynn/porcelain/internal/naming"
)

func TestLegacyVirtualModelsRoutesReturn404(t *testing.T) {
	t.Setenv(naming.EnvBrokerAPIKeyTarget, "ukey")
	up := chimeraBrokerStubForUILogs(t)
	t.Cleanup(up.Close)

	rt := runtimeForUILogs(t, up.URL)
	seedChimeraTestVM(t, rt, "0.1.0", []string{"m"})
	front := httptest.NewServer(NewMux(rt, testLog(), nil, NewUIOptions()))
	t.Cleanup(front.Close)
	client := vmTestLoginClient(t, front.URL, "gw-ui-secret")

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/ui/virtual-models"},
		{http.MethodPost, "/api/ui/virtual-models"},
		{http.MethodGet, "/api/ui/virtual-models/1"},
		{http.MethodPost, "/api/ui/virtual-models/1/routing/evaluate"},
		{http.MethodGet, "/api/ui/virtual-models/1/harness"},
		{http.MethodPut, "/api/ui/virtual-models/1/fallback"},
		{http.MethodPost, "/api/ui/virtual-models/1/routing/generate"},
	}
	for _, tc := range cases {
		req, err := http.NewRequest(tc.method, front.URL+tc.path, strings.NewReader(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		if tc.method == http.MethodPost || tc.method == http.MethodPut {
			req.Header.Set("Content-Type", "application/json")
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("%s %s: status=%d want 404", tc.method, tc.path, res.StatusCode)
		}
	}
}

func TestUIStateUsesAssistantFieldsNotLegacyVirtualModels(t *testing.T) {
	t.Setenv(naming.EnvBrokerAPIKeyTarget, "ukey")
	up := chimeraBrokerStubForUILogs(t)
	t.Cleanup(up.Close)

	rt := runtimeForUILogs(t, up.URL)
	seedChimeraTestVM(t, rt, "0.1.0", []string{"m"})
	front := httptest.NewServer(NewMux(rt, testLog(), nil, NewUIOptions()))
	t.Cleanup(front.Close)
	client := vmTestLoginClient(t, front.URL, "gw-ui-secret")

	res, err := client.Get(front.URL + "/api/ui/state")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("state status=%d body=%s", res.StatusCode, b)
	}
	var st map[string]any
	if err := json.NewDecoder(res.Body).Decode(&st); err != nil {
		t.Fatal(err)
	}
	gw, ok := st["gateway"].(map[string]any)
	if !ok {
		t.Fatalf("gateway section missing: %+v", st)
	}
	if _, ok := gw["virtual_models"]; ok {
		t.Fatalf("gateway.virtual_models should be absent: %+v", gw)
	}
	if _, ok := gw["virtual_model_id"]; ok {
		t.Fatalf("gateway.virtual_model_id should be absent: %+v", gw)
	}
	if gw["assistant_id"] != "Chimera-0.1.0" {
		t.Fatalf("gateway.assistant_id=%v", gw["assistant_id"])
	}
	assistants, ok := gw["assistants"].([]any)
	if !ok || len(assistants) == 0 {
		t.Fatalf("gateway.assistants=%v", gw["assistants"])
	}
}

func TestUIAssistantsListJSONKey(t *testing.T) {
	t.Setenv(naming.EnvBrokerAPIKeyTarget, "ukey")
	up := chimeraBrokerStubForUILogs(t)
	t.Cleanup(up.Close)

	rt := runtimeForUILogs(t, up.URL)
	seedChimeraTestVM(t, rt, "0.1.0", []string{"m"})
	front := httptest.NewServer(NewMux(rt, testLog(), nil, NewUIOptions()))
	t.Cleanup(front.Close)
	client := vmTestLoginClient(t, front.URL, "gw-ui-secret")

	res, err := client.Get(front.URL + "/api/ui/assistants")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("assistants status=%d body=%s", res.StatusCode, b)
	}
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["assistants"]; !ok {
		t.Fatalf("response missing assistants key: %s", raw)
	}
	if _, ok := doc["virtual_models"]; ok {
		t.Fatalf("response must not include virtual_models: %s", raw)
	}
}

func TestUIAssistantRoutingEvaluateIgnoresLegacyVirtualModelIDField(t *testing.T) {
	t.Setenv(naming.EnvBrokerAPIKeyTarget, "ukey")
	up := chimeraBrokerStubForUILogs(t)
	t.Cleanup(up.Close)

	rt := runtimeForUILogs(t, up.URL)
	seedChimeraTestVMWithPolicy(t, rt, "0.1.0", []string{"m"}, "m")
	front := httptest.NewServer(NewMux(rt, testLog(), nil, NewUIOptions()))
	t.Cleanup(front.Close)
	client := vmTestLoginClient(t, front.URL, "gw-ui-secret")

	body := `{"virtual_model_id":"Wrong-9.9.9","fallback_chain":["m"]}`
	res, err := client.Post(front.URL+"/api/ui/assistants/1/routing/evaluate", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("evaluate status=%d body=%s", res.StatusCode, b)
	}
	var out struct {
		InitialModel string `json:"initial_model"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.InitialModel != "m" {
		t.Fatalf("initial_model=%q want m (path assistant, not virtual_model_id body field)", out.InitialModel)
	}
}
