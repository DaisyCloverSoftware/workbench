#!/usr/bin/env bash
set -euo pipefail

BASE='https://dev-rum-ra.daisycloversoftware.uk'
EXPECTED_VERSION='rat-candidate+cf92e170c8b8728cb59c5b22c424e6472d048b49'
THUMB='/brand/rum-thumb.webp?v=ad234fbada71f3be'
MASCOT='/brand/rat-pirate-mascot.webp?v=0c4ade813ec3e41a'
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

version="$(curl -fsS --connect-timeout 10 --max-time 30 -H 'Cache-Control: no-cache, no-store' "$BASE/VERSION?proof=cf92e170")"
[[ "$version" == "$EXPECTED_VERSION" ]] || {
  echo "ERROR: VERSION mismatch: expected $EXPECTED_VERSION got $version" >&2
  exit 1
}
printf 'version=%s\n' "$version"

check_asset() {
  local label="$1"
  local path="$2"
  local expected_bytes="$3"
  local expected_sha="$4"
  local out="$TMP/$label"
  local headers="$TMP/$label.headers"
  curl -fsS --connect-timeout 10 --max-time 30 -D "$headers" \
    -H 'Cache-Control: no-cache, no-store' -H 'Pragma: no-cache' \
    "$BASE$path" -o "$out"
  local bytes sha magic mime cf_cache age
  bytes="$(stat -c '%s' "$out")"
  sha="$(sha256sum "$out" | awk '{print $1}')"
  magic="$(od -An -tx1 -N12 "$out" | tr -d ' \n')"
  mime="$(file -b --mime-type "$out")"
  cf_cache="$(awk 'BEGIN{IGNORECASE=1} /^cf-cache-status:/ {gsub(/\r/,""); sub(/^[^:]+:[[:space:]]*/,""); print; exit}' "$headers")"
  age="$(awk 'BEGIN{IGNORECASE=1} /^age:/ {gsub(/\r/,""); sub(/^[^:]+:[[:space:]]*/,""); print; exit}' "$headers")"
  [[ "$bytes" == "$expected_bytes" ]] || { echo "ERROR: $label size $bytes != $expected_bytes" >&2; exit 1; }
  [[ "$sha" == "$expected_sha" ]] || { echo "ERROR: $label sha $sha != $expected_sha" >&2; exit 1; }
  [[ "$mime" == 'image/webp' ]] || { echo "ERROR: $label mime $mime" >&2; exit 1; }
  [[ "$magic" == 52494646*57454250 ]] || { echo "ERROR: $label magic $magic" >&2; exit 1; }
  printf '%s_url=%s\n' "$label" "$path"
  printf '%s_bytes=%s\n' "$label" "$bytes"
  printf '%s_sha256=%s\n' "$label" "$sha"
  printf '%s_magic12=%s\n' "$label" "$magic"
  printf '%s_mime=%s\n' "$label" "$mime"
  printf '%s_cf_cache=%s\n' "$label" "${cf_cache:-none}"
  printf '%s_age=%s\n' "$label" "${age:-none}"
}

check_asset thumb "$THUMB" 14232 38643e3cad4e1e9e22b1726a9fdbc8cb415e6630ce60db9601c9c734627c853c
check_asset mascot "$MASCOT" 9000 279cccc86f46fb4c50fb63eecfdd58f7ffe1b5c3bf6e32f34b8fe5c8de0402b3

echo 'runtime_asset_proof=PASS'
