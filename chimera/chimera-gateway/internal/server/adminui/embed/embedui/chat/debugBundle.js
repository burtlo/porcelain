/**
 * Formats a paste-friendly troubleshooting bundle for operator chat turns.
 */
(function () {
  "use strict";

  function priorUserText(messages, assistantMsg) {
    if (!messages || !assistantMsg) return "";
    for (var i = 0; i < messages.length; i++) {
      if (messages[i].id !== assistantMsg.id) continue;
      for (var j = i - 1; j >= 0; j--) {
        if (messages[j].role === "user") return String(messages[j].content || "").trim();
      }
      break;
    }
    return "";
  }

  function workspaceLine(ctx) {
    if (!ctx || !ctx.workspace) return "(none — chat workspace selector Default)";
    var ws = ctx.workspace;
    var parts = [];
    if (ws.project_id) parts.push("project=" + ws.project_id);
    if (ws.flavor_id) parts.push("flavor=" + ws.flavor_id);
    if (ws.id != null) parts.push("workspace_row_id=" + ws.id);
    return parts.length ? parts.join(", ") : JSON.stringify(ws);
  }

  function toolCallsFromSummary(summary) {
    if (!summary || !summary.execution) return "";
    var calls = summary.execution.tool_calls;
    if (!calls || !calls.length) return "";
    var names = [];
    for (var i = 0; i < calls.length; i++) {
      if (calls[i] && calls[i].name) names.push(String(calls[i].name));
    }
    return names.join(", ");
  }

  function formatTurnDebugBundle(msg, ctx) {
    ctx = ctx || {};
    var summary = msg && msg.harnessSummary;
    var requestId =
      (msg && msg.requestId) ||
      (summary && summary.request_id) ||
      "";
    var conversationId =
      (msg && msg.conversationId) ||
      (ctx.conversationId) ||
      (summary && summary.conversation_id) ||
      "";
    var turnIndex =
      msg && msg.turnIndex != null
        ? msg.turnIndex
        : summary && summary.turn_index != null
          ? summary.turn_index
          : "";
    var lines = [
      "Chimera chat troubleshooting bundle",
      "generated_at: " + new Date().toISOString(),
      "",
      "## Correlation (grep supervisor log)",
      "request_id: " + (requestId || "(missing — retry after gateway rebuild)"),
      "conversation_id: " + (conversationId || "(unknown)"),
      "turn_index: " + (turnIndex !== "" ? String(turnIndex) : "(unknown)"),
      "",
      "## Routing",
      "assistant_id: " +
        ((msg && (msg.selectedModel || msg.upstreamModel)) || ctx.selectedModel || ""),
      "upstream_model: " + (msg && msg.upstreamModel || ""),
      "workspace: " + workspaceLine(ctx),
      "",
      "## User message",
      priorUserText(ctx.messages, msg) || "(unknown)",
      "",
      "## Assistant reply (excerpt)",
      String((msg && msg.content) || "").slice(0, 1200)
    ];
    if (msg && msg.ragHits && msg.ragHits.length) {
      lines.push("", "## RAG hits: " + msg.ragHits.length);
    }
    var tools = toolCallsFromSummary(summary);
    if (tools) {
      lines.push("", "## Harness tool calls (summary): " + tools);
    }
    if (summary) {
      lines.push(
        "",
        "## Harness summary (JSON)",
        "(snapshot from early in the turn; tool_calls may be empty even when tools ran — grep harness.tool.* in supervisor log)",
        JSON.stringify(summary, null, 2)
      );
    }
    lines.push(
      "",
      "## Log hint",
      'grep "' + (requestId || "REQUEST_ID") + '" data/locus-desktop-supervisor.log'
    );
    return lines.join("\n");
  }

  globalThis.ChimeraChat = globalThis.ChimeraChat || {};
  globalThis.ChimeraChat.DebugBundle = {
    formatTurnDebugBundle: formatTurnDebugBundle
  };
})();
