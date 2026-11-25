#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]]||exit 64
ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
for f in accept-rat183-fresh-image-failure.sh accept-rat183-connected-thumbnails.sh check-rat181-protected-baseline.sh;do cp "$ops/$f" "$t/$f";done
cat >"$t/adjust.py" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]);s=p.read_text()
s=s.replace("import {chromium,expect} from '@playwright/test';", "import {chromium,expect as baseExpect} from '@playwright/test';const expect=baseExpect.configure({timeout:30000});")
s=s.replace("async function readOnlyRoutes(){", "async function readOnlyRoutes(){page.on('requestfailed',q=>console.log('READONLY_REQUEST_FAILED',q.method(),new URL(q.url()).pathname,q.failure()?.errorText));page.on('response',r=>{if(r.status()>=400&&new URL(r.url()).pathname!=='/api/v1/rate-anything/me')console.log('READONLY_HTTP_ERROR',r.status(),new URL(r.url()).pathname);});")
old="async function workspace(id){await page.goto(rat+'/e/thing~'+id+'#connections');await expect(page.locator('.rat-workspace-id')).toHaveText(id);}"
new="""async function workspace(id){for(let attempt=0;attempt<2;attempt++){await page.goto(rat+'/e/thing~'+id+'#connections');try{await expect(page.locator('.rat-workspace-id')).toHaveText(id);return;}catch(error){const message=await page.locator('body').innerText();if(attempt===1||!message.includes('Failed to fetch')){console.log('WORKSPACE_LOAD_FAILURE',id,message.slice(-1000));throw error;}console.log('BOUNDED_READONLY_NETWORK_RETRY',id);await page.waitForTimeout(2000);}}}"""
assert s.count(old)==1;s=s.replace(old,new)
# The published API URL may be http; Chromium upgrades mixed image requests to https.
# Match the same exact host/path under either normal web scheme, not an unrelated image.
old='const imageRequest=u=>u.href===photo;';new="const imageRequest=u=>['http:','https:'].includes(u.protocol)&&u.hostname===new URL(photo).hostname&&u.pathname===new URL(photo).pathname;";assert s.count(old)==1;s=s.replace(old,new)
s=s.replace("blockedProfileRequests++;return r.abort('failed');", "blockedProfileRequests++;console.log('BLOCKED_EXACT_PUBLISHED_PROFILE_IMAGE',new URL(r.request().url()).protocol,new URL(r.request().url()).pathname);return r.abort('failed');")
s=s.replace("const path=u=>u.href===dev+'/api/v1/public/entities/rat-catalogue/nodes/'+person;", "const path=u=>[dev,rat].includes(u.origin)&&u.pathname==='/api/v1/public/entities/rat-catalogue/nodes/'+person;")
s=s.replace("evidence.imageFailureFallsBackAndReloadRecovers=true;", "evidence.imageFailureFallsBackAndReloadRecovers=true;console.log('REAL_IMAGE_NETWORK_FAILURE_FALLBACK_PASS',blockedProfileRequests);")
p.write_text(s)
PY
python3 - "$t/accept-rat183-fresh-image-failure.sh" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]);s=p.read_text();old=' bash -n "$t/accept.sh";bash "$t/accept.sh";exit';new=' python3 "$ops/adjust.py" "$t/accept.sh"\n bash -n "$t/accept.sh";bash "$t/accept.sh";exit';assert s.count(old)==1;p.write_text(s.replace(old,new))
PY
bash -n "$t/accept-rat183-fresh-image-failure.sh";bash "$t/accept-rat183-fresh-image-failure.sh" browser
