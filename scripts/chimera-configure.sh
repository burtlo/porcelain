#!/usr/bin/env bash
# Materialize local Chimera config from examples (never overwrites existing files).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COPY="$ROOT/scripts/configure-copy.sh"
missing=0
"$COPY" chimera.example.yaml chimera.yaml || missing=1
"$COPY" api-keys.example.yaml api-keys.yaml || missing=1
"$COPY" indexer.example.yaml indexer.yaml || missing=1
exit "$missing"
