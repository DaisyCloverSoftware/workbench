#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]] || exit 64
sha=b135949c8d97072cca3b24c2611984ae6d6fc30c
t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
git -C "$t" init -q;git -C "$t" remote add origin https://github.com/DaisyCloverSoftware/rum.git;git -C "$t" fetch -q --depth=1 origin fix/rat-persist-explicit-linked-things-20261122; [[ $(git -C "$t" rev-parse FETCH_HEAD) == "$sha" ]] || exit 78;git -C "$t" checkout -q --detach FETCH_HEAD
cd "$t/apps/rate-anything";suite=src/standaloneJourney.browser.test.ts
npm ci --no-audit --no-fund --ignore-scripts >"$t/install" 2>&1;npx playwright install chromium --only-shell >"$t/chromium" 2>&1
# This dedicated disposable copy uses the pinned Playwright browser (not the system Snap).
# Only browser selection changes in the test harness; no application code or assertion changes and no push.
python3 - "$suite" <<'PY'
import sys
from pathlib import Path
p=Path(sys.argv[1]);s=p.read_text();a="  const executablePath = ['/usr/bin/chromium', '/usr/bin/chromium-browser', '/usr/bin/google-chrome', '/usr/bin/google-chrome-stable'].find(existsSync)\n  browser = await chromium.launch({ headless: true, ...(executablePath ? { executablePath } : {}) })";assert s.count(a)==1;p.write_text(s.replace(a,'  browser = await chromium.launch({ headless: true })'))
PY
set +e; npx vitest run "$suite" --reporter=verbose >"$t/test" 2>&1; rc=$?;set -e;tail -48 "$t/test";echo "APP_SOURCE=$sha PINNED_CHROMIUM_LOCAL_EXIT=$rc";exit "$rc"
