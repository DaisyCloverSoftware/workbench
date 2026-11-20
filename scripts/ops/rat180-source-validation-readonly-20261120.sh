#!/usr/bin/env bash
set -Eeuo pipefail
# Exact-SHA source-only RAT180 tests, bounded output, disposable checkout.
[[ $# -eq 1 && "$1" == '061beefd23bd71cf5902ff2a7724ff78acb515f6' ]] || exit 64
temp="$(mktemp -d)"
trap 'rm -rf "$temp"' EXIT
git -C "$temp" init -q
git -C "$temp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$temp" fetch -q --depth=1 origin "$1"
git -C "$temp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$temp" rev-parse HEAD)" == "$1" ]] || exit 78
cd "$temp/apps/api"
runcheck(){
  local title="$1";shift
  if "$@" >"$temp/$title.log" 2>&1;then
    printf 'RAT180_%s=PASS\n' "$title"
    tail -n 5 "$temp/$title.log" | cut -c1-240
  else
    local code="$?"
    printf 'RAT180_%s=FAIL exit=%s\n' "$title" "$code"
    tail -n 70 "$temp/$title.log" | cut -c1-240
    exit "$code"
  fi
}
printf 'RAT180_SOURCE=%s\n' "$1"
runcheck COMPOSER composer install --no-interaction --prefer-dist --no-progress --no-scripts --quiet
runcheck PINT vendor/bin/pint --test --no-ansi
runcheck PHPSTAN vendor/bin/phpstan analyse --memory-limit=1G --no-progress
runcheck FOCUSED_PHPUNIT vendor/bin/phpunit tests/Feature/Entity/RatExplorerPrivacyTest.php tests/Feature/Entity/UniversalSearchAddAnythingTest.php
echo RAT180_LOCAL_VALIDATION_COMPLETE=true
