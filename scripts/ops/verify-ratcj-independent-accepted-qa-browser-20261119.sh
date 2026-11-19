#!/usr/bin/env bash
set -Eeuo pipefail
# Real read-only accepted-public-rating browser proof, separately scoped DEV QA Thing, no CJ writes.
[[ $# -eq 5 ]] || exit 64
src="$1";api="$2";rat="$3";run="$4";entityId="$5"
[[ "$src" =~ ^[0-9a-f]{40}$ && "$entityId" =~ ^[0-9a-z]{26}$ ]] || exit 64
[[ "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$run" =~ ^ratcj-public-accepted-qa-browser-20261119-r[0-9]+$ ]] || exit 64
dir="/tmp/$run";[[ ! -e "$dir" ]] || exit 65;mkdir -m 700 "$dir"
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --retry 3 --max-time 20 https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$src" ]] || exit 78
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$src"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$src" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for p in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser;do if [[ -x "$p" ]];then chrome="$p";break;fi;done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
CHROME="$chrome" REPORT_DIR="$dir" REPORT_RUN="$run" SOURCE_SHA="$src" QA_ENTITY="$entityId" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import {chromium,expect} from '@playwright/test'
import {stat,readFile,writeFile} from 'node:fs/promises'
import {createHash} from 'node:crypto'
const RAT='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk'
const QA=process.env.QA_ENTITY
const NAME='RAT Public Accepted Rating QA 20261119'
const REVIEW='Isolated DEV accepted-rating card verification only.'
const report={status:'NOT_ASSESSED',run:process.env.REPORT_RUN,source:process.env.SOURCE_SHA,qaEntity:QA,checks:[],screenshots:[],errors:[]}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
try{
 const ctx=await browser.newContext({viewport:{width:1440,height:940}})
 const nodeResponse=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+QA,{headers:{'Cache-Control':'no-cache'}})
 assert.equal(nodeResponse.status(),200)
 const node=(await nodeResponse.json()).data
 assert.equal(node.active.name,NAME)
 assert.equal(node.active.id,QA)
 assert.equal(node.active.reputation?.ratingCount,1,'Canonical accepted-only public reputation count')
 const ratingsResponse=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+QA+'/ratings',{headers:{'Cache-Control':'no-cache',Origin:RAT}})
 assert.equal(ratingsResponse.status(),200)
 const data=(await ratingsResponse.json()).data
 assert.deepEqual(data.counts,{publicSubmitted:1,accepted:1,awaitingVerdict:0,inReview:0})
 assert.equal(data.items.length,1)
 const card=data.items[0],ratingId=card.id
 assert.equal(card.value,4)
 assert.equal(card.review,REVIEW)
 assert.deepEqual(card.verdict,{state:'accepted',keepVotes:5,sinBinVotes:0,score:5,countsTowardAccepted:true})
 report.qaRating=ratingId
 report.checks.push('canonical public accepted rating API count 1, card 1, five real Judge Keep votes')
 const page=await ctx.newPage()
 const pageErrors=[],httpErrors=[],authErrors=[]
 page.on('pageerror',e=>pageErrors.push(e.message))
 page.on('console',m=>{if(m.type()==='error')pageErrors.push(m.text())})
 page.on('response',r=>{if(r.status()===401)authErrors.push({method:r.request().method(),url:r.url()});else if(r.status()>=400)httpErrors.push({status:r.status(),url:r.url()})})
 await page.route('**/*',r=>['GET','HEAD','OPTIONS'].includes(r.request().method())?r.continue():r.abort('blockedbyclient'))
 const url=RAT+'/e/rat-pr179-public-accepted-qa-20261119~'+QA+'#ratings'
 for(const width of [1440,768,390]){
  await page.setViewportSize({width,height:width===390?850:940})
  for(const stage of ['initial','refreshed']){
   if(stage==='initial')await page.goto(url,{waitUntil:'domcontentloaded',timeout:45000})
   else await page.reload({waitUntil:'domcontentloaded',timeout:45000})
   await expect(page.locator('.rat-workspace-identity h1')).toHaveText(NAME,{timeout:35000})
   const panel=page.locator('#ratings')
   await expect(panel.locator('.rat-public-rating-card')).toHaveCount(1,{timeout:30000})
   await expect(panel.locator('.rat-public-rating-card')).toContainText(REVIEW)
   await expect(panel.locator('.rat-public-rating-card')).toContainText('4/5')
   await expect(panel.locator('.rat-public-rating-card')).toContainText('Accepted into public reputation')
   await expect(panel.locator('.rat-public-rating-card')).toContainText('Keep 5')
   await expect(panel).toContainText('1 accepted')
   await expect(panel).toContainText('0 awaiting community verdict')
   await expect(panel).toContainText('Counted in reputation')
   await expect(panel).toContainText('Accepted public ratings')
   await expect(page).toHaveURL(/#ratings$/)
   await expect.poll(()=>panel.evaluate(el=>{const rect=el.getBoundingClientRect();return rect.top >= -6 && rect.top < innerHeight}),{timeout:20000}).toBe(true)
   const spill=await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)
   assert.ok(spill<=2,'Horizontal overflow '+width)
  }
  const file='ratcj-accepted-qa-'+width+'-refreshed.jpg'
  const filepath=process.env.REPORT_DIR+'/'+file
  await page.screenshot({path:filepath,type:'jpeg',quality:68,fullPage:true,animations:'disabled'})
  const bytes=(await stat(filepath)).size
  assert.ok(bytes>12000&&bytes<950000)
  const sha256=createHash('sha256').update(await readFile(filepath)).digest('hex')
  report.screenshots.push({name:file,width,bytes,sha256})
  report.checks.push('accepted public card '+width+'px direct deep-link, refreshed, visible and non-overflow')
  console.log('RATCJ_ACCEPTED_QA_SCREENSHOT='+JSON.stringify({name:file,bytes,sha256}))
 }
 assert.deepEqual(httpErrors,[])
 assert.ok(authErrors.every(x=>x.method==='GET'&&x.url===RAT+'/api/v1/rate-anything/me'))
 const expected='Failed to load resource: the server responded with a status of 401 ()'
 assert.equal(pageErrors.filter(x=>x===expected).length,authErrors.length)
 assert.deepEqual(pageErrors.filter(x=>x!==expected),[])
 report.status='PASS'
 await writeFile(process.env.REPORT_DIR+'/report.json',JSON.stringify(report,null,2))
 await ctx.close()
 console.log('RATCJ_ACCEPTED_QA_BROWSER_PASS='+JSON.stringify({widths:[1440,768,390],screenshots:3,checks:report.checks.length,qaEntity:QA,acceptedRating:ratingId}))
}catch(e){report.status='FAIL';report.errors.push(String(e));await writeFile(process.env.REPORT_DIR+'/report.json',JSON.stringify(report,null,2));throw e}
finally{await browser.close()}
NODE
echo RATCJ_ACCEPTED_QA_BROWSER_COMPLETED=true
