package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/lynn/porcelain/internal/naming"
	"github.com/lynn/porcelain/internal/operatorapi"
)

func TestUIAssistants_CRUDViaAssistantsRoute(t *testing.T) {
	t.Setenv(naming.EnvBrokerAPIKeyTarget, "ukey")
	broker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(broker.Close)

	dir := t.TempDir()
	gwPath := filepath.Join(dir, naming.GatewayConfigFileTarget)
	writeGateway(t, gwPath, broker.URL, []string{"groq/x"}, "")
	tokPath := filepath.Join(dir, naming.APIKeysFileTarget)
	writeTokens(t, tokPath, "gw-assistants-crud", "tenant-a")
	routePath := filepath.Join(dir, naming.RoutingPolicyFileTarget)
	if err := os.WriteFile(routePath, []byte("rules: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rt := mustRuntime(t, gwPath)
	seedChimeraTestVM(t, rt, "0.1.0", []string{"groq/x"})
	front := httptest.NewServer(NewMux(rt, testLog(), nil, NewUIOptions()))
	t.Cleanup(front.Close)
	client := uiLoginClient(t, front.URL, "gw-assistants-crud")

	listRes, err := client.Get(front.URL + "/api/ui/assistants")
	if err != nil {
		t.Fatal(err)
	}
	defer listRes.Body.Close()
	if listRes.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(listRes.Body)
		t.Fatalf("list status=%d body=%s", listRes.StatusCode, b)
	}
	var list operatorapi.AssistantListResponse
	if err := json.NewDecoder(listRes.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list.Assistants) != 1 || list.Assistants[0].ModelID != "Chimera-0.1.0" {
		t.Fatalf("seeded list: %+v", list.Assistants)
	}

	createRes, err := client.Post(front.URL+"/api/ui/assistants", "application/json",
		strings.NewReader(`{"name":"Demo","version":"2.0.0","description":"test assistant"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer createRes.Body.Close()
	if createRes.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(createRes.Body)
		t.Fatalf("create status=%d body=%s", createRes.StatusCode, b)
	}
	var created operatorapi.AssistantDetail
	if err := json.NewDecoder(createRes.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ModelID != "Demo-2.0.0" || created.ID < 2 {
		t.Fatalf("created: %+v", created)
	}

	getRes, err := client.Get(front.URL + "/api/ui/assistants/" + strconv.FormatInt(created.ID, 10))
	if err != nil {
		t.Fatal(err)
	}
	defer getRes.Body.Close()
	if getRes.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(getRes.Body)
		t.Fatalf("get status=%d body=%s", getRes.StatusCode, b)
	}
	var got operatorapi.AssistantDetail
	if err := json.NewDecoder(getRes.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "Demo" {
		t.Fatalf("get: %+v", got)
	}

	putReq, err := http.NewRequest(http.MethodPut, front.URL+"/api/ui/assistants/"+strconv.FormatInt(created.ID, 10),
		strings.NewReader(`{"name":"Demo Renamed"}`))
	if err != nil {
		t.Fatal(err)
	}
	putReq.Header.Set("Content-Type", "application/json")
	putRes, err := client.Do(putReq)
	if err != nil {
		t.Fatal(err)
	}
	defer putRes.Body.Close()
	if putRes.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(putRes.Body)
		t.Fatalf("update status=%d body=%s", putRes.StatusCode, b)
	}
	var updated operatorapi.AssistantDetail
	if err := json.NewDecoder(putRes.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Demo Renamed" {
		t.Fatalf("updated: %+v", updated)
	}

	fbReq, err := http.NewRequest(http.MethodPut, front.URL+"/api/ui/assistants/"+strconv.FormatInt(created.ID, 10)+"/fallback",
		strings.NewReader(`{"fallback_chain":["groq/a","groq/b"]}`))
	if err != nil {
		t.Fatal(err)
	}
	fbReq.Header.Set("Content-Type", "application/json")
	fbRes, err := client.Do(fbReq)
	if err != nil {
		t.Fatal(err)
	}
	defer fbRes.Body.Close()
	if fbRes.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(fbRes.Body)
		t.Fatalf("fallback status=%d body=%s", fbRes.StatusCode, b)
	}

	delReq, err := http.NewRequest(http.MethodDelete, front.URL+"/api/ui/assistants/"+strconv.FormatInt(created.ID, 10), nil)
	if err != nil {
		t.Fatal(err)
	}
	delRes, err := client.Do(delReq)
	if err != nil {
		t.Fatal(err)
	}
	delRes.Body.Close()
	if delRes.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status=%d", delRes.StatusCode)
	}

	listRes2, err := client.Get(front.URL + "/api/ui/assistants")
	if err != nil {
		t.Fatal(err)
	}
	defer listRes2.Body.Close()
	var list2 operatorapi.AssistantListResponse
	if err := json.NewDecoder(listRes2.Body).Decode(&list2); err != nil {
		t.Fatal(err)
	}
	if len(list2.Assistants) != 1 {
		t.Fatalf("after delete list: %+v", list2.Assistants)
	}
}

func TestUIAssistants_VirtualModelsRouteReturnsNotFound(t *testing.T) {
	t.Setenv(naming.EnvBrokerAPIKeyTarget, "ukey")
	broker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(broker.Close)

	dir := t.TempDir()
	gwPath := filepath.Join(dir, naming.GatewayConfigFileTarget)
	writeGateway(t, gwPath, broker.URL, []string{"groq/x"}, "")
	tokPath := filepath.Join(dir, naming.APIKeysFileTarget)
	writeTokens(t, tokPath, "gw-assistants-404", "tenant-a")
	routePath := filepath.Join(dir, naming.RoutingPolicyFileTarget)
	if err := os.WriteFile(routePath, []byte("rules: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rt := mustRuntime(t, gwPath)
	front := httptest.NewServer(NewMux(rt, testLog(), nil, NewUIOptions()))
	t.Cleanup(front.Close)
	client := uiLoginClient(t, front.URL, "gw-assistants-404")

	legacyPaths := []string{
		"/api/ui/virtual-models",
		"/api/ui/virtual-models/1",
		"/api/ui/virtual-models/1/fallback",
		"/api/ui/virtual-models/1/routing-policy",
	}
	for _, path := range legacyPaths {
		res, err := client.Get(front.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("%s status=%d want 404", path, res.StatusCode)
		}
	}

	postRes, err := client.Post(front.URL+"/api/ui/virtual-models", "application/json",
		strings.NewReader(`{"name":"X","version":"1.0.0"}`))
	if err != nil {
		t.Fatal(err)
	}
	postRes.Body.Close()
	if postRes.StatusCode != http.StatusNotFound {
		t.Fatalf("POST virtual-models status=%d want 404", postRes.StatusCode)
	}
}
