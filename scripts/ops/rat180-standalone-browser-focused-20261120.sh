#!/usr/bin/env bash
set -Eeuo pipefail
# Browser-test-only exact-SHA diagnosis on a disposable RAT checkout.
[[ $# -eq 1 && "$1" == '061beefd23bd71cf5902ff2a7724ff78acb515f6' ]] || exit 64
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$1"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$1" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >"$tmp/npm.log" 2>&1 || { tail -n 35 "$tmp/npm.log";exit 1; }
if ! command -v chromium >/dev/null 2>&1 && ! command -v google-chrome >/dev/null 2>&1;then
 npx playwright install chromium >"$tmp/browser.log" 2>&1 || { tail -n 35 "$tmp/browser.log";exit 1; }
fi
printf 'RAT180_BROWSER_SOURCE=%s\n' "$1"
if npm test -- src/standaloneJourney.browser.test.ts >"$tmp/test.log" 2>&1;then
 echo RAT180_EXISTING_STANDALONE_BROWSER_TEST=PASS
 tail -n 15 "$tmp/test.log" | cut -c1-240
else
 code=$?
 echo "RAT180_EXISTING_STANDALONE_BROWSER_TEST=FAIL exit=$code"
 tail -n 65 "$tmp/test.log" | cut -c1-240
 exit "$code"
fi
