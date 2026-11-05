# Gateway HTTP routes (generated)

DO NOT EDIT; run: `make contracts-generate` or `go run ./internal/docgen/routes/cmd`.

Auth values: **public** (no gateway token), **bearer** (gateway API token), **session** (operator UI cookie), **tenant** (session + tenant scope for RAG).

| Method | Path | Auth | Source | Notes |
|--------|------|------|--------|-------|
| GET | `/api/ui/assistants` | session | `chimera\chimera-gateway\internal\server\adminui\api\assistants\register.go:14` | operator UI (ui != nil) |
| POST | `/api/ui/assistants` | session | `chimera\chimera-gateway\internal\server\adminui\api\assistants\register.go:17` | operator UI (ui != nil) |
| GET | `/api/ui/assistants/{id}` | session | `chimera\chimera-gateway\internal\server\adminui\api\assistants\register.go:20` | operator UI (ui != nil) |
| PUT | `/api/ui/assistants/{id}` | session | `chimera\chimera-gateway\internal\server\adminui\api\assistants\register.go:23` | operator UI (ui != nil) |
| DELETE | `/api/ui/assistants/{id}` | session | `chimera\chimera-gateway\internal\server\adminui\api\assistants\register.go:26` | operator UI (ui != nil) |
| PUT | `/api/ui/assistants/{id}/fallback` | session | `chimera\chimera-gateway\internal\server\adminui\api\assistants\register.go:29` | operator UI (ui != nil) |
| PUT | `/api/ui/assistants/{id}/routing-policy` | session | `chimera\chimera-gateway\internal\server\adminui\api\assistants\register.go:32` | operator UI (ui != nil) |
| PUT | `/api/ui/assistants/{id}/tool-router` | session | `chimera\chimera-gateway\internal\server\adminui\api\assistants\register.go:35` | operator UI (ui != nil) |
| GET | `/api/ui/assistants/{id}/harness` | session | `chimera\chimera-gateway\internal\server\adminui\api\assistants\register.go:38` | operator UI (ui != nil) |
| PUT | `/api/ui/assistants/{id}/harness` | session | `chimera\chimera-gateway\internal\server\adminui\api\assistants\register.go:41` | operator UI (ui != nil) |
| POST | `/api/ui/assistants/{id}/routing/generate` | session | `chimera\chimera-gateway\internal\server\adminui\api\assistants\register.go:44` | operator UI (ui != nil) |
| POST | `/api/ui/assistants/{id}/routing/evaluate` | session | `chimera\chimera-gateway\internal\server\adminui\api\assistants\register.go:47` | operator UI (ui != nil) |
| POST | `/api/ui/assistants/{id}/harness/evaluate` | session | `chimera\chimera-gateway\internal\server\adminui\api\assistants\register.go:50` | operator UI (ui != nil) |
| GET | `/ui/login` | public | `chimera\chimera-gateway\internal\server\adminui\api\auth\register.go:14` | operator UI (ui != nil) |
| POST | `/api/ui/login` | public | `chimera\chimera-gateway\internal\server\adminui\api\auth\register.go:17` | operator UI (ui != nil) |
| GET | `/api/ui/conversations` | session | `chimera\chimera-gateway\internal\server\adminui\api\conversations\register.go:14` | operator UI (ui != nil) |
| GET | `/api/ui/conversations/{conversation_id}` | session | `chimera\chimera-gateway\internal\server\adminui\api\conversations\register.go:17` | operator UI (ui != nil) |
| PATCH | `/api/ui/conversations/{conversation_id}` | session | `chimera\chimera-gateway\internal\server\adminui\api\conversations\register.go:20` | operator UI (ui != nil) |
| POST | `/api/ui/conversations/{conversation_id}/flag` | session | `chimera\chimera-gateway\internal\server\adminui\api\conversations\register.go:23` | operator UI (ui != nil) |
| DELETE | `/api/ui/conversations/{conversation_id}` | session | `chimera\chimera-gateway\internal\server\adminui\api\conversations\register.go:26` | operator UI (ui != nil) |
| GET | `/api/ui/indexer/config` | session | `chimera\chimera-gateway\internal\server\adminui\api\indexer\register.go:14` | operator UI (ui != nil) |
| PUT | `/api/ui/indexer/config` | session | `chimera\chimera-gateway\internal\server\adminui\api\indexer\register.go:17` | operator UI (ui != nil) |
| GET | `/api/ui/indexer/workspaces` | session | `chimera\chimera-gateway\internal\server\adminui\api\indexer\register.go:20` | operator UI (ui != nil) |
| POST | `/api/ui/indexer/workspaces` | session | `chimera\chimera-gateway\internal\server\adminui\api\indexer\register.go:23` | operator UI (ui != nil) |
| PUT | `/api/ui/indexer/workspaces/{id}` | session | `chimera\chimera-gateway\internal\server\adminui\api\indexer\register.go:26` | operator UI (ui != nil) |
| DELETE | `/api/ui/indexer/workspaces/{id}` | session | `chimera\chimera-gateway\internal\server\adminui\api\indexer\register.go:29` | operator UI (ui != nil) |
| POST | `/api/ui/indexer/workspaces/{id}/paths` | session | `chimera\chimera-gateway\internal\server\adminui\api\indexer\register.go:32` | operator UI (ui != nil) |
| PUT | `/api/ui/indexer/workspace-paths/{pathid}` | session | `chimera\chimera-gateway\internal\server\adminui\api\indexer\register.go:35` | operator UI (ui != nil) |
| DELETE | `/api/ui/indexer/workspace-paths/{pathid}` | session | `chimera\chimera-gateway\internal\server\adminui\api\indexer\register.go:38` | operator UI (ui != nil) |
| POST | `/api/ui/indexer/workspaces/{id}/reindex` | session | `chimera\chimera-gateway\internal\server\adminui\api\indexer\register.go:41` | operator UI (ui != nil) |
| POST | `/api/ui/indexer/reindex-all` | session | `chimera\chimera-gateway\internal\server\adminui\api\indexer\register.go:44` | operator UI (ui != nil) |
| GET | `/api/ui/indexer/corpus/stale` | session | `chimera\chimera-gateway\internal\server\adminui\api\indexer\register.go:47` | operator UI (ui != nil) |
| GET | `/api/ui/logs` | session | `chimera\chimera-gateway\internal\server\adminui\api\logs\register.go:14` | operator UI (ui != nil) |
| GET | `/api/ui/logs/stream` | session | `chimera\chimera-gateway\internal\server\adminui\api\logs\register.go:17` | operator UI (ui != nil) |
| GET | `/api/ui/metrics` | session | `chimera\chimera-gateway\internal\server\adminui\api\metrics\register.go:14` | operator UI (ui != nil) |
| GET | `/api/ui/chimera-broker/providers` | session | `chimera\chimera-gateway\internal\server\adminui\api\providers\register.go:14` | operator UI (ui != nil) |
| GET | `/api/ui/providers/catalog` | session | `chimera\chimera-gateway\internal\server\adminui\api\providers\register.go:17` | operator UI (ui != nil) |
| GET | `/api/ui/providers/{provider_id}/models` | session | `chimera\chimera-gateway\internal\server\adminui\api\providers\register.go:20` | operator UI (ui != nil) |
| PUT | `/api/ui/providers/{provider_id}/models` | session | `chimera\chimera-gateway\internal\server\adminui\api\providers\register.go:23` | operator UI (ui != nil) |
| POST | `/api/ui/providers/{provider_id}/models/apply-free-tier` | session | `chimera\chimera-gateway\internal\server\adminui\api\providers\register.go:26` | operator UI (ui != nil) |
| POST | `/api/ui/rag/search` | tenant | `chimera\chimera-gateway\internal\server\adminui\api\rag\register.go:14` | operator UI (ui != nil) |
| GET | `/api/ui/rag/embedding` | session | `chimera\chimera-gateway\internal\server\adminui\api\rag\register.go:17` | operator UI (ui != nil) |
| PUT | `/api/ui/rag/embedding` | session | `chimera\chimera-gateway\internal\server\adminui\api\rag\register.go:20` | operator UI (ui != nil) |
| POST | `/api/ui/provider/{provider}/keys` | session | `chimera\chimera-gateway\internal\server\adminui\api\save\register.go:17` | operator UI (ui != nil) |
| POST | `/api/ui/provider/{provider}/keys/delete` | session | `chimera\chimera-gateway\internal\server\adminui\api\save\register.go:18` | operator UI (ui != nil) |
| POST | `/api/ui/provider/ollama/base_url` | session | `chimera\chimera-gateway\internal\server\adminui\api\save\register.go:20` | operator UI (ui != nil) |
| POST | `/api/ui/logout` | session | `chimera\chimera-gateway\internal\server\adminui\api\save\register.go:23` | operator UI (ui != nil) |
| GET | `/api/ui/state` | session | `chimera\chimera-gateway\internal\server\adminui\api\state\register.go:14` | operator UI (ui != nil) |
| GET | `/api/ui/tokens` | session | `chimera\chimera-gateway\internal\server\adminui\api\tokens\register.go:14` | operator UI (ui != nil) |
| POST | `/api/ui/tokens` | session | `chimera\chimera-gateway\internal\server\adminui\api\tokens\register.go:17` | operator UI (ui != nil) |
| POST | `/api/ui/tokens/delete` | session | `chimera\chimera-gateway\internal\server\adminui\api\tokens\register.go:20` | operator UI (ui != nil) |
| GET | `/ui` | public | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:15` | operator UI (ui != nil) |
| GET | `/ui/chat` | session | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:27` | operator UI (ui != nil) |
| GET | `/ui/search` | session | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:28` | operator UI (ui != nil) |
| GET | `/ui/settings` | session | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:29` | operator UI (ui != nil) |
| GET | `/ui/settings/gallery` | session | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:30` | operator UI (ui != nil) |
| GET | `/ui/assets/ui.css` | public | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:33` | operator UI (ui != nil) |
| GET | `/ui/assets/theme-tokens.css` | public | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:34` | operator UI (ui != nil) |
| GET | `/ui/assets/fonts.css` | public | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:35` | operator UI (ui != nil) |
| GET | `/ui/assets/embed-theme.js` | public | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:36` | operator UI (ui != nil) |
| GET | `/ui/assets/fonts/` | public | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:38` | operator UI (ui != nil) |
| GET | `/ui/assets/settings.css` | session | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:40` | operator UI (ui != nil) |
| GET | `/ui/assets/styles/` | session | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:41` | operator UI (ui != nil) |
| GET | `/ui/assets/ui/` | session | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:42` | operator UI (ui != nil) |
| GET | `/ui/assets/shared/` | session | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:43` | operator UI (ui != nil) |
| GET | `/ui/assets/settings.js` | session | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:44` | operator UI (ui != nil) |
| GET | `/ui/assets/settings/main.js` | session | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:45` | operator UI (ui != nil) |
| GET | `/ui/assets/settings/` | session | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:46` | operator UI (ui != nil) |
| GET | `/ui/assets/gallery/` | session | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:47` | operator UI (ui != nil) |
| GET | `/ui/assets/chat/` | session | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:48` | operator UI (ui != nil) |
| GET | `/ui/assets/search/` | session | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:49` | operator UI (ui != nil) |
| GET | `/ui/assets/shell/` | session | `chimera\chimera-gateway\internal\server\adminui\embed\routes.go:50` | operator UI (ui != nil) |
| GET | `/` | public | `chimera\chimera-gateway\internal\server\server.go:145` | redirects to /ui when operator UI enabled |
| GET | `/healthz` | public | `chimera\chimera-gateway\internal\server\server.go:266` |  |
| GET | `/health` | public | `chimera\chimera-gateway\internal\server\server.go:275` |  |
| GET | `/status` | public | `chimera\chimera-gateway\internal\server\server.go:322` |  |
| GET | `/ui/models` | public | `chimera\chimera-gateway\internal\server\server.go:326` | merged models without bearer (browser) |
| GET | `/v1/models` | bearer | `chimera\chimera-gateway\internal\server\server.go:336` |  |
| POST | `/v1/chat/completions` | bearer | `chimera\chimera-gateway\internal\server\server.go:344` |  |
| POST | `/v1/ingest` | bearer | `chimera\chimera-gateway\internal\server\server.go:352` |  |
| GET | `/v1/ingest/session/` | bearer | `chimera\chimera-gateway\internal\server\server.go:359` |  |
| GET | `/v1/ingest/session` | bearer | `chimera\chimera-gateway\internal\server\server.go:362` |  |
| GET | `/v1/indexer/config` | bearer | `chimera\chimera-gateway\internal\server\server.go:366` |  |
| GET | `/v1/indexer/workspaces` | bearer | `chimera\chimera-gateway\internal\server\server.go:373` |  |
| GET | `/v1/indexer/storage/health` | bearer | `chimera\chimera-gateway\internal\server\server.go:380` |  |
| GET | `/v1/indexer/storage/stats` | bearer | `chimera\chimera-gateway\internal\server\server.go:387` |  |
| GET | `/v1/indexer/corpus/inventory` | bearer | `chimera\chimera-gateway\internal\server\server.go:394` |  |
| GET,PUT | `/v1/indexer/corpus/stale` | bearer | `chimera\chimera-gateway\internal\server\server.go:401` |  |
| * | `/v1/rag/segments` | bearer | `chimera\chimera-gateway\internal\server\server.go:412` |  |
| * | `/v1/rag/context` | bearer | `chimera\chimera-gateway\internal\server\server.go:415` |  |
| * | `/v1/rag/adjacent` | bearer | `chimera\chimera-gateway\internal\server\server.go:418` |  |
| * | `/v1/rag/tools` | bearer | `chimera\chimera-gateway\internal\server\server.go:421` |  |
| GET | `/` | public | `chimera\chimera-gateway\internal\server\ui_bootstrap.go:22` | UI bootstrap mux (pre-operator SQLite) |
| GET | `/ui` | public | `chimera\chimera-gateway\internal\server\ui_bootstrap.go:26` | UI bootstrap mux (pre-operator SQLite) |
| GET | `/ui/setup` | public | `chimera\chimera-gateway\internal\server\ui_bootstrap.go:34` | UI bootstrap mux (pre-operator SQLite) |
| GET | `/ui/assets/embed-theme.js` | public | `chimera\chimera-gateway\internal\server\ui_bootstrap.go:35` | UI bootstrap mux (pre-operator SQLite) |
| GET | `/ui/assets/theme-tokens.css` | public | `chimera\chimera-gateway\internal\server\ui_bootstrap.go:36` | UI bootstrap mux (pre-operator SQLite) |
| GET | `/ui/assets/ui.css` | public | `chimera\chimera-gateway\internal\server\ui_bootstrap.go:37` | UI bootstrap mux (pre-operator SQLite) |
| GET | `/ui/login` | public | `chimera\chimera-gateway\internal\server\ui_bootstrap.go:38` | UI bootstrap mux (pre-operator SQLite) |
| GET | `/ui/panel` | public | `chimera\chimera-gateway\internal\server\ui_bootstrap.go:45` | UI bootstrap mux (pre-operator SQLite) |
| GET | `/healthz` | public | `chimera\chimera-gateway\internal\server\ui_bootstrap.go:53` | UI bootstrap mux (pre-operator SQLite) |
| GET | `/readyz` | public | `chimera\chimera-gateway\internal\server\ui_bootstrap.go:62` | UI bootstrap mux (pre-operator SQLite) |
| GET | `/health` | public | `chimera\chimera-gateway\internal\server\ui_bootstrap.go:71` | UI bootstrap mux (pre-operator SQLite) |
| GET | `/status` | public | `chimera\chimera-gateway\internal\server\ui_bootstrap.go:89` | UI bootstrap mux (pre-operator SQLite) |
| POST | `/api/ui/setup/token` | public | `chimera\chimera-gateway\internal\server\ui_bootstrap.go:93` | UI bootstrap mux (pre-operator SQLite) |
| POST | `/api/ui/setup/complete` | public | `chimera\chimera-gateway\internal\server\ui_bootstrap.go:94` | UI bootstrap mux (pre-operator SQLite) |
| * | `/healthz` | public | `chimera\chimera-supervisor\internal\control\http.go:107` | chimera-supervisor control plane |
| * | `/readyz` | public | `chimera\chimera-supervisor\internal\control\http.go:113` | chimera-supervisor control plane |
| * | `/status` | public | `chimera\chimera-supervisor\internal\control\http.go:127` | chimera-supervisor control plane |
| POST | `/shutdown` | public | `chimera\chimera-supervisor\internal\control\http.go:189` | chimera-supervisor control plane |
| * | `/metrics` | public | `chimera\chimera-supervisor\internal\control\http.go:208` | chimera-supervisor control plane |

