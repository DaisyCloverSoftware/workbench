#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then echo "usage: $0 <exact-rum-sha>" >&2; exit 64; fi
SHA="$1"
[[ "$SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "full lowercase SHA required" >&2; exit 64; }
REPOSITORY="DaisyCloverSoftware/rum"
IMAGE="docker.io/library/node:22-bookworm"

for command in gh git mktemp; do command -v "$command" >/dev/null 2>&1 || { echo "missing $command" >&2; exit 2; }; done
TOKEN="${GH_TOKEN:-}"; [[ -n "$TOKEN" ]] || TOKEN="$(gh auth token 2>/dev/null || true)"
[[ -n "$TOKEN" ]] || { echo "GitHub token unavailable" >&2; exit 2; }
if command -v podman >/dev/null 2>&1; then RUNTIME=podman; elif command -v docker >/dev/null 2>&1; then RUNTIME=docker; else echo "no container runtime" >&2; exit 2; fi

tmp="$(mktemp -d)"; cid=""
cleanup() { [[ -z "$cid" ]] || "$RUNTIME" rm -f "$cid" >/dev/null 2>&1 || true; rm -rf "$tmp"; }
trap cleanup EXIT HUP INT TERM
GH_TOKEN="$TOKEN" gh repo clone "$REPOSITORY" "$tmp/rum" -- --no-checkout --filter=blob:none >/dev/null
git -C "$tmp/rum" checkout --detach "$SHA" >/dev/null
[[ "$(git -C "$tmp/rum" rev-parse HEAD)" == "$SHA" ]] || { echo "checkout mismatch" >&2; exit 78; }

"$RUNTIME" pull "$IMAGE" >/dev/null
cid="$("$RUNTIME" create --user 0:0 --entrypoint tail "$IMAGE" -f /dev/null)"
"$RUNTIME" start "$cid" >/dev/null
"$RUNTIME" exec --user 0:0 "$cid" mkdir -p /work
"$RUNTIME" cp "$tmp/rum/apps/web/." "$cid:/work"
"$RUNTIME" exec --user 0:0 -w /work "$cid" sh -lc 'npm ci --no-audit --no-fund >/dev/null && npm run lint && npm run typecheck && npm test -- --run && npm run build'
printf 'RUM_WEB_EXACT_CHECK_OK=%s\n' "$SHA"
