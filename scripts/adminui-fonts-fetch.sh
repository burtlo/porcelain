#!/usr/bin/env bash
# Fetch upstream font sources into chimera/.deps for `make adminui-fonts`.
# Not required for normal builds — committed embedui/fonts/*.woff2 are used instead.
# Prefer sibling clones when present (../material-design-icons, ../fonts).
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ROOT="$REPO_ROOT"
# shellcheck source=deps-lock.sh
source "$ROOT/scripts/deps-lock.sh"

DEPS="$ROOT/chimera/.deps"
MS_DIR="$DEPS/material-design-icons"
GF_DIR="$DEPS/google-fonts"
MS_REF="$(deps_lock_get MATERIAL_ICONS_GIT_REF)"
GF_REF="$(deps_lock_get GOOGLE_FONTS_GIT_REF)"
MS_URL="$(deps_lock_get MATERIAL_ICONS_GIT_URL)"
GF_URL="$(deps_lock_get GOOGLE_FONTS_GIT_URL)"

need_curl() {
  command -v curl >/dev/null 2>&1 || {
    echo "adminui-fonts-fetch: curl is required" >&2
    exit 1
  }
}

fetch_file() {
  local url="$1" dest="$2"
  mkdir -p "$(dirname "$dest")"
  if [[ -f "$dest" && "${FORCE:-}" != "1" ]]; then
    echo "adminui-fonts-fetch: keep $dest (FORCE=1 to refresh)"
    return 0
  fi
  echo "adminui-fonts-fetch: GET $url"
  curl -fsSL "$url" -o "$dest.tmp"
  mv "$dest.tmp" "$dest"
}

github_raw_base() {
  # https://github.com/org/repo.git -> https://raw.githubusercontent.com/org/repo
  local url="$1"
  url="${url%.git}"
  case "$url" in
  https://github.com/*)
    url="https://raw.githubusercontent.com/${url#https://github.com/}"
    ;;
  esac
  printf '%s\n' "$url"
}

ms_ttf_name='MaterialSymbolsOutlined[FILL,GRAD,opsz,wght].ttf'
ms_cp_name='MaterialSymbolsOutlined[FILL,GRAD,opsz,wght].codepoints'
hk_ttf_name='HankenGrotesk[wght].ttf'

ms_raw="$(github_raw_base "$MS_URL")/${MS_REF}/variablefont/MaterialSymbolsOutlined%5BFILL%2CGRAD%2Copsz%2Cwght%5D.ttf"
ms_cp_raw="$(github_raw_base "$MS_URL")/${MS_REF}/variablefont/MaterialSymbolsOutlined%5BFILL%2CGRAD%2Copsz%2Cwght%5D.codepoints"
hk_raw="$(github_raw_base "$GF_URL")/${GF_REF}/ofl/hankengrotesk/HankenGrotesk%5Bwght%5D.ttf"
hk_ofl="$(github_raw_base "$GF_URL")/${GF_REF}/ofl/hankengrotesk/OFL.txt"

need_curl

if [[ -d "$ROOT/../material-design-icons/variablefont" && "${FORCE:-}" != "1" ]]; then
  echo "adminui-fonts-fetch: sibling ../material-design-icons present; skipping material download"
else
  fetch_file "$ms_raw" "$MS_DIR/variablefont/${ms_ttf_name}"
  fetch_file "$ms_cp_raw" "$MS_DIR/variablefont/${ms_cp_name}"
fi

if [[ -d "$ROOT/../fonts/ofl/hankengrotesk" && "${FORCE:-}" != "1" ]]; then
  echo "adminui-fonts-fetch: sibling ../fonts present; skipping Hanken download"
else
  fetch_file "$hk_raw" "$GF_DIR/ofl/hankengrotesk/${hk_ttf_name}"
  curl -fsSL "$hk_ofl" -o "$GF_DIR/ofl/hankengrotesk/OFL.txt" 2>/dev/null || true
fi

echo "adminui-fonts-fetch: done (sources under $DEPS or siblings)"
