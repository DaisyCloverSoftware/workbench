#!/usr/bin/env bash
set -euo pipefail
# The target is a SYNTHETIC, separately named, level-two claimed DEV business.
# Browser submits a correction through real RAT host, checks pending moderation,
# and never changes its canonical description. No LIVE, no resets or owner data.
[[ $# -ge 1 ]] || exit 64
mode="$1"; shift
root='/tmp/rat170-claimed-correction'
if [[ "$mode" == image ]]; then
 [[ $# -eq 2 ]] || exit 64
 rid="$1"; stage="$2"
 [[ "$rid" =~ ^[a-z0-9-]{8,42}$ && "$stage" =~ ^[0-9]{2}-[a-z0-9-]+$ ]] || exit 64
 f="$root/$rid/$stage.jpg"
 [[ -f "$f" && ! -L "$f" ]] || exit 66
 echo "RAT170_CLAIMED_IMAGE_STAGE=$stage"
 echo "RAT170_CLAIMED_IMAGE_BYTES=$(wc -c < "$f")"
 printf 'RAT170_CLAIMED_IMAGE_BASE64='
 base64 -w0 "$f"; printf '\n'
 exit 0
fi
[[ "$mode" == run && $# -eq 4 ]] || exit 64
api_sha="$1"; frontend_sha="$2"; entity_id="$3"; rid="$4"
[[ "$api_sha" =~ ^[0-9a-f]{40}$ && "$frontend_sha" =~ ^[0-9a-f]{40}$ ]] || exit 64
[[ "$entity_id" =~ ^[a-z0-9]{26}$ && "$rid" =~ ^[a-z0-9-]{8,42}$ ]] || exit 64
mkdir -p "$root"; chmod 700 "$root"
output="$root/$rid";[[ ! -e "$output" ]] || { echo 'Previous test evidence exists; reuse instead' >&2; exit 65; }
mkdir -m 700 "$output"
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$frontend_sha"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$frontend_sha" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=""
for c in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do
 if [[ -x "$c" ]]; then chrome="$c"; break; fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
API_SHA="$api_sha" FRONT_SHA="$frontend_sha" ENTITY_ID="$entity_id" RID="$rid" OUT="$output" CHROMIUM_PATH="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { writeFile } from 'node:fs/promises'
import { chromium, expect } from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk'
const api=process.env.API_SHA,front=process.env.FRONT_SHA,entity=process.env.ENTITY_ID,rid=process.env.RID,out=process.env.OUT
const report={api,front,entity,runId:rid,syntheticClaim:true,result:'UNVERIFIED',checks:[],screenshots:[],errors:[],requests:[]}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROMIUM_PATH||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const context=await browser.newContext({viewport:{width:1240,height:900}})
const page=await context.newPage()
let stage='preflight'
const note=x=>{report.checks.push(x);console.log('RAT170_CLAIMED_PASS='+x)}
const shot=async x=>{await page.screenshot({path:out+'/'+x+'.jpg',type:'jpeg',quality:40,animations:'disabled'});report.screenshots.push(x)}
page.on('pageerror',e=>report.errors.push('pageerror: '+e.message))
page.on('response',r=>{
 const u=new URL(r.url())
 if([RAT,DEV].includes(u.origin)&&u.pathname.startsWith('/api/')&&(r.request().method()!=='GET'||r.status()>=400))report.requests.push({method:r.request().method(),path:u.pathname,status:r.status()})
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
 const root=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+entity)
 assert.equal(root.status(),200)
 const info=(await root.json()).data.active
 assert.equal(info.id,entity)
 assert.ok(info.name.startsWith('RATDEV Claimed Business Fixture'))
 const description=info.description
 report.thingName=info.name
 note('synthetic-claimed-business-public-and-exact-versions')
 stage='open-edit'
 await page.goto(RAT+'/e/claimed-test~'+entity,{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toHaveText(info.name,{timeout:25000})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const editor=page.locator('.rat-workspace-profile .entity-about')
 await expect(editor.getByRole('button',{name:'Create account',exact:true})).toBeVisible({timeout:15000})
 stage='register-outsider'
 await editor.getByRole('button',{name:'Create account',exact:true}).click()
 const username='ratclm'+rid.replace(/[^a-z0-9]/g,'').slice(-12)
 const password=randomBytes(21).toString('base64url')+'Aa1!'
 await editor.getByLabel('Email',{exact:true}).fill(username+'@example.com')
 await editor.getByLabel('Username',{exact:true}).fill(username)
 await editor.getByLabel('Password',{exact:true}).fill(password)
 await editor.getByLabel('Confirm password').fill(password)
 await editor.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await editor.getByRole('checkbox',{name:/Privacy Notice/}).check()
 await editor.getByRole('button',{name:'Create RAT account and continue'}).click()
 await expect(editor.getByRole('button',{name:'Record edit'})).toBeVisible({timeout:25000})
 const me=await context.request.get(RAT+'/api/v1/rate-anything/me')
 assert.equal(me.status(),200)
 report.outsiderId=(await me.json()).data.id
 await shot('01-claimed-edit-open')
 note('distinct-authenticated-member-opens-claimed-thing')
 stage='submit-correction'
 const proposal='Community-correction proposal requiring verified claimant review '+rid
 await editor.getByLabel('Proposed Thing description').fill(proposal)
 const promise=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/rate-anything/things/'+entity+'/edits'&&r.request().method()==='POST',{timeout:30000})
 await editor.getByRole('button',{name:'Record edit'}).click()
 const saved=await promise
 assert.equal(saved.status(),201)
 const payload=await saved.json()
 assert.equal(payload.meta.applied,false)
 assert.equal(payload.data.source,'correction_suggestion')
 assert.equal(payload.data.moderationStatus,'pending')
 assert.equal(payload.data.outcome,'pending')
 assert.equal(payload.data.editorUserId,report.outsiderId)
 report.revisionId=payload.data.id
 await expect(page.getByText('Edit recorded as a governed proposal pending review.')).toBeVisible({timeout:15000})
 await expect(page.locator('.rat-workspace-description')).toHaveText(description)
 await shot('02-claimed-correction-pending')
 note('nonclaimant-claimed-thing-correction-pending-without-direct-mutation')
 stage='persistence'
 await page.reload({waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-description')).toHaveText(description,{timeout:25000})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 await expect(page.locator('.rat-workspace-profile .entity-about').getByText('pending',{exact:false}).first()).toBeVisible({timeout:18000})
 note('claimed-correction-persists-but-canonical-description-unchanged')
 report.result='PASS'
}catch(e){
 report.result='FAIL';report.failedStage=stage
 report.errors.push(e instanceof Error?e.stack||e.message:String(e))
 console.error('RAT170_CLAIMED_FAILED_STAGE='+stage)
 console.error('RAT170_CLAIMED_ERROR='+String(e).slice(0,1500))
 try{await shot('99-failure')}catch{}
}finally{
 await writeFile(out+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT170_CLAIMED_REPORT='+JSON.stringify(report))
 await browser.close()
}
if(report.result!=='PASS')process.exit(1)
NODE
