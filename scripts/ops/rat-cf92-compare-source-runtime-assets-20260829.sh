#!/usr/bin/env bash
set -euo pipefail

RUM_REPO='https://github.com/DaisyCloverSoftware/rum.git'
RUM_COMMIT='cf92e170c8b8728cb59c5b22c424e6472d048b49'
BASE='https://dev-rum-ra.daisycloversoftware.uk/brand'
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

git clone --quiet --no-checkout "$RUM_REPO" "$TMP/rum"
cd "$TMP/rum"
git fetch --quiet origin "$RUM_COMMIT"
git checkout --quiet --detach "$RUM_COMMIT"

for name in rum-thumb.webp rat-pirate-mascot.webp; do
  src="apps/rate-anything/public/brand/$name"
  printf '%s_source_bytes=%s\n' "$name" "$(stat -c '%s' "$src")"
  printf '%s_source_sha256=%s\n' "$name" "$(sha256sum "$src" | awk '{print $1}')"
  printf '%s_source_blob=%s\n' "$name" "$(git hash-object "$src")"
  printf '%s_source_magic12=%s\n' "$name" "$(od -An -tx1 -N12 "$src" | tr -d ' \n')"
done

probe() {
  local label="$1"
  local url="$2"
  local out="$TMP/$label.bin"
  local headers="$TMP/$label.headers"
  curl -fsS --connect-timeout 10 --max-time 30 -D "$headers" \
    -H 'Cache-Control: no-cache, no-store' -H 'Pragma: no-cache' "$url" -o "$out"
  printf '%s_bytes=%s\n' "$label" "$(stat -c '%s' "$out")"
  printf '%s_sha256=%s\n' "$label" "$(sha256sum "$out" | awk '{print $1}')"
  printf '%s_magic12=%s\n' "$label" "$(od -An -tx1 -N12 "$out" | tr -d ' \n')"
  printf '%s_mime=%s\n' "$label" "$(file -b --mime-type "$out")"
  printf '%s_cf_cache=%s\n' "$label" "$(awk 'BEGIN{IGNORECASE=1} /^cf-cache-status:/ {gsub(/\r/,""); sub(/^[^:]+:[[:space:]]*/,""); print; exit}' "$headers")"
  printf '%s_age=%s\n' "$label" "$(awk 'BEGIN{IGNORECASE=1} /^age:/ {gsub(/\r/,""); sub(/^[^:]+:[[:space:]]*/,""); print; exit}' "$headers")"
}

probe thumb_key "$BASE/rum-thumb.webp?v=ad234fbada71f3be"
probe thumb_unique "$BASE/rum-thumb.webp?v=ad234fbada71f3be&proof=$(date +%s%N)"
probe mascot_key "$BASE/rat-pirate-mascot.webp?v=0c4ade813ec3e41a"
probe mascot_unique "$BASE/rat-pirate-mascot.webp?v=0c4ade813ec3e41a&proof=$(date +%s%N)"
