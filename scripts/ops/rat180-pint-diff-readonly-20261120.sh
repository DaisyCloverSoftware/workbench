#!/usr/bin/env bash
set -Eeuo pipefail
# RAT180 exact-head Pint diagnosis in a disposable isolated checkout only.
# Never modifies the registered rum checkout, cluster, images or application data.
[[ $# -eq 1 && "$1" == 'ef100f6eb4d1b5400610d9879529ae78e7072ff9' ]] || exit 64
temp="$(mktemp -d)"
trap 'rm -rf "$temp"' EXIT
git -C "$temp" init -q
git -C "$temp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$temp" fetch -q --depth=1 origin "$1"
git -C "$temp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$temp" rev-parse HEAD)" == "$1" ]] || exit 78
cd "$temp/apps/api"
composer install --no-interaction --prefer-dist --no-progress --no-scripts --quiet >/dev/null
printf 'PINT_DIAG_SOURCE=%s\n' "$1"
vendor/bin/pint --test app/Http/Controllers/Api/V1/RatCatalogueController.php app/Http/Controllers/Api/V1/UniversalEntityController.php --no-ansi || true
vendor/bin/pint app/Http/Controllers/Api/V1/UniversalEntityController.php --no-ansi >/dev/null
git -C "$temp" diff -- apps/api/app/Http/Controllers/Api/V1/UniversalEntityController.php
echo RAT180_DISPOSABLE_PINT_DIFF_COMPLETE=true
