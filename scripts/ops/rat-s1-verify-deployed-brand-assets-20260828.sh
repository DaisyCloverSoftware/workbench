#!/usr/bin/env bash
set -euo pipefail

BASE='https://dev-rum-ra.daisycloversoftware.uk/brand'
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

for name in rat-wordmark.webp rum-thumb.webp rat-pirate-mascot.webp; do
  out="$TMP/$name"
  headers="$TMP/$name.headers"
  curl -fsS --connect-timeout 10 --max-time 30 -D "$headers" "$BASE/$name" -o "$out"
  mime="$(file -b --mime-type "$out" 2>/dev/null || true)"
  bytes="$(stat -c '%s' "$out")"
  sha="$(sha256sum "$out" | awk '{print $1}')"
  magic="$(od -An -tx1 -N12 "$out" | tr -d ' \n')"
  ctype="$(awk 'BEGIN{IGNORECASE=1} /^content-type:/ {gsub(/\r/,""); sub(/^[^:]+:[[:space:]]*/,""); print; exit}' "$headers")"
  printf '%s|http_content_type=%s|file_mime=%s|bytes=%s|sha256=%s|magic12=%s\n' "$name" "$ctype" "$mime" "$bytes" "$sha" "$magic"
done

echo RAT_DEPLOYED_BRAND_ASSET_PROBE_DONE
