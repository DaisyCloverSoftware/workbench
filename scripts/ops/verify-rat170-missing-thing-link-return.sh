#!/usr/bin/env bash
set -euo pipefail
# Isolated RAT only: one harmless missing Thing through the established
# pending + initial-rating journey, then return and connect to the originating
# existing DEV test Thing. No LIVE or owner data, migrations or resets.
[[ $# -ge 1 ]] || exit 64
mode="$1"; shift
root='/tmp/rat170-missing-connection-acceptance'
if [[ "$mode" == image ]]; then
  [[ $# -eq 2 ]] || exit 64
  rid="$1"; stage="$2"
  [[ "$rid" =~ ^[a-z0-9-]{8,42}$ && "$stage" =~ ^[0-9]{2}-[a-z0-9-]+$ ]] || exit 64
  path="$root/$rid/$stage.jpg"
  [[ -f "$path" && ! -L "$path" ]] || exit 66
  echo "RAT170_MISSING_IMAGE_STAGE=$stage"
  echo "RAT170_MISSING_IMAGE_BYTES=$(wc -c < "$path")"
  printf 'RAT170_MISSING_IMAGE_BASE64='
  base64 -w0 "$path"; printf '\n'
  exit 0
fi
[[ "$mode" == run && $# -eq 5 ]] || exit 64
api_sha="$1"; front_sha="$2"; root_id="$3"; rid="$4"; type_label="$5"
[[ "$api_sha" =~ ^[0-9a-f]{40}$ && "$front_sha" =~ ^[0-9a-f]{40}$ ]] || exit 64
[[ "$root_id" =~ ^[0-9a-z]{26}$ && "$rid" =~ ^[a-z0-9-]{8,42}$ ]] || exit 64
mkdir -p "$root"; chmod 700 "$root"
output="$root/$rid"
[[ ! -e "$output" ]] || { echo 'Evidence for this attempt exists; reuse rather than rerun' >&2; exit 65; }
mkdir -m 700 "$output"
temp="$(mktemp -d)"; trap 'rm -rf "$temp"' EXIT
git -C "$temp" init -q
git -C "$temp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$temp" fetch -q --depth=1 origin "$front_sha"
git -C "$temp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$temp" rev-parse HEAD)" == "$front_sha" ]] || exit 78
cd "$temp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=""
for c in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do
  if [[ -x "$c" ]]; then chrome="$c"; break; fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
API_SHA="$api_sha" FRONT_SHA="$front_sha" ROOT_ID="$root_id" RUN_ID="$rid" TYPE_LABEL="$type_label" OUT="$output" CHROME_PATH="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { writeFile } from 'node:fs/promises'
import { chromium, expect } from '@playwright/test'

const RAT='https://dev-rum-ra.daisycloversoftware.uk', DEV='https://dev-rum.daisycloversoftware.uk'
const api=process.env.API_SHA, front=process.env.FRONT_SHA, rootId=process.env.ROOT_ID
const rid=process.env.RUN_ID, label=process.env.TYPE_LABEL, out=process.env.OUT
const name='Eclipse Parchment Transit '+rid
const report={apiSource:api,frontendSource:front,rootId,runId:rid,thingName:name,typeLabel:label,result:'UNVERIFIED',screenshots:[],checks:[],errors:[],requests:[]}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const context=await browser.newContext({viewport:{width:1365,height:960}})
const page=await context.newPage()
let stage='preflight'
const note=(x)=>{report.checks.push(x);console.log('RAT170_MISSING_PASS='+x)}
const shot=async n=>{await page.screenshot({path:out+'/'+n+'.jpg',type:'jpeg',quality:42,animations:'disabled'});report.screenshots.push(n);console.log('RAT170_MISSING_SCREENSHOT='+n)}
page.on('pageerror',e=>report.errors.push('pageerror '+e.message))
page.on('response',r=>{
 const u=new URL(r.url());if([RAT,DEV].includes(u.origin)&&u.pathname.startsWith('/api/')&&(r.request().method()!=='GET'||r.status()>=400))report.requests.push({method:r.request().method(),path:u.pathname,status:r.status()})
})
await context.route('**/*',route=>{
 const r=route.request(),u=new URL(r.url())
 if(u.hostname==='rateurmate.online'||(!['GET','HEAD','OPTIONS'].includes(r.method())&&![RAT,DEV].includes(u.origin)))return route.abort('blockedbyclient')
 return route.continue()
})
try{
  const version=await context.request.get(RAT+'/VERSION',{headers:{'Cache-Control':'no-cache'}})
  assert.equal(version.status(),200);assert.equal((await version.text()).trim(),'rat-candidate+'+front)
  const health=await context.request.get(DEV+'/api/v1/health')
  assert.equal(health.status(),200);assert.ok(String((await health.json()).version).includes(api))
  note('compatible-serving-versions')
  stage='root-workspace'
  const rootResp=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+rootId)
  assert.equal(rootResp.status(),200)
  report.rootName=(await rootResp.json()).data.active.name
  await page.goto(RAT+'/e/test~'+encodeURIComponent(rootId),{waitUntil:'domcontentloaded'})
  await expect(page.locator('.rat-workspace-header h1')).toHaveText(report.rootName,{timeout:30000})
  await page.getByRole('button',{name:'Add a connection'}).click()
  const conn=page.locator('.rat-community-panel')
  await expect(conn).toBeVisible({timeout:12000})
  stage='community-auth'
  await expect(conn.getByRole('button',{name:'Create account',exact:true})).toBeVisible({timeout:15000})
  await conn.getByRole('button',{name:'Create account',exact:true}).click()
  const username='ratmiss'+rid.replace(/[^a-z0-9]/g,'').slice(-11)
  const password=randomBytes(22).toString('base64url')+'Aa1!'
  await conn.getByLabel('Email',{exact:true}).fill(username+'@example.com')
  await conn.getByLabel('Username',{exact:true}).fill(username)
  await conn.getByLabel('Password',{exact:true}).fill(password)
  await conn.getByLabel('Confirm password').fill(password)
  await conn.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
  await conn.getByRole('checkbox',{name:/Privacy Notice/}).check()
  await conn.getByRole('button',{name:'Create RAT account and continue'}).click()
  await expect(conn.getByLabel('Search Things to connect')).toBeVisible({timeout:22000})
  const me=await context.request.get(RAT+'/api/v1/rate-anything/me')
  assert.equal(me.status(),200);report.memberId=(await me.json()).data.id
  note('distinct-authenticated-contributor')
  stage='genuine-missing-search'
  await conn.getByLabel('Search Things to connect').fill(name)
  const searchPromise=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/rate-anything/things/'+rootId+'/connections/search'&&r.request().method()==='GET',{timeout:25000})
  await conn.getByRole('button',{name:'Search existing Things'}).click()
  const search=await searchPromise
  assert.equal(search.status(),200)
  const searchData=await search.json()
  assert.equal(searchData.meta.count,0,'test Thing name unexpectedly already exists')
  await expect(conn.getByRole('button',{name:/None of these — add a missing Thing/})).toBeVisible()
  await shot('01-missing-thing-choice')
  note('server-backed-missing-thing-search')
  stage='create-missing-panel'
  await conn.getByRole('button',{name:/None of these — add a missing Thing/}).click()
  const dialog=page.getByRole('dialog',{name:'Add a Thing',exact:true})
  await expect(dialog).toBeVisible({timeout:20000})
  await expect(dialog.getByLabel('Thing name',{exact:true})).toHaveValue(name)
  await dialog.getByLabel('Thing description').fill('A harmless separate DEV test resource for completing an originated connection.')
  const picker=dialog.getByRole('combobox',{name:'What kind of thing is it?'})
  await picker.fill(label)
  await expect(dialog.getByRole('button',{name:label+' Existing type',exact:true})).toBeVisible({timeout:18000})
  await picker.press('Enter')
  await expect(dialog.getByText('Selected: '+label,{exact:true})).toBeVisible()
  await shot('02-missing-creation-form')
  const createPromise=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/rate-anything/standalone-things'&&r.request().method()==='POST',{timeout:30000})
  await dialog.getByRole('button',{name:'Check and add',exact:true}).click()
  const created=await createPromise
  assert.equal(created.status(),201)
  const response=await created.json()
  assert.equal(response.data.publicationState,'pending')
  assert.equal(response.data.name,name)
  report.createdThingId=response.data.id
  report.ratingJourneyId=response.meta.journey.id
  assert.ok(response.data.classifications.some(x=>x.name===label))
  note('canonical-missing-thing-created-pending-with-type-reused')
  stage='initial-rating'
  await expect(page.getByRole('heading',{name:'What’s your verdict on '+name+'?'})).toBeVisible({timeout:30000})
  await expect(page.locator('[data-rating-step="Value"]')).toBeVisible({timeout:20000})
  await shot('03-rating-continuation')
  await page.getByRole('radio',{name:'5: Exceptional. Strongest approval'}).click()
  await page.getByRole('button',{name:'Continue',exact:true}).click()
  await expect(page.locator('[data-rating-step="Category"]')).toBeVisible({timeout:15000})
  await page.getByLabel('Search categories').fill('quality')
  await page.getByRole('button',{name:/^Quality/i}).first().click()
  await page.getByRole('button',{name:'Continue',exact:true}).click()
  await expect(page.locator('[data-rating-step="Reason"]')).toBeVisible()
  await page.getByLabel('Why this verdict?').fill('A harmless test rating required before this community Thing becomes public.')
  await page.getByRole('button',{name:'Continue',exact:true}).click()
  await expect(page.locator('[data-rating-step="Review"]')).toBeVisible()
  const ratingPromise=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/rate-anything/rating'&&r.request().method()==='POST',{timeout:30000})
  await page.getByRole('button',{name:'Submit rating'}).click()
  const rated=await ratingPromise
  assert.equal(rated.status(),201)
  const ratedData=await rated.json()
  assert.equal(ratedData.meta.thingId,report.createdThingId)
  report.ratingId=ratedData.data.id
  await expect(page.getByText('Rating submitted',{exact:true})).toBeVisible({timeout:25000})
  note('initial-rating-submitted')
  stage='return-to-origin'
  await page.getByRole('button',{name:'Back to RAT'}).click()
  await expect(page.locator('.rat-workspace-header h1')).toHaveText(report.rootName,{timeout:30000})
  await expect(page.locator('.rat-community-panel')).toBeVisible({timeout:30000})
  await expect(page.locator('.rat-community-choice').getByText(name,{exact:false})).toBeVisible({timeout:30000})
  await shot('04-return-connection-intent')
  note('original-root-and-intent-preserved-after-rating')
  stage='complete-connection'
  const savedPromise=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/rate-anything/things/'+rootId+'/connections'&&r.request().method()==='POST',{timeout:30000})
  await page.getByRole('button',{name:'Confirm connection'}).click()
  const saved=await savedPromise
  assert.equal(saved.status(),201)
  const edge=await saved.json()
  assert.equal(edge.data.sourceEntityId,rootId)
  assert.equal(edge.data.targetEntityId,report.createdThingId)
  assert.equal(edge.data.createdByUserId,report.memberId)
  report.relationshipId=edge.data.id
  await expect(page.locator('.rat-workspace-connection').filter({hasText:name}).first()).toBeVisible({timeout:30000})
  await shot('05-new-connection-visible')
  await page.reload({waitUntil:'domcontentloaded'})
  await expect(page.locator('.rat-workspace-connection').filter({hasText:name}).first()).toBeVisible({timeout:25000})
  note('created-rated-and-linked-thing-persists')
  report.result='PASS'
}catch(e){
 report.result='FAIL';report.failedStage=stage
 report.errors.push(e instanceof Error?e.stack||e.message:String(e))
 console.error('RAT170_MISSING_FAILED_STAGE='+stage)
 console.error('RAT170_MISSING_ERROR='+String(e).slice(0,1600))
 try{await shot('99-failure')}catch{}
}finally{
 await writeFile(out+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT170_MISSING_LINK_REPORT='+JSON.stringify(report))
 await browser.close()
}
if(report.result!=='PASS')process.exit(1)
NODE
