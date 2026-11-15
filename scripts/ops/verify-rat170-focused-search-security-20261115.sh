#!/usr/bin/env bash
set -Eeuo pipefail
# Scope: deployed DEV browser search diagnosis; one disposable member account
# through ordinary registration; no Thing creation, image success, or moderation.
if [[ $# -gt 0 && "$1" == image ]]; then
  [[ $# -eq 3 ]] || exit 64
  id="$2"; name="$3"
  [[ "$id" =~ ^[a-z0-9-]{8,42}$ && "$name" =~ ^[0-9]{2}-[a-z0-9-]+$ ]] || exit 64
  file="/tmp/rat170-focused-$id/$name.jpg"
  [[ -f "$file" && ! -L "$file" ]] || exit 66
  echo "RAT170_FOCUSED_SCREENSHOT_STAGE=$name"
  printf 'RAT170_FOCUSED_SCREENSHOT_BASE64='; base64 -w0 "$file"; printf '\n'
  exit 0
fi
[[ $# -eq 2 ]] || { echo 'usage: <exact-source-sha> <unique-run-id>' >&2; exit 64; }
source_sha="$1"; run_id="$2"
[[ "$source_sha" =~ ^[0-9a-f]{40}$ && "$run_id" =~ ^[a-z0-9-]{8,42}$ ]] || exit 64
dir="/tmp/rat170-focused-$run_id"
[[ ! -e "$dir" ]] || { echo 'Do not repeat an existing run id'; exit 65; }
mkdir -m 700 "$dir"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$source_sha"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$source_sha" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for b in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do
  if [[ -x "$b" ]]; then chrome="$b"; break; fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
RAT170_FOCUSED_SHA="$source_sha" RAT170_FOCUSED_RUN="$run_id" RAT170_FOCUSED_OUT="$dir" CHROME="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { writeFile,stat } from 'node:fs/promises'
import { chromium,expect } from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk'
const src=process.env.RAT170_FOCUSED_SHA,rid=process.env.RAT170_FOCUSED_RUN,out=process.env.RAT170_FOCUSED_OUT
const oldQuery='zzzz-no-such-entity-rat170-rat170-owner-20261115-r17'
const missing='qzvxnplmksuntopaz-614927551'
const ownerFixture='01m7fvzcen292rvqeqj03evtxe'
const assets=['01m7fwa27f5rekx7cqzbg3ttjz','01m7fwa5bdp4a3hxeq9cmtdwhw','01m7fwa7va06zckhbezb9apj70','01m7fwaarm0926qnt4phea32xa']
const report={source:src,runId:rid,status:'UNVERIFIED',checks:[],screenshots:[],createdAccount:false,createdThings:0,createdMedia:0,oldQuery:null,missingQuery:null,rejectedPosts:[],errors:[],blockedRequests:[]}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const ctx=await browser.newContext({viewport:{width:1440,height:930}}),page=await ctx.newPage()
const anon=await browser.newContext({viewport:{width:400,height:850}})
const pass=s=>{report.checks.push(s);console.log('RAT170_FOCUSED_PASS='+s)}
const shot=async name=>{
 const path=out+'/'+name+'.jpg'
 await page.screenshot({path,type:'jpeg',quality:40,animations:'disabled',fullPage:true})
 let f=await stat(path);report.screenshots.push({stage:name,bytes:f.size,path})
 console.log('RAT170_FOCUSED_SCREENSHOT='+name+' bytes='+f.size)
}
let phase='preflight'
await page.route('**/*',route=>{
 const req=route.request(),u=new URL(req.url()),method=req.method()
 const allowed=method==='POST'&&u.origin===RAT&&(u.pathname==='/api/v1/rate-anything/auth/register'||u.pathname==='/api/v1/rate-anything/things/'+ownerFixture+'/images/link')
 if(u.hostname==='rateurmate.online'||(!['GET','HEAD','OPTIONS'].includes(method)&&!allowed)){
  report.blockedRequests.push({method,host:u.hostname,path:u.pathname})
  return route.abort('blockedbyclient')
 }
 return route.continue()
})
const catalogue=async query=>{
 const r=await anon.request.get(DEV+'/api/v1/public/entities/rat-catalogue/search?q='+encodeURIComponent(query),{headers:{Accept:'application/json'}})
 assert.equal(r.status(),200,'canonical catalogue HTTP status')
 const json=await r.json()
 assert.equal(json.meta.source,'rum-dev-canonical')
 assert.equal(json.meta.privacy,'private-rum-people-excluded')
 assert.equal(typeof json.meta.found,'boolean')
 return {found:json.meta.found,name:json.data?.active?.name??null,id:json.data?.active?.id??null,query:json.meta.query??null}
}
const link=async url=>{
 const response=await page.evaluate(async ({path,url})=>{
  const raw=document.cookie.split('; ').find(t=>t.startsWith('XSRF-TOKEN='))
  const token=raw?decodeURIComponent(raw.slice('XSRF-TOKEN='.length)):''
  const result=await fetch(path,{method:'POST',credentials:'same-origin',headers:{'Content-Type':'application/json','Accept':'application/json',...(token?{'X-XSRF-TOKEN':token}:{})},body:JSON.stringify({slot:'profile',url})})
  return {status:result.status,body:(await result.text()).slice(0,240)}
 },{path:'/api/v1/rate-anything/things/'+ownerFixture+'/images/link',url})
 report.rejectedPosts.push({urlKind:url.includes('/redirect')?'redirect':'invalid',status:response.status})
 return response
}
try{
 const version=await anon.request.get(RAT+'/VERSION',{headers:{'Cache-Control':'no-cache'}})
 assert.equal(version.status(),200);assert.equal((await version.text()).trim(),'rat-candidate+'+src)
 const health=await anon.request.get(DEV+'/api/v1/health')
 assert.equal(health.status(),200);assert.ok(String((await health.json()).version).includes(src))
 pass('exact-deployed-api-and-rat-source')
 phase='old-query-response'
 report.oldQuery=await catalogue(oldQuery)
 console.log('RAT170_OLD_QUERY_RESULT='+JSON.stringify(report.oldQuery))
 if(report.oldQuery.found)pass('old-unmatched-fixture-actually-matched-existing-thing')
 else pass('old-query-now-returned-actual-no-match')
 phase='genuine-no-match'
 report.missingQuery=await catalogue(missing)
 assert.equal(report.missingQuery.found,false,'known genuine-no-match returned an existing record')
 assert.equal(report.missingQuery.name,null)
 pass('canonical-http-200-genuine-found-false')
 await page.goto(RAT+'/search?q='+encodeURIComponent(missing),{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-message')).toContainText('No RAT-visible Thing matched',{timeout:25000})
 await expect(page.getByRole('button',{name:'Add a Thing',exact:true})).toBeVisible()
 await expect(page.getByRole('button',{name:'Add a Thing',exact:true})).toBeEnabled()
 assert.equal(new URL(page.url()).searchParams.get('q'),missing)
 pass('genuine-no-match-url-and-add-a-thing-visible')
 await shot('01-genuine-no-match')
 phase='open-prefilled-panel'
 await page.getByRole('button',{name:'Add a Thing',exact:true}).click()
 const modal=page.getByRole('dialog',{name:'Add a Thing',exact:true})
 await expect(modal).toBeVisible({timeout:15000})
 await expect(modal.getByText('Adding: '+missing+'. Your details stay in this form while you sign in.')).toBeVisible({timeout:15000})
 pass('click-opens-existing-add-panel-retains-searched-name-before-login')
 await shot('02-add-panel-anonymous')
 phase='create-disposable-member'
 await modal.getByRole('button',{name:'Create account',exact:true}).click()
 const name='ratnoc'+rid.replace(/[^a-z0-9]/g,'').slice(-12)
 const password=randomBytes(24).toString('base64url')+'Aa1!'
 await modal.getByLabel('Email',{exact:true}).fill(name+'@example.com')
 await modal.getByLabel('Username',{exact:true}).fill(name)
 await modal.getByLabel('Password',{exact:true}).fill(password)
 await modal.getByLabel('Confirm password').fill(password)
 await modal.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await modal.getByRole('checkbox',{name:/Privacy Notice/}).check()
 const registered=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/rate-anything/auth/register'&&r.request().method()==='POST',{timeout:30000})
 await modal.getByRole('button',{name:'Create RAT account and continue'}).click()
 const reg=await registered
 assert.ok([200,201].includes(reg.status()),'ordinary registration failed status '+reg.status())
 report.createdAccount=true
 await expect(modal.getByLabel('Thing name',{exact:true})).toHaveValue(missing,{timeout:25000})
 await expect(modal.getByLabel('Thing description')).toBeVisible()
 await expect(modal.getByRole('button',{name:'Check and add'})).toBeVisible()
 pass('real-authenticated-creation-form-name-prefilled-without-creating-thing')
 await shot('03-prefilled-creation-form')
 await modal.getByRole('button',{name:'Close',exact:true}).click()
 phase='old-search-browser'
 await page.goto(RAT+'/search?q='+encodeURIComponent(oldQuery),{waitUntil:'domcontentloaded'})
 if(report.oldQuery.found){
  await expect(page.locator('.rat-active-card')).toContainText(report.oldQuery.name,{timeout:26000})
  await expect(page.getByRole('button',{name:'Add a Thing',exact:true})).toHaveCount(0)
  pass('old-query-found-existing-result-so-no-add-button-is-correct')
 }else{
  await expect(page.locator('.rat-message')).toContainText('No RAT-visible Thing matched',{timeout:26000})
  await expect(page.getByRole('button',{name:'Add a Thing',exact:true})).toBeVisible()
  pass('old-query-found-false-now-offers-add')
 }
 phase='error-distinction'
 await page.route('**/api/v1/public/entities/rat-catalogue/search?*',route=>route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({detail:'Simulated catalogue outage in browser acceptance only'})}))
 await page.goto(RAT+'/search?q='+encodeURIComponent('zznocatalogueresult'+rid),{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-message')).toContainText('Simulated catalogue outage',{timeout:25000})
 await expect(page.getByRole('button',{name:'Add a Thing',exact:true})).toHaveCount(0)
 pass('http-503-simulated-error-not-misclassified-as-genuine-no-match')
 await page.unroute('**/api/v1/public/entities/rat-catalogue/search?*')
 phase='evidence-ui'
 await page.goto(RAT+'/e/qa~'+ownerFixture,{waitUntil:'domcontentloaded'})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const form=page.locator('.rat-thing-edit-form')
 await expect(form).toBeVisible({timeout:24000})
 await form.getByRole('button',{name:'Image',exact:true}).click()
 await expect(form.getByLabel('Suggested Thing image URL')).toBeVisible()
 await expect(form.getByRole('button',{name:'Use image URL'})).toBeVisible()
 await expect(form.getByRole('button',{name:'Upload image'})).toBeVisible()
 await expect(form.getByText('Supporting details (optional)')).toBeVisible()
 await expect(form.getByLabel('Edit evidence URL')).toBeHidden()
 await form.getByText('Supporting details (optional)').click()
 await expect(form.getByLabel('Edit evidence URL')).toBeVisible()
 await expect(form.getByLabel('Edit evidence note')).toBeVisible()
 await expect(form.getByText(/not the profile image URL/)).toBeVisible()
 pass('image-url-distinct-from-collapsed-optional-evidence-url-and-note')
 await shot('04-evidence-url-separation')
 phase='public-pending'
 for(const asset of assets){
  const r=await anon.request.get(DEV+'/api/v1/public/entities/rat-image/'+asset+'/content')
  assert.equal(r.status(),404,'Pending media publicly accessible: '+asset)
  const generic=await anon.request.get(DEV+'/api/v1/media/'+asset+'/content')
  assert.ok([401,403,404].includes(generic.status()),'Pending generic-media content public: '+asset)
 }
 pass('all-four-pending-images-denied-by-public-and-generic-media-urls')
 phase='redirect'
 const response=await link('https://httpbin.org/redirect/1')
 assert.equal(response.status,422,'Redirecting source was accepted or failed non-validation: '+response.status+' '+response.body)
 pass('public-https-redirect-image-source-rejected-http-422')
 assert.equal(report.blockedRequests.length,0,'Unexpected mutating or LIVE browser requests')
 report.status='PASS'
 console.log('RAT170_FOCUSED_SECURITY_SEARCH_PASS=true')
}catch(e){
 report.status='FAILED'
 report.errors.push({phase,message:String(e).slice(0,1500)})
 console.error('RAT170_FOCUSED_FAILED_PHASE='+phase)
 console.error('RAT170_FOCUSED_ERROR='+String(e).slice(0,1500))
 try{await shot('99-failure')}catch{}
 process.exitCode=1
}finally{
 await writeFile(out+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT170_FOCUSED_REPORT='+JSON.stringify(report))
 await browser.close()
}
NODE
