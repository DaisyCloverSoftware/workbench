#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only, unique no-match search and Add a Thing visibility on deployed RAT.
[[ $# -eq 3 ]] || exit 64
src="$1";api="$2";rat="$3"
[[ "$src" =~ ^[0-9a-f]{40}$ && "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ && "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --max-time 20 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$src" ]] || exit 78
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q;git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$src"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$src" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for p in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser;do
 [[ ! -x "$p" ]] || { chrome="$p";break; }
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
CHROME="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { chromium, expect } from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk'
const query='rat170-no-match-'+randomBytes(12).toString('hex')+'-unknown'
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
try{
 const context=await browser.newContext({viewport:{width:390,height:844}})
 const p=await context.newPage()
 const events=[]
 p.on('response',r=>{if(r.url().includes('rat-catalogue/search'))events.push({url:r.url(),status:r.status()})})
 const endpoint=DEV+'/api/v1/public/entities/rat-catalogue/search?q='+encodeURIComponent(query)
 const r=await context.request.get(endpoint)
 assert.equal(r.status(),200)
 const j=await r.json()
 console.log('RAT170_NO_MATCH_CANONICAL='+JSON.stringify({query,meta:j.meta,dataId:j.data?.active?.id||null}))
 assert.equal(j.meta.source,'rum-dev-canonical')
 assert.equal(j.meta.found,false,'Unexpected canonical match for random query')
 await p.goto(RAT+'/search?q='+encodeURIComponent(query),{waitUntil:'domcontentloaded',timeout:35000})
 try{
  await expect(p.locator('.rat-message')).toBeVisible({timeout:22000})
  await expect(p.locator('.rat-message button.rat-primary-action')).toHaveText('Add a Thing',{timeout:12000})
 }catch(e){
  console.log('RAT170_NO_MATCH_FAILURE_DETAIL='+JSON.stringify({url:p.url(),events,text:(await p.locator('body').innerText()).slice(0,2400),error:String(e).slice(0,700)}))
  throw e
 }
 console.log('RAT170_NO_MATCH_ADD_VISIBLE=true')
 await p.locator('.rat-message button.rat-primary-action').click()
 const title=await p.locator('body').innerText()
 assert.ok(/Add a Thing|Create.*Thing|Sign in|Create account/i.test(title))
 console.log('RAT170_NO_MATCH_ADD_ENTRY_OPENS=true')
 const types=await context.request.get(RAT+'/api/v1/rate-anything/thing-types?q=YouTube%20Channel')
 assert.equal(types.status(),200)
 assert.ok(JSON.stringify(await types.json()).toLowerCase().includes('youtube channel'))
 console.log('RAT170_YOUTUBE_CHANNEL_CATALOGUE_PASS=true')
 const overflow=await p.evaluate(()=>document.documentElement.scrollWidth-innerWidth)
 assert.ok(overflow<=2,'mobile no-match horizontal overflow '+overflow)
 console.log('RAT170_MOBILE_NO_MATCH_PASS=true')
}finally{await browser.close()}
NODE
