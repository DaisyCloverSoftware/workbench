#!/usr/bin/env bash
set -euo pipefail
# Genuine isolated RAT creation -> initial rating -> originating connection.
# Restricted to separate harmless DEV test data. No LIVE, reset or migration.
[[ $# -eq 3 ]] || { echo 'usage: <source-sha> <run-id> <root-entity-ulid>' >&2; exit 64; }
sha="$1"; rid="$2"; rootId="$3"
[[ "$sha" =~ ^[0-9a-f]{40}$ && "$rid" =~ ^[a-z0-9-]{8,42}$ && "$rootId" =~ ^[a-z0-9]{26}$ ]] || exit 64
dir="/tmp/rat-community-missing-$rid"
[[ ! -e "$dir" ]] || { echo 'Existing evidence: do not re-run a mutating test'; exit 65; }
mkdir -m 700 "$dir"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$sha"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$sha" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=""
for p in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do
 if [[ -x "$p" ]]; then chrome="$p"; break; fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
SOURCE="$sha" RID="$rid" ROOT_ID="$rootId" OUT="$dir" CHROMIUM_PATH="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { writeFile } from 'node:fs/promises'
import { chromium,expect } from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk'
const sha=process.env.SOURCE, id=process.env.RID, root=process.env.ROOT_ID, out=process.env.OUT
const name='Iridescent Monocle Transit '+id, description='Separate harmless non-person linked Thing created only for DEV acceptance.'
const report={source:sha,runId:id,rootId:root,newThingName:name,result:'UNVERIFIED',steps:[],screenshots:[],requests:[],errors:[]}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROMIUM_PATH||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const ctx=await browser.newContext({viewport:{width:1250,height:960}})
const page=await ctx.newPage()
let stage='preflight'
const mark=(label)=>{report.steps.push(label);console.log('RAT_COMMUNITY_MISSING_PASS='+label)}
const shot=async label=>{await page.screenshot({path:out+'/'+label+'.jpg',type:'jpeg',quality:46,animations:'disabled'});report.screenshots.push(label)}
const requestAt=(path)=>page.waitForResponse(r=>new URL(r.url()).pathname===path&&r.request().method()==='POST',{timeout:30000})
page.on('pageerror',e=>report.errors.push(e.message))
page.on('response',r=>{
 const u=new URL(r.url())
 if([RAT,DEV].includes(u.origin)&&u.pathname.startsWith('/api/')&&(r.request().method()!=='GET'||r.status()>=400)){
  report.requests.push({path:u.pathname,method:r.request().method(),status:r.status()})
 }
})
await ctx.route('**/*',route=>{
 const u=new URL(route.request().url()),m=route.request().method()
 if(u.hostname==='rateurmate.online'||(!['GET','HEAD','OPTIONS'].includes(m)&&![RAT,DEV].includes(u.origin)))return route.abort('blockedbyclient')
 return route.continue()
})
try {
 const version=await ctx.request.get(RAT+'/VERSION',{headers:{'Cache-Control':'no-cache'}})
 assert.equal(version.status(),200)
 assert.equal((await version.text()).trim(),'rat-candidate+'+sha)
 const health=await ctx.request.get(DEV+'/api/v1/health')
 assert.equal(health.status(),200)
 assert.ok(String((await health.json()).version).includes(sha))
 const rootRes=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+root)
 assert.equal(rootRes.status(),200)
 report.rootName=(await rootRes.json()).data.active.name
 mark('exact-head-and-public-root')
 stage='connect-entry'
 await page.goto(RAT+'/e/dev-root~'+root,{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toHaveText(report.rootName,{timeout:27000})
 await page.getByRole('button',{name:'Add a connection'}).click()
 const panel=page.locator('.rat-community-panel')
 await expect(panel.getByText('Sign in to contribute a connection.')).toBeVisible({timeout:13000})
 await panel.getByRole('button',{name:'Create account',exact:true}).click()
 const username='raticon'+id.replace(/[^a-z0-9]/g,'').slice(-12)
 const pw=randomBytes(20).toString('base64url')+'Aa1!'
 await panel.getByLabel('Email',{exact:true}).fill(username+'@example.com')
 await panel.getByLabel('Username',{exact:true}).fill(username)
 await panel.getByLabel('Password',{exact:true}).fill(pw)
 await panel.getByLabel('Confirm password').fill(pw)
 await panel.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await panel.getByRole('checkbox',{name:/Privacy Notice/}).check()
 await panel.getByRole('button',{name:'Create RAT account and continue'}).click()
 await expect(panel.getByLabel('Search Things to connect')).toBeVisible({timeout:20000})
 report.testMember=username
 mark('independent-account-authenticated')
 stage='search-missing'
 await panel.getByLabel('Search Things to connect').fill(name)
 const get=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/rate-anything/things/'+root+'/connections/search'&&r.request().method()==='GET',{timeout:20000})
 await panel.getByRole('button',{name:'Search existing Things'}).click()
 const result=await get
 assert.equal(result.status(),200)
 const body=await result.json()
 assert.equal(body.data.length,0,'A real canonical Thing already has this test name')
 await expect(panel.getByText(/No eligible published Thing matched/)).toBeVisible()
 await shot('01-missing-connection-search')
 mark('canonical-search-confirms-missing')
 stage='create-missing'
 await panel.getByRole('button',{name:'None of these — add a missing Thing'}).click()
 const dialog=page.getByRole('dialog',{name:'Add a Thing',exact:true})
 await expect(dialog).toBeVisible({timeout:10000})
 await expect(dialog.getByLabel('Thing name',{exact:true})).toHaveValue(name)
 await dialog.getByLabel('Thing description').fill(description)
 const picker=dialog.getByRole('combobox',{name:'What kind of thing is it?'})
 await picker.fill('Product')
 await expect(dialog.getByRole('button',{name:/^Product General Thing kind/})).toBeVisible({timeout:15000})
 await picker.press('Enter')
 await expect(dialog.getByText('Selected: Product')).toBeVisible()
 await shot('02-missing-thing-create')
 const createResp=requestAt('/api/v1/rate-anything/standalone-things')
 await dialog.getByRole('button',{name:'Check and add'}).click()
 const created=await createResp
 assert.equal(created.status(),201)
 const createdBody=await created.json()
 assert.equal(createdBody.data.name,name)
 assert.equal(createdBody.data.publicationState,'pending')
 assert.equal(createdBody.data.entityType,'product')
 report.newThingId=createdBody.data.id
 report.journeyId=createdBody.meta.journey.id
 mark('canonical-pending-thing-created-no-autoclaim')
 stage='initial-rating'
 await expect(page.locator('[data-rating-step="Value"]')).toBeVisible({timeout:25000})
 assert.equal(new URL(page.url()).pathname,'/rate/'+report.newThingId)
 const returnTo=new URLSearchParams(new URL(page.url()).search).get('return')||''
 assert.ok(returnTo.includes(root)&&returnTo.includes('connect='+report.newThingId))
 report.returnPath=returnTo
 await shot('03-pending-rating-context')
 await page.getByRole('radio',{name:'5: Exceptional. Strongest approval'}).click()
 await page.getByRole('button',{name:'Continue',exact:true}).click()
 await expect(page.locator('[data-rating-step="Category"]')).toBeVisible()
 await page.getByLabel('Search categories').fill('quality')
 await expect(page.getByRole('button',{name:/^Quality/i}).first()).toBeVisible({timeout:15000})
 await page.getByRole('button',{name:/^Quality/i}).first().click()
 await page.getByRole('button',{name:'Continue',exact:true}).click()
 await page.getByLabel('Why this verdict?').fill('Harmless independent DEV rating to publish a missing test Thing.')
 await page.getByRole('button',{name:'Continue',exact:true}).click()
 await expect(page.locator('[data-rating-step="Review"]')).toBeVisible()
 const rateResp=requestAt('/api/v1/rate-anything/rating')
 await page.getByRole('button',{name:'Submit rating'}).click()
 const rated=await rateResp
 assert.equal(rated.status(),201)
 report.ratingEventId=(await rated.json()).data.id
 await expect(page.getByText('Rating submitted',{exact:true})).toBeVisible({timeout:19000})
 await shot('04-initial-rating-saved')
 mark('initial-rating-submitted')
 stage='return-to-origin'
 await page.getByRole('button',{name:'Done',exact:true}).click()
 await expect(page.locator('[data-rating-state="complete"]')).toBeVisible({timeout:14000})
 await page.getByRole('button',{name:'Back to RAT'}).first().click()
 await expect(page.locator('.rat-workspace-header h1')).toHaveText(report.rootName,{timeout:22000})
 assert.equal(new URLSearchParams(new URL(page.url()).search).get('connect'),report.newThingId)
 const conn=page.locator('.rat-community-panel')
 await expect(conn.getByRole('button',{name:'Confirm connection'})).toBeVisible({timeout:16000})
 await expect(conn.getByText(new RegExp(name))).toBeVisible()
 await shot('05-origin-restored-selected-new-thing')
 mark('original-thing-and-connection-intent-restored')
 stage='connect-new-thing'
 const linkResp=requestAt('/api/v1/rate-anything/things/'+root+'/connections')
 await conn.getByRole('button',{name:'Confirm connection'}).click()
 const linked=await linkResp
 assert.equal(linked.status(),201)
 const edge=await linked.json()
 assert.equal(edge.meta.created,true)
 assert.equal(edge.data.sourceEntityId,root)
 assert.equal(edge.data.targetEntityId,report.newThingId)
 assert.equal(edge.data.verificationState,'unverified')
 report.relationshipId=edge.data.id
 await expect(page.locator('.rat-workspace-connection').filter({hasText:name})).toBeVisible({timeout:20000})
 await shot('06-linked-new-thing-visible')
 await page.reload({waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-connection').filter({hasText:name})).toBeVisible({timeout:22000})
 mark('new-thing-connection-persisted-after-refresh')
 const check=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+report.newThingId)
 assert.equal(check.status(),200)
 assert.equal((await check.json()).data.active.id,report.newThingId)
 mark('published-canonical-identity-readback')
 report.result='PASS'
}catch(e){
 report.result='FAIL';report.failedStage=stage
 report.errors.push(e instanceof Error?e.stack||e.message:String(e))
 console.error('RAT_COMMUNITY_MISSING_FAILED_STAGE='+stage)
 console.error('RAT_COMMUNITY_MISSING_ERROR='+String(e).slice(0,2200))
 try{await shot('99-failure')}catch{}
}finally{
 await writeFile(out+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT_COMMUNITY_MISSING_REPORT='+JSON.stringify(report))
 await browser.close()
}
if(report.result!=='PASS')process.exit(1)
NODE
