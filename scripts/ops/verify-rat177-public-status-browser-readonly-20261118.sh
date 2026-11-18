#!/usr/bin/env bash
set -Eeuo pipefail
# Real deployed CJ public header responsive check with Chromium.
# Strictly GET/HEAD/OPTIONS; no accounts, edits, ratings or voting mutations.
[[ $# -eq 4 ]] || exit 64
src="$1";api="$2";rat="$3";rid="$4"
[[ "$src" == '628296e5c57665814799bfedf1ab3ef8f6f163df' ]] || exit 78
[[ "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[a-f0-9]{64}$ ]] || exit 64
[[ "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[a-f0-9]{64}$ ]] || exit 64
[[ "$api" == 'ghcr.io/daisycloversoftware/rum-api@sha256:c68f9e4902b7ab71abab2cf0ea50244bf80dd93809385add84b764ff1fc34d0c' ]] || exit 78
[[ "$rat" == 'ghcr.io/daisycloversoftware/rum-rate-anything@sha256:6a506b08359eea423340c467ee6f330e22556c1a29002d0351323528123c8518' ]] || exit 78
[[ "$rid" =~ ^[a-z0-9-]{16,44}$ ]] || exit 64
out="/tmp/rat177-public-header-$rid"
[[ ! -e "$out" ]] || exit 65
mkdir -m 700 "$out"
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --max-time 20 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$src" ]] || exit 78
health="$(curl -fsS --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$health" "$src" <<'PY'
import json,sys
obj=json.loads(sys.argv[1]); assert obj["status"]=="ok" and obj["service"]=="rum-api"
assert sys.argv[2] in obj["version"]
PY
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
# Keep Chromium user-data profiles inside this writable isolated run directory.
export TMPDIR="$tmp"
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$src"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$src" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for p in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser;do
 if [[ -x "$p" ]];then chrome="$p";break;fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
CHROME="$chrome" READ_OUT="$out" READ_RUN="$rid" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import {writeFile,stat} from 'node:fs/promises'
import {chromium,expect} from '@playwright/test'
const SITE='https://dev-rum-ra.daisycloversoftware.uk',CJ='01m07d2f9fj05f4ypdbwp087qj'
const report={run:process.env.READ_RUN,status:'NOT_ASSESSED',checks:[],screenshots:[],errors:[]}
const pass=x=>{report.checks.push(x);console.log('RAT177_PUBLIC_HEADER_PASS='+x)}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const ctx=await browser.newContext({viewport:{width:1440,height:940}})
const page=await ctx.newPage()
await page.route('**/*',r=>['GET','HEAD','OPTIONS'].includes(r.request().method())?r.continue():r.abort('blockedbyclient'))
let stage='starting'
try{
 for(const width of [1440,768,390]){
  stage='header-'+width
  await page.setViewportSize({width,height:width===390?860:940})
  await page.goto(SITE+'/e/cj-investigates~'+CJ,{waitUntil:'domcontentloaded',timeout:40000})
  await expect(page.locator('.rat-workspace-header h1')).toHaveText('CJ Investigates',{timeout:30000})
  const score=page.locator('.rat-workspace-score')
  await expect(score.locator('.rum-rating__empty')).toBeVisible({timeout:25000})
  const visibleText=await score.innerText()
  const pending=(visibleText.match(/At least \d+ ratings? awaiting community verdict/g)||[]).length
  // Anonymous visitors may not receive community Judge counts. Confirm
  // the appropriate branded empty state rather than fabricating votes.
  assert.ok(pending===0||pending===1,'Duplicate awaiting verdict in header')
  if(pending===0){
   await expect(score.locator('.rum-rating__empty')).toHaveText('No accepted rating yet')
   await expect(score).toContainText('No established public reputation yet')
  } else {
   await expect(score.locator('.rum-rating__empty')).toContainText('awaiting community verdict')
   assert.equal((visibleText.match(/awaiting community verdict/g)||[]).length,1)
  }
  await expect(score.locator('img[alt="RUM Thumb"]')).toBeVisible()
  await expect(score.getByRole('button',{name:'Rate',exact:true})).toBeVisible()
  await expect(score.locator('a[href="#ratings"]')).toBeVisible()
  const overflow=await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)
  assert.ok(overflow<=2,'Horizontal header/page overflow '+width+'px: '+overflow)
  const name='cj-'+width+'-public-header.png',path=process.env.READ_OUT+'/'+name
  await page.locator('.rat-workspace-header').screenshot({path,animations:'disabled'})
  const bytes=(await stat(path)).size
  assert.ok(bytes>1500&&bytes<650000)
  report.screenshots.push({name,bytes})
  pass('CJ-'+width+'-one-branded-pending-or-appropriate-public-empty-state-no-overflow')
 }
 const context=await browser.newContext({viewport:{width:390,height:850}})
 const fresh=await context.newPage()
 await fresh.goto(SITE+'/e/cj-investigates~'+CJ,{waitUntil:'domcontentloaded'})
 await expect(fresh.locator('.rat-workspace-header h1')).toHaveText('CJ Investigates')
 await expect(fresh.locator('.rat-workspace-score .rum-rating__empty')).toBeVisible({timeout:25000})
 await context.close()
 pass('independent-public-browser-context-preserves-header')
 report.status='PASS'
 console.log('RAT177_PUBLIC_HEADER_REAL_BROWSER_PASS=true')
}catch(e){
 report.status='FAIL'
 report.errors.push({stage,message:String(e).slice(0,1100)})
 console.log('RAT177_PUBLIC_HEADER_BROWSER_ERROR='+JSON.stringify(report.errors[0]))
 process.exitCode=1
}finally{
 await writeFile(process.env.READ_OUT+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT177_PUBLIC_HEADER_REPORT='+JSON.stringify(report))
 await browser.close()
}
NODE