#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only corrected mobile RAT search screenshot, must wait for published artwork.
[[ $# -eq 1 && "$1" == 'rat177-mobile-search-loaded-20261118-r3' ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == 'ghcr.io/daisycloversoftware/rum-rate-anything@sha256:ec11bfec432ff10d508850a0a071b3af3787bff656d2decca90eed5951cd0003' ]] || exit 78
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:c68f9e4902b7ab71abab2cf0ea50244bf80dd93809385add84b764ff1fc34d0c' ]] || exit 78
[[ "$(curl -fsS --retry 3 --max-time 20 https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == 'rat-candidate+887b5e02f4be6ec7e52371666effa9eb47892d6e' ]] || exit 78
out="/tmp/rat177-cj-search-avatar-$1"
[[ ! -e "$out" ]] || exit 65
mkdir -m 700 "$out"
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
export TMPDIR="$tmp"
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin 887b5e02f4be6ec7e52371666effa9eb47892d6e
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == '887b5e02f4be6ec7e52371666effa9eb47892d6e' ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for p in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do if [[ -x "$p" ]];then chrome="$p";break;fi;done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
SCREEN_OUT="$out" CHROME="$chrome" node --input-type=module <<'NODE'
import {chromium,expect} from '@playwright/test'
import assert from 'node:assert/strict'
import {writeFile,stat} from 'node:fs/promises'
const dir=process.env.SCREEN_OUT
const report={run:'rat177-mobile-search-loaded-20261118-r3',status:'NOT_ASSESSED',checks:[],screenshots:[],errors:[]}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
try{
 const ctx=await browser.newContext({viewport:{width:390,height:850}})
 const page=await ctx.newPage()
 await page.route('**/*',r=>['GET','HEAD','OPTIONS'].includes(r.request().method())?r.continue():r.abort('blockedbyclient'))
 await page.goto('https://dev-rum-ra.daisycloversoftware.uk/search?q=cj&active=01m07d2f9fj05f4ypdbwp087qj',{waitUntil:'domcontentloaded',timeout:45000})
 await expect(page.locator('.rat-active-card h1')).toHaveText('CJ Investigates',{timeout:30000})
 await expect(page.locator('.rat-active-card__type')).toHaveText('YouTube Channel')
 const avatar=page.locator('.rat-active-card__avatar img')
 await expect(avatar).toBeVisible({timeout:25000})
 await expect.poll(()=>avatar.evaluate(el=>el.complete&&el.naturalWidth>0&&el.naturalHeight>0),{timeout:30000}).toBe(true)
 assert.ok((await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth))<=2,'mobile search overflow')
 const filename='cj-390-search-full.jpg'
 await page.screenshot({path:dir+'/'+filename,type:'jpeg',quality:70,fullPage:true,animations:'disabled'})
 const bytes=(await stat(dir+'/'+filename)).size
 assert.ok(bytes>8000&&bytes<800000)
 report.screenshots.push({name:filename,bytes})
 report.checks.push('new-exact-head-channel-label-and-published-search-profile-picture-loaded-no-mobile-overflow')
 report.status='PASS'
 console.log('RAT177_MOBILE_SEARCH_LOADED_SCREENSHOT_PASS=true')
}catch(e){report.status='FAIL';report.errors.push(String(e).slice(0,1200));process.exitCode=1;console.log('RAT177_MOBILE_SEARCH_LOADED_SCREENSHOT_FAIL='+report.errors[0])}
finally{await writeFile(dir+'/report.json',JSON.stringify(report,null,2));await browser.close();console.log('RAT177_MOBILE_SEARCH_REPORT='+JSON.stringify(report))}
NODE
