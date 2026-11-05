package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
	uimcp "github.com/lynn/porcelain/chimera/chimera-gateway/internal/server/adminui/api/mcp"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/server/adminui/handler"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/server/adminui/session"
	gruntime "github.com/lynn/porcelain/chimera/chimera-gateway/internal/server/runtime"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/testsupport"
	"github.com/lynn/porcelain/chimera/internal/config"
	"github.com/lynn/porcelain/internal/operatorapi"
)

func testEnv(t *testing.T) (*http.ServeMux, *handler.Handler, *operatorstore.Store, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := operatorstore.Open(filepath.Join(dir, "op.sqlite"), testsupport.GatewayOperatorMigrationsDir(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ui := session.NewUIOptions()
	rt := &gruntime.Runtime{}
	rt.SetOperatorStoreForTest(store)
	rt.SetResolvedForTest(&config.Resolved{
		MCP: config.MCP{
			CatalogMaxTools: 32,
			Servers:         []config.MCPServer{{ID: "yaml_seed", Transport: "stdio", Command: "true"}},
		},
	})
	h := handler.New(rt, nil, ui)
	mux := http.NewServeMux()
	uimcp.Register(mux, h)
	const principal = "tenant-a"
	sid, err := ui.Sessions.Issue(principal)
	if err != nil {
		t.Fatal(err)
	}
	return mux, h, store, sid
}

func authedRequest(method, path, sid, cookieName string, body []byte) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	r.AddCookie(&http.Cookie{Name: cookieName, Value: sid})
	return r
}

func TestMCPAPI_serversYAMLThenSQLite(t *testing.T) {
	mux, h, store, sid := testEnv(t)
	ctx := context.Background()

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, authedRequest(http.MethodGet, "/api/ui/mcp/servers", sid, h.CookieName(), nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	var list operatorapi.MCPServersListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list.Source != "yaml" || len(list.Servers) != 1 || list.Servers[0].ServerID != "yaml_seed" {
		t.Fatalf("yaml list=%+v", list)
	}

	putBody, _ := json.Marshal(operatorapi.MCPServerUpsertRequest{
		Transport: "stdio",
		Command:   "go",
		Args:      []string{"run", "./fake"},
	})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, authedRequest(http.MethodPut, "/api/ui/mcp/servers/fake_echo", sid, h.CookieName(), putBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("put status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, authedRequest(http.MethodGet, "/api/ui/mcp/servers", sid, h.CookieName(), nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list.Source != "sqlite" || len(list.Servers) != 1 || list.Servers[0].ServerID != "fake_echo" {
		t.Fatalf("sqlite list=%+v", list)
	}

	vm, err := store.CreateAssistant(ctx, operatorstore.CreateAssistantInput{
		ModelID: "Dev-1.0", Name: "Dev", Version: "1.0", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	saveBody, _ := json.Marshal(operatorapi.MCPAssistantBindingsSaveRequest{
		Bindings: []operatorapi.MCPAssistantServerBinding{{
			ServerID: "fake_echo",
			Enabled:  true,
			Tools: []operatorapi.MCPAssistantToolPermission{
				{ToolName: "echo", Enabled: true},
				{ToolName: "shell_run", Enabled: false},
			},
		}},
	})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, authedRequest(http.MethodPut, "/api/ui/assistants/"+itoa(vm.ID)+"/mcp", sid, h.CookieName(), saveBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("bindings put status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, authedRequest(http.MethodGet, "/api/ui/assistants/"+itoa(vm.ID)+"/mcp", sid, h.CookieName(), nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("bindings get status=%d", rec.Code)
	}
	var got operatorapi.MCPAssistantBindingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Bindings) != 1 || !got.Bindings[0].Enabled {
		t.Fatalf("bindings=%+v", got.Bindings)
	}
	toolState := map[string]bool{}
	for _, tp := range got.Bindings[0].Tools {
		toolState[tp.ToolName] = tp.Enabled
	}
	if toolState["echo"] != true || toolState["shell_run"] != false {
		t.Fatalf("tools=%v", toolState)
	}
}

func TestMCPAPI_assistantBindingsNotFound(t *testing.T) {
	mux, h, _, sid := testEnv(t)
	saveBody, _ := json.Marshal(operatorapi.MCPAssistantBindingsSaveRequest{Bindings: []operatorapi.MCPAssistantServerBinding{}})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, authedRequest(http.MethodPut, "/api/ui/assistants/99999/mcp", sid, h.CookieName(), saveBody))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMCPAPI_invalidAssistantID(t *testing.T) {
	mux, h, _, sid := testEnv(t)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, authedRequest(http.MethodGet, "/api/ui/assistants/abc/mcp", sid, h.CookieName(), nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestMCPAPI_serverToolsWithoutManager(t *testing.T) {
	mux, h, _, sid := testEnv(t)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, authedRequest(http.MethodGet, "/api/ui/mcp/servers/fake_echo/tools", sid, h.CookieName(), nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMCPAPI_unauthorized(t *testing.T) {
	mux, _, _, _ := testEnv(t)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/ui/mcp/servers", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rec.Code)
	}
}

func itoa(id int64) string {
	if id <= 0 {
		return "0"
	}
	var b [32]byte
	i := len(b)
	for id > 0 {
		i--
		b[i] = byte('0' + id%10)
		id /= 10
	}
	return string(b[i:])
}
