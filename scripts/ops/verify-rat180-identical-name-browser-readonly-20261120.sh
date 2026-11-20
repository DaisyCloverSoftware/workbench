#!/usr/bin/env bash
set -Eeuo pipefail
# Real Chromium against isolated DEV only. Never submits authentication, entities or ratings.
[[ $# -eq 4 ]] || exit 64
source="$1"; api="$2"; rat="$3"; run="$4"
[[ "$source" == '061beefd23bd71cf5902ff2a7724ff78acb515f6' ]] || exit 78
[[ "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$run" =~ ^rat180-[a-z0-9-]{8,34}$ ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
version="$(curl -fsS --retry 4 --max-time 20 https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')"
[[ "$version" == "rat-candidate+$source" ]] || exit 78
work="$(mktemp -d)";trap 'rm -rf "$work"' EXIT
out="/tmp/rat180-browser-$run"
[[ ! -e "$out" ]] || exit 65
mkdir -m 700 "$out"
git -C "$work" init -q
git -C "$work" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$work" fetch -q --depth=1 origin "$source"
git -C "$work" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$work" rev-parse HEAD)" == "$source" ]] || exit 78
cd "$work/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >"$work/npm.log" 2>&1 || { tail -n 25 "$work/npm.log";exit 1; }
npx playwright install chromium >"$work/browser.log" 2>&1 || { tail -n 25 "$work/browser.log";exit 1; }
REPORT_DIR="$out" REPORT_RUN="$run" SOURCE_SHA="$source" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import {chromium,expect} from '@playwright/test'
import {readFile,stat,writeFile} from 'node:fs/promises'
import {createHash} from 'node:crypto'
const RAT='https://dev-rum-ra.daisycloversoftware.uk'
const DEV='https://dev-rum.daisycloversoftware.uk'
const PERSON='01m0mw9hkbynssr7kbc0gk534j'
const KEY='community_788e6ae3bfcd391d111e3ca31a1332ebf92906d0d0868e7cf7bb'
const dir=process.env.REPORT_DIR
const report={status:'NOT_ASSESSED',source:process.env.SOURCE_SHA,run:process.env.REPORT_RUN,checks:[],screenshots:[],errors:[],personId:PERSON,channelId:null}
const pass=(x)=>{report.checks.push(x);console.log('RAT180_BROWSER_PASS='+x)}
const browser=await chromium.launch({headless:true,args:['--no-sandbox','--disable-dev-shm-usage']})
const context=await browser.newContext({viewport:{width:1440,height:940}})
const page=await context.newPage()
await page.route('**/*',route=>{
 const u=new URL(route.request().url())
 const method=route.request().method()
 if(!['GET','HEAD','OPTIONS'].includes(method)||!['dev-rum-ra.daisycloversoftware.uk','dev-rum.daisycloversoftware.uk'].includes(u.hostname))return route.abort('blockedbyclient')
 return route.continue()
})
const screenshot=async(width,label)=>{
 const name='colin-furze-'+label+'-'+width+'.jpg'
 const path=dir+'/'+name
 await page.screenshot({path,type:'jpeg',quality:69,fullPage:true,animations:'disabled'})
 const bytes=(await stat(path)).size
 assert.ok(bytes>5000&&bytes<900000,'Invalid screenshot size '+name+': '+bytes)
 const sha=createHash('sha256').update(await readFile(path)).digest('hex')
 report.screenshots.push({name,bytes,sha256:sha})
 console.log('RAT180_SCREENSHOT='+name+' bytes='+bytes+' sha256='+sha)
}
let stage='canonical-search'
try{
 const res=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/search?q=Colin%20Furze')
 assert.equal(res.status(),200)
 const data=await res.json()
 assert.equal(data.meta.audience,'rat-public')
 assert.equal(data.meta.privacy,'private-rum-people-excluded')
 assert.equal(data.data.active.id,PERSON)
 assert.equal(data.data.active.name,'Colin Furze')
 assert.equal(data.data.active.ratingOwner,'rum')
 const matches=data.meta.matches??[]
 assert.ok(matches.some(item=>item.id===PERSON),'Existing person missing')
 const channels=matches.filter(item=>item.name==='Colin Furze'&&item.entityType==='digital_resource'&&item.classifications?.some(c=>c.primary&&c.key===KEY))
 assert.ok(channels.length<=1,'Duplicate YouTube channel matches')
 const channel=channels[0]??null
 if(channel)report.channelId=channel.id
 pass('canonical-public-person-id-and-rum-rating-owner-preserved')
 for(const width of [1440,768,390]){
  stage='person-'+width
  await page.setViewportSize({width,height:width===390?850:940})
  await page.goto(RAT+'/search?q=Colin%20Furze&active='+PERSON,{waitUntil:'domcontentloaded',timeout:45000})
  await expect(page.locator('.rat-active-card h1')).toHaveText('Colin Furze',{timeout:30000})
  await expect(page.locator('.rat-active-card__actions').getByRole('button',{name:'Rate',exact:true})).toBeVisible()
  await expect(page.locator('.rat-active-card__actions').getByRole('button',{name:'See ratings',exact:true})).toBeVisible()
  await expect(page.getByRole('button',{name:'Add another Thing with this name'})).toBeVisible()
  assert.equal(new URL(page.url()).searchParams.get('active'),PERSON)
  assert.ok((await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth))<=2,'Horizontal overflow persona '+width)
  await screenshot(width,'persona')
  pass('person-'+width+'-selected-card-rating-controls-and-add-another')
 }
 stage='creation-dialog'
 await page.getByRole('button',{name:'Add another Thing with this name'}).click()
 await expect(page.getByRole('dialog',{name:'Add a Thing'})).toBeVisible()
 await expect(page.getByRole('dialog',{name:'Add a Thing'})).toContainText('Colin Furze')
 await page.getByRole('dialog',{name:'Add a Thing'}).getByRole('button',{name:'Close'}).click()
 pass('same-name-creation-dialog-with-no-write')
 if(!channel){
  report.status='PARTIAL_CHANNEL_NOT_PRESENT'
  console.log('RAT180_CHANNEL_NOT_YET_PRESENT=true')
 }else{
  stage='channel-node'
  const response=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+encodeURIComponent(channel.id))
  assert.equal(response.status(),200)
  const subject=(await response.json()).data.active
  assert.equal(subject.id,channel.id);assert.equal(subject.name,'Colin Furze');assert.equal(subject.ratingOwner,'rat')
  pass('separate-canonical-channel-id-and-rat-owner')
  for(const width of [1440,768,390]){
   stage='channel-'+width
   await page.setViewportSize({width,height:width===390?850:940})
   await page.goto(RAT+'/search?q=Colin%20Furze&active='+PERSON,{waitUntil:'domcontentloaded',timeout:45000})
   await expect(page.locator('.rat-active-card h1')).toHaveText('Colin Furze',{timeout:30000})
   const alternative=page.getByRole('region',{name:'Other matching Things'}).getByRole('button',{name:/Colin Furze/})
   await expect(alternative).toHaveCount(1)
   await expect(alternative).toContainText('YouTube Channel')
   await alternative.click()
   await expect(page.locator('.rat-active-card__type')).toHaveText('YouTube Channel',{timeout:30000})
   assert.equal(new URL(page.url()).searchParams.get('active'),channel.id)
   await page.reload({waitUntil:'domcontentloaded'})
   await expect(page.locator('.rat-active-card__type')).toHaveText('YouTube Channel',{timeout:30000})
   await expect(page.locator('.rat-active-card__actions').getByRole('button',{name:'Rate',exact:true})).toBeVisible()
   await expect(page.locator('.rat-active-card__actions').getByRole('button',{name:'See ratings',exact:true})).toBeVisible()
   assert.ok((await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth))<=2,'Horizontal overflow channel '+width)
   await screenshot(width,'channel')
   pass('channel-'+width+'-rebase-refresh-and-rating-controls')
  }
  report.status='PASS_READONLY'
 }
}catch(error){report.status='FAIL';report.errors.push({stage,message:String(error).slice(0,2200)});console.log('RAT180_BROWSER_ERROR='+JSON.stringify(report.errors.at(-1)));process.exitCode=1}
finally{await writeFile(dir+'/report.json',JSON.stringify(report,null,2));console.log('RAT180_BROWSER_REPORT='+JSON.stringify(report));await browser.close()}
NODE
