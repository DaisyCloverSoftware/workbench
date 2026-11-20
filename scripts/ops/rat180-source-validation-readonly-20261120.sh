#!/usr/bin/env bash
set -Eeuo pipefail
# Source-only exact-head RAT180 verification in a disposable checkout.
# No registered checkout, Kubernetes, DB deployment, data or PR mutation.
[[ $# -eq 1 && "$1" == '061beefd23bd71cf5902ff2a7724ff78acb515f6' ]] || exit 64
temp="$(mktemp -d)"
trap 'rm -rf "$temp"' EXIT
git -C "$temp" init -q
git -C "$temp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$temp" fetch -q --depth=1 origin "$1"
git -C "$temp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$temp" rev-parse HEAD)" == "$1" ]] || exit 78
cd "$temp/apps/api"
composer install --no-interaction --prefer-dist --no-progress --no-scripts --quiet >/dev/null
printf 'SOURCE_SHA=%s\n' "$1"
vendor/bin/pint --test --no-ansi
echo 'PINT_COMPLETE_PASS=true'
vendor/bin/phpstan analyse --memory-limit=1G --no-progress
echo 'PHPSTAN_COMPLETE_PASS=true'
vendor/bin/phpunit tests/Feature/Entity/RatExplorerPrivacyTest.php tests/Feature/Entity/UniversalSearchAddAnythingTest.php
echo 'FOCUSED_PHPUNIT_COMPLETE_PASS=true'
