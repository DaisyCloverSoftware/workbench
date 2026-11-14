#!/usr/bin/env bash
set -euo pipefail
[[ $# -eq 5 ]] || { echo 'usage: <source> <run-id> <root-id> <target-id> <target-name>'; exit 64; }
source="$1"; rid="$2"; rootId="$3"; targetId="$4"; targetName="$5"
[[ "$source" =~ ^[0-9a-f]{40}$ && "$rid" =~ ^[a-z0-9-]{8,42}$ ]] || exit 64
[[ "$rootId" =~ ^[a-z0-9]{26}$ && "$targetId" =~ ^[a-z0-9]{26}$ ]] || exit 64
dir="/tmp/rat-community-browser-$rid"
[[ ! -e "$dir" ]] || { echo 'Previous run exists; recover instead'; exit 65; }
mkdir -m 700 "$dir"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$source"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$source" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=""
for p in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do
 if [[ -x "$p" ]]; then chrome="$p"; break; fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
SOURCE="$source" RID="$rid" ROOT_ID="$rootId" TARGET_ID="$targetId" TARGET_NAME="$targetName" OUT="$dir" CHROMIUM_PATH="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { writeFile } from 'node:fs/promises'
import { chromium, expect } from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk', DEV='https://dev-rum.daisycloversoftware.uk'
const source=process.env.SOURCE, root=process.env.ROOT_ID, target=process.env.TARGET_ID
const targetName=process.env.TARGET_NAME, out=process.env.OUT, rid=process.env.RID
const report={source,root,target,runId:rid,result:'UNVERIFIED',checks:[],screenshots:[],errors:[],requests:[]}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROMIUM_PATH||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const context=await browser.newContext({viewport:{width:1280,height:980}})
const page=await context.newPage()
let stage='versions'
const note=(name)=>{report.checks.push(name);console.log('RAT_COMMUNITY_CHECK='+name)}
const screenshot=async name=>{await page.screenshot({path:out+'/'+name+'.jpg',type:'jpeg',quality:40,animations:'disabled'});report.screenshots.push(name);console.log('RAT_COMMUNITY_IMAGE='+name)}
const path='/api/v1/rate-anything/things/'+root
const waitPost=(suffix)=>page.waitForResponse(r=>new URL(r.url()).pathname===path+suffix&&r.request().method()==='POST',{timeout:30000})
page.on('pageerror',e=>report.errors.push('browser error '+e.message))
page.on('response',r=>{
 const u=new URL(r.url())
 if([RAT,DEV].includes(u.origin)&&u.pathname.startsWith('/api/')&&(r.request().method()!=='GET'||r.status()>=400))report.requests.push({method:r.request().method(),path:u.pathname,status:r.status()})
})
await context.route('**/*',route=>{
 const req=route.request(),url=new URL(req.url())
 if(url.hostname==='rateurmate.online'||(!['GET','HEAD','OPTIONS'].includes(req.method())&&![RAT,DEV].includes(url.origin)))return route.abort('blockedbyclient')
 return route.continue()
})
try{
 const v=await context.request.get(RAT+'/VERSION',{headers:{'Cache-Control':'no-cache'}})
 assert.equal(v.status(),200)
 assert.equal((await v.text()).trim(),'rat-candidate+'+source)
 const health=await context.request.get(DEV+'/api/v1/health')
 assert.equal(health.status(),200)
 assert.ok(String((await health.json()).version).includes(source))
 note('both-served-component-versions')
 stage='public-roots'
 const a=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+root)
 const b=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+target)
 assert.equal(a.status(),200);assert.equal(b.status(),200)
 report.rootName=(await a.json()).data.active.name
 assert.equal((await b.json()).data.active.id,target)
 await page.goto(RAT+'/e/community~'+root,{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toHaveText(report.rootName,{timeout:30000})
 await expect(page.getByRole('button',{name:'Edit or suggest a correction'})).toBeVisible()
 await expect(page.getByRole('button',{name:'Add a connection'})).toBeVisible()
 await screenshot('01-community-controls')
 note('existing-workspace-actions-visible')
 stage='register-unrelated-member'
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const editor=page.locator('.rat-workspace-profile .entity-about')
 await expect(editor.getByRole('button',{name:'Create account',exact:true})).toBeVisible({timeout:16000})
 await editor.getByRole('button',{name:'Create account',exact:true}).click()
 const username='ratcom'+rid.replace(/[^a-z0-9]/g,'').slice(-12)
 const password=randomBytes(22).toString('base64url')+'Aa1!'
 await editor.getByLabel('Email',{exact:true}).fill(username+'@example.com')
 await editor.getByLabel('Username',{exact:true}).fill(username)
 await editor.getByLabel('Password',{exact:true}).fill(password)
 await editor.getByLabel('Confirm password').fill(password)
 await editor.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await editor.getByRole('checkbox',{name:/Privacy Notice/}).check()
 await editor.getByRole('button',{name:'Create RAT account and continue'}).click()
 await expect(editor.getByRole('button',{name:'Record edit'})).toBeVisible({timeout:23000})
 const me=await context.request.get(RAT+'/api/v1/rate-anything/me')
 assert.equal(me.status(),200)
 report.member=(await me.json()).data.id
 note('separate-member-authenticated')
 stage='edit-unclaimed'
 const desc='Unclaimed community edit for DEV only '+rid
 await editor.getByLabel('Proposed Thing description').fill(desc)
 const editResp=waitPost('/edits')
 await editor.getByRole('button',{name:'Record edit'}).click()
 const edit=await editResp
 assert.equal(edit.status(),201)
 const editBody=await edit.json()
 assert.equal(editBody.meta.applied,true)
 assert.equal(editBody.data.editorUserId,report.member)
 await expect(page.locator('.rat-workspace-description')).toHaveText(desc,{timeout:25000})
 await screenshot('02-noncreator-edit-applied')
 note('unclaimed-noncreator-edit-applied')
 stage='image-moderation'
 await editor.getByRole('button',{name:'Image'}).click()
 await editor.getByLabel('Suggested Thing image URL').fill('https://example.org/community-qa-'+rid+'.png')
 const imgResp=waitPost('/edits')
 await editor.getByRole('button',{name:'Record edit'}).click()
 const img=await imgResp
 assert.equal(img.status(),201)
 const imgBody=await img.json()
 assert.equal(imgBody.meta.applied,false)
 assert.equal(imgBody.data.outcome,'pending')
 assert.equal(imgBody.data.moderationStatus,'pending')
 await screenshot('03-image-pending-moderation')
 note('image-suggestion-pending-not-published')
 stage='connect-existing'
 await page.getByRole('button',{name:'Add a connection'}).click()
 const panel=page.locator('.rat-community-panel')
 await expect(panel.getByLabel('Search Things to connect')).toBeVisible({timeout:12000})
 await panel.getByLabel('Search Things to connect').fill(targetName)
 const lookup=page.waitForResponse(r=>new URL(r.url()).pathname===path+'/connections/search'&&r.request().method()==='GET',{timeout:20000})
 await panel.getByRole('button',{name:'Search existing Things'}).click()
 const lookupResp=await lookup
 assert.equal(lookupResp.status(),200)
 assert.ok((await lookupResp.json()).data.some(x=>x.id===target))
 await panel.locator('.rat-community-choices button').filter({hasText:targetName}).first().click()
 await expect(panel.getByRole('button',{name:'Confirm connection'})).toBeVisible()
 await screenshot('04-connection-confirmation')
 const connectResp=waitPost('/connections')
 await panel.getByRole('button',{name:'Confirm connection'}).click()
 const connected=await connectResp
 assert.equal(connected.status(),201)
 const result=await connected.json()
 assert.equal(result.meta.created,true)
 assert.equal(result.data.sourceEntityId,root)
 assert.equal(result.data.targetEntityId,target)
 assert.equal(result.data.createdByUserId,report.member)
 assert.equal(result.data.verificationState,'unverified')
 report.relationshipId=result.data.id
 await expect(page.locator('.rat-workspace-connection').filter({hasText:targetName}).first()).toBeVisible({timeout:22000})
 await screenshot('05-connected-public-things')
 note('canonical-relationship-created-by-unrelated-member')
 stage='persistence'
 await page.reload({waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-description')).toHaveText(desc,{timeout:25000})
 await expect(page.locator('.rat-workspace-connection').filter({hasText:targetName}).first()).toBeVisible()
 note('edit-and-connection-survive-refresh')
 stage='anonymous-safety'
 const anonymous=await browser.newContext()
 const unauth=await anonymous.request.post(RAT+path+'/connections',{data:{targetEntityId:target},headers:{Accept:'application/json'}})
 const unauthEdit=await anonymous.request.post(RAT+path+'/edits',{data:{field:'description',value:'Anonymous override'},headers:{Accept:'application/json'}})
 assert.equal(unauth.status(),401);assert.equal(unauthEdit.status(),401)
 await anonymous.close()
 note('anonymous-write-rejected')
 report.result='PASS'
}catch(e){
 report.result='FAIL';report.failedStage=stage
 report.errors.push(e instanceof Error?e.stack||e.message:String(e))
 console.error('RAT_COMMUNITY_FAILED_STAGE='+stage)
 console.error('RAT_COMMUNITY_ERROR='+String(e).slice(0,1900))
 try{await screenshot('99-failure')}catch{}
}finally{
 await writeFile(out+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT_COMMUNITY_REPORT='+JSON.stringify(report))
 await browser.close()
}
if(report.result!=='PASS')process.exit(1)
NODE
