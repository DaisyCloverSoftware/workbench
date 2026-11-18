#!/usr/bin/env bash
set -Eeuo pipefail
# Source-only deterministic formatting diff on exact PR177 head; no app DB or deployment.
[[ $# -eq 1 && "$1" == '6cf8f1e921e9f9b0c35369ab380d4103886a935e' ]] || exit 64
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
git -C "$work" init -q
git -C "$work" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$work" fetch -q --depth=1 origin "$1"
git -C "$work" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$work" rev-parse HEAD)" == "$1" ]] || exit 78
cd "$work/apps/api"
if [[ ! -x vendor/bin/pint ]]; then
 composer install --no-interaction --prefer-dist --no-progress --no-scripts --quiet >/dev/null
fi
vendor/bin/pint app/Services/ThingTypeChangeService.php --no-ansi >/dev/null
git -C "$work" diff -- apps/api/app/Services/ThingTypeChangeService.php
echo RAT177_PINT_DIFF_GENERATED=true
