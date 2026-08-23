#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-rum-candidate-sha>" >&2
  exit 64
fi

for command in podman mktemp sed grep; do
  command -v "$command" >/dev/null 2>&1 || {
    echo "required command unavailable: $command" >&2
    exit 2
  }
done

# The RUM API Dockerfile uses Docker-compatible external-stage syntax:
#   COPY --from=composer:2 ...
# On this Podman/Buildah host the unqualified external image is not pulled
# automatically. Pre-stage an equivalent local short-name alias without
# changing the candidate Dockerfile or build context.
podman pull docker.io/library/composer:2 >/dev/null
podman tag docker.io/library/composer:2 composer:2
podman image inspect composer:2 >/dev/null

# The guarded publisher uses --pull=always. On Buildah that also forces a
# registry pull for COPY --from=composer:2, resolving the local short name as
# localhost/composer:2 and failing. Run a disposable tooling-only copy with
# --pull=newer instead: registry-backed base images are refreshed when their
# digest differs, while a failed pull is suppressed when the required local
# Composer image exists.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
compat_script="$(mktemp)"
cleanup() { rm -f "$compat_script"; }
trap cleanup EXIT HUP INT TERM
sed 's/--pull=always/--pull=newer/' "$SCRIPT_DIR/rum-candidate-full-tag-publisher.sh" >"$compat_script"
grep -q -- '--pull=newer' "$compat_script" || {
  echo "compatibility rewrite failed" >&2
  exit 2
}
if grep -q -- '--pull=always' "$compat_script"; then
  echo "compatibility rewrite left an always-pull directive" >&2
  exit 2
fi
bash "$compat_script" "$1"
