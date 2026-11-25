#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 2 && $1 == 2359178af19fdd9b5b45d103c892cdc36c2ad086 && $2 == rat182-20261124-isolated-release-r2 ]] || exit 64
ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
# Test-harness correction only. The approved stylesheet intentionally hides the
# existing mascot below 1600px. Exercise both sides without changing any app code.
[[ $(sha256sum "$ops/accept-rat182-shared-discovery.sh" | cut -d' ' -f1) == fa70a0893068eb665a1c94d231afa8fbea662c39306d64eeceb5163117cb922c ]]
python3 - "$ops" "$t" <<'PY'
from pathlib import Path
import shlex,sys
ops,t=map(Path,sys.argv[1:]);s=(ops/'accept-rat182-shared-discovery.sh').read_text()
old='ops=$(cd "$(dirname "$0")" && pwd)';assert s.count(old)==1;s=s.replace(old,'ops='+shlex.quote(str(ops)))
old="await expect(page.locator('.rat-results-mascot')).toBeVisible();"
new="const mascot=page.locator('.rat-results-mascot');await expect(mascot).toBeHidden();await page.setViewportSize({width:1920,height:1000});await expect(mascot).toBeVisible();await expect.poll(()=>mascot.evaluate(img=>img.complete&&img.naturalWidth>0),{timeout:15000}).toBe(true);await page.setViewportSize({width:1440,height:1000});await expect(mascot).toBeHidden();"
assert s.count(old)==1;s=s.replace(old,new)
old='evidence.brandAndThumbsLoaded=true;';assert s.count(old)==1;s=s.replace(old,old+'evidence.responsiveMascotWidths=[1440,1920];')
(t/'accept.sh').write_text(s)
(t/'public.mjs').write_text(s.split("<<'JS'\n",1)[1].split('\nJS\n',1)[0])
PY
bash -n "$t/accept.sh";node --check "$t/public.mjs"
bash "$t/accept.sh" "$@"
echo RAT182_PUBLIC_RESPONSIVE_ACCEPTANCE_PASS
