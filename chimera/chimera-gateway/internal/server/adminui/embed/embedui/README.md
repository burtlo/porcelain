# Operator embed UI (`embedui/`)

Static HTML/CSS/JS served by the gateway admin UI (`adminui/embed`). Session auth applies to all routes except login/setup and shared primitive assets.

## Pages

| HTTP route | File | Role |
|------------|------|------|
| `GET /ui` | `index.html` | App shell (iframe: chat default, settings via ribbon) |
| `GET /ui/chat` | `chat.html` | Chat |
| `GET /ui/settings` | `settings.html` | Operator settings + summarized log feed |
| `GET /ui/settings/gallery` | `settings/gallery.html` | Component gallery (styleguide) |
| `GET /ui/login` | `login.html` | Login (registered in `api/auth`) |
| `GET /ui/setup` | `setup.html` | First-run setup (bootstrap mux) |

Legacy routes (`/ui/logs`, `/ui/desktop`, `/ui/gallery`, …) are **not** registered.

## Asset URL map

| URL prefix | Directory | Notes |
|------------|-----------|--------|
| `/ui/assets/fonts.css` | `fonts.css` | Local `@font-face` (Hanken + Material Symbols) |
| `/ui/assets/fonts/**` | `fonts/**` | Generated `.woff2`/`.ttf` subsets + `icons.txt` + `codepoints.js` |
| `/ui/assets/settings.css` | `settings.css` | Style composition entry |
| `/ui/assets/settings.js` | `settings_entry.js` | Boots `ChimeraSettings.Main()` |
| `/ui/assets/settings/main.js` | `settings_app.js` | Main app IIFE |
| `/ui/assets/settings/**` | `settings/**` | Modules, generated `contracts.js` / `operator_copy.js` |
| `/ui/assets/gallery/**` | `gallery/**` | Gallery-only CSS/JS (not under `settings/`) |
| `/ui/assets/styles/**` | `styles/**` | Shared layout tokens used by settings + gallery |
| `/ui/assets/ui/**` | `ui/**` | `ChimeraUI` primitives |
| `/ui/assets/shared/**` | `shared/**` | `ChimeraShared` admin primitives (settings + wizard) |
| `/ui/assets/theme-tokens.css` | `theme-tokens.css` | Design tokens |
| `/ui/assets/ui.css` | `ui.css` | Shared primitives (login/setup too) |

## JavaScript layout

- **`globalThis.ChimeraSettings`** — settings app (`settings/` modules). Log stream uses **`/api/ui/logs`** (API name unchanged).
- **`globalThis.ChimeraUI`** — shared presentation components (`ui/components/`).
- **`globalThis.ChimeraShared`** — operator admin primitives (`shared/`; credentials, status, configure, scoped evlog).
- **Codegen:** `settings/contracts.js` ← `go run ./internal/naming/cmd/gencontracts`; `settings/operator_copy.js` ← `go run ./internal/operatorcopy/cmd/genjs`.

## Directory guide

```
embedui/
  index.html, chat.html, settings.html   # top-level pages
  settings_entry.js, settings_app.js     # served as settings.js + settings/main.js
  settings.css                           # @imports styles/*
  settings/                              # ChimeraSettings modules (see settings/README.md)
  gallery/                               # gallery static assets only
  settings/gallery.html                  # gallery page HTML
  styles/                                # CSS building blocks
  shared/                                # ChimeraShared (see shared/README.md)
  ui/                                    # ChimeraUI components
  fonts/                                 # icons.txt (manual) + committed woff2/ttf + codepoints.js
  fonts.css                              # @font-face entry (generated)
  scripts/                               # maintainer tools (not served)
```

## Local fonts (offline / restricted networks)

Operator UI does **not** load Google Fonts at runtime. Subsetted **Hanken Grotesk** and **Material Symbols Outlined** ship under `fonts/` and are embedded in the gateway binary.

**Icon source of truth:** hand-maintained [`fonts/icons.txt`](fonts/icons.txt) (one Material Symbols name per line).

| Task | Purpose |
|------|---------|
| Edit `fonts/icons.txt` | Add/remove icon names you use in the UI |
| `make adminui-fonts` | Fetch sources (if needed) + rebuild subsets + `codepoints.js` |
| `make adminui-fonts-check` | Verify generated assets match `icons.txt` |
| `make adminui-fonts-fetch` | Download upstream TTFs into `chimera/.deps` only |

**Not a hard dependency for builds:** committed `fonts/*.{woff2,ttf}` are enough for `make chimera-build`. Regeneration needs either sibling clones (`../material-design-icons`, `../fonts`), `make adminui-fonts-fetch`, or `ADMINUI_MATERIAL_ICONS_SRC` / `ADMINUI_FONTS_SRC`.

Dynamic icon helpers call `ChimeraMaterialIcons.char(name)` from `fonts/codepoints.js` so the desktop webview does not rely on OpenType ligatures.

## Local iteration

Set `CHIMERA_ADMINUI_ROOT` to `chimera/chimera-gateway/internal/server/adminui/embed` (loopback listen only) to serve files from disk without rebuilding.
