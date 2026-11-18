#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only exact-source test of Judge voting markup, CSS, tiny official assets.
[[ $# -eq 1 && "$1" == "67d19d785c752c9a692a8e0a1aa67da9f7d4d508" ]] || exit 64
src="$1"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$src"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$src" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
npx vitest run src/ratJudgeWorkspace.test.ts src/ratRatingRoute.test.ts
npx tsc -b --pretty false
echo RAT176_COMPACT_VOTE_SOURCE_FOCUSED_PASS=true
