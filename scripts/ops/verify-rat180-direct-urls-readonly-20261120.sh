#!/usr/bin/env bash
set -Eeuo pipefail
# Isolated real Chromium direct-route diagnostic, read-only, no screenshot output.
[[ $# -eq 1 && "$1" == '061beefd23bd71cf5902ff2a7724ff78acb515f6' ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:7411cc02eb6568e6c6d228b551683294bfac49a6f89c320078af5550adce5d56' ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == 'ghcr.io/daisycloversoftware/rum-rate-anything@sha256:9a33ea315f52d5e14a5ece0e57895aae8d4eb4d7d7e9955110d49fc9676c64d9' ]] || exit 78
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q;git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$1";git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$1" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >"$tmp/npm.log" 2>&1 || { tail -n 12 "$tmp/npm.log";exit 1; }
npx playwright install chromium >"$tmp/chromium.log" 2>&1 || { tail -n 12 "$tmp/chromium.log";exit 1; }
node --input-type=module <<'NODE'
import {chromium,expect} from '@playwright/test'
import assert from 'node:assert/strict'
const RAT='https://dev-rum-ra.daisycloversoftware.uk'
const P='01m0mw9hkbynssr7kbc0gk534j'
const C='01m7x6ez57zpsgbv486y9t0y9h'
const results=[]
let browser
try {
 browser=await chromium.launch({headless:true,args:['--no-sandbox','--disable-dev-shm-usage']})
 const context=await browser.newContext({viewport:{width:1440,height:940}})
 const page=await context.newPage()
 await page.route('**/*',r=>{
   const q=r.request(),u=new URL(q.url())
   if(['GET','HEAD','OPTIONS'].includes(q.method())&&['dev-rum-ra.daisycloversoftware.uk','dev-rum.daisycloversoftware.uk'].includes(u.hostname))return r.continue()
   return r.abort('blockedbyclient')
 })
 const failed=[]
 page.on('response',r=>{if(r.status()>=400&&new URL(r.url()).pathname.startsWith('/api/'))failed.push({status:r.status(),path:new URL(r.url()).pathname})})
 const probe=async(id,label)=>{
   const route=RAT+'/search?q=Colin%20Furze&active='+id
   for(let attempt=1;attempt<=2;attempt++){
     await page.goto(route,{waitUntil:'domcontentloaded',timeout:45000})
     try{
       await expect(page.locator('.rat-active-card h1')).toHaveText('Colin Furze',{timeout:20000})
       await expect(page.locator('.rat-active-card__type')).toHaveText(label)
       assert.equal(new URL(page.url()).searchParams.get('active'),id)
       await expect(page.locator('.rat-active-card__actions').getByRole('button',{name:'Rate',exact:true})).toBeVisible()
       await expect(page.locator('.rat-active-card__actions').getByRole('button',{name:'See ratings',exact:true})).toBeVisible()
       console.log('RAT180_DIRECT_PASS='+id+' '+label)
       results.push('direct-'+id)
       await page.reload({waitUntil:'domcontentloaded',timeout:45000})
       await expect(page.locator('.rat-active-card__type')).toHaveText(label,{timeout:20000})
       console.log('RAT180_REFRESH_PASS='+id+' '+label)
       results.push('refresh-'+id)
       return
     }catch(e){
       const m=(await page.locator('.rat-message').allTextContents()).join(' ').slice(0,360)
       console.log('RAT180_DIRECT_DIAG='+JSON.stringify({id,attempt,url:page.url(),message:m,failedResponses:failed.slice(-8),error:String(e).slice(0,240)}))
       if(attempt===2)throw e
       await page.waitForTimeout(3000)
     }
   }
 }
 await probe(C,'YouTube Channel')
 await probe(P,'YouTube creator')
 console.log('RAT180_DIRECT_SUMMARY='+JSON.stringify({status:'PASS',results,failedResponses:failed}))
}catch(e){console.log('RAT180_DIRECT_FAILURE='+String(e).slice(0,1800));process.exitCode=1}
finally{if(browser)await browser.close()}
NODE
