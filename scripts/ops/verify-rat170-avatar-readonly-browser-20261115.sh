#!/usr/bin/env bash
set -Eeuo pipefail
# Strictly read-only visual check of the pre-existing disposable RAT QA entity.
[[ $# -eq 3 ]] || exit 64
src="$1"; api="$2"; rat="$3"
[[ "$src" =~ ^[0-9a-f]{40}$ && "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ && "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --max-time 20 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$src" ]] || exit 78
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
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
CHROME="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { chromium, expect } from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk', DEV='https://dev-rum.daisycloversoftware.uk'
const id='01m7fvzcen292rvqeqj03evtxe'
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
try{
 const ctx=await browser.newContext({viewport:{width:390,height:844}})
 const p=await ctx.newPage()
 const events=[]
 p.on('response', r=>{
   if(r.url().includes('rat-catalogue/nodes/'+id))events.push({status:r.status(),url:r.url()})
 })
 p.on('console',msg=>{if(msg.type()==='error')events.push({console:msg.text().slice(0,350)})})
 const canonical=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+id)
 assert.equal(canonical.status(),200)
 const node=(await canonical.json()).data.active
 assert.ok(node.images?.profile?.url,'No current published profile in canonical DEV')
 const src=node.images.profile.url
 console.log('RAT170_AVATAR_CANONICAL_CURRENT='+src)
 await p.goto(RAT+'/e/qa~'+id,{waitUntil:'domcontentloaded'})
 await expect(p.locator('.rat-workspace-header h1')).toBeVisible({timeout:30000})
 const selector='.rat-workspace-avatar > img'
 let img=await p.locator(selector).count()
 let loading=await p.locator('.rat-workspace-loading').allTextContents()
 console.log('RAT170_AVATAR_FIRST_RENDER='+JSON.stringify({images:img,loading,events}).slice(0,1500))
 if(!img){await p.reload({waitUntil:'domcontentloaded'});await expect(p.locator('.rat-workspace-header h1')).toBeVisible({timeout:30000});img=await p.locator(selector).count()}
 const finalSrc=img?await p.locator(selector).getAttribute('src'):null
 console.log('RAT170_AVATAR_FINAL_RENDER='+JSON.stringify({images:img,src:finalSrc,events}).slice(0,2400))
 assert.ok(finalSrc&&finalSrc.includes('/api/v1/public/entities/rat-image/'),'Public avatar absent on refreshed page')
 assert.equal(finalSrc,src,'DOM avatar differs from canonical public image')
 await expect.poll(async()=>p.locator(selector).evaluate(el=>el.complete&&el.naturalWidth>0),{timeout:16000}).toBe(true)
 const resp=await ctx.request.get(finalSrc)
 assert.equal(resp.status(),200)
 assert.ok((resp.headers()['content-type']||'').includes('image/webp'))
 console.log('RAT170_AVATAR_READONLY_BROWSER_PASS=true')
}finally{await browser.close()}
NODE
