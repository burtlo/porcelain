/**
 * Assistant cards for /ui/settings summarized feed (sectioned routing stack).
 */
globalThis.ChimeraSettings = globalThis.ChimeraSettings || {};
globalThis.ChimeraSettings.Render = globalThis.ChimeraSettings.Render || {};
globalThis.ChimeraSettings.Render.Cards = globalThis.ChimeraSettings.Render.Cards || {};

globalThis.ChimeraSettings.Render.Cards.mountAdminAssistants = function (ctx) {
  var escapeHtml = ctx.escapeHtml;
  var formatInt = typeof ctx.formatInt === "function" ? ctx.formatInt : function (n) { return String(n); };
  var operatorCardChevronHtml = ctx.operatorCardChevronHtml;
  var sgOpHealthPillHtml = ctx.sgOpHealthPillHtml;
  var fallbackChainToYAML = ctx.fallbackChainToYAML;
  var parseFallbackChainInput = ctx.parseFallbackChainInput;
  var parseRoutingRulesFromYAML = ctx.parseRoutingRulesFromYAML;
  var adminModelUsageById = ctx.adminModelUsageById;
  var adminExtractProviderModel = ctx.adminExtractProviderModel;
  var adminProviderTierSpan = ctx.adminProviderTierSpan;
  var adminScopedEvlogPanelFromEvents = ctx.adminScopedEvlogPanelFromEvents;
  var operatorConfigureBtnInline = ctx.operatorConfigureBtnInline;
  var getFlat = ctx.getFlat;
  var entryCache = ctx.entryCache;

  function vmUi(ctx, id) {
    var key = String(id);
    if (!ctx.assistantUi) ctx.assistantUi = {};
    if (!ctx.assistantUi[key]) {
      ctx.assistantUi[key] = {
        panelOpen: false,
        hydrated: false,
        detailLoading: false,
        identityEditing: false,
        fallbackEditing: false,
        routingEditing: false,
        routerEditing: false,
        fallbackTouched: false,
        fallbackDraft: null,
        policyTouched: false,
        routerModelsTouched: false,
        routerThresholdTouched: false,
        routerEnabledTouched: false,
        policyDraft: null,
        routerModelsDraft: null,
        routerThresholdDraft: null,
        routerEnabledDraft: null,
        sectionOpen: { identity: true, fallback: true }
      };
    }
    return ctx.assistantUi[key];
  }

  function vmSectionOpenAttr(ui, sectionKey) {
    ui = ui || {};
    if (!ui.sectionOpen) ui.sectionOpen = { identity: true, fallback: true };
    if (ui.sectionOpen[sectionKey]) return " open";
    if (sectionKey === "identity") return " open";
    if (sectionKey === "fallback" && ui.fallbackEditing) return " open";
    if (sectionKey === "routing" && ui.routingEditing) return " open";
    if (sectionKey === "router" && ui.routerEditing) return " open";
    return "";
  }

  var ET = globalThis.ChimeraShared && globalThis.ChimeraShared.EditToolbar;
  var YE = globalThis.ChimeraShared && globalThis.ChimeraShared.YamlEditor;

  function vmToolbarIconBtn(action, vmId, title, iconName, extraClass) {
    if (ET && typeof ET.iconBtnHtml === "function") {
      return ET.iconBtnHtml(escapeHtml, {
        action: action,
        title: title,
        icon: iconName,
        extraClass: extraClass,
        dataAttrs: { "vm-id": String(vmId) }
      });
    }
    return "";
  }

  function vmAttrHidden(isHidden) {
    return isHidden ? " hidden" : "";
  }

  function vmYamlTextareaHtml(pfx, taId, yaml, touched, rows, readonly) {
    if (YE && typeof YE.textareaWrapHtml === "function") {
      return YE.textareaWrapHtml(escapeHtml, {
        id: pfx + taId,
        yaml: yaml,
        touched: touched,
        rows: rows != null ? rows : 8,
        readonly: readonly,
        spellcheck: false
      });
    }
    return "";
  }

  function vmSectionToolbarHtml(vm, leadingHtml, editOpts) {
    editOpts = editOpts || {};
    var vmId = vm.id;
    var editing = !!editOpts.editing;
    var actions = "";
    if (editing && editOpts.generateAction) {
      actions +=
        '<button class="sg-op-btn sg-op-btn--ghost" type="button" data-admin-action="' +
        escapeHtml(String(editOpts.generateAction)) +
        '" data-vm-id="' +
        escapeHtml(String(vmId)) +
        '">Generate from catalog</button>';
    }
    if (editing && editOpts.deleteAction) {
      actions += vmToolbarIconBtn(
        editOpts.deleteAction,
        vmId,
        editOpts.deleteTitle || "Delete assistant",
        "delete_forever",
        "vm-identity-btn-delete"
      );
    }
    if (editing && editOpts.saveAction) {
      actions += vmToolbarIconBtn(editOpts.saveAction, vmId, editOpts.saveTitle || "Keep changes", "keep");
    }
    if (editing && editOpts.refreshAction) {
      actions += vmToolbarIconBtn(editOpts.refreshAction, vmId, editOpts.refreshTitle || "Revert", "refresh");
    }
    if (editing && editOpts.cancelAction) {
      actions += vmToolbarIconBtn(editOpts.cancelAction, vmId, editOpts.cancelTitle || "Cancel", "cancel");
    }
    if (!editing && editOpts.configureAction) {
      actions += vmConfigureBtn(editOpts.configureAction, vmId, editOpts.configureTitle);
    }
    return (
      '<div class="sum-asst-section__toolbar">' +
      (leadingHtml ? '<div class="sum-asst-section__toolbar-leading">' + leadingHtml + "</div>" : "") +
      '<div class="sum-asst-section__toolbar-actions">' +
      actions +
      "</div></div>"
    );
  }

  function vmDetail(ctx, id) {
    if (!ctx.assistantDetails) return null;
    return ctx.assistantDetails[String(id)] || null;
  }

  function mergeVm(summary, detail) {
    var base = summary || {};
    if (!detail) return base;
    var out = {};
    var keys = [
      "id", "model_id", "name", "version", "description", "enabled", "visibility",
      "fallback_depth", "routing_policy_enabled", "tool_router_enabled", "router_models",
      "routing_policy_yaml", "fallback_chain", "fallback_unavailable", "tool_router_confidence_threshold",
      "harness_modules"
    ];
    for (var i = 0; i < keys.length; i++) {
      var k = keys[i];
      if (detail[k] !== undefined && detail[k] !== null) out[k] = detail[k];
      else if (base[k] !== undefined && base[k] !== null) out[k] = base[k];
    }
    if (detail.fallback_chain && detail.fallback_chain.length) {
      out.fallback_depth = detail.fallback_chain.length;
    }
    return out;
  }

  function adminScopedEventsForAssistant(modelId) {
    modelId = String(modelId || "").trim();
    var out = [];
    for (var i = entryCache.length - 1; i >= 0 && out.length < 18; i--) {
      var ev = entryCache[i];
      var f = getFlat(ev.parsed);
      var vmField = String(f.assistant_id || "").trim();
      var msg = String(f.msg || f.message || "").toLowerCase();
      var isHarness = msg.indexOf("harness.") === 0;
      // Harness events are strictly VM-scoped: never leak an untagged event into
      // every VM card while retaining existing routing lines' historical scope.
      if (isHarness && (!vmField || vmField !== modelId)) continue;
      if (!isHarness && modelId && vmField && vmField !== modelId) continue;
      if (
        isHarness ||
        msg.indexOf("routing") >= 0 ||
        msg.indexOf("fallback") >= 0 ||
        msg.indexOf("failover") >= 0 ||
        msg.indexOf("tool_router") >= 0 ||
        msg.indexOf("tool-router") >= 0 ||
        msg.indexOf("assistant") >= 0
      ) {
        out.push(ev);
      }
    }
    return out;
  }

  function vmEnabledPill(vm) {
    if (vm && vm.enabled) return sgOpHealthPillHtml("enabled", "ok");
    return sgOpHealthPillHtml("disabled", "unknown");
  }

  function vmVisibilityPill(vm) {
    var vis = String((vm && vm.visibility) || "public").toLowerCase() === "private" ? "private" : "public";
    return sgOpHealthPillHtml(vis, vis === "private" ? "unknown" : "metric");
  }

  function vmCardTitleText(vm) {
    vm = vm || {};
    var name = String(vm.name != null ? vm.name : "").trim();
    var version = String(vm.version != null ? vm.version : "").trim();
    if (name && version) return name + " · " + version;
    if (name) return name;
    if (version) return version;
    var modelId = String(vm.model_id != null ? vm.model_id : "").trim();
    return modelId || "Assistant";
  }

  function vmCardSubtitleText(vm) {
    return String((vm && vm.description != null ? vm.description : "") || "").trim();
  }

  function vmMetricsHtml(vm) {
    return vmEnabledPill(vm) + vmVisibilityPill(vm);
  }

  function syncAssistantCardHeader(cardEl, vm) {
    if (!cardEl || !vm) return;
    var titleEl = cardEl.querySelector(".sum-main .sum-title");
    var subEl = cardEl.querySelector(".sum-main .sum-sub");
    var metricsEl = cardEl.querySelector(".sum-metrics");
    if (titleEl) titleEl.textContent = vmCardTitleText(vm);
    if (subEl) subEl.textContent = vmCardSubtitleText(vm);
    if (metricsEl) metricsEl.innerHTML = vmMetricsHtml(vm);
  }

  function vmIdPrefix(rowId) {
    return "vm-" + String(rowId) + "-";
  }

  function vmRouterToggleHtml(id, action, vmId, pressed, ariaLabel) {
    return (
      '<button class="sum-router-toggle" type="button" id="' +
      escapeHtml(String(id || "")) +
      '" data-admin-action="' +
      escapeHtml(String(action || "")) +
      '" data-vm-id="' +
      escapeHtml(String(vmId)) +
      '" aria-label="' +
      escapeHtml(String(ariaLabel || "")) +
      '" aria-pressed="' +
      (pressed ? "true" : "false") +
      '"><span class="sum-router-toggle__track"><span class="sum-router-toggle__thumb"></span></span></button>'
    );
  }

  function vmCardUsageTogglesHtml(vm, pfx) {
    var visPublic = String(vm.visibility || "public").toLowerCase() !== "private";
    var enabled = !!vm.enabled;
    return (
      '<div class="sum-asst-card-toggles">' +
      '<span class="sum-asst-hdr-toggle">' +
      '<span class="sum-asst-hdr-toggle-label">Enabled</span>' +
      vmRouterToggleHtml(
        pfx + "enabled-toggle",
        "vm-identity-enabled-toggle",
        vm.id,
        enabled,
        "Toggle enabled"
      ) +
      '<span class="sum-asst-hdr-toggle-state muted">' +
      escapeHtml(enabled ? "on" : "off") +
      "</span></span>" +
      '<span class="sum-asst-hdr-toggle">' +
      '<span class="sum-asst-hdr-toggle-label">Visibility</span>' +
      vmRouterToggleHtml(
        pfx + "visibility-toggle",
        "vm-identity-visibility-toggle",
        vm.id,
        visPublic,
        "Toggle visibility (on = public)"
      ) +
      '<span class="sum-asst-hdr-toggle-state muted">' +
      escapeHtml(visPublic ? "public" : "private") +
      "</span></span></div>"
    );
  }

  function vmSectionHdrToggleHtml(label, id, action, vmId, pressed, ariaLabel) {
    return (
      '<span class="sum-asst-section__hdr-toggles">' +
      '<span class="sum-asst-hdr-toggle">' +
      '<span class="sum-asst-hdr-toggle-label">' +
      escapeHtml(String(label || "")) +
      "</span>" +
      vmRouterToggleHtml(id, action, vmId, pressed, ariaLabel) +
      '<span class="sum-asst-hdr-toggle-state muted">' +
      escapeHtml(pressed ? "on" : "off") +
      "</span></span></span>"
    );
  }

  function vmSectionHeaderHtml(title, opts) {
    return vmSectionHeaderParts(title, opts).summary;
  }

  /** Disclosure header: toggles live inside <summary> so they stay visible when collapsed. */
  function vmSectionHeaderParts(title, opts) {
    opts = opts || {};
    var controls = opts.controlsHtml != null ? String(opts.controlsHtml) : "";
    var desc = opts.desc != null ? String(opts.desc) : "";
    var trailInRow = "";
    if (controls) {
      trailInRow =
        '<span class="sum-asst-section__hdr-trail" data-sum-card-no-toggle>' +
        '<span class="sum-asst-section__trail">' +
        controls +
        "</span></span>";
    }
    var summary =
      '<summary class="sum-asst-section__hdr">' +
      '<span class="sum-asst-section__hdr-row">' +
      '<span class="sum-asst-section__lead">' +
      operatorCardChevronHtml() +
      "</span>" +
      '<span class="sum-asst-section__title"><span class="sum-section-label">' +
      escapeHtml(String(title || "")) +
      "</span></span>" +
      trailInRow +
      "</span>";
    if (desc) {
      summary +=
        '<span class="sum-asst-section__hdr-desc-row">' +
        '<p class="sg-op-card-note sg-op-card-note--tight sum-asst-section__hdr-desc">' +
        escapeHtml(desc) +
        "</p></span>";
    }
    summary += "</summary>";
    return { summary: summary, trail: "" };
  }

  function vmConfigureBtn(action, vmId, title) {
    return vmToolbarIconBtn(action, vmId, title != null ? String(title) : "Configure", "settings");
  }

  function buildClientUsageBlock(vm, pfx, loading) {
    if (loading) {
      return (
        '<div class="sum-asst-section sum-asst-section--bar" data-ui-part="assistant.client-usage">' +
        '<div class="sum-asst-section__hdr sum-asst-section__hdr--bar"><p class="muted">Loading…</p></div></div>'
      );
    }
    return (
      '<div class="sum-asst-section sum-asst-section--bar" data-ui-part="assistant.client-usage">' +
      '<div class="sum-asst-section__hdr sum-asst-section__hdr--bar" aria-label="Model access">' +
      '<span class="sum-asst-section__hdr-row">' +
      '<span class="sum-asst-section__trail">' +
      vmCardUsageTogglesHtml(vm, pfx) +
      "</span></span></div></div>"
    );
  }

  function buildIdentityReadonlyHtml(vm) {
    var name = String(vm.name != null ? vm.name : "").trim() || "—";
    var version = String(vm.version != null ? vm.version : "").trim() || "—";
    var desc = String(vm.description != null ? vm.description : "").trim() || "—";
    var modelId = String(vm.model_id || "").trim() || "—";
    return (
      '<dl class="sg-op-kv sg-op-kv--vm-identity">' +
      "<dt>Client model id</dt><dd><code class=\"sum-mono-id\">" +
      escapeHtml(modelId) +
      "</code></dd>" +
      "<dt>Name</dt><dd>" +
      escapeHtml(name) +
      "</dd>" +
      "<dt>Version</dt><dd>" +
      escapeHtml(version) +
      "</dd>" +
      "<dt>Description</dt><dd>" +
      escapeHtml(desc) +
      "</dd></dl>"
    );
  }

  function buildIdentityEditHtml(vm, pfx) {
    var name = String(vm.name != null ? vm.name : "");
    var version = String(vm.version != null ? vm.version : "");
    var desc = String(vm.description != null ? vm.description : "");
    return (
      '<label class="sg-op-label" for="' +
      pfx +
      'name">Name</label>' +
      '<input id="' +
      pfx +
      'name" class="sg-op-input" type="text" value="' +
      escapeHtml(name) +
      '"/>' +
      '<label class="sg-op-label" for="' +
      pfx +
      'version">Version</label>' +
      '<input id="' +
      pfx +
      'version" class="sg-op-input" type="text" value="' +
      escapeHtml(version) +
      '"/>' +
      '<label class="sg-op-label" for="' +
      pfx +
      'description">Description</label>' +
      '<textarea id="' +
      pfx +
      'description" class="sg-op-input" rows="3" spellcheck="false">' +
      escapeHtml(desc) +
      "</textarea>"
    );
  }

  function buildIdentitySection(vm, ui, pfx, loading) {
    var body = "";
    if (loading) {
      body = '<p class="muted">Loading…</p>';
    } else {
      body =
        vmSectionToolbarHtml(vm, "", {
          editing: ui.identityEditing,
          deleteAction: "vm-identity-delete",
          deleteTitle: "Delete assistant",
          saveAction: "vm-identity-save",
          saveTitle: "Keep identity",
          refreshAction: "vm-identity-refresh",
          refreshTitle: "Revert identity to last saved",
          cancelAction: "vm-identity-cancel",
          cancelTitle: "Cancel identity edit",
          configureAction: "vm-identity-configure",
          configureTitle: "Configure identity"
        }) +
        '<div id="' +
        pfx +
        'identity-view"' +
        (ui.identityEditing ? " hidden" : "") +
        ">" +
        buildIdentityReadonlyHtml(vm) +
        "</div>" +
        '<div id="' +
        pfx +
        'identity-edit"' +
        (ui.identityEditing ? "" : " hidden") +
        ">" +
        buildIdentityEditHtml(vm, pfx) +
        "</div>";
    }
    return (
      '<details class="sum-asst-section" data-vm-section="identity" data-ui-part="assistant.identity"' +
      vmSectionOpenAttr(ui, "identity") +
      ">" +
      vmSectionHeaderHtml("Identity") +
      '<div class="sum-asst-section__body">' +
      body +
      "</div></details>"
    );
  }

  function buildFallbackSection(vm, ui, pfx, gw, loading) {
    var chain = Array.isArray(vm.fallback_chain) ? vm.fallback_chain : [];
    var fallbackYAML = fallbackChainToYAML(chain);
    if (ui.fallbackDraft != null) {
      fallbackYAML = String(ui.fallbackDraft);
    } else if (ui.fallbackTouched && typeof document !== "undefined") {
      var fbTa = document.getElementById(pfx + "fallback-yaml-ta");
      if (fbTa) fallbackYAML = String(fbTa.value || fallbackYAML);
    }
    var displayChain = chain;
    if (ui.fallbackTouched) {
      try {
        displayChain = parseFallbackChainInput(fallbackYAML);
      } catch (_e) {
        displayChain = chain;
      }
    }
    var usesByModel = adminModelUsageById();
    var unavailableSet = {};
    var unavailList = Array.isArray(vm.fallback_unavailable) ? vm.fallback_unavailable : [];
    for (var ufi = 0; ufi < unavailList.length; ufi++) {
      var umid = String(unavailList[ufi] || "").trim();
      if (umid) unavailableSet[umid] = true;
    }
    var tableRows = "";
    for (var i = 0; i < displayChain.length; i++) {
      var mid = String(displayChain[i] || "");
      var pm = adminExtractProviderModel(mid);
      var rowCls = unavailableSet[mid] ? " sg-op-vm-fallback-row--unavailable" : "";
      var unavailBadge = unavailableSet[mid]
        ? ' <span class="sg-op-vm-fallback-unavail-badge" title="Operator marked unavailable for this tenant">unavailable</span>'
        : "";
      tableRows +=
        "<tr" +
        (rowCls ? ' class="' + rowCls.trim() + '"' : "") +
        ">" +
        '<td class="num">' +
        escapeHtml(String(i + 1)) +
        "</td>" +
        "<td>" +
        adminProviderTierSpan(pm.provider) +
        "</td>" +
        '<td><code class="sum-mono-id">' +
        escapeHtml(mid) +
        "</code>" +
        unavailBadge +
        "</td>" +
        '<td class="num">' +
        escapeHtml(formatInt(usesByModel[mid] || 0)) +
        "</td>" +
        "</tr>";
    }
    if (!tableRows) {
      tableRows = '<tr><td colspan="4" class="muted">No fallback routes configured.</td></tr>';
    }
    var body = "";
    if (loading) {
      body = '<p class="muted">Loading fallback chain…</p>';
    } else {
      body =
        vmSectionToolbarHtml(vm, "", {
          editing: ui.fallbackEditing,
          generateAction: "vm-fallback-generate",
          saveAction: "vm-fallback-save",
          saveTitle: "Keep fallback chain",
          refreshAction: "vm-fallback-refresh",
          refreshTitle: "Revert fallback to last saved",
          cancelAction: "vm-fallback-cancel",
          cancelTitle: "Cancel fallback edit",
          configureAction: "vm-fallback-configure",
          configureTitle: "Configure fallback"
          }
        ) +
        '<div class="sum-asst-fallback-panel">' +
        '<div id="' +
        pfx +
        'fallback-table" class="sum-asst-fallback-panel__view"' +
        vmAttrHidden(ui.fallbackEditing) +
        '><div class="sum-metrics-table-wrap sg-op-fallback-table-scroll"><table class="sum-metrics-table"><thead><tr><th class="num">Order</th><th>Provider</th><th>Model</th><th class="num">Uses (24h)</th></tr></thead><tbody>' +
        tableRows +
        "</tbody></table></div></div>" +
        '<div id="' +
        pfx +
        'fallback-yaml" class="sum-asst-fallback-panel__view"' +
        vmAttrHidden(!ui.fallbackEditing) +
        ">" +
        vmYamlTextareaHtml(pfx, "fallback-yaml-ta", fallbackYAML, ui.fallbackTouched, 8) +
        "</div></div>";
    }
    return (
      '<details class="sum-asst-section" data-vm-section="fallback" data-ui-part="assistant.fallback"' +
      vmSectionOpenAttr(ui, "fallback") +
      ">" +
      vmSectionHeaderHtml("Fallback chain", {
        desc: "Required. Ordered upstream model ids for failover after the routing policy picks an initial model."
      }) +
      '<div class="sum-asst-section__body">' +
      body +
      "</div></details>"
    );
  }

  function buildRoutingSection(vm, ui, pfx, gw, loading) {
    var policy = String(vm.routing_policy_yaml || "");
    var policyYAML = policy;
    if (ui.policyDraft != null) {
      policyYAML = String(ui.policyDraft);
    } else if (ui.policyTouched && typeof document !== "undefined") {
      var routeTa = document.getElementById(pfx + "routing-yaml-ta");
      if (routeTa) policyYAML = String(routeTa.value || policyYAML);
    }
    var polEnabled = !!vm.routing_policy_enabled;
    var displayPolicy = policy;
    if (ui.policyTouched && !ui.routingEditing) {
      displayPolicy = policyYAML;
    }
    var routingRulesRows = parseRoutingRulesFromYAML(displayPolicy);
    var usesByModel = adminModelUsageById();
    var tableRows = "";
    for (var ri = 0; ri < routingRulesRows.length; ri++) {
      var rr = routingRulesRows[ri] || {};
      var matchVal = "";
      if (rr.whenInline) {
        matchVal = rr.whenInline === "{}" ? "(catch-all)" : rr.whenInline;
      } else if (rr.whenParts && rr.whenParts.length) {
        matchVal = rr.whenParts.join("; ");
      } else {
        matchVal = "(catch-all)";
      }
      var modelCell = "—";
      if (rr.models && rr.models.length) {
        var parts = [];
        for (var mi = 0; mi < rr.models.length; mi++) {
          parts.push('<code class="sum-mono-id">' + escapeHtml(rr.models[mi]) + "</code>");
        }
        modelCell = parts.join(", ");
      }
      var hits = 0;
      for (var hm = 0; hm < (rr.models || []).length; hm++) {
        hits += Number(usesByModel[rr.models[hm]] || 0);
      }
      tableRows +=
        "<tr>" +
        '<td><code class="sum-mono-id">' +
        escapeHtml(rr.name || "unnamed") +
        "</code></td>" +
        '<td><code class="sum-mono-id">' +
        escapeHtml(matchVal) +
        "</code></td>" +
        "<td>" +
        modelCell +
        "</td>" +
        '<td class="num">' +
        escapeHtml(formatInt(hits)) +
        "</td>" +
        "</tr>";
    }
    if (!tableRows) {
      tableRows = '<tr><td colspan="4" class="muted">No routing rules configured.</td></tr>';
    }
    var body = "";
    if (loading) {
      body = '<p class="muted">Loading routing policy…</p>';
    } else {
      body =
        vmSectionToolbarHtml(vm, "", {
          editing: ui.routingEditing,
          generateAction: "vm-routing-generate",
          saveAction: "vm-routing-save",
          saveTitle: "Keep routing policy",
          refreshAction: "vm-routing-refresh",
          refreshTitle: "Revert routing policy to last saved",
          cancelAction: "vm-routing-cancel",
          cancelTitle: "Cancel routing edit",
          configureAction: "vm-routing-configure",
          configureTitle: "Configure routing"
          }
        ) +
        '<div class="sum-asst-routing-panel">' +
        '<div id="' +
        pfx +
        'routing-table" class="sum-asst-routing-panel__view"' +
        vmAttrHidden(ui.routingEditing) +
        '><div class="sum-metrics-table-wrap sg-op-routing-table-scroll"><table class="sum-metrics-table"><thead><tr><th>Name</th><th>Match</th><th>Models</th><th class="num">Hits (24h)</th></tr></thead><tbody>' +
        tableRows +
        "</tbody></table></div></div>" +
        '<div id="' +
        pfx +
        'routing-yaml" class="sum-asst-routing-panel__view"' +
        vmAttrHidden(!ui.routingEditing) +
        ">" +
        vmYamlTextareaHtml(pfx, "routing-yaml-ta", policyYAML, ui.policyTouched, 10) +
        "</div></div>";
    }
    return (
      '<details class="sum-asst-section" data-vm-section="routing" data-ui-part="assistant.routing"' +
      vmSectionOpenAttr(ui, "routing") +
      ">" +
      vmSectionHeaderHtml("Routing policy", {
        desc: "Optional. First matching rule selects the initial upstream model; otherwise ambiguous default or fallback head applies.",
        controlsHtml: vmSectionHdrToggleHtml(
          "Enabled",
          pfx + "routing-enabled",
          "vm-routing-enabled-toggle",
          vm.id,
          polEnabled,
          "Toggle routing policy enabled"
        )
      }) +
      '<div class="sum-asst-section__body">' +
      body +
      "</div></details>"
    );
  }

  function buildToolRouterSection(vm, ui, pfx, loading) {
    var routerModels = Array.isArray(vm.router_models) ? vm.router_models : [];
    var thresholdSaved = String(vm.tool_router_confidence_threshold != null ? vm.tool_router_confidence_threshold : 0.5);
    var thresholdEdit =
      ui.routerThresholdTouched && ui.routerThresholdDraft != null ? String(ui.routerThresholdDraft) : thresholdSaved;
    var thresholdDisplay = ui.routerEditing ? thresholdEdit : thresholdSaved;
    var routerEnabled =
      ui.routerEnabledTouched && ui.routerEnabledDraft != null ? !!ui.routerEnabledDraft : !!vm.tool_router_enabled;
    var routerYAML = ui.routerModelsTouched
      ? ui.routerModelsDraft != null
        ? String(ui.routerModelsDraft)
        : fallbackChainToYAML(routerModels)
      : fallbackChainToYAML(routerModels);
    var routerChain = routerModels;
    if (ui.routerModelsTouched) {
      try {
        routerChain = parseFallbackChainInput(routerYAML);
      } catch (_e) {
        routerChain = routerModels;
      }
    }
    var usesByModel = adminModelUsageById();
    var tableRows = "";
    for (var i = 0; i < routerChain.length; i++) {
      var rid = String(routerChain[i] || "");
      var rpm = adminExtractProviderModel(rid);
      tableRows +=
        "<tr>" +
        '<td class="num">' +
        escapeHtml(String(i + 1)) +
        "</td>" +
        "<td>" +
        adminProviderTierSpan(rpm.provider) +
        "</td>" +
        '<td><code class="sum-mono-id">' +
        escapeHtml(rid) +
        "</code></td>" +
        '<td class="num">' +
        escapeHtml(formatInt(usesByModel[rid] || 0)) +
        "</td>" +
        "</tr>";
    }
    if (!tableRows) {
      tableRows = '<tr><td colspan="4" class="muted">No router models configured.</td></tr>';
    }
    var body = "";
    if (loading) {
      body = '<p class="muted">Loading tool-router settings…</p>';
    } else {
      body =
        vmSectionToolbarHtml(vm, "", {
          editing: ui.routerEditing,
          saveAction: "vm-router-save",
          saveTitle: "Keep tool router",
          refreshAction: "vm-router-refresh",
          refreshTitle: "Revert tool router to last saved",
          cancelAction: "vm-router-cancel",
          cancelTitle: "Cancel tool router edit",
          configureAction: "vm-router-configure",
          configureTitle: "Configure tool router"
          }
        ) +
        '<div class="sg-op-head-row sum-asst-router-threshold-row">' +
        '<label class="sg-op-label sg-op-label--inline" for="' +
        pfx +
        'router-threshold">Confidence threshold</label>' +
        '<div class="sg-op-head-actions">' +
        '<input id="' +
        pfx +
        'router-threshold" class="sg-op-input' +
        (ui.routerEditing ? "" : " sg-op-input--readonly") +
        '" type="number" min="0" max="1" step="0.05" value="' +
        escapeHtml(thresholdDisplay) +
        '"' +
        (ui.routerEditing ? "" : " readonly") +
        ' aria-label="Tool router confidence threshold" /></div></div>' +
        '<div id="' +
        pfx +
        'router-table" class="sum-asst-router-panel__view"' +
        vmAttrHidden(ui.routerEditing) +
        '><div class="sum-metrics-table-wrap sg-op-router-table-scroll"><table class="sum-metrics-table"><thead><tr><th class="num">Order</th><th>Provider</th><th>Model</th><th class="num">Uses (24h)</th></tr></thead><tbody>' +
        tableRows +
        "</tbody></table></div></div>" +
        '<div id="' +
        pfx +
        'router-edit" class="sum-asst-router-panel__view"' +
        vmAttrHidden(!ui.routerEditing) +
        ">" +
        vmYamlTextareaHtml(pfx, "router-yaml-ta", routerYAML, ui.routerModelsTouched, 6) +
        "</div>";
    }
    return (
      '<details class="sum-asst-section" data-vm-section="router" data-ui-part="assistant.tool-router"' +
      vmSectionOpenAttr(ui, "router") +
      ">" +
      vmSectionHeaderHtml("Tool router", {
        desc: "Optional. Slims tools before the main completion when enabled and router models are set.",
        controlsHtml: vmSectionHdrToggleHtml(
          "Enabled",
          pfx + "router-enabled",
          "vm-router-enabled-toggle",
          vm.id,
          routerEnabled,
          "Toggle tool router enabled"
        )
      }) +
      '<div class="sum-asst-section__body">' +
      body +
      "</div></details>"
    );
  }

  function harnessModuleLabel(moduleId) {
    switch (String(moduleId || "")) {
      case "retrieval":
        return "Retrieval";
      case "intent":
        return "Intent";
      case "evaluator":
        return "Evaluator";
      case "escalation":
        return "Escalation";
      case "tool_executor":
        return "Workspace tools";
      default:
        return String(moduleId || "Module");
    }
  }

  function harnessModuleDesc(moduleId) {
    switch (String(moduleId || "")) {
      case "retrieval":
        return "Per-assistant RAG inject (still requires gateway search enabled).";
      case "intent":
        return "Classify turn intent before primary completion.";
      case "evaluator":
        return "Score primary output and apply the selected stream policy.";
      case "escalation":
        return "Re-retrieve or walk fallback when evaluation recommends it.";
      case "tool_executor":
        return "Gateway-injected workspace tools and optional MCP sidecars (per-assistant bind and per-tool allowlist).";
      default:
        return "";
    }
  }

  function retrievalConfigFromModule(module) {
    var cfg = (module && module.config_json) || {};
    if (typeof cfg === "string") {
      try { cfg = JSON.parse(cfg); } catch (_e) { cfg = {}; }
    }
    return cfg && typeof cfg === "object" ? cfg : {};
  }

  function retrievalKnobsHtml(module, pfx, vmId) {
    var cfg = retrievalConfigFromModule(module);
    var strategy = String(cfg.compress_strategy || "truncate").toLowerCase();
    if (strategy !== "none" && strategy !== "truncate" && strategy !== "summarize") strategy = "truncate";
    return (
      '<div class="sum-asst-harness-retrieval" data-ui-part="assistant.harness.retrieval-config">' +
      '<label class="sg-op-label">Top K<input id="' + pfx + 'retrieval-top-k" class="sg-op-input" type="number" min="1" step="1" value="' + escapeHtml(String(cfg.top_k || "")) + '" placeholder="Gateway default" /></label>' +
      '<label class="sg-op-label">Score floor<input id="' + pfx + 'retrieval-score-floor" class="sg-op-input" type="number" min="0" max="1" step="0.01" value="' + escapeHtml(String(cfg.score_floor || "")) + '" placeholder="Gateway default" /></label>' +
      '<label class="sg-op-label">Compression<select id="' + pfx + 'retrieval-compress-strategy" class="sg-op-input"><option value="none"' + (strategy === "none" ? " selected" : "") + '>None</option><option value="truncate"' + (strategy === "truncate" ? " selected" : "") + '>Truncate</option><option value="summarize"' + (strategy === "summarize" ? " selected" : "") + '>Summarize</option></select></label>' +
      '<label class="sg-op-label">Max context chars<input id="' + pfx + 'retrieval-max-context-chars" class="sg-op-input" type="number" min="1" step="100" value="' + escapeHtml(String(cfg.max_context_chars || "")) + '" placeholder="12000" /></label>' +
      '<button type="button" class="sg-op-btn sg-op-btn--ghost" data-admin-action="vm-harness-retrieval-save" data-vm-id="' + escapeHtml(String(vmId)) + '">Keep retrieval settings</button>' +
      "</div>"
    );
  }

  function intentKnobsHtml(module, pfx, vmId) {
    var cfg = retrievalConfigFromModule(module);
    var mode = String(cfg.mode || "heuristic").toLowerCase() === "llm" ? "llm" : "heuristic";
    return (
      '<div class="sum-asst-harness-intent" data-ui-part="assistant.harness.intent-config">' +
      '<label class="sg-op-label">Classification mode<select id="' + pfx + 'intent-mode" class="sg-op-input"><option value="heuristic"' + (mode === "heuristic" ? " selected" : "") + '>Heuristic</option><option value="llm"' + (mode === "llm" ? " selected" : "") + '>LLM assist</option></select></label>' +
      '<label class="sg-op-label">Classifier model id<input id="' + pfx + 'intent-model-id" class="sg-op-input" type="text" value="' + escapeHtml(String(cfg.model_id || "")) + '" placeholder="provider/model" /></label>' +
      '<p class="sg-op-card-note sg-op-card-note--tight">Heuristic mode never calls an upstream classifier. LLM assist falls back to heuristic classification when unavailable.</p>' +
      '<button type="button" class="sg-op-btn sg-op-btn--ghost" data-admin-action="vm-harness-intent-save" data-vm-id="' + escapeHtml(String(vmId)) + '">Keep intent settings</button>' +
      "</div>"
    );
  }

  function evaluatorKnobsHtml(module, pfx, vmId) {
    var cfg = retrievalConfigFromModule(module);
    var mode = String(cfg.mode || "single_pass").toLowerCase() === "multi_draft" ? "multi_draft" : "single_pass";
    var policy = String(cfg.stream_policy || "immediate").toLowerCase();
    if (policy !== "gate_on_evaluator" && policy !== "buffer_until_complete") policy = "immediate";
    return (
      '<div class="sum-asst-harness-evaluator" data-ui-part="assistant.harness.evaluator-config">' +
      '<label class="sg-op-label">Mode<select id="' + pfx + 'evaluator-mode" class="sg-op-input"><option value="single_pass"' + (mode === "single_pass" ? " selected" : "") + '>Single pass</option><option value="multi_draft"' + (mode === "multi_draft" ? " selected" : "") + '>Multi-draft</option></select></label>' +
      '<label class="sg-op-label">Evaluator model id<input id="' + pfx + 'evaluator-model-id" class="sg-op-input" type="text" value="' + escapeHtml(String(cfg.model_id || "")) + '" placeholder="provider/model" /></label>' +
      '<label class="sg-op-label">Draft count<input id="' + pfx + 'evaluator-draft-count" class="sg-op-input" type="number" min="1" max="8" step="1" value="' + escapeHtml(String(cfg.draft_count || 3)) + '" /></label>' +
      '<label class="sg-op-label">Synthesis model id<input id="' + pfx + 'evaluator-synthesize-model-id" class="sg-op-input" type="text" value="' + escapeHtml(String(cfg.synthesize_model_id || "")) + '" placeholder="provider/model" /></label>' +
      '<label class="sg-op-label">Stream policy<select id="' + pfx + 'evaluator-stream-policy" class="sg-op-input"><option value="immediate"' + (policy === "immediate" ? " selected" : "") + '>Immediate</option><option value="gate_on_evaluator"' + (policy === "gate_on_evaluator" ? " selected" : "") + '>Gate on evaluator</option><option value="buffer_until_complete"' + (policy === "buffer_until_complete" ? " selected" : "") + '>Buffer until complete</option></select></label>' +
      '<label class="sg-op-label">Min confidence<input id="' + pfx + 'evaluator-min-confidence" class="sg-op-input" type="number" min="0" max="1" step="0.01" value="' + escapeHtml(String(cfg.min_confidence || "")) + '" /></label>' +
      '<label class="sg-op-label">Hallucination risk max<input id="' + pfx + 'evaluator-hallucination-risk-max" class="sg-op-input" type="number" min="0" max="1" step="0.01" value="' + escapeHtml(String(cfg.hallucination_risk_max || "")) + '" /></label>' +
      '<button type="button" class="sg-op-btn sg-op-btn--ghost" data-admin-action="vm-harness-evaluator-save" data-vm-id="' + escapeHtml(String(vmId)) + '">Keep evaluator settings</button></div>'
    );
  }

  function mcpBindingEnabled(bindings, serverId) {
    var list = Array.isArray(bindings) ? bindings : [];
    for (var i = 0; i < list.length; i++) {
      if (String(list[i].server_id || "") === serverId) return !!list[i].enabled;
    }
    return false;
  }

  function mcpToolEnabled(bindings, serverId, toolName) {
    var list = Array.isArray(bindings) ? bindings : [];
    for (var i = 0; i < list.length; i++) {
      if (String(list[i].server_id || "") !== serverId) continue;
      var tools = Array.isArray(list[i].tools) ? list[i].tools : [];
      for (var j = 0; j < tools.length; j++) {
        if (String(tools[j].tool_name || "") === toolName) return !!tools[j].enabled;
      }
    }
    return false;
  }

  function toolExecutorMcpKnobsHtml(pfx, vmId) {
    var cache = (ctx.adminStateCache && ctx.adminStateCache.mcp) || {};
    var servers = Array.isArray(cache.servers) ? cache.servers : [];
    var bindings = (cache.bindingsByVm && cache.bindingsByVm[String(vmId)]) || [];
    if (!servers.length) {
      return (
        '<div class="sum-asst-harness-mcp" data-ui-part="assistant.harness.mcp-config">' +
        '<p class="sg-op-card-note sg-op-card-note--tight">No MCP servers in operator store. Add rows via <code class="sum-mono-id">PUT /api/ui/mcp/servers/{id}</code> or keep using <code class="sum-mono-id">chimera.yaml</code> until the first server row exists.</p></div>'
      );
    }
    var rows = "";
    for (var si = 0; si < servers.length; si++) {
      var srv = servers[si] || {};
      var sid = String(srv.server_id || "");
      var state = String(srv.state || "stopped");
      var err = String(srv.error || "").trim();
      var pillVariant = state === "running" ? "ok" : state === "error" ? "down" : "unknown";
      var pillLabel = state === "error" && err ? state + ": " + err : state;
      var bindOn = mcpBindingEnabled(bindings, sid);
      var tools = Array.isArray(cache.toolsByServer && cache.toolsByServer[sid]) ? cache.toolsByServer[sid] : [];
      var toolRows = "";
      for (var ti = 0; ti < tools.length; ti++) {
        var tname = String(tools[ti] || "");
        if (!tname) continue;
        var tid = pfx + "mcp-tool-" + sid.replace(/[^a-z0-9_-]/gi, "-") + "-" + tname.replace(/[^a-z0-9_-]/gi, "-");
        toolRows +=
          '<label class="sg-op-label sg-op-label--inline"><input type="checkbox" id="' +
          escapeHtml(tid) +
          '" data-mcp-server="' +
          escapeHtml(sid) +
          '" data-mcp-tool="' +
          escapeHtml(tname) +
          '"' +
          (mcpToolEnabled(bindings, sid, tname) ? " checked" : "") +
          (bindOn ? "" : " disabled") +
          " /> " +
          escapeHtml(tname) +
          "</label>";
      }
      rows +=
        "<tr><td><strong>" +
        escapeHtml(sid) +
        '</strong><div class="sg-op-card-note sg-op-card-note--tight">' +
        escapeHtml(String(srv.transport || "stdio")) +
        (srv.disabled ? " · disabled" : "") +
        "</div></td><td>" +
        (sgOpHealthPillHtml
          ? sgOpHealthPillHtml(pillLabel, pillVariant, err ? { title: err } : undefined)
          : escapeHtml(pillLabel)) +
        '</td><td><label class="sg-op-label sg-op-label--inline"><input type="checkbox" class="sum-asst-mcp-bind" id="' +
        escapeHtml(pfx + "mcp-bind-" + sid) +
        '" data-mcp-server="' +
        escapeHtml(sid) +
        '"' +
        (bindOn ? " checked" : "") +
        " /> Bind</label></td></tr>" +
        (toolRows ? '<tr><td colspan="3"><div class="sum-asst-mcp-tools">' + toolRows + "</div></td></tr>" : "");
    }
    return (
      '<div class="sum-asst-harness-mcp" data-ui-part="assistant.harness.mcp-config">' +
      '<p class="sg-op-card-note">MCP sidecars for this assistant (operator SQLite overrides YAML when server rows exist). Catalog rebuilds on the next harness turn after save.</p>' +
      '<div class="sum-metrics-table-wrap"><table class="sum-metrics-table"><thead><tr><th>Server</th><th>Health</th><th>Bind</th></tr></thead><tbody>' +
      rows +
      '</tbody></table></div><button type="button" class="sg-op-btn sg-op-btn--ghost" data-admin-action="vm-mcp-bindings-save" data-vm-id="' +
      escapeHtml(String(vmId)) +
      '">Keep MCP bindings</button></div>'
    );
  }

  function escalationKnobsHtml(module, pfx, vmId) {
    var cfg = retrievalConfigFromModule(module);
    var actions = Array.isArray(cfg.on_fail) ? cfg.on_fail.join(", ") : "";
    var maxRounds = cfg.max_rounds != null ? cfg.max_rounds : 2;
    return (
      '<div class="sum-asst-harness-escalation" data-ui-part="assistant.harness.escalation-config">' +
      '<label class="sg-op-label">Max rounds<input id="' + pfx + 'escalation-max-rounds" class="sg-op-input" type="number" min="0" max="2" step="1" value="' + escapeHtml(String(maxRounds)) + '" /></label>' +
      '<label class="sg-op-label">On failure (comma separated)<input id="' + pfx + 'escalation-on-fail" class="sg-op-input" type="text" value="' + escapeHtml(actions) + '" placeholder="re_retrieve, fallback_chain" /></label>' +
      '<label class="sg-op-label">Human surfaces (one Name | URL per line)<textarea id="' + pfx + 'escalation-human-surfaces" class="sg-op-input" rows="3" placeholder="Support desk | https://example.test/escalate">' + escapeHtml((Array.isArray(cfg.human_surfaces) ? cfg.human_surfaces : []).map(function (s) { return String(s.name || "") + " | " + String(s.url || ""); }).join("\n")) + '</textarea></label>' +
      '<label class="sg-op-label">Privacy disclosure<textarea id="' + pfx + 'escalation-privacy-disclosure" class="sg-op-input" rows="2">' + escapeHtml(String(cfg.privacy_disclosure || "")) + '</textarea></label>' +
      '<label class="sg-op-label">Paste-back delimiter<input id="' + pfx + 'escalation-paste-back-delimiter" class="sg-op-input" type="text" value="' + escapeHtml(String(cfg.paste_back_delimiter || "<<<CHIMERA_HUMAN_ANSWER>>>")) + '" /></label>' +
      '<button type="button" class="sg-op-btn sg-op-btn--ghost" data-admin-action="vm-harness-escalation-save" data-vm-id="' + escapeHtml(String(vmId)) + '">Keep escalation settings</button></div>'
    );
  }

  function buildHarnessSection(vm, ui, pfx, loading) {
    var modules = Array.isArray(vm.harness_modules) ? vm.harness_modules : [];
    var body = "";
    if (loading) {
      body = '<p class="muted">Loading harness modules…</p>';
    } else if (!modules.length) {
      body = '<p class="muted">No harness modules loaded.</p>';
    } else {
      var rows = "";
      for (var i = 0; i < modules.length; i++) {
        var m = modules[i] || {};
        var mid = String(m.module_id || "");
        var locked = m.configurable === false;
        var en = !!m.enabled && !locked;
        var reason = String(m.disabled_reason || "").trim();
        var toggleId = pfx + "harness-" + mid.replace(/[^a-z0-9_-]/gi, "-");
        var toggleHtml = locked
          ? '<button class="sum-router-toggle" type="button" id="' +
            escapeHtml(toggleId) +
            '" aria-pressed="false" disabled aria-disabled="true" title="' +
            escapeHtml(reason || "Not available yet") +
            '"><span class="sum-router-toggle__track"><span class="sum-router-toggle__thumb"></span></span></button>'
          : vmRouterToggleHtml(
              toggleId,
              "vm-harness-module-toggle",
              vm.id,
              en,
              "Toggle " + harnessModuleLabel(mid)
            ).replace(
              'data-admin-action="vm-harness-module-toggle"',
              'data-admin-action="vm-harness-module-toggle" data-harness-module="' + escapeHtml(mid) + '"'
            );
        var stateLabel = en ? "On" : "Off";
        var retrievalKnobs = mid === "retrieval" && en ? retrievalKnobsHtml(m, pfx, vm.id) : "";
        var intentKnobs = mid === "intent" && en ? intentKnobsHtml(m, pfx, vm.id) : "";
        var evaluatorKnobs = mid === "evaluator" && en ? evaluatorKnobsHtml(m, pfx, vm.id) : "";
        var escalationKnobs = mid === "escalation" && en ? escalationKnobsHtml(m, pfx, vm.id) : "";
        var mcpKnobs = mid === "tool_executor" && en ? toolExecutorMcpKnobsHtml(pfx, vm.id) : "";
        rows +=
          "<tr data-harness-module-row=\"" +
          escapeHtml(mid) +
          '">' +
          "<td><strong>" +
          escapeHtml(harnessModuleLabel(mid)) +
          '</strong><div class="sg-op-card-note sg-op-card-note--tight">' +
          escapeHtml(harnessModuleDesc(mid)) +
          (reason ? " — " + escapeHtml(reason) : "") +
          "</div></td>" +
          '<td class="sum-asst-harness-toggle-cell">' +
          '<span class="sum-asst-hdr-toggle">' +
          toggleHtml +
          '<span class="sum-asst-hdr-toggle-state muted">' +
          escapeHtml(stateLabel) +
          "</span></span></td></tr>" +
          ((retrievalKnobs || intentKnobs || evaluatorKnobs || escalationKnobs || mcpKnobs) ? '<tr><td colspan="2">' + retrievalKnobs + intentKnobs + evaluatorKnobs + escalationKnobs + mcpKnobs + "</td></tr>" : "");
      }
      body =
        '<div class="sum-metrics-table-wrap"><table class="sum-metrics-table sum-asst-harness-table"><thead><tr><th>Module</th><th class="num">Enabled</th></tr></thead><tbody>' +
        rows +
        "</tbody></table></div>";
    }
    return (
      '<details class="sum-asst-section" data-vm-section="harness" data-ui-part="assistant.harness"' +
      vmSectionOpenAttr(ui, "harness") +
      ">" +
      vmSectionHeaderHtml("Harness", {
        desc: "Turn-harness modules for this assistant. Toggle independently of the routing stack above."
      }) +
      '<div class="sum-asst-section__body">' +
      body +
      "</div></details>"
    );
  }

  function buildAssistantCardHtml(vmSummary) {
    vmSummary = vmSummary || {};
    var rowId = String(vmSummary.id != null ? vmSummary.id : "");
    var detail = vmDetail(ctx, rowId);
    var ui = vmUi(ctx, rowId);
    var vm = mergeVm(vmSummary, detail);
    var loading = !!ui.detailLoading && !detail;
    var gw = (ctx.adminStateCache && ctx.adminStateCache.gateway) || {};
    var pfx = vmIdPrefix(rowId);
    var modelId = String(vm.model_id || "").trim() || "—";
    var title = vmCardTitleText(vm);
    var subtitle = vmCardSubtitleText(vm);

    var sections =
      buildClientUsageBlock(vm, pfx, loading) +
      buildIdentitySection(vm, ui, pfx, loading) +
      buildFallbackSection(vm, ui, pfx, gw, loading) +
      buildRoutingSection(vm, ui, pfx, gw, loading) +
      buildToolRouterSection(vm, ui, pfx, loading) +
      buildHarnessSection(vm, ui, pfx, loading) +
      adminScopedEvlogPanelFromEvents(
        "Scoped log — " + modelId,
        "vm-" + rowId + "-routing",
        adminScopedEventsForAssistant(modelId),
        { uiPart: "assistant.scoped-evlog" }
      );

    return (
      '<details class="sum-card sum-card--assistant" id="assistant-' +
      escapeHtml(rowId) +
      '" data-assistant-id="' +
      escapeHtml(rowId) +
      '">' +
      '<summary data-ui-part="assistant.summary">' +
      '<span class="sum-avatar sum-av-svc-chimera-gateway">As</span>' +
      '<span class="sum-main"><span class="sum-title">' +
      escapeHtml(title) +
      "</span>" +
      '<span class="sum-sub sum-sub--clamp">' +
      escapeHtml(subtitle) +
      "</span></span>" +
      '<span class="sum-metrics">' +
      vmMetricsHtml(vm) +
      "</span>" +
      operatorCardChevronHtml() +
      "</summary>" +
      '<div class="sum-body sum-body--assistant">' +
      sections +
      "</div></details>"
    );
  }

  function buildAssistantsSectionIntroHtml(count) {
    return (
      '<div class="sum-workspaces-intro">' +
      '<p class="sum-workspaces-intro-lead">Operator-managed assistants (<strong>' +
      escapeHtml(String(count)) +
      "</strong>).Define assistants with different routing policies and strategies.</p>" +
      "</div>"
    );
  }

  function draftModelIdPreview(draft) {
    draft = draft || {};
    var custom = String(draft.model_id != null ? draft.model_id : "").trim();
    if (custom) return custom;
    var name = String(draft.name != null ? draft.name : "").trim();
    var version = String(draft.version != null ? draft.version : "").trim();
    if (name && version) return name + "-" + version;
    if (name) return name;
    return "—";
  }

  function syncAssistantDraftCardChrome(cardEl, draft) {
    if (!cardEl || !draft) return;
    var preview = draftModelIdPreview(draft);
    var codeEl = cardEl.querySelector(".sum-main--workspace-draft .sum-mono-id");
    if (codeEl) codeEl.textContent = preview;
    var saveBtn = cardEl.querySelector('[data-admin-action="vm-draft-save"]');
    if (saveBtn) {
      if (draft.saving) saveBtn.setAttribute("disabled", "");
      else saveBtn.removeAttribute("disabled");
    }
    var msg = draft.msg ? String(draft.msg) : "";
    var hint = cardEl.querySelector(".ws-draft-hint");
    if (msg) {
      if (hint) {
        hint.textContent = msg;
      } else {
        var body = cardEl.querySelector(".sum-body");
        if (body) {
          var p = document.createElement("p");
          p.className = "muted ws-draft-hint";
          p.textContent = msg;
          body.appendChild(p);
        }
      }
    } else if (hint) {
      hint.remove();
    }
  }

  function buildAssistantDraftCardHtml(draft) {
    draft = draft || {};
    var draftId = String(draft.id != null ? draft.id : "");
    var name = String(draft.name != null ? draft.name : "");
    var version = String(draft.version != null ? draft.version : "");
    var desc = String(draft.description != null ? draft.description : "");
    var modelId = String(draft.model_id != null ? draft.model_id : "");
    var preview = draftModelIdPreview(draft);
    var msg = draft.msg ? String(draft.msg) : "";
    return (
      '<article class="sum-card sum-card--assistant-draft sum-card--workspace-draft" id="assistant-draft-' +
      escapeHtml(draftId) +
      '" data-assistant-draft="' +
      escapeHtml(draftId) +
      '">' +
      '<header class="sum-card__workspace-draft-hdr">' +
      '<span class="sum-avatar sum-av-svc-chimera-gateway">As</span>' +
      '<span class="sum-main sum-main--workspace-draft"><span class="sum-title">New assistant</span>' +
      '<span class="sum-sub sum-sub--clamp muted">Client model id: <code class="sum-mono-id">' +
      escapeHtml(preview) +
      "</code></span></span>" +
      '<span class="ws-draft-actions">' +
      '<button type="button" class="sg-op-btn sg-op-btn--ghost ws-draft-btn-cancel" data-admin-action="vm-draft-cancel" data-vm-draft-id="' +
      escapeHtml(draftId) +
      '">Cancel</button>' +
      '<button type="button" class="sg-op-btn ws-draft-btn-save" data-admin-action="vm-draft-save" data-vm-draft-id="' +
      escapeHtml(draftId) +
      '"' +
      (draft.saving ? " disabled" : "") +
      ">Save</button></span></header>" +
      '<div class="sum-body sum-body--assistant" data-ui-part="assistant-draft.form">' +
      '<div class="sg-op-card-note sg-op-card-note--tight">Save to create the assistant, then configure fallback (required), routing, and tool-router on the new card.</div>' +
      '<div class="ws-draft-fields">' +
      '<div class="ws-draft-field"><label class="ws-draft-field-label">Name</label>' +
      '<input class="ws-draft-input" data-vm-draft-field="name" data-vm-draft-id="' +
      escapeHtml(draftId) +
      '" type="text" value="' +
      escapeHtml(name) +
      '" placeholder="e.g. Chimera" /></div>' +
      '<div class="ws-draft-field"><label class="ws-draft-field-label">Version</label>' +
      '<input class="ws-draft-input" data-vm-draft-field="version" data-vm-draft-id="' +
      escapeHtml(draftId) +
      '" type="text" value="' +
      escapeHtml(version) +
      '" placeholder="e.g. 0.3.0" /></div>' +
      '<div class="ws-draft-field"><label class="ws-draft-field-label">Model id (optional)</label>' +
      '<input class="ws-draft-input" data-vm-draft-field="model_id" data-vm-draft-id="' +
      escapeHtml(draftId) +
      '" type="text" value="' +
      escapeHtml(modelId) +
      '" placeholder="Defaults to name-version" /></div>' +
      '<div class="ws-draft-field ws-draft-field--wide"><label class="ws-draft-field-label">Description</label>' +
      '<textarea class="ws-draft-input" data-vm-draft-field="description" data-vm-draft-id="' +
      escapeHtml(draftId) +
      '" rows="3" spellcheck="false" placeholder="Short summary for operators">' +
      escapeHtml(desc) +
      "</textarea></div>" +
      "</div>" +
      (msg ? '<p class="muted ws-draft-hint">' + escapeHtml(msg) + "</p>" : "") +
      "</div></article>"
    );
  }

  function buildAssistantsSectionBreakHtml(count) {
    var intro = buildAssistantsSectionIntroHtml(count);
    if (typeof ctx.operatorSectionHeadHtml !== "function") {
      return (
        '<div class="sum-section-label sum-feed-section-title">Assistants</div>' +
        intro
      );
    }
    var addBtn = "";
    if (typeof ctx.operatorSectionAddBtn === "function") {
      var hasDraft = ctx.assistantDrafts && ctx.assistantDrafts.length > 0;
      addBtn = ctx.operatorSectionAddBtn(
        { "data-admin-action": "vm-add" },
        "Add assistant",
        hasDraft
          ? { disabled: true, title: "Finish or cancel the draft assistant first" }
          : { title: "Create a new assistant" }
      );
    }
    return (
      '<div class="sg-op-assistants-section" id="sg-op-assistants-section">' +
      ctx.operatorSectionHeadHtml("Assistants", "network_intel_node", {
        actionHtml: addBtn
      }) +
      intro +
      "</div>"
    );
  }

  ctx.buildAssistantCardHtml = buildAssistantCardHtml;
  ctx.buildAssistantDraftCardHtml = buildAssistantDraftCardHtml;
  ctx.buildAssistantsSectionIntroHtml = buildAssistantsSectionIntroHtml;
  ctx.buildAssistantsSectionBreakHtml = buildAssistantsSectionBreakHtml;
  ctx.adminScopedEventsForAssistant = adminScopedEventsForAssistant;
  ctx.vmCardTitleText = vmCardTitleText;
  ctx.vmCardSubtitleText = vmCardSubtitleText;
  ctx.syncAssistantCardHeader = syncAssistantCardHeader;
  ctx.syncAssistantDraftCardChrome = syncAssistantDraftCardChrome;
  ctx.mergeAssistantForRender = function (summary, id) {
    return mergeVm(summary, vmDetail(ctx, id));
  };
};
