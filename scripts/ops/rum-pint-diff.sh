#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <exact-rum-candidate-sha>" >&2
  exit 64
fi
SHA="$1"
[[ "$SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "full lowercase SHA required" >&2; exit 64; }

REPOSITORY="DaisyCloverSoftware/rum"
IMAGE="ghcr.io/daisycloversoftware/rum-api@sha256:f4e700314dd2fe2b8b6dabee5f640f35b9463d914f8a2be79ff781862a80e978"

for command in gh git mktemp; do command -v "$command" >/dev/null 2>&1 || { echo "missing $command" >&2; exit 2; }; done
TOKEN="${GH_TOKEN:-}"
[[ -n "$TOKEN" ]] || TOKEN="$(gh auth token 2>/dev/null || true)"
[[ -n "$TOKEN" ]] || { echo "GitHub token unavailable" >&2; exit 2; }

if command -v podman >/dev/null 2>&1; then
  RUNTIME=podman
elif command -v docker >/dev/null 2>&1; then
  RUNTIME=docker
else
  echo "no container runtime" >&2; exit 2
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
GH_TOKEN="$TOKEN" gh repo clone "$REPOSITORY" "$tmp/rum" -- --no-checkout --filter=blob:none >/dev/null
git -C "$tmp/rum" checkout --detach "$SHA" >/dev/null
[[ "$(git -C "$tmp/rum" rev-parse HEAD)" == "$SHA" ]] || { echo "checkout mismatch" >&2; exit 78; }

# Rootless Podman must be able to traverse the disposable scratch parent. The
# checkout contains no credentials; only the temporary source tree is exposed.
chmod 755 "$tmp" "$tmp/rum"

"$RUNTIME" pull "$IMAGE" >/dev/null
if [[ "$RUNTIME" == "podman" ]]; then
  VOLUME="$tmp/rum:/work:Z"
else
  VOLUME="$tmp/rum:/work"
fi
"$RUNTIME" run --rm \
  -v "$VOLUME" \
  -w /work/apps/api \
  "$IMAGE" \
  sh -lc 'test -f composer.json && composer install --no-interaction --prefer-dist --no-progress >/dev/null && vendor/bin/pint app/Http/Controllers/Api/V1/RatingController.php app/Services/RateAnythingRatingService.php >/dev/null'

printf '%s\n' '--- PINT DIFF START ---'
git -C "$tmp/rum" diff -- apps/api/app/Http/Controllers/Api/V1/RatingController.php apps/api/app/Services/RateAnythingRatingService.php
printf '%s\n' '--- PINT DIFF END ---'
