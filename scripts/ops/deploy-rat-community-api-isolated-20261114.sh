#!/usr/bin/env bash
set -euo pipefail
# Guard the exact previous PR169 API image, then reuse the reviewed bounded
# API-only operation. Never run a migration, seed, reset, Helm or LIVE mutation.
[[ $# -eq 2 ]] || { echo "usage: $0 <source-sha> <api-image@sha256>" >&2; exit 64; }
source_sha="$1"; candidate="$2"
[[ "$source_sha" == 'e3bb818f1825d778f5084460057ba7ea8bfb422b' ]] || { echo 'Blocked: unexpected PR170 source SHA' >&2; exit 78; }
[[ "$source_sha" =~ ^[0-9a-f]{40}$ ]] || exit 64
[[ "$candidate" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
previous='ghcr.io/daisycloversoftware/rum-api@sha256:24b789fefa9fbb12f02e4baaa14b14fcdb81afedfd9775982986a3d53d3109d2'
kctl() { if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
current="$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')"
[[ "$current" == "$previous" || "$current" == "$candidate" ]] || {
  echo "Blocked: canonical DEV API was superseded: $current" >&2; exit 78;
}
rat_current="$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')"
old_front='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:83c476fc811fb50a9ec1d04117e04dc9c5d319b1314ba99c349d222f9e1c12d9'
[[ "$rat_current" == "$old_front" ]] || { echo "Blocked: RAT frontend moved since API preflight: $rat_current" >&2; exit 78; }
bash "$(dirname "$0")/deploy-rat-search-api-repair.sh" "$source_sha" "$candidate"
