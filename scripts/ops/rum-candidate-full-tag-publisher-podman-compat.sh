#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-rum-candidate-sha>" >&2
  exit 64
fi

for command in podman; do
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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec bash "$SCRIPT_DIR/rum-candidate-full-tag-publisher.sh" "$1"
