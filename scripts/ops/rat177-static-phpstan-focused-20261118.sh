#!/usr/bin/env bash
set -Eeuo pipefail
# Exact-source PHPStan on temporary scratch; never touch canonical DB or running workloads.
[[ $# -eq 1 && "$1" == "9eeb261bd4cec483d629b88eaebe2ae5b2c2ab9e" ]] || exit 64
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$1"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$1" ]] || exit 78
cd "$tmp/apps/api"
composer install --no-interaction --prefer-dist --no-progress --no-scripts --quiet >/dev/null
vendor/bin/pint --test --no-ansi
vendor/bin/phpstan analyse --memory-limit=1G --no-progress --no-ansi
echo RAT177_PHPSTAN_PINT_EXACT_SOURCE_PASS=true
