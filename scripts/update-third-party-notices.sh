#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
GO_BIN="${GO_BIN:-go}"
OUT_FILE="${OUT_FILE:-THIRD_PARTY_NOTICES.md}"
PLATFORMS="${PLATFORMS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64}"
GENERATED_ON="${GENERATED_ON:-$(date -u +%Y-%m-%d)}"

if [[ "$OUT_FILE" = /* ]]; then
	out_path="$OUT_FILE"
else
	out_path="$ROOT_DIR/$OUT_FILE"
fi

cd "$ROOT_DIR/scripts/third_party_notices"

GOWORK=off "$GO_BIN" run . \
	--root "$ROOT_DIR" \
	--out "$out_path" \
	--platforms "$PLATFORMS" \
	--generated-on "$GENERATED_ON" \
	--go "$GO_BIN"
