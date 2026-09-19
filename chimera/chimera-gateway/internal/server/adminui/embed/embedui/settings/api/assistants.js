/**
 * Assistant detail fetch + adminStateCache summary sync.
 * Exports: ChimeraSettings.Api.mountAssistantsApi(ctx, bridge)
 */
globalThis.ChimeraSettings = globalThis.ChimeraSettings || {};
globalThis.ChimeraSettings.Api = globalThis.ChimeraSettings.Api || {};

globalThis.ChimeraSettings.Api.mountAssistantsApi = function (ctx, bridge) {
  bridge = bridge || {};

  function markUnauthorized() {
    if (typeof bridge.markUnauthorized === "function") bridge.markUnauthorized();
    else if (typeof ctx.markUiUnauthorized === "function") ctx.markUiUnauthorized();
  }

  function syncAssistantSummaryFromDetail(detail) {
    if (!detail || detail.id == null) return;
    var gw = ctx.adminStateCache && ctx.adminStateCache.gateway;
    if (!gw || !gw.assistants) return;
    var key = String(detail.id);
    for (var i = 0; i < gw.assistants.length; i++) {
      if (gw.assistants[i] && String(gw.assistants[i].id) === key) {
        var row = gw.assistants[i];
        row.enabled = !!detail.enabled;
        row.name = detail.name;
        row.version = detail.version;
        row.description = detail.description;
        row.visibility = detail.visibility;
        row.routing_policy_enabled = !!detail.routing_policy_enabled;
        row.tool_router_enabled = !!detail.tool_router_enabled;
        row.router_models = detail.router_models;
        row.fallback_depth = detail.fallback_chain && detail.fallback_chain.length ? detail.fallback_chain.length : 0;
        break;
      }
    }
  }

  function fetchAssistantDetail(vmId, force) {
    if (ctx.uiUnauthorized) return Promise.resolve(null);
    var key = String(vmId);
    if (!ctx.assistantDetails) ctx.assistantDetails = {};
    if (!ctx.assistantUi) ctx.assistantUi = {};
    var ui = ctx.assistantUi[key];
    if (!ui) {
      ui = ctx.assistantUi[key] = { panelOpen: false, hydrated: false };
    }
    if (!force && ctx.assistantDetails[key]) {
      return Promise.resolve(ctx.assistantDetails[key]);
    }
    ui.detailLoading = true;
    return fetch("/api/ui/assistants/" + encodeURIComponent(key), { credentials: "same-origin" })
      .then(function (r) {
        if (r.status === 401) {
          markUnauthorized();
          return null;
        }
        if (!r.ok) throw new Error("HTTP " + r.status);
        return r.json();
      })
      .then(function (j) {
        ui.detailLoading = false;
        if (!j) return null;
        ctx.assistantDetails[key] = j;
        syncAssistantSummaryFromDetail(j);
        return j;
      })
      .catch(function (e) {
        ui.detailLoading = false;
        throw e;
      });
  }

  ctx.fetchAssistantDetail = fetchAssistantDetail;

  return {
    fetchAssistantDetail: fetchAssistantDetail,
    syncAssistantSummaryFromDetail: syncAssistantSummaryFromDetail
  };
};
