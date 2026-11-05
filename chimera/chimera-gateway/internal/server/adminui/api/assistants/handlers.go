package assistants

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/routing"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/routinggen"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/server/adminui/handler"
	"github.com/lynn/porcelain/chimera/internal/brokerclient"
	"github.com/lynn/porcelain/chimera/internal/config"
	"github.com/lynn/porcelain/chimera/internal/modelcatalog"
	"github.com/lynn/porcelain/internal/operatorapi"
)

const operatorTenantID = ""

func vmSummary(vm operatorstore.Assistant) operatorapi.AssistantSummary {
	return operatorapi.AssistantSummary{
		ID:                   vm.ID,
		ModelID:              vm.ModelID,
		Name:                 vm.Name,
		Version:              vm.Version,
		Description:          vm.Description,
		Enabled:              vm.Enabled,
		Visibility:           vm.Visibility,
		FallbackDepth:        len(vm.FallbackChain),
		RoutingPolicyEnabled: vm.RoutingPolicyEnabled,
		ToolRouterEnabled:    vm.ToolRouterEnabled,
		RouterModels:         vm.RouterModels,
	}
}

func vmDetail(vm operatorstore.Assistant) operatorapi.AssistantDetail {
	return operatorapi.AssistantDetail{
		AssistantSummary:     vmSummary(vm),
		RoutingPolicyYAML:    vm.RoutingPolicyYAML,
		FallbackChain:        vm.FallbackChain,
		ToolRouterConfidence: vm.ToolRouterConfidence,
		HarnessModules:       harnessModulesAPI(vm.HarnessModules),
		CreatedByPrincipalID: vm.CreatedByPrincipalID,
		CreatedAt:            vm.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:            vm.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func harnessModulesAPI(mods []operatorstore.HarnessModule) []operatorapi.AssistantHarnessModule {
	if len(mods) == 0 {
		mods = operatorstore.DefaultHarnessModules(false)
	}
	out := make([]operatorapi.AssistantHarnessModule, 0, len(mods))
	for _, m := range mods {
		item := operatorapi.AssistantHarnessModule{
			ModuleID:     m.ModuleID,
			Enabled:      m.Enabled,
			ConfigJSON:   json.RawMessage(operatorstoreNormalizeConfig(m.ConfigJSON)),
			Configurable: true,
		}
		out = append(out, item)
	}
	return out
}

func operatorstoreNormalizeConfig(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || !json.Valid([]byte(s)) {
		return "{}"
	}
	return s
}

func vmDetailForSession(h *handler.Handler, r *http.Request, vm operatorstore.Assistant) operatorapi.AssistantDetail {
	out := vmDetail(vm)
	if h == nil || h.RT == nil {
		return out
	}
	reg := h.RT.ProviderModels()
	if reg == nil {
		return out
	}
	snap := reg.Snapshot(h.SessionTenantID(r))
	var unavail []string
	for _, mid := range out.FallbackChain {
		mid = strings.TrimSpace(mid)
		if mid == "" || snap.IsAvailable(mid) {
			continue
		}
		unavail = append(unavail, mid)
	}
	if len(unavail) > 0 {
		out.FallbackUnavailable = unavail
	}
	return out
}

func filterModelsBySessionTenant(h *handler.Handler, r *http.Request, ids []string) []string {
	if h == nil || h.RT == nil {
		return ids
	}
	reg := h.RT.ProviderModels()
	if reg == nil {
		return ids
	}
	return reg.FilterAvailable(h.SessionTenantID(r), ids)
}

func operatorStore(h *handler.Handler) *operatorstore.Store {
	if h == nil || h.RT == nil {
		return nil
	}
	return h.RT.OperatorStore()
}

func reloadRegistry(h *handler.Handler, ctx context.Context) {
	if h == nil || h.RT == nil {
		return
	}
	_ = h.RT.ReloadAssistants(ctx)
}

func parseVMID(r *http.Request) (int64, bool) {
	idStr := strings.TrimSpace(r.PathValue("id"))
	if idStr == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func handleListGET(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	st := operatorStore(h)
	if st == nil {
		http.Error(w, "operator store unavailable", http.StatusServiceUnavailable)
		return
	}
	vms, err := st.ListAssistants(r.Context(), operatorTenantID, operatorTenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]operatorapi.AssistantSummary, 0, len(vms))
	for _, vm := range vms {
		out = append(out, vmSummary(vm))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(operatorapi.AssistantListResponse{Assistants: out})
}

func handleCreatePOST(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	st := operatorStore(h)
	if st == nil {
		http.Error(w, "operator store unavailable", http.StatusServiceUnavailable)
		return
	}
	var body operatorapi.AssistantCreateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	ragOn := false
	if h != nil && h.RT != nil {
		if res, _ := h.RT.Snapshot(); res != nil && res.RAG.Enabled {
			ragOn = true
		}
	}
	vm, err := st.CreateAssistant(r.Context(), operatorstore.CreateAssistantInput{
		ModelID:                 body.ModelID,
		Name:                    body.Name,
		Version:                 body.Version,
		Description:             body.Description,
		Visibility:              body.Visibility,
		TenantID:                operatorTenantID,
		Enabled:                 true,
		DefaultRetrievalEnabled: ragOn,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reloadRegistry(h, r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(vmDetail(*vm))
}

func handleGetGET(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	st := operatorStore(h)
	if st == nil {
		http.Error(w, "operator store unavailable", http.StatusServiceUnavailable)
		return
	}
	id, ok := parseVMID(r)
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	vm, err := st.GetAssistantByID(r.Context(), operatorTenantID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if vm == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(vmDetailForSession(h, r, *vm))
}

func handleUpdatePUT(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	st := operatorStore(h)
	if st == nil {
		http.Error(w, "operator store unavailable", http.StatusServiceUnavailable)
		return
	}
	id, ok := parseVMID(r)
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body operatorapi.AssistantUpdateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	name, version, desc := "", "", ""
	if body.Name != nil {
		name = *body.Name
	}
	if body.Version != nil {
		version = *body.Version
	}
	if body.Description != nil {
		desc = *body.Description
	}
	if err := st.UpdateAssistantMetadata(r.Context(), operatorTenantID, id, name, version, desc, body.Enabled, body.Visibility); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reloadRegistry(h, r.Context())
	handleGetGET(h, w, r)
}

func handleDeleteDELETE(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	st := operatorStore(h)
	if st == nil {
		http.Error(w, "operator store unavailable", http.StatusServiceUnavailable)
		return
	}
	id, ok := parseVMID(r)
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := st.DeleteAssistant(r.Context(), operatorTenantID, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	reloadRegistry(h, r.Context())
	w.WriteHeader(http.StatusNoContent)
}

func handleFallbackPUT(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	st := operatorStore(h)
	if st == nil {
		http.Error(w, "operator store unavailable", http.StatusServiceUnavailable)
		return
	}
	id, ok := parseVMID(r)
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body operatorapi.AssistantFallbackSaveRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := st.SetAssistantFallback(r.Context(), operatorTenantID, id, body.FallbackChain); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reloadRegistry(h, r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "fallback_chain": body.FallbackChain})
}

func handleRoutingPolicyPUT(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	st := operatorStore(h)
	if st == nil {
		http.Error(w, "operator store unavailable", http.StatusServiceUnavailable)
		return
	}
	id, ok := parseVMID(r)
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body operatorapi.AssistantRoutingPolicySaveRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.Enabled && len(strings.TrimSpace(body.RoutingPolicyYAML)) > 0 {
		if err := routing.ValidatePolicyYAML([]byte(body.RoutingPolicyYAML)); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if err := st.SetAssistantRoutingPolicy(r.Context(), operatorTenantID, id, body.Enabled, body.RoutingPolicyYAML); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reloadRegistry(h, r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func handleToolRouterPUT(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	st := operatorStore(h)
	if st == nil {
		http.Error(w, "operator store unavailable", http.StatusServiceUnavailable)
		return
	}
	id, ok := parseVMID(r)
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body operatorapi.AssistantToolRouterSaveRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	th := body.ConfidenceThreshold
	if th <= 0 {
		th = 0.5
	}
	if err := st.SetAssistantToolRouter(r.Context(), operatorTenantID, id, body.Enabled, body.RouterModels, th); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reloadRegistry(h, r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func handleHarnessGET(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	st := operatorStore(h)
	if st == nil {
		http.Error(w, "operator store unavailable", http.StatusServiceUnavailable)
		return
	}
	id, ok := parseVMID(r)
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	vm, err := st.GetAssistantByID(r.Context(), operatorTenantID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if vm == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(operatorapi.AssistantHarnessResponse{
		Modules: harnessModulesAPI(vm.HarnessModules),
	})
}

func handleHarnessPUT(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	st := operatorStore(h)
	if st == nil {
		http.Error(w, "operator store unavailable", http.StatusServiceUnavailable)
		return
	}
	id, ok := parseVMID(r)
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body operatorapi.AssistantHarnessSaveRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	mods := make([]operatorstore.HarnessModule, 0, len(body.Modules))
	for _, m := range body.Modules {
		cfg := "{}"
		if len(m.ConfigJSON) > 0 && json.Valid(m.ConfigJSON) {
			cfg = string(m.ConfigJSON)
		}
		mods = append(mods, operatorstore.HarnessModule{
			ModuleID:   m.ModuleID,
			Enabled:    m.Enabled,
			ConfigJSON: cfg,
		})
	}
	if err := st.SetAssistantHarness(r.Context(), operatorTenantID, id, mods); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reloadRegistry(h, r.Context())
	vm, err := st.GetAssistantByID(r.Context(), operatorTenantID, id)
	if err != nil || vm == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(operatorapi.AssistantHarnessResponse{
		Modules: harnessModulesAPI(vm.HarnessModules),
	})
}

func filterByProviderPrefix(ids []string, prefix string) []string {
	prefix = strings.TrimSpace(strings.ToLower(prefix))
	if prefix == "" {
		return ids
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if strings.HasPrefix(strings.ToLower(id), prefix) {
			out = append(out, id)
		}
	}
	return out
}

func handleGeneratePOST(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	st := operatorStore(h)
	if st == nil {
		http.Error(w, "operator store unavailable", http.StatusServiceUnavailable)
		return
	}
	id, ok := parseVMID(r)
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	vm, err := st.GetAssistantByID(r.Context(), operatorTenantID, id)
	if err != nil || vm == nil {
		http.NotFound(w, r)
		return
	}
	var body operatorapi.AssistantGenerateRequest
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body)

	h.RT.Sync()
	res, _ := h.RT.Snapshot()
	apiKey := h.RT.UpstreamAPIKey()
	if apiKey == "" {
		http.Error(w, "missing chimera-broker API key", http.StatusServiceUnavailable)
		return
	}
	stCode, catBody, ok := brokerclient.FetchOpenAIModels(r.Context(), res.UpstreamBaseURL, apiKey, configHealthTimeout(res), h.Log)
	if !ok {
		http.Error(w, "failed to list models from chimera-broker", http.StatusBadGateway)
		return
	}
	ids, err := routinggen.ExtractCatalogModelIDs(catBody, vm.ModelID)
	if err != nil {
		http.Error(w, "invalid chimera-broker models JSON", http.StatusBadGateway)
		return
	}
	_ = stCode
	pool := filterByProviderPrefix(ids, body.ProviderPrefix)
	pool = filterModelsBySessionTenant(h, r, pool)
	if len(pool) == 0 {
		http.Error(w, "no models left after operator availability filter", http.StatusBadRequest)
		return
	}
	pool, capabilityDecisions := modelcatalog.FilterGeneralCandidates(pool)
	if len(pool) == 0 {
		http.Error(w, "no general text/chat models left after capability filter", http.StatusBadRequest)
		return
	}
	preference := body.PreferredModels
	if len(preference) == 0 {
		preference = vm.FallbackChain
	}
	localFirst := true
	if body.LocalFirst != nil {
		localFirst = *body.LocalFirst
	}
	chain := modelcatalog.OrderByPreference(pool, preference, localFirst)
	routeYAML, err := routinggen.BuildRoutingPolicyYAML(chain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := routing.ValidatePolicyYAML(routeYAML); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.Save {
		if err := st.SetAssistantFallback(r.Context(), operatorTenantID, id, chain); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := st.SetAssistantRoutingPolicy(r.Context(), operatorTenantID, id, true, string(routeYAML)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		reloadRegistry(h, r.Context())
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(operatorapi.RoutingGenerateResponse{
		OK:                  true,
		Saved:               body.Save,
		FallbackChain:       chain,
		RoutingPolicyYAML:   string(routeYAML),
		ModelsBrokerCatalog: len(ids),
		ModelsUsed:          len(chain),
		ModelDecisions:      operatorModelDecisions(capabilityDecisions),
	})
}

func operatorModelDecisions(in []modelcatalog.Decision) []operatorapi.ModelDecision {
	out := make([]operatorapi.ModelDecision, 0, len(in))
	for _, decision := range in {
		out = append(out, operatorapi.ModelDecision{
			ModelID: decision.ModelID,
			Status:  string(decision.Status),
			Role:    decision.Role,
			Reason:  decision.Reason,
		})
	}
	return out
}

func handleEvaluatePOST(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	st := operatorStore(h)
	if st == nil {
		http.Error(w, "operator store unavailable", http.StatusServiceUnavailable)
		return
	}
	id, ok := parseVMID(r)
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	vm, err := st.GetAssistantByID(r.Context(), operatorTenantID, id)
	if err != nil || vm == nil {
		http.NotFound(w, r)
		return
	}
	var body operatorapi.RoutingEvaluateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	policyYAML := []byte(body.RoutingPolicyYAML)
	if len(policyYAML) == 0 {
		policyYAML = []byte(vm.RoutingPolicyYAML)
	}
	chain := body.FallbackChain
	if len(chain) == 0 {
		chain = vm.FallbackChain
	}
	vmID := strings.TrimSpace(body.AssistantID)
	if vmID == "" {
		vmID = vm.ModelID
	}
	reqBody := map[string]json.RawMessage{"model": json.RawMessage(`"` + vmID + `"`)}
	if len(body.Messages) > 0 {
		reqBody["messages"] = body.Messages
	}
	initial, via, err := routing.EvaluatePick(policyYAML, reqBody, chain, vmID, h.Log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	start := routing.StartingFallbackIndex(initial, chain)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(operatorapi.RoutingEvaluateResponse{
		OK:                  true,
		InitialModel:        initial,
		Via:                 string(via),
		FallbackStartIndex:  start,
		FallbackFromInitial: chain[start:],
	})
}

func handleHarnessEvaluatePOST(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	st := operatorStore(h)
	if st == nil || h == nil || h.RT == nil {
		http.Error(w, "operator store unavailable", http.StatusServiceUnavailable)
		return
	}
	id, ok := parseVMID(r)
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	vm, err := st.GetAssistantByID(r.Context(), operatorTenantID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if vm == nil {
		http.NotFound(w, r)
		return
	}
	var input operatorapi.AssistantHarnessEvaluateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.Message) == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}
	reg := h.RT.Assistants()
	if reg == nil {
		http.Error(w, "assistant registry unavailable", http.StatusServiceUnavailable)
		return
	}
	resolved, err := reg.Resolve(vm.ModelID, operatorTenantID)
	if err != nil {
		http.Error(w, "assistant unavailable for evaluation", http.StatusBadRequest)
		return
	}
	res, _ := h.RT.Snapshot()
	if res == nil {
		http.Error(w, "gateway config unavailable", http.StatusServiceUnavailable)
		return
	}
	messages, _ := json.Marshal([]map[string]string{{"role": "user", "content": input.Message}})
	tc := &harness.TurnContext{
		Resolved: res,
		Stack: harness.VMStack{
			VM: resolved, Fallback: append([]string(nil), resolved.FallbackChain...),
			ToolEnabled: resolved.ToolRouterEnabled, RouterModels: resolved.RouterModels,
			ToolThresh: resolved.ToolRouterConfidence,
		},
		RouteLog: h.Log, TenantID: operatorTenantID, ProjectID: strings.TrimSpace(input.Project),
		FlavorID: strings.TrimSpace(input.Flavor), OperatorStore: st,
	}
	if err := harness.PrePrimaryRunner().Run(r.Context(), tc, harness.Body{
		"messages": json.RawMessage(messages),
	}); err != nil {
		http.Error(w, "harness evaluation failed", http.StatusBadRequest)
		return
	}
	redacted, err := harness.RedactedJSON(tc.Envelope)
	if err != nil {
		http.Error(w, "harness evaluation encoding failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(operatorapi.AssistantHarnessEvaluateResponse{OK: true, Envelope: redacted})
}

func configHealthTimeout(res *config.Resolved) time.Duration {
	if res == nil || res.HealthTimeoutMs <= 0 {
		return 5 * time.Second
	}
	return time.Duration(res.HealthTimeoutMs) * time.Millisecond
}
