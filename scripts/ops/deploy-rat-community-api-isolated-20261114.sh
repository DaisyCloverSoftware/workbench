#!/usr/bin/env bash
set -euo pipefail
# Guard the exact previous PR169 API image, then reuse the reviewed bounded
# API-only operation. Never run a migration, seed, reset, Helm or LIVE mutation.
[[ $# -eq 2 ]] || { echo "usage: $0 <source-sha> <api-image@sha256>" >&2; exit 64; }
source_sha="$1"; candidate="$2"
[[ "$source_sha" =~ ^[0-9a-f]{40}$ ]] || exit 64
[[ "$candidate" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
previous='ghcr.io/daisycloversoftware/rum-api@sha256:71ec4ce2fa083270275f475e9a6c7c427088c0ed568263f6abbe5670ca9ba443'
kctl() { if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
current="$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')"
[[ "$current" == "$previous" || "$current" == "$candidate" ]] || {
  echo "Blocked: canonical DEV API was superseded: $current" >&2; exit 78;
}
rat_current="$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')"
old_front='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:9d067ce6244ef9504e67c478d06975c2bf22a1f3c41c1c240c7134b011c15778'
[[ "$rat_current" == "$old_front" ]] || { echo "Blocked: RAT frontend moved since API preflight: $rat_current" >&2; exit 78; }
bash "$(dirname "$0")/deploy-rat-search-api-repair.sh" "$source_sha" "$candidate"
