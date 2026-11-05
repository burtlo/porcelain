/**
 * Render production card HTML into /ui/settings/gallery fixture mount points.
 * Requires the same shared + settings card modules as the live summarized feed.
 */
(function () {
  function mountGalleryCardCtx() {
    var escapeHtml = ChimeraSettings.escapeHtml;
    var ctx = {
      escapeHtml: escapeHtml,
      strHash: ChimeraSettings.strHash,
      getFlat: function (p) {
        return (p && p.rawFlat) || {};
      },
      entryCache: [],
      entryInstant: function () {
        return null;
      },
      formatInt: function (n) {
        return String(n != null ? n : 0);
      },
      RECENT_CARD_STATUS_N: 8,
      sumEvlogPanelHtml: function (o) {
        return (
          '<div class="sum-evlog sum-evlog--stub" data-gallery-evlog-stub>' +
          escapeHtml(o && o.title ? o.title : "Scoped log") +
          "</div>"
        );
      },
      sumEvlogBuildTbodyFromServiceEntries: function () {
        return "";
      },
      sumEvlogBuildTbodyFromConvEvents: function () {
        return "";
      },
      sumEvlogCountWarnFailFromEntries: function () {
        return { warn: 0, fail: 0 };
      },
      scopedEvlogTitle: function (t) {
        return String(t || "Scoped log");
      },
      primaryLogMessage: function (parsed) {
        var flat = (parsed && parsed.rawFlat) || {};
        return String(flat.msg || flat.message || "Harness event");
      },
      formatMergedConversationSubtitle: function () {
        return "";
      },
      serviceSummaryStatusPillHtml: function () {
        return "";
      },
      operatorCardChevronHtml: function () {
        return '<span class="sum-card-chevron" aria-hidden="true">›</span>';
      },
      inferServiceBadge: function () {
        return { lab: "chimera-gateway" };
      },
      adminVisibleProviderIds: ["groq", "ollama"],
      adminProviderKeyDraft: { groq: "gsk-gallery-demo" },
      adminOllamaUrlDraft: "http://127.0.0.1:11434",
      adminProviderModelsEditingId: null,
      adminProviderModelsDraft: {},
      adminProviderModelsCache: {},
      adminStateCache: {
        providers: {
          groq: {
            keys: [{ name: "gallery", key_configured: true }],
            ok: true,
            key_configured: true,
            models_configured: true,
            models_available_count: 2,
            models_unavailable_count: 0
          },
          ollama: {
            keys: [],
            ok: true,
            ollama_base_url: "http://127.0.0.1:11434"
          }
        },
        gateway: {
          assistants: [
            {
              id: 42,
              model_id: "Chimera-0.2.0",
              name: "Chimera",
              version: "0.2.0",
              description: "Gallery fixture — retrieval truncate / empty-query skip",
              enabled: true,
              visibility: "public",
              fallback_depth: 2,
              routing_policy_enabled: true,
              tool_router_enabled: false,
              router_models: []
            },
            {
              id: 43,
              model_id: "Research-1.0",
              name: "Research",
              version: "1.0",
              description: "Gallery fixture — retrieval summarize",
              enabled: true,
              visibility: "public",
              fallback_depth: 1,
              routing_policy_enabled: false,
              tool_router_enabled: false,
              router_models: []
            },
            {
              id: 44,
              model_id: "Engineer-1.0",
              name: "Engineer",
              version: "1.0",
              description: "Gallery fixture — LLM-assisted intent",
              enabled: true,
              visibility: "public",
              fallback_depth: 1,
              routing_policy_enabled: false,
              tool_router_enabled: false,
              router_models: []
            },
            {
              id: 45,
              model_id: "Review-Gated-1.0",
              name: "Review Gated",
              version: "1.0",
              description: "Gallery fixture — evaluator gate on evaluator",
              enabled: true,
              visibility: "public",
              fallback_depth: 2,
              routing_policy_enabled: false,
              tool_router_enabled: false,
              router_models: []
            },
            {
              id: 46,
              model_id: "Review-Buffered-1.0",
              name: "Review Buffered",
              version: "1.0",
              description: "Gallery fixture — evaluator buffer until complete",
              enabled: true,
              visibility: "public",
              fallback_depth: 2,
              routing_policy_enabled: false,
              tool_router_enabled: false,
              router_models: []
            },
            {
              id: 47,
              model_id: "Research-Ensemble-1.0",
              name: "Research Ensemble",
              version: "1.0",
              description: "Gallery fixture — multi-draft evaluator",
              enabled: true, visibility: "public", fallback_depth: 3, routing_policy_enabled: false, tool_router_enabled: false, router_models: []
            },
            {
              id: 48,
              model_id: "Support-Escalation-1.0",
              name: "Support Escalation",
              version: "1.0",
              description: "Gallery fixture — human escalation and paste-back",
              enabled: true, visibility: "public", fallback_depth: 2, routing_policy_enabled: false, tool_router_enabled: false, router_models: []
            }
          ]
        }
      },
      chimeraBrokerProviderSnapshot: {
        fetchedClientMs: Date.now(),
        data: { providers: [{ id: "groq", model_ids: ["groq/free", "groq/paid"] }] }
      },
      metricsCache: {
        day_rollups: [{ provider: "groq", model_id: "groq/free", calls: 12, status: 200 }]
      },
      gatewayOverviewCache: {
        assistant_id: "virtual/claude-opus-proxy",
        service_overview: {
          refreshed_at: new Date().toISOString(),
          services: {
            "chimera-broker": { state: "up" },
            "chimera-vectorstore": { state: "up" },
            "chimera-indexer": { worker: "idle" }
          }
        }
      },
      tokenListCache: [{ tenant_id: "tenant-a", label: "Gallery", index: 0 }],
      tokenLabelByTenant: { "tenant-a": "Gallery" },
      assistantDrafts: [],
      assistantUi: {
        "42": { panelOpen: true, hydrated: true, sectionOpen: { identity: true, fallback: true, harness: true } },
        "43": { panelOpen: true, hydrated: true, sectionOpen: { harness: true } }
        ,"44": { panelOpen: true, hydrated: true, sectionOpen: { harness: true } }
        ,"45": { panelOpen: true, hydrated: true, sectionOpen: { harness: true } }
        ,"46": { panelOpen: true, hydrated: true, sectionOpen: { harness: true } }
        ,"47": { panelOpen: true, hydrated: true, sectionOpen: { harness: true } }
        ,"48": { panelOpen: true, hydrated: true, sectionOpen: { harness: true } }
      },
      assistantDetails: {
        "42": {
          fallback_chain: ["groq/free"],
          fallback_unavailable: [],
          routing_policy: "ambiguous_default_model: groq/free\nrules: []\n",
          router_models: ["- groq/free"],
          harness_modules: [
            { module_id: "retrieval", enabled: true, configurable: true, config_json: { top_k: 4, score_floor: 0.68, compress_strategy: "truncate", max_context_chars: 8000, skip_if: ["empty_query"] } },
            { module_id: "intent", enabled: false, configurable: true, config_json: {} },
            { module_id: "evaluator", enabled: false, configurable: true, config_json: {} },
            { module_id: "escalation", enabled: false, configurable: true, config_json: {} },
            {
              module_id: "tool_executor",
              enabled: false,
              configurable: true,
              config_json: {}
            }
          ]
        },
        "43": {
          fallback_chain: ["groq/free"],
          fallback_unavailable: [],
          routing_policy: "ambiguous_default_model: groq/free\nrules: []\n",
          router_models: [],
          harness_modules: [
            { module_id: "retrieval", enabled: true, configurable: true, config_json: { top_k: 10, score_floor: 0.55, compress_strategy: "summarize", max_context_chars: 16000, summarize_model_id: "groq/free" } },
            { module_id: "intent", enabled: true, configurable: true, config_json: {} },
            { module_id: "evaluator", enabled: true, configurable: true, config_json: { mode: "single_pass", model_id: "groq/free", min_confidence: 0.72, hallucination_risk_max: 0.2, stream_policy: "immediate" } },
            { module_id: "escalation", enabled: true, configurable: true, config_json: { max_rounds: 2, on_fail: ["re_retrieve", "fallback_chain", "ensemble", "human"] } },
            {
              module_id: "tool_executor",
              enabled: true,
              configurable: true,
              config_json: {
                max_tool_rounds: 5,
                mcp: { enabled: true, preview_tools: ["fake_echo__echo"] }
              }
            }
          ]
        },
        "44": {
          fallback_chain: ["groq/free"],
          fallback_unavailable: [],
          harness_modules: [
            { module_id: "retrieval", enabled: false, configurable: true, config_json: {} },
            { module_id: "intent", enabled: true, configurable: true, config_json: { mode: "llm", model_id: "groq/free" } },
            { module_id: "evaluator", enabled: false, configurable: true, config_json: {} },
            { module_id: "escalation", enabled: false, configurable: true, config_json: {} },
            { module_id: "tool_executor", enabled: false, configurable: true, config_json: {} }
          ]
        },
        "45": {
          fallback_chain: ["groq/free", "groq/paid"],
          fallback_unavailable: [],
          harness_modules: [
            { module_id: "retrieval", enabled: true, configurable: true, config_json: { top_k: 6, compress_strategy: "truncate" } },
            { module_id: "intent", enabled: false, configurable: true, config_json: {} },
            { module_id: "evaluator", enabled: true, configurable: true, config_json: { mode: "single_pass", model_id: "groq/free", min_confidence: 0.8, stream_policy: "gate_on_evaluator" } },
            { module_id: "escalation", enabled: true, configurable: true, config_json: { max_rounds: 1, on_fail: ["fallback_chain"] } },
            { module_id: "tool_executor", enabled: false, configurable: true, config_json: {} }
          ]
        },
        "46": {
          fallback_chain: ["groq/free", "groq/paid"],
          fallback_unavailable: [],
          harness_modules: [
            { module_id: "retrieval", enabled: true, configurable: true, config_json: { top_k: 6, compress_strategy: "truncate" } },
            { module_id: "intent", enabled: false, configurable: true, config_json: {} },
            { module_id: "evaluator", enabled: true, configurable: true, config_json: { mode: "single_pass", model_id: "groq/free", min_confidence: 0.8, stream_policy: "buffer_until_complete" } },
            { module_id: "escalation", enabled: true, configurable: true, config_json: { max_rounds: 2, on_fail: ["re_retrieve", "fallback_chain"] } },
            { module_id: "tool_executor", enabled: false, configurable: true, config_json: {} }
          ]
        },
        "47": {
          fallback_chain: ["groq/free", "groq/paid", "ollama/local"],
          fallback_unavailable: [],
          harness_modules: [
            { module_id: "retrieval", enabled: true, configurable: true, config_json: {} },
            { module_id: "intent", enabled: false, configurable: true, config_json: {} },
            { module_id: "evaluator", enabled: true, configurable: true, config_json: { mode: "multi_draft", model_id: "groq/free", draft_count: 3, synthesize_model_id: "groq/paid", stream_policy: "buffer_until_complete", min_confidence: 0.8 } },
            { module_id: "escalation", enabled: true, configurable: true, config_json: { max_rounds: 2, on_fail: ["re_retrieve", "fallback_chain"] } },
            { module_id: "tool_executor", enabled: false, configurable: true, config_json: {} }
          ]
        },
        "48": {
          fallback_chain: ["groq/free", "groq/paid"],
          fallback_unavailable: [],
          harness_modules: [
            { module_id: "retrieval", enabled: false, configurable: true, config_json: {} },
            { module_id: "intent", enabled: false, configurable: true, config_json: {} },
            { module_id: "evaluator", enabled: true, configurable: true, config_json: { mode: "single_pass", model_id: "groq/free", stream_policy: "buffer_until_complete", min_confidence: 0.9 } },
            { module_id: "escalation", enabled: true, configurable: true, config_json: { max_rounds: 2, on_fail: ["re_retrieve", "fallback_chain", "human"], human_surfaces: [{ name: "Support desk", url: "https://support.example.test/escalate" }], privacy_disclosure: "Remove customer secrets before sharing.", paste_back_delimiter: "<<<CHIMERA_HUMAN_ANSWER>>>" } },
            { module_id: "tool_executor", enabled: false, configurable: true, config_json: {} }
          ]
        }
      },
      workspaceDrafts: [
        {
          id: 1,
          projectId: "acme-docs",
          flavorId: "main",
          paths: [{ path: "C:\\\\data\\\\gallery-draft" }]
        }
      ],
      workspaceManagedEditId: null,
      workspaceManagedStaging: null,
      lastIndexerOperatorWorkspacesNested: [
        {
          id: "3",
          project_id: "acme-docs",
          flavor_id: "main",
          sensitivity: "public",
          allow_cloud: true,
          allow_cloud_summary_only: false,
          file_action_policy: "read",
          paths: [{ id: 10, path: "C:\\\\data\\\\managed" }]
        },
        {
          id: "4",
          project_id: "private-ledger",
          flavor_id: "restricted",
          sensitivity: "private",
          allow_cloud: false,
          allow_cloud_summary_only: false,
          file_action_policy: "none",
          paths: [{ id: 11, path: "C:\\\\data\\\\private-ledger" }]
        },
        {
          id: "5",
          project_id: "automation",
          flavor_id: "main",
          sensitivity: "internal",
          allow_cloud: true,
          allow_cloud_summary_only: true,
          file_action_policy: "read_write",
          paths: [{ id: 12, path: "C:\\\\data\\\\automation" }]
        }
      ],
      operatorWsFullLogCtx: {},
      workspaceDesktopFeaturesAvailable: function () {
        return false;
      },
      wrapDesktopOnlyLockedControl: function (html) {
        return html;
      },
      resolveLogsOperatorUserLabel: function () {
        return "Gallery operator";
      },
      ragEmbeddingCache: {
        model: "ollama/nomic-embed-text:latest",
        dim: 768,
        status: "ok",
        candidates: [
          { id: "ollama/nomic-embed-text:latest", embedding_likely: true, known_dim: 768 },
          { id: "groq/llama3", embedding_likely: false }
        ]
      },
      ragEmbeddingDraftModel: "groq/llama3"
    };
    ChimeraSettings.Render.mountSumEvlog(ctx);
    ChimeraSettings.Render.Cards.mountAll(ctx);
    if (typeof ChimeraSettings.Render.Cards.mountSummarizedFeedCards === "function") {
      ChimeraSettings.Render.Cards.mountSummarizedFeedCards(ctx);
    }
    return ctx;
  }

  function setHtml(id, html) {
    var el = document.getElementById(id);
    if (!el) return;
    el.innerHTML = html || "";
  }

  function renderFixtures() {
    if (!globalThis.ChimeraSettings || !ChimeraSettings.Render || !ChimeraSettings.Render.Cards) return;
    var ctx = mountGalleryCardCtx();
    setHtml("gallery-fixture-overview", ctx.buildGatewayOverviewCardHtml());
    setHtml(
      "gallery-fixture-provider-groq",
      ctx.buildAdminProviderCardHtml("groq", "Groq", "Gq", "LPU inference — gallery fixture")
    );
    setHtml(
      "gallery-fixture-provider-ollama",
      ctx.buildAdminProviderCardHtml("ollama", "Ollama", "Ol", "Local chat + embeddings")
    );
    if (typeof ctx.ragEmbeddingPanelHtml === "function") {
      setHtml("gallery-fixture-rag-embedding", ctx.ragEmbeddingPanelHtml());
      ctx.ragEmbeddingDraftModel = null;
      ctx.ragEmbeddingPostSaveBanner = true;
      setHtml("gallery-fixture-rag-embedding-saved", ctx.ragEmbeddingPanelHtml());
      ctx.ragEmbeddingPostSaveBanner = false;
      ctx.ragEmbeddingDraftModel = "groq/llama3";
    }
    if (ctx.workspaceDrafts && ctx.workspaceDrafts.length && typeof ctx.buildWorkspaceDraftCardHtml === "function") {
      setHtml("gallery-fixture-workspace-draft", ctx.buildWorkspaceDraftCardHtml(ctx.workspaceDrafts[0]));
    }
    var vmList =
      ctx.adminStateCache &&
      ctx.adminStateCache.gateway &&
      Array.isArray(ctx.adminStateCache.gateway.assistants)
        ? ctx.adminStateCache.gateway.assistants
        : [];
    if (vmList.length && typeof ctx.buildAssistantCardHtml === "function") {
      setHtml("gallery-fixture-assistant", ctx.buildAssistantCardHtml(vmList[0]));
      if (vmList[1]) {
        setHtml("gallery-fixture-assistant-harness-on", ctx.buildAssistantCardHtml(vmList[1]));
      }
      if (vmList[2]) {
        setHtml("gallery-fixture-assistant-intent-llm", ctx.buildAssistantCardHtml(vmList[2]));
      }
      if (vmList[3]) setHtml("gallery-fixture-assistant-evaluator-gated", ctx.buildAssistantCardHtml(vmList[3]));
      if (vmList[4]) setHtml("gallery-fixture-assistant-evaluator-buffered", ctx.buildAssistantCardHtml(vmList[4]));
      if (vmList[5]) setHtml("gallery-fixture-assistant-evaluator-multi-draft", ctx.buildAssistantCardHtml(vmList[5]));
      if (vmList[6]) setHtml("gallery-fixture-assistant-human-escalation", ctx.buildAssistantCardHtml(vmList[6]));
    }
    if (ctx.lastIndexerOperatorWorkspacesNested && typeof ctx.buildIndexerOperatorWorkspaceCard === "function") {
      var workspaces = ctx.lastIndexerOperatorWorkspacesNested;
      function policyFixture(workspace, mountID) {
        if (!workspace) return;
        var wsNum = Number(workspace.id);
        ctx.workspaceManagedEditId = wsNum;
        ctx.workspaceManagedStaging = {
          wsNum: wsNum,
          paths: (workspace.paths || []).map(function (path) {
            return { id: path.id, path: path.path };
          })
        };
        setHtml(mountID, ctx.buildIndexerOperatorWorkspaceCard(workspace, {}));
      }
      policyFixture(workspaces[0], "gallery-fixture-opws");
      policyFixture(workspaces[1], "gallery-fixture-opws-private");
      policyFixture(workspaces[2], "gallery-fixture-opws-read-write");
      ctx.workspaceManagedEditId = null;
      ctx.workspaceManagedStaging = null;
    }
    if (typeof ctx.buildIndexerStaleSnapshotCard === "function") {
      setHtml(
        "gallery-fixture-indexer-stale",
        ctx.buildIndexerStaleSnapshotCard("gallery-stale-bucket", {
          userLabel: "Gallery",
          projectId: "acme-docs",
          flavorId: "main",
          paths: ["C:\\\\data\\\\stale-watch"]
        })
      );
    }
    if (typeof ctx.buildConvCard === "function") {
      var turnEvents = [
        { seq: 1, ts: "2026-07-25T14:00:00Z", parsed: { rawFlat: { msg: "conversation.received", turn_index: 3 } } },
        { seq: 2, ts: "2026-07-25T14:00:01Z", parsed: { rawFlat: { msg: "harness.stage.started", turn_index: 3, assistant_id: "Research-1.0", stage: "retrieval" } } },
        { seq: 3, ts: "2026-07-25T14:00:02Z", parsed: { rawFlat: { msg: "harness.stage.completed", turn_index: 3, assistant_id: "Research-1.0", stage: "retrieval" } } },
        { seq: 4, ts: "2026-07-25T14:00:03Z", parsed: { rawFlat: { msg: "harness.stage.completed", turn_index: 3, assistant_id: "Research-1.0", stage: "primary" } } },
        { seq: 5, ts: "2026-07-25T14:00:04Z", parsed: { rawFlat: { msg: "harness.stage.completed", turn_index: 3, assistant_id: "Research-1.0", stage: "evaluator" } } },
        { seq: 6, ts: "2026-07-25T14:00:05Z", parsed: { rawFlat: { msg: "conversation.delivered", turn_index: 3 } } }
      ];
      setHtml("gallery-fixture-conversation-harness", ctx.buildConvCard({ pid: "tenant-a", cid: "gallery-harness-turn", events: turnEvents }));
    }
    var Messages = globalThis.ChimeraChat && ChimeraChat.Render && ChimeraChat.Render.Messages;
    if (Messages && typeof Messages.renderMessage === "function") {
      var fixtureMessage = {
        id: "gallery-harness-message",
        role: "assistant",
        content: "The indexed workspace confirms the requested behavior.",
        harnessSummary: {
          intent: { task_type: "code", domain: "repository", complexity: "medium", tags: ["workspace"] },
          plan: { primary_model_id: "groq/free" },
          retrieval: { ran: true, hits_count: 4, compress_strategy: "summarize" },
          execution: { resolved_model_id: "groq/free" },
          evaluation: { ran: true, recommend_escalation: false }
        }
      };
      setHtml("gallery-fixture-chat-turn-details-collapsed", Messages.renderMessage(fixtureMessage));
      setHtml("gallery-fixture-chat-turn-details-expanded", Messages.renderMessage(fixtureMessage).replace('class="chat-turn-details"', 'class="chat-turn-details" open'));
      setHtml("gallery-fixture-chat-human-escalation", Messages.renderMessage({
        id: "gallery-human-escalation",
        role: "assistant",
        content: "This request needs an external review after internal attempts were exhausted.\n\nPrivacy: Remove customer secrets before sharing.\n\nEscalation surfaces:\n- Support desk: https://support.example.test/escalate\n\nPaste the external answer in your next message after:\n<<<CHIMERA_HUMAN_ANSWER>>>"
      }));
    }
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", renderFixtures);
  } else {
    renderFixtures();
  }
})();
