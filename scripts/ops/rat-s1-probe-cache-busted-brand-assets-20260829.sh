#!/usr/bin/env bash
set -euo pipefail

BASE='https://dev-rum-ra.daisycloversoftware.uk/brand'
VERSION='60cab55d5bd868913da833e60c15cbe938afa494'
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

for name in rum-thumb.webp rat-pirate-mascot.webp; do
  for mode in plain busted; do
    if [[ "$mode" == plain ]]; then
      url="$BASE/$name"
    else
      url="$BASE/$name?v=$VERSION"
    fi
    out="$TMP/${name}.${mode}"
    headers="$TMP/${name}.${mode}.headers"
    curl -fsS --connect-timeout 10 --max-time 30 -D "$headers" \
      -H 'Cache-Control: no-cache, no-store' -H 'Pragma: no-cache' \
      "$url" -o "$out"
    bytes="$(stat -c '%s' "$out")"
    sha="$(sha256sum "$out" | awk '{print $1}')"
    magic="$(od -An -tx1 -N12 "$out" | tr -d ' \n')"
    mime="$(file -b --mime-type "$out" 2>/dev/null || true)"
    cache_control="$(awk 'BEGIN{IGNORECASE=1} /^cache-control:/ {gsub(/\r/,""); sub(/^[^:]+:[[:space:]]*/,""); print; exit}' "$headers")"
    age="$(awk 'BEGIN{IGNORECASE=1} /^age:/ {gsub(/\r/,""); sub(/^[^:]+:[[:space:]]*/,""); print; exit}' "$headers")"
    cf_cache="$(awk 'BEGIN{IGNORECASE=1} /^cf-cache-status:/ {gsub(/\r/,""); sub(/^[^:]+:[[:space:]]*/,""); print; exit}' "$headers")"
    via="$(awk 'BEGIN{IGNORECASE=1} /^via:/ {gsub(/\r/,""); sub(/^[^:]+:[[:space:]]*/,""); print; exit}' "$headers")"
    printf '%s|%s|bytes=%s|sha256=%s|magic12=%s|mime=%s|cache_control=%s|age=%s|cf_cache=%s|via=%s\n' \
      "$name" "$mode" "$bytes" "$sha" "$magic" "$mime" "$cache_control" "$age" "$cf_cache" "$via"
  done
done
