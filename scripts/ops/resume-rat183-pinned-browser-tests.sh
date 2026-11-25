#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]]||exit 64
ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
# Same runner-only browser adaptation used by diagnose-rat181-standalone-browser-test.sh.
# Restore the unrelated browser-launch file before build/commit; all assertions run unchanged.
python3 - "$ops/resume-rat183-checkpoint.sh" "$t/run.sh" <<'PY'
from pathlib import Path
import sys
s=Path(sys.argv[1]).read_text();old='npm run lint;npm run typecheck;npm test;npm run build';assert s.count(old)==1
new='''npm run lint;npm run typecheck
python3 - <<'BROWSER'
from pathlib import Path
p=Path('src/standaloneJourney.browser.test.ts');s=p.read_text()
a="  const executablePath = ['/usr/bin/chromium', '/usr/bin/chromium-browser', '/usr/bin/google-chrome', '/usr/bin/google-chrome-stable'].find(existsSync)\\n  browser = await chromium.launch({ headless: true, ...(executablePath ? { executablePath } : {}) })"
assert s.count(a)==1;p.write_text(s.replace(a,'  browser = await chromium.launch({ headless: true })'))
BROWSER
npm test
git checkout -- src/standaloneJourney.browser.test.ts
npm run build'''
Path(sys.argv[2]).write_text(s.replace(old,new))
PY
bash -n "$t/run.sh";bash "$t/run.sh" fix-test
root=$(git -C "$ops" rev-parse --show-toplevel);git -C "$root" ls-tree -r --name-only HEAD -- scripts/ops | grep -E '(rat183|rum-dev-delete-rating)'
