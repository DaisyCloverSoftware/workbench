#!/usr/bin/env bash
set -Eeuo pipefail
# Bounded regression browser journey after the four image submissions.
# Mutations are strictly confined to the named disposable public Thing:
# description revision, legacy pending image reference with withdrawal,
# and one verified community connection. No owner/other record changes.
[[ $# -eq 2 ]] || exit 64
sha="$1"; rid="$2"
[[ "$sha" =~ ^[0-9a-f]{40}$ && "$rid" =~ ^[a-z0-9-]{8,42}$ ]] || exit 64
dir="/tmp/rat170-post-images-$rid"
[[ ! -e "$dir" ]] || { echo 'Existing run ID; inspect its result, do not repeat mutations'; exit 65; }
mkdir -m 700 "$dir"
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$sha"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$sha" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for b in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do if [[ -x "$b" ]]; then chrome="$b";break;fi;done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
RAT170_SHA="$sha" RAT170_RID="$rid" RAT170_OUT="$dir" CHROME="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { writeFile,stat } from 'node:fs/promises'
import { chromium,expect } from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk'
const root='01m7fvzcen292rvqeqj03evtxe',target='01m7fw9xhpfmdw0cmpaf3zngkm'
const sha=process.env.RAT170_SHA,rid=process.env.RAT170_RID,out=process.env.RAT170_OUT
const report={source:sha,runId:rid,root,target,status:'UNVERIFIED',checks:[],screenshots:[],errors:[],revisions:[]}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const context=await browser.newContext({viewport:{width:1440,height:940}})
const page=await context.newPage()
let stage='version'
const pass=x=>{report.checks.push(x);console.log('RAT170_POSTIMAGE_PASS='+x)}
const snap=async x=>{
 const path=out+'/'+x+'.jpg'
 await page.screenshot({path,type:'jpeg',quality:45,animations:'disabled'})
 const f=await stat(path);report.screenshots.push({stage:x,bytes:f.size,path})
 console.log('RAT170_POSTIMAGE_SCREENSHOT='+x+' bytes='+f.size)
}
await page.route('**/*',route=>{
 const q=route.request(),u=new URL(q.url())
 if(u.hostname==='rateurmate.online'||(!['GET','HEAD','OPTIONS'].includes(q.method())&&![RAT,DEV].includes(u.origin)))return route.abort('blockedbyclient')
 return route.continue()
})
const post=async (path,body)=>page.evaluate(async ({path,body})=>{
 const raw=document.cookie.split('; ').find(v=>v.startsWith('XSRF-TOKEN='))
 const token=raw?decodeURIComponent(raw.slice('XSRF-TOKEN='.length)):''
 const r=await fetch(path,{method:'POST',credentials:'same-origin',headers:{'Accept':'application/json','Content-Type':'application/json',...(token?{'X-XSRF-TOKEN':token}:{})},body:JSON.stringify(body)})
 return {status:r.status,data:await r.json().catch(()=>({}))}
},{path,body})
try{
 const version=await context.request.get(RAT+'/VERSION',{headers:{'Cache-Control':'no-cache'}})
 assert.equal(version.status(),200);assert.equal((await version.text()).trim(),'rat-candidate+'+sha)
 const original=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+root)
 assert.equal(original.status(),200)
 assert.equal((await original.json()).data.active.name,'RAT170 Image QA profile-url rat170img4way-20261115-r13')
 pass('exact-candidate-and-existing-disposable-entity')
 stage='member'
 await page.goto(RAT+'/e/qa~'+root,{waitUntil:'domcontentloaded'})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const about=page.locator('.rat-workspace-profile .entity-about')
 await about.getByRole('button',{name:'Create account',exact:true}).click()
 const username='ratpost'+rid.replace(/[^a-z0-9]/g,'').slice(-13)
 const password=randomBytes(22).toString('base64url')+'Aa1!'
 await about.getByLabel('Email',{exact:true}).fill(username+'@example.com')
 await about.getByLabel('Username',{exact:true}).fill(username)
 await about.getByLabel('Password',{exact:true}).fill(password)
 await about.getByLabel('Confirm password').fill(password)
 await about.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await about.getByRole('checkbox',{name:/Privacy Notice/}).check()
 await about.getByRole('button',{name:'Create RAT account and continue'}).click()
 const form=about.locator('.rat-thing-edit-form')
 await expect(form).toBeVisible({timeout:24000})
 const me=await context.request.get(RAT+'/api/v1/rate-anything/me')
 assert.equal(me.status(),200);report.memberId=(await me.json()).data.id
 pass('new-independent-noncreator-contributor')
 stage='community-edit'
 const desc='Isolated community edit acceptance on previously created disposable image fixture '+rid
 await form.getByRole('button',{name:'Description',exact:true}).click()
 await form.getByLabel('Proposed Thing description').fill(desc)
 const editResponse=page.waitForResponse(x=>x.request().method()==='POST'&&new URL(x.url()).pathname==='/api/v1/rate-anything/things/'+root+'/edits',{timeout:25000})
 await form.getByRole('button',{name:'Record edit',exact:true}).click()
 const edit=await editResponse
 assert.equal(edit.status(),201)
 const data=await edit.json()
 assert.equal(data.meta.applied,true);assert.equal(data.data.field,'description')
 pass('noncreator-unclaimed-community-edit-applied')
 const refreshed=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+root)
 assert.equal(refreshed.status(),200)
 assert.equal((await refreshed.json()).data.active.description,desc)
 pass('community-edit-persisted-publicly')
 stage='reference-revisions'
 const base='/api/v1/rate-anything/things/'+root+'/edits'
 const one=await post(base,{field:'image_url',value:'https://example.org/disposable-only-'+rid+'-one.png'})
 assert.equal(one.status,201,JSON.stringify(one.data).slice(0,350))
 assert.equal(one.data.data.outcome,'pending')
 report.revisions.push(one.data.data.id)
 const two=await post(base,{field:'image_url',value:'https://example.org/disposable-only-'+rid+'-two.png',supersedesRevisionId:one.data.data.id})
 assert.equal(two.status,201,JSON.stringify(two.data).slice(0,350))
 assert.equal(two.data.data.outcome,'pending')
 report.revisions.push(two.data.data.id)
 const history=await context.request.get(RAT+base)
 assert.equal(history.status(),200)
 const rows=(await history.json()).data
 assert.equal(rows.find(x=>x.id===one.data.data.id).outcome,'superseded')
 assert.equal(rows.find(x=>x.id===two.data.data.id).outcome,'pending')
 pass('pending-reference-revision-superseded-attributably')
 const withdrew=await post(base+'/'+two.data.data.id+'/withdraw',{})
 assert.equal(withdrew.status,200,JSON.stringify(withdrew.data).slice(0,350))
 assert.equal(withdrew.data.data.outcome,'withdrawn')
 pass('pending-reference-revision-withdrawn')
 stage='connection'
 await page.goto(RAT+'/e/qa~'+root,{waitUntil:'domcontentloaded'})
 await page.getByRole('button',{name:'Add a connection'}).click()
 const panel=page.locator('.rat-community-panel')
 await expect(panel.getByLabel('Search Things to connect')).toBeVisible({timeout:18000})
 await panel.getByLabel('Search Things to connect').fill('Purple Lantern Product Fixture')
 await panel.getByRole('button',{name:'Search existing Things'}).click()
 await expect(panel.locator('.rat-community-choices button').filter({hasText:'Purple Lantern Product Fixture'}).first()).toBeVisible({timeout:18000})
 await panel.locator('.rat-community-choices button').filter({hasText:'Purple Lantern Product Fixture'}).first().click()
 const connectionResponse=page.waitForResponse(x=>x.request().method()==='POST'&&new URL(x.url()).pathname==='/api/v1/rate-anything/things/'+root+'/connections',{timeout:30000})
 await panel.getByRole('button',{name:'Confirm connection'}).click()
 const connection=await connectionResponse
 assert.equal(connection.status(),201)
 const connectionBody=await connection.json()
 assert.equal(connectionBody.data.targetEntityId,target)
 await expect(page.locator('.rat-workspace-connection').filter({hasText:'Purple Lantern Product Fixture'}).first()).toBeVisible({timeout:22000})
 await page.reload({waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-connection').filter({hasText:'Purple Lantern Product Fixture'}).first()).toBeVisible({timeout:22000})
 pass('canonical-connection-created-and-refresh-persisted')
 await snap('01-connected-disposable-thing')
 stage='private-url'
 const rejected=await post('/api/v1/rate-anything/things/'+root+'/images/link',{slot:'profile',url:'https://127.0.0.1/unsafe.png'})
 assert.ok([422,400].includes(rejected.status),'Private-network URL not rejected '+rejected.status)
 pass('unsafe-remote-image-destination-rejected-'+rejected.status)
 stage='no-match'
 const unmatched='qzvxnplmksuntopaz-614927551'
 await page.goto(RAT+'/search?q='+unmatched,{waitUntil:'domcontentloaded'})
 const message=page.locator('.rat-message')
 await expect(message).toContainText('No RAT-visible Thing matched',{timeout:25000})
 await expect(page.getByRole('button',{name:'Add a Thing',exact:true})).toBeVisible({timeout:12000})
 pass('genuine-no-match-add-a-thing-visible')
 await snap('02-no-match-add-a-thing')
 stage='privacy'
 const privacy=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/search?q=RAT%20Private%20Person%20Sentinel')
 assert.equal(privacy.status(),200)
 assert.equal((await privacy.json()).meta.privacy,'private-rum-people-excluded')
 pass('private-person-catalogue-filtering')
 report.status='PASS'
 console.log('RAT170_POSTIMAGE_BROWSER_PASS=true')
}catch(e){
 report.status='FAILED'
 const msg=String(e).slice(0,1300);report.errors.push({stage,message:msg})
 console.error('RAT170_POSTIMAGE_FAILED_STAGE='+stage)
 console.error('RAT170_POSTIMAGE_ERROR='+msg)
 try{await snap('99-failure')}catch{}
 process.exitCode=1
}finally{
 await writeFile(out+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT170_POSTIMAGE_REPORT='+JSON.stringify(report))
 await browser.close()
}
NODE
