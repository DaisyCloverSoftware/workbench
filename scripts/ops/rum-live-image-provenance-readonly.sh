#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <web-digest> <api-digest>" >&2
  exit 64
fi

WEB_DIGEST="$1"
API_DIGEST="$2"
for digest in "$WEB_DIGEST" "$API_DIGEST"; do
  [[ "$digest" =~ ^sha256:[0-9a-f]{64}$ ]] || {
    echo "digest must be sha256:<64 lowercase hex>" >&2
    exit 64
  }
done

for command in gh podman mktemp timeout; do
  command -v "$command" >/dev/null 2>&1 || {
    echo "required command unavailable: $command" >&2
    exit 2
  }
done

TOKEN="${GH_TOKEN:-}"
if [[ -z "$TOKEN" ]]; then
  TOKEN="$(gh auth token 2>/dev/null || true)"
fi
[[ -n "$TOKEN" ]] || {
  echo "no GitHub token is available for private GHCR inspection" >&2
  exit 2
}

ACTOR="$(GH_TOKEN="$TOKEN" gh api user --jq .login 2>/dev/null || true)"
[[ -n "$ACTOR" ]] || ACTOR="mattbrownsett"
AUTHDIR="$(mktemp -d)"
AUTHFILE="$AUTHDIR/auth.json"
cleanup() { rm -rf "$AUTHDIR"; }
trap cleanup EXIT HUP INT TERM
printf '%s' "$TOKEN" | podman login --authfile "$AUTHFILE" ghcr.io -u "$ACTOR" --password-stdin >/dev/null
unset TOKEN GH_TOKEN GHCR_TOKEN

inspect_component() {
  local component="$1"
  local digest="$2"
  local ref="ghcr.io/daisycloversoftware/${component}@${digest}"
  timeout 90s podman pull --authfile "$AUTHFILE" "$ref" >/dev/null
  local actual revision source version created
  actual="$(podman image inspect "$ref" --format '{{.Digest}}')"
  revision="$(podman image inspect "$ref" --format '{{ index .Labels "org.opencontainers.image.revision" }}')"
  source="$(podman image inspect "$ref" --format '{{ index .Labels "org.opencontainers.image.source" }}')"
  version="$(podman image inspect "$ref" --format '{{ index .Labels "org.opencontainers.image.version" }}')"
  created="$(podman image inspect "$ref" --format '{{.Created}}')"
  [[ "$actual" == "$digest" ]] || { echo "digest mismatch for $component" >&2; exit 78; }
  printf '%s_DIGEST=%s\n' "${component^^}" "$actual"
  printf '%s_REVISION=%s\n' "${component^^}" "$revision"
  printf '%s_SOURCE=%s\n' "${component^^}" "$source"
  printf '%s_VERSION=%s\n' "${component^^}" "$version"
  printf '%s_CREATED=%s\n' "${component^^}" "$created"
}

inspect_component rum-web "$WEB_DIGEST"
inspect_component rum-api "$API_DIGEST"
printf 'LIVE_RUNTIME_MUTATION=NO\n'
printf 'REGISTRY_MUTATION=NO\n'
printf 'LOCAL_IMAGE_CACHE_ONLY=YES\n'
