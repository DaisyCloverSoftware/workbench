#!/usr/bin/env bash
set -Eeuo pipefail
# Public read-only RAT174 no-image fallback visual check; no account, rating or media mutations.
[[ $# -eq 3 ]] || exit 64
src="$1";api="$2";rat="$3"
[[ "$src" == "cd1910ba808919d74f748456fd04202da052ddf6" ]] || exit 78
[[ "$api" == "ghcr.io/daisycloversoftware/rum-api@sha256:5fc614b352a596e41d35a33f55b9f4b01d4c6c2bfbe2d7dda700d92267761a13" ]] || exit 78
[[ "$rat" == "ghcr.io/daisycloversoftware/rum-rate-anything@sha256:31651a79f1bd59c8ff6c04a627e392a10b7756a7579633be81230cf96673ad7f" ]] || exit 78
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --max-time 20 https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$src" ]] || exit 78
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$src"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$src" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for p in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser;do
 if [[ -x "$p" ]]; then chrome="$p";break;fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
CHROME="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import {chromium,expect} from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk'
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
try{
 for(const c of [{q:'DJI',id:'01m784s4xmkdbfxxd3mzwhp991',name:'DJI'},{q:'CJ',id:'01m07d2f9fj05f4ypdbwp087qj',name:'CJ Investigates'}]){
  for(const width of c.q==='DJI'?[1440,390]:[1440]){
   const ctx=await browser.newContext({viewport:{width,height:900}})
   const nodeResponse=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+c.id)
   assert.equal(nodeResponse.status(),200)
   const payload=await nodeResponse.json(),node=payload.data
   assert.equal(payload.meta?.privacy,'private-rum-people-excluded')
   assert.equal(node.active.name,c.name)
   assert.equal(node.active.images?.profile??null,null,'Unexpected published or private main image')
   assert.ok(node.topLinked.length>0,'No linked fallback fixture')
   const page=await ctx.newPage()
   await page.goto(RAT+'/search?q='+c.q+'&active='+c.id,{waitUntil:'domcontentloaded',timeout:40000})
   await expect(page.locator('.rat-active-card h1')).toContainText(c.name,{timeout:30000})
   await expect(page.locator('.rat-active-card__avatar > img')).toHaveCount(0)
   await expect(page.locator('.rat-active-card__avatar')).toContainText(c.name.slice(0,2).toUpperCase())
   const cards=page.locator('.rat-top-linked .rat-linked-card')
   await expect(cards).toHaveCount(node.topLinked.length)
   for(let i=0;i<node.topLinked.length;i++){
    const item=node.topLinked[i],card=cards.nth(i)
    assert.equal(item.images?.profile??null,null,'Unexpected linked published image fixture')
    await expect(card.locator('.rat-linked-card__media > img')).toHaveCount(0)
    await expect(card.locator('.rat-linked-card__media')).toContainText(item.name.slice(0,2).toUpperCase())
   }
   assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)<=2,'Page horizontal overflow')
   console.log('RAT174_REAL_BROWSER_FALLBACK_PASS='+JSON.stringify({query:c.q,width,active:c.name,linked:node.topLinked.length,images:0,privacy:payload.meta.privacy}))
   if(c.q==='DJI'){
    const png=await page.locator('.rat-top-linked').screenshot({type:'png',animations:'disabled'})
    assert.ok(png.length>1000 && png.length<100000)
    console.log('RAT174_FALLBACK_LINKED_B64_'+width+'='+png.toString('base64'))
   }
   await ctx.close()
  }
 }
 console.log('RAT174_PUBLIC_INITIALS_FALLBACK_BROWSER_PASS=true')
}finally{await browser.close()}
NODE