#!/usr/bin/env bash
set -Eeuo pipefail
# Real Chromium 1440/768/390 acceptance of canonical pending/accepted RAT rating cards; GET-only.
[[ $# -eq 4 ]] || exit 64
src="$1"; api="$2"; rat="$3"; run="$4"
[[ "$src" =~ ^[0-9a-f]{40}$ ]] || exit 64
[[ "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$run" =~ ^ratcj-rating-final-20261118-r[0-9]+$ ]] || exit 64
out="/tmp/$run";[[ ! -e "$out" ]] || exit 65;mkdir -m 700 "$out"
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --retry 3 --max-time 25 https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$src" ]] || exit 78
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
CHROME="$chrome" REPORT_DIR="$out" REPORT_RUN="$run" SOURCE_SHA="$src" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import {chromium,expect} from '@playwright/test'
import {createHash} from 'node:crypto'
import {writeFile,readFile,stat} from 'node:fs/promises'
const RAT='https://dev-rum-ra.daisycloversoftware.uk'
const DEV='https://dev-rum.daisycloversoftware.uk'
const CJ='01m07d2f9fj05f4ypdbwp087qj'
const KEEP='01m0mvrxv8c3s12q0w7gwmzt03'
const REVIEW="I have been watching CJ's progress over the years and he is starting to make some great content."
const report={status:'NOT_ASSESSED',run:process.env.REPORT_RUN,source:process.env.SOURCE_SHA,entityId:CJ,originalRating:KEEP,checks:[],screenshots:[],issues:[]}
const pass=(x)=>{report.checks.push(x);console.log('RATCJ_FINAL_PUBLIC_RATING_PASS='+x)}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
try {
 const context=await browser.newContext({viewport:{width:1440,height:940}})
 const page=await context.newPage()
 const errors=[],httpFailures=[],anonymous=[]
 page.on('pageerror',e=>errors.push(e.message))
 page.on('console',m=>{if(m.type()==='error')errors.push(m.text())})
 page.on('response',r=>{
  if(r.status()===401)anonymous.push({url:r.url(),method:r.request().method()})
  else if(r.status()>=400)httpFailures.push({status:r.status(),url:r.url(),method:r.request().method()})
 })
 await page.route('**/*',r=>['GET','HEAD','OPTIONS'].includes(r.request().method())?r.continue():r.abort('blockedbyclient'))
 const c=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+CJ,{headers:{'Cache-Control':'no-cache'}})
 assert.equal(c.status(),200)
 const n=(await c.json()).data
 assert.equal(n.active.id,CJ);assert.equal(n.active.name,'CJ Investigates')
 assert.ok(n.active.classifications.some(x=>x.primary&&x.name==='YouTube Channel'))
 assert.deepEqual(n.categories,[]);assert.deepEqual(n.topLinked,[])
 assert.equal(n.active.reputation,null,'Awaiting Judge vote must not appear as a public RUM score')
 assert.ok(n.active.images?.profile?.url && n.active.images?.cover?.url)
 const rr=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+CJ+'/ratings',{headers:{'Cache-Control':'no-cache','Origin':RAT}})
 assert.equal(rr.status(),200)
 const rating=(await rr.json()).data
 assert.deepEqual(rating.counts,{publicSubmitted:1,accepted:0,awaitingVerdict:1,inReview:0})
 assert.equal(rating.items.length,1)
 const item=rating.items[0]
 assert.equal(item.id,KEEP);assert.equal(item.value,5);assert.equal(item.review,REVIEW)
 assert.equal(item.verdict.state,'needs_verdict')
 assert.equal(item.verdict.keepVotes,0);assert.equal(item.verdict.sinBinVotes,0)
 assert.equal(item.verdict.score,0);assert.equal(item.verdict.countsTowardAccepted,false)
 pass('canonical-RAT-source-single-pending-owner-rating-no-public-reputation-unaccepted')
 const slug=n.active.slug?.trim()||'thing'
 const detail=RAT+'/e/'+encodeURIComponent(slug)+'~'+encodeURIComponent(CJ)
 const search=RAT+'/search?q=CJ%20Investigates&active='+CJ
 const decoded=async selector=>{
  const img=page.locator(selector).first()
  await expect(img).toBeVisible({timeout:30000})
  await expect.poll(()=>img.evaluate(e=>e.complete&&e.naturalWidth>0&&e.naturalHeight>0),{timeout:25000}).toBe(true)
  await img.evaluate(e=>e.decode())
 }
 const capture=async (width,kind)=>{
  const name='cj-rating-clean-'+width+'-'+kind+'.jpg'
  const dest=process.env.REPORT_DIR+'/'+name
  await page.screenshot({path:dest,type:'jpeg',quality:68,fullPage:true,animations:'disabled'})
  const bytes=(await stat(dest)).size
  assert.ok(bytes>12000 && bytes<950000,name+' unexpected bytes='+bytes)
  const sha256=createHash('sha256').update(await readFile(dest)).digest('hex')
  report.screenshots.push({name,bytes,sha256,width,kind})
  console.log('RATCJ_FINAL_RATING_CAPTURE='+JSON.stringify({name,bytes,sha256}))
 }
 const noHorizontalOverflow=async width=>{
  const spill=await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)
  assert.ok(spill<=2,'Horizontal overflow width '+width+': '+spill)
 }
 const searchCheck=async width=>{
  await expect(page.locator('.rat-active-card h1')).toHaveText('CJ Investigates',{timeout:35000})
  await expect(page.locator('.rat-active-card__type')).toHaveText('YouTube Channel')
  await expect(page.locator('.rat-top-linked')).toHaveCount(0)
  await expect(page.locator('section.rat-category:not(.rat-category--empty)')).toHaveCount(0)
  await expect(page.locator('.rat-active-card__actions').getByRole('button',{name:'See ratings'})).toBeVisible()
  await expect(page.locator('.rat-active-card .rum-rating')).toContainText('Not rated yet')
  await decoded('.rat-active-card__avatar img')
  const text=await page.locator('body').innerText()
  assert.ok(!text.includes('testing1234')&&!text.includes('Quillstone')&&!text.includes('3.3'), 'Unwanted search text')
  await noHorizontalOverflow(width)
 }
 const detailCheck=async width=>{
  await expect(page.locator('.rat-workspace-identity h1')).toHaveText('CJ Investigates',{timeout:35000})
  await expect(page.locator('.rat-workspace-kickers')).toContainText('YouTube Channel')
  await expect(page.locator('#ratings h2')).toHaveText('Reputation snapshot')
  await expect(page.locator('#ratings .rat-workspace-facts')).toContainText('Counted in reputation')
  await expect(page.locator('#ratings .rat-workspace-facts')).toContainText('Accepted public ratings')
  const panel=page.locator('#ratings .rat-public-ratings')
  await expect(panel.locator('.rat-public-rating-card')).toHaveCount(1,{timeout:35000})
  const card=panel.locator('.rat-public-rating-card').first()
  await expect(card).toContainText('5/5')
  await expect(card).toContainText(REVIEW)
  await expect(card).toContainText('Awaiting community verdict')
  await expect(card).toContainText('Keep 0')
  await expect(card).toContainText('Sin Bin 0')
  await expect(card).toContainText('Net 0')
  await expect(panel).toContainText('0 accepted')
  await expect(panel).toContainText('1 awaiting community verdict')
  await expect(page.locator('.rat-workspace-score .rum-rating')).toContainText('awaiting community verdict')
  await expect(page.locator('#connections .rat-workspace-connection-group')).toHaveCount(0)
  await decoded('.rat-workspace-avatar img')
  await decoded('.rat-workspace-cover img')
  const t=await page.locator('body').innerText()
  assert.ok(!t.includes('testing1234')&&!t.includes('Quillstone')&&!t.includes('3.3'))
  await noHorizontalOverflow(width)
 }
 for(const width of [1440,768,390]){
  await page.setViewportSize({width,height:width===390?850:940})
  await page.goto(search,{waitUntil:'domcontentloaded',timeout:45000})
  await searchCheck(width)
  await page.reload({waitUntil:'domcontentloaded',timeout:45000})
  await searchCheck(width)
  await capture(width,'search-refreshed')
  pass('search-'+width+'-no-ineligible-score-or-test-links-and-refresh')
  await page.locator('.rat-active-card__actions').getByRole('button',{name:'See ratings'}).click()
  await expect(page).toHaveURL(/#ratings$/)
  await detailCheck(width)
  const visibleRatings = () => page.locator('#ratings').evaluate(el => {
   const r=el.getBoundingClientRect()
   return r.top >= -3 && r.top < innerHeight
  })
  await expect.poll(visibleRatings,{timeout:25000}).toBe(true)
  await page.reload({waitUntil:'domcontentloaded',timeout:45000})
  await detailCheck(width)
  await expect.poll(visibleRatings,{timeout:25000}).toBe(true)
  await expect(page.locator('#ratings .rat-public-rating-card')).toHaveCount(1)
  await capture(width,'thing-see-ratings-refreshed')
  pass('thing-'+width+'-one-5of5-review-0votes-pending-visible-and-refresh')
 }
 // A new anonymous browser context must independently resolve the
 // direct ratings deep-link with no cached login/session or prior React state.
 const freshContext=await browser.newContext({viewport:{width:390,height:850}})
 const freshPage=await freshContext.newPage()
 freshPage.on('pageerror',e=>errors.push(e.message))
 freshPage.on('console',m=>{if(m.type()==='error')errors.push(m.text())})
 freshPage.on('response',r=>{
  if(r.status()===401)anonymous.push({method:r.request().method(),url:r.url()})
  else if(r.status()>=400)httpFailures.push({status:r.status(),url:r.url(),method:r.request().method()})
 })
 await freshPage.route('**/*',r=>['GET','HEAD','OPTIONS'].includes(r.request().method())?r.continue():r.abort('blockedbyclient'))
 await freshPage.goto(detail+'#ratings',{waitUntil:'domcontentloaded',timeout:45000})
 await expect(freshPage.locator('.rat-workspace-identity h1')).toHaveText('CJ Investigates',{timeout:35000})
 await expect(freshPage.locator('#ratings .rat-public-rating-card')).toHaveCount(1,{timeout:35000})
 await expect(freshPage.locator('#ratings .rat-public-rating-card')).toContainText(REVIEW)
 await expect(freshPage.locator('#ratings .rat-public-ratings')).toContainText('0 accepted')
 await expect(freshPage.locator('#ratings .rat-public-ratings')).toContainText('1 awaiting community verdict')
 await expect.poll(()=>freshPage.locator('#ratings').evaluate(el=>{const b=el.getBoundingClientRect();return b.top>=-3 && b.top<innerHeight}),{timeout:25000}).toBe(true)
 await freshContext.close()
 pass('fresh-anonymous-context-direct-ratings-anchor-pending-original-visible')
 // Previously accepted unrelated channel grouping and branded thumbnail still hold.
 const other=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/01m7bswfh0a3evf271ye4taaf0')
 assert.equal(other.status(),200)
 const otherNode=(await other.json()).data
 assert.ok(otherNode.categories.some(c=>c.key==='channels'&&c.items.some(i=>i.id==='01m7jss48cpff8c59j6mdbbb7f')))
 pass('unrelated-professional-channel-category-intact')
 assert.deepEqual(httpFailures,[],'Unexpected HTTP failures')
 assert.ok(anonymous.every(r=>r.method==='GET'&&r.url===RAT+'/api/v1/rate-anything/me'),'Unexpected 401')
 const anonConsole='Failed to load resource: the server responded with a status of 401 ()'
 assert.equal(errors.filter(x=>x===anonConsole).length,anonymous.length,'Anonymous /me 401 console mismatch')
 assert.deepEqual(errors.filter(x=>x!==anonConsole),[],'Unexpected browser errors')
 report.status='PASS';report.anonymousMe401Count=anonymous.length
 await writeFile(process.env.REPORT_DIR+'/report.json',JSON.stringify(report,null,2))
 await context.close()
 console.log('RATCJ_FINAL_RATING_BROWSER_VERIFIED='+JSON.stringify({checks:report.checks.length,screenshots:report.screenshots.length,viewports:[1440,768,390],activeRating:KEEP,accepted:0,pending:1,votes:0}))
}catch(e){
 report.status='FAIL';report.issues.push(String(e))
 await writeFile(process.env.REPORT_DIR+'/report.json',JSON.stringify(report,null,2))
 throw e
}finally{await browser.close()}
NODE
echo RATCJ_FINAL_CHROMIUM_SOURCE_PINNED_PASS=true
