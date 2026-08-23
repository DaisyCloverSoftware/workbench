#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-rum-candidate-sha>" >&2
  exit 64
fi

CANDIDATE_SHA="$1"
[[ "$CANDIDATE_SHA" =~ ^[0-9a-f]{40}$ ]] || {
  echo "candidate SHA must be a full 40-character lowercase hexadecimal commit" >&2
  exit 64
}

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
AUTHFILE="$(mktemp)"
cleanup() {
  rm -f "$AUTHFILE"
}
trap cleanup EXIT HUP INT TERM

printf '%s' "$TOKEN" | podman login --authfile "$AUTHFILE" ghcr.io -u "$ACTOR" --password-stdin >/dev/null
unset TOKEN GH_TOKEN GHCR_TOKEN

TAG="sha-${CANDIDATE_SHA:0:7}"
missing=0
for component in rum-web rum-api; do
  ref="ghcr.io/daisycloversoftware/${component}:${TAG}"
  if ! timeout 90s podman pull --authfile "$AUTHFILE" "$ref" >/dev/null 2>&1; then
    printf '%s_STATUS=UNAVAILABLE\n' "${component^^}"
    missing=1
    continue
  fi
  revision="$(podman image inspect "$ref" --format '{{ index .Labels "org.opencontainers.image.revision" }}')"
  digest="$(podman image inspect "$ref" --format '{{.Digest}}')"
  [[ "$revision" == "$CANDIDATE_SHA" ]] || {
    echo "REGISTRY INSPECTION BLOCKED: ${ref} revision ${revision} != ${CANDIDATE_SHA}" >&2
    exit 78
  }
  [[ "$digest" == sha256:* ]] || {
    echo "REGISTRY INSPECTION BLOCKED: ${ref} has no immutable digest" >&2
    exit 78
  }
  printf '%s_STATUS=PRESENT\n' "${component^^}"
  printf '%s_DIGEST=%s\n' "${component^^}" "$digest"
  printf '%s_REVISION=%s\n' "${component^^}" "$revision"
done

printf 'RUM_CANDIDATE_SHA=%s\n' "$CANDIDATE_SHA"
printf 'RUM_CANDIDATE_TAG=%s\n' "$TAG"
printf 'REGISTRY_MUTATION=NO\n'
printf 'LIVE_RUNTIME_AFFECTED=NO\n'
printf 'RATE_ANYTHING_AFFECTED=NO\n'
exit "$missing"
