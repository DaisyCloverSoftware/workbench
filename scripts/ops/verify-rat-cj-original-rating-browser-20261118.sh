#!/usr/bin/env bash
set -Eeuo pipefail
# Real read-only Chromium acceptance for exact-ID CJ rating restoration, 1440/390, search/workspace/ratings.
[[ $# -eq 0 ]] || exit 64
run='ratcj-owner-rating-final-browser-20261118-r1'
out="/tmp/$run"
[[ ! -e "$out" ]] || exit 65
mkdir -m 700 "$out"
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
api='ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430'
rat='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:ec11bfec432ff10d508850a0a071b3af3787bff656d2decca90eed5951cd0003'
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --retry 3 --max-time 20 https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == 'rat-candidate+887b5e02f4be6ec7e52371666effa9eb47892d6e' ]] || exit 78
temp="$(mktemp -d)"
trap 'rm -rf "$temp"' EXIT
git -C "$temp" init -q
git -C "$temp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$temp" fetch -q --depth=1 origin 887b5e02f4be6ec7e52371666effa9eb47892d6e
git -C "$temp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$temp" rev-parse HEAD)" == '887b5e02f4be6ec7e52371666effa9eb47892d6e' ]] || exit 78
cd "$temp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for p in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser;do if [[ -x "$p" ]];then chrome="$p";break;fi;done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
CHROME="$chrome" REPORT_DIR="$out" REPORT_RUN="$run" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import {stat,readFile,writeFile} from 'node:fs/promises'
import {createHash} from 'node:crypto'
import {chromium,expect} from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk'
const DEV='https://dev-rum.daisycloversoftware.uk'
const CJ='01m07d2f9fj05f4ypdbwp087qj'
const ORIGINAL='01m0mvrxv8c3s12q0w7gwmzt03'
const report={status:'NOT_ASSESSED',run:process.env.REPORT_RUN,id:CJ,keep:ORIGINAL,checks:[],screenshots:[],errors:[]}
const pass=x=>{report.checks.push(x);console.log('RATCJ_RATING_BROWSER_PASS='+x)}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
try{
 const ctx=await browser.newContext({viewport:{width:1440,height:940}})
 const page=await ctx.newPage()
 const errors=[],httpFailures=[],expected401=[]
 page.on('pageerror',e=>errors.push(e.message))
 page.on('console',m=>{if(m.type()==='error')errors.push(m.text())})
 page.on('response',r=>{if(r.status()===401)expected401.push({method:r.request().method(),url:r.url()});else if(r.status()>=400)httpFailures.push({status:r.status(),url:r.url()})})
 await page.route('**/*',r=>['GET','HEAD','OPTIONS'].includes(r.request().method())?r.continue():r.abort('blockedbyclient'))
 const nodeRes=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+CJ,{headers:{'Cache-Control':'no-cache'}})
 assert.equal(nodeRes.status(),200)
 const node=(await nodeRes.json()).data
 assert.equal(node.active.id,CJ);assert.equal(node.active.name,'CJ Investigates')
 assert.equal(node.active.classifications.find(x=>x.primary)?.name,'YouTube Channel')
 assert.deepEqual(node.categories,[]);assert.deepEqual(node.topLinked,[])
 assert.ok(node.active.images?.profile?.url && node.active.images?.cover?.url,'CJ published photos')
 const repRes=await ctx.request.get(DEV+'/api/v1/public/entities/'+CJ+'/reputation',{headers:{'Cache-Control':'no-cache'}})
 assert.equal(repRes.status(),200)
 const rep=await repRes.json()
 assert.equal(rep.meta.entityId,CJ)
 assert.equal(rep.meta.status,'rated')
 assert.equal(rep.data.ratingCount,1)
 assert.ok(Number.isInteger(rep.data.score)&&rep.data.score>=0&&rep.data.score<=100)
 assert.equal(node.active.reputation?.ratingCount,1)
 assert.equal(node.active.reputation?.score,rep.data.score)
 pass('canonical-public-API-single-eligible-owner-reputation-and-zero-linked-items')
 const slug=node.active.slug?.trim()||'thing'
 const detail=RAT+'/e/'+encodeURIComponent(slug)+'~'+encodeURIComponent(CJ)
 const search=RAT+'/search?q=CJ%20Investigates&active='+CJ
 const noOverflow=async width=>{
  const n=await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)
  assert.ok(n<=2,'Unexpected horizontal overflow width '+width+': '+n)
 }
 const decoded=async selector=>{
  const img=page.locator(selector).first()
  await expect(img).toBeVisible({timeout:25000})
  await expect.poll(()=>img.evaluate(e=>e.complete&&e.naturalWidth>0&&e.naturalHeight>0),{timeout:30000}).toBe(true)
  await img.evaluate(e=>e.decode())
 }
 const capture=async (width,view)=>{
  const name='cj-owner-rating-'+width+'-'+view+'-refreshed.jpg'
  const path=process.env.REPORT_DIR+'/'+name
  await page.screenshot({path,type:'jpeg',quality:68,animations:'disabled',fullPage:true})
  const bytes=(await stat(path)).size
  assert.ok(bytes>12000&&bytes<900000,'Unexpected screenshot bytes '+name+': '+bytes)
  const sha256=createHash('sha256').update(await readFile(path)).digest('hex')
  report.screenshots.push({name,width,view,bytes,sha256})
  console.log('RATCJ_RATING_SCREENSHOT='+JSON.stringify({name,bytes,sha256}))
 }
 const searchCheck=async width=>{
  await expect(page.locator('.rat-active-card h1')).toHaveText('CJ Investigates',{timeout:35000})
  await expect(page.locator('.rat-active-card__type')).toHaveText('YouTube Channel')
  await expect(page.locator('.rat-top-linked')).toHaveCount(0)
  await expect(page.locator('section.rat-category:not(.rat-category--empty)')).toHaveCount(0)
  await expect(page.locator('.rat-linked-card')).toHaveCount(0)
  await expect(page.locator('section.rat-category--empty h2')).toHaveText('No public linked Things yet')
  await expect(page.locator('.rat-active-card__actions').getByRole('button',{name:'See ratings'})).toBeVisible()
  await decoded('.rat-active-card__avatar img')
  const text=await page.locator('body').innerText()
  assert.ok(!/Quillstone|RUM DEV linked check|testing1234/.test(text))
  await noOverflow(width)
 }
 const workspaceCheck=async width=>{
  await expect(page.locator('.rat-workspace-identity h1')).toHaveText('CJ Investigates',{timeout:35000})
  await expect(page.locator('.rat-workspace-kickers')).toContainText('YouTube Channel')
  await expect(page.locator('#connections .rat-workspace-connection-group')).toHaveCount(0)
  await expect(page.locator('#connections .rat-workspace-empty')).toContainText('No public connections are attached to this Thing yet.')
  await expect(page.locator('#ratings h2')).toHaveText('Reputation snapshot')
  await expect(page.locator('#ratings .rat-workspace-facts')).toContainText('1')
  await expect(page.locator('.rat-workspace-attention h2')).toHaveText('Community verdicts')
  await expect(page.locator('.rat-workspace-attention')).toContainText('Sign in to see community verdicts',{timeout:25000})
  await decoded('.rat-workspace-avatar img')
  await decoded('.rat-workspace-cover img')
  const text=await page.locator('body').innerText()
  assert.ok(!/Quillstone|RUM DEV linked check|testing1234/.test(text))
  await noOverflow(width)
 }
 for(const width of [1440,390]){
  await page.setViewportSize({width,height:width===390?850:940})
  await page.goto(search,{waitUntil:'domcontentloaded',timeout:45000})
  await searchCheck(width)
  await page.reload({waitUntil:'domcontentloaded',timeout:45000})
  await searchCheck(width)
  await capture(width,'search')
  pass('search-'+width+'-canonical-CJ-no-QA-links-after-refresh')
  await page.goto(detail,{waitUntil:'domcontentloaded',timeout:45000})
  await workspaceCheck(width)
  await page.reload({waitUntil:'domcontentloaded',timeout:45000})
  await workspaceCheck(width)
  await capture(width,'workspace')
  pass('workspace-'+width+'-reputation-one-no-QA-and-photos-after-refresh')
  await page.locator('.rat-workspace-score a.rat-secondary-action').click()
  await expect(page).toHaveURL(/#ratings$/)
  await expect(page.locator('#ratings h2')).toHaveText('Reputation snapshot')
  await expect(page.locator('#ratings .rat-workspace-facts')).toContainText('1')
  await capture(width,'see-ratings')
  pass('see-ratings-'+width+'-anchor-reputation-and-community-verdict-eligibility')
 }
 assert.deepEqual(httpFailures,[],'Unexpected HTTP failures')
 assert.ok(expected401.every(r=>r.method==='GET'&&r.url===RAT+'/api/v1/rate-anything/me'),'Unexpected 401')
 const expectedConsole='Failed to load resource: the server responded with a status of 401 ()'
 assert.equal(errors.filter(x=>x===expectedConsole).length,expected401.length,'Anonymous account 401 console mismatch')
 assert.deepEqual(errors.filter(x=>x!==expectedConsole),[],'Unexpected browser/page errors')
 report.status='PASS'
 report.publicReputation={ratingCount:rep.data.ratingCount,score:rep.data.score,confidence:rep.data.confidenceLabel}
 report.expectedAnonymousAccountChecks=expected401.length
 await writeFile(process.env.REPORT_DIR+'/report.json',JSON.stringify(report,null,2))
 await ctx.close()
 console.log('RATCJ_FINAL_RATING_BROWSER_ACCEPTANCE='+JSON.stringify({status:'PASS',checks:report.checks.length,screenshots:report.screenshots.length,widths:[1440,390],refreshVerified:true,publicRatingCount:1,sourceOfScore:'canonical-existing-algorithm'}))
}catch(e){
 report.status='FAIL';report.errors.push(String(e))
 await writeFile(process.env.REPORT_DIR+'/report.json',JSON.stringify(report,null,2))
 throw e
}finally{await browser.close()}
NODE
echo RATCJ_FINAL_RATING_BROWSER_SCRIPT_COMPLETED=true
