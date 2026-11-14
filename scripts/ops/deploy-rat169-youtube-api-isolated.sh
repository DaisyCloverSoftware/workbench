#!/usr/bin/env bash
set -euo pipefail
# Guard the exact previous PR169 API image, then reuse the reviewed bounded
# API-only operation. Never run a migration, seed, reset, Helm or LIVE mutation.
[[ $# -eq 2 ]] || { echo "usage: $0 <source-sha> <api-image@sha256>" >&2; exit 64; }
source_sha="$1"; candidate="$2"
[[ "$source_sha" =~ ^[0-9a-f]{40}$ ]] || exit 64
[[ "$candidate" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
previous='ghcr.io/daisycloversoftware/rum-api@sha256:3f313da265f9b19c166073fa6642922cf8ee2e897318b22ce747826a841395f3'
kctl() { if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
current="$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')"
[[ "$current" == "$previous" || "$current" == "$candidate" ]] || {
  echo "Blocked: canonical DEV API was superseded: $current" >&2; exit 78;
}
rat_current="$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')"
old_front='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:9e9feff67611009f1f9193345754fd929cd9dd1449e6d3b30ae3a5486c085638'
[[ "$rat_current" == "$old_front" ]] || { echo "Blocked: RAT frontend moved since API preflight: $rat_current" >&2; exit 78; }
bash "$(dirname "$0")/deploy-rat-search-api-repair.sh" "$source_sha" "$candidate"
