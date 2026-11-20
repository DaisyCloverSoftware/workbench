#!/usr/bin/env bash
set -Eeuo pipefail
# Independent read-only functional Chromium assertions. No screenshot or report
# file is generated. Do not modify the existing screenshot recorder.
[[ $# -eq 1 && "$1" == "061beefd23bd71cf5902ff2a7724ff78acb515f6" ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
api='ghcr.io/daisycloversoftware/rum-api@sha256:7411cc02eb6568e6c6d228b551683294bfac49a6f89c320078af5550adce5d56'
rat='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:9a33ea315f52d5e14a5ece0e57895aae8d4eb4d7d7e9955110d49fc9676c64d9'
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --max-time 25 https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$1" ]] || exit 78
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$1"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$1" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >"$tmp/npm.log" 2>&1 || { tail -n 12 "$tmp/npm.log";exit 1; }
npm_config_cache="$tmp/npm-cache" npx playwright install chromium >"$tmp/browser.log" 2>&1 || { tail -n 12 "$tmp/browser.log";exit 1; }
node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import {chromium,expect} from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk'
const DEV='https://dev-rum.daisycloversoftware.uk'
const P='01m0mw9hkbynssr7kbc0gk534j', C='01m7x6ez57zpsgbv486y9t0y9h'
const REL='01m7x7kn332fqjxyj9b78zrmh0'
const log=[]
const pass=x=>{log.push(x);console.log('RAT180_NAV_PASS='+x)}
let browser
try {
 browser=await chromium.launch({headless:true,args:['--no-sandbox','--disable-dev-shm-usage']})
 const context=await browser.newContext({viewport:{width:1440,height:940}})
 const page=await context.newPage()
 await page.route('**/*',route=>{
   const req=route.request(),url=new URL(req.url())
   if(['GET','HEAD','OPTIONS'].includes(req.method())&&['dev-rum-ra.daisycloversoftware.uk','dev-rum.daisycloversoftware.uk'].includes(url.hostname))return route.continue()
   return route.abort('blockedbyclient')
 })
 const response=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/search?q=Colin%20Furze')
 assert.equal(response.status(),200)
 const info=await response.json()
 assert.equal(info.data.active.id,P)
 assert.ok(info.meta.matches.some(x=>x.id===P&&x.ratingOwner==='rum'))
 assert.ok(info.meta.matches.some(x=>x.id===C&&x.ratingOwner==='rat'&&x.name==='Colin Furze'))
 pass('canonical-same-name-distinct-ids-and-rating-owners')
 const url=id=>RAT+'/search?q=Colin%20Furze&active='+id
 const selected=async(id,label)=>{
   await expect(page.locator('.rat-active-card h1')).toHaveText('Colin Furze',{timeout:30000})
   await expect(page.locator('.rat-active-card__type')).toHaveText(label,{timeout:30000})
   assert.equal(new URL(page.url()).searchParams.get('active'),id)
   await expect(page.locator('.rat-active-card__actions').getByRole('button',{name:'Rate',exact:true})).toBeVisible()
   await expect(page.locator('.rat-active-card__actions').getByRole('button',{name:'See ratings',exact:true})).toBeVisible()
   assert.ok(await page.locator('.rat-active-card .rum-rating img[alt="RUM Thumb"]').count()>0,'No RUM thumb rating controls')
 }
 const visit=async address=>{
   for(let attempt=1;attempt<=3;attempt++){
     try{return await page.goto(address,{waitUntil:'domcontentloaded',timeout:45000})}
     catch(e){if(attempt===3||!String(e).includes('ERR_NETWORK_CHANGED'))throw e;await page.waitForTimeout(1200)}
   }
 }
 for(const width of [1440,768,390]){
   await page.setViewportSize({width,height:width===390?850:940})
   await visit(url(P))
   await selected(P,'YouTube creator')
   await expect(page.getByRole('button',{name:'Add another Thing with this name'})).toBeVisible()
   const other=page.getByRole('region',{name:'Other matching Things'}).getByRole('button',{name:/Colin Furze/})
   await expect(other).toHaveCount(1)
   await expect(other).toContainText('YouTube Channel')
   await other.click()
   await selected(C,'YouTube Channel')
   pass('width-'+width+'-select-channel-as-main-by-canonical-id')
   await page.reload({waitUntil:'domcontentloaded'})
   await selected(C,'YouTube Channel')
   pass('width-'+width+'-channel-refresh-persists')
   await page.goBack({waitUntil:'domcontentloaded'})
   await selected(P,'YouTube creator')
   await page.goForward({waitUntil:'domcontentloaded'})
   await selected(C,'YouTube Channel')
   pass('width-'+width+'-browser-back-forward-keeps-canonical-ids')
   await visit(url(P))
   await selected(P,'YouTube creator')
   const linked=page.locator('.rat-top-linked .rat-linked-card, .rat-category .rat-linked-card').filter({hasText:'Colin Furze'}).filter({hasText:'YouTube Channel'}).first()
   await expect(linked).toBeVisible({timeout:30000})
   await linked.click()
   await selected(C,'YouTube Channel')
   pass('width-'+width+'-person-linked-channel-rebases')
   assert.ok((await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth))<=2,'Horizontal scroll overflow '+width)
   pass('width-'+width+'-no-overflow-native-thumbs-and-branding-preserved')
 }
 await visit(url(C))
 await selected(C,'YouTube Channel')
 await page.reload({waitUntil:'domcontentloaded'})
 await selected(C,'YouTube Channel')
 pass('direct-channel-canonical-url-and-refresh')
 await visit(url(P))
 await selected(P,'YouTube creator')
 await page.reload({waitUntil:'domcontentloaded'})
 await selected(P,'YouTube creator')
 pass('direct-person-canonical-url-and-refresh')
 const personNode=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+P)
 assert.equal(personNode.status(),200)
 const pn=await personNode.json()
 assert.ok(JSON.stringify(pn.data).includes(REL),'Link edge not in canonical node')
 pass('public-relationship-provenance-id-in-canonical-node')
 console.log('RAT180_NAV_SUMMARY='+JSON.stringify({status:'PASS',steps:log,personId:P,channelId:C,relationshipId:REL,viewports:[1440,768,390]}))
} catch(e) {
 console.log('RAT180_NAV_FAILURE='+String(e).slice(0,2600))
 process.exitCode=1
} finally {if(browser)await browser.close()}
NODE
