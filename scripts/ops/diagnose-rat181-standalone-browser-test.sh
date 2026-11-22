#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]] || exit 64
sha=b135949c8d97072cca3b24c2611984ae6d6fc30c
t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
git -C "$t" init -q;git -C "$t" remote add origin https://github.com/DaisyCloverSoftware/rum.git;git -C "$t" fetch -q --depth=1 origin fix/rat-persist-explicit-linked-things-20261122; [[ $(git -C "$t" rev-parse FETCH_HEAD) == "$sha" ]] || exit 78;git -C "$t" checkout -q --detach FETCH_HEAD
cd "$t/apps/rate-anything"; mapfile -t tests < <(git ls-files | grep -Ei 'standalone.*browser.*test\.(ts|tsx)$'); printf 'Matching suite: %s\n' "${tests[@]}"; [[ ${#tests[@]} -gt 0 ]] || exit 78
npm ci --no-audit --no-fund --ignore-scripts >"$t/install" 2>&1;npx playwright install chromium --only-shell >"$t/chromium" 2>&1
# Local application tests only. This does not publish or promote, and cannot count as a GitHub CI result.
set +e; npx vitest run "${tests[@]}" --reporter=verbose >"$t/test" 2>&1; rc=$?;set -e;tail -65 "$t/test";echo "EXACT_SOURCE=$sha LOCAL_RAT_STANDALONE_EXIT=$rc";exit "$rc"
