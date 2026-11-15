#!/usr/bin/env bash
set -euo pipefail
# One-time deployed browser test: only existing synthetic root; never owner data.
if [[ $# -gt 0 && "$1" == image ]]; then
 [[ $# -eq 3 ]] || exit 64
 id="$2"; name="$3"
 [[ "$id" =~ ^[a-z0-9-]{8,42}$ && "$name" =~ ^[0-9]{2}-[a-z0-9-]+$ ]] || exit 64
 file="/tmp/rat170-issue8-$id/$name.jpg"
 [[ -f "$file" && ! -L "$file" ]] || exit 66
 echo "RAT170_ISSUE8_SCREENSHOT_STAGE=$name"
 printf 'RAT170_ISSUE8_SCREENSHOT_BASE64='; base64 -w0 "$file"; printf '\n'
 exit 0
fi
[[ $# -eq 2 ]] || exit 64
sha="$1"; id="$2"
[[ "$sha" == '4153c5be7268a95a94fb038b3e58a02e2375d273' && "$id" =~ ^[a-z0-9-]{8,42}$ ]] || exit 78
out="/tmp/rat170-issue8-$id"
[[ ! -e "$out" ]] || { echo 'Prior test data exists; do not duplicate'; exit 65; }
mkdir -m 700 "$out"
temp="$(mktemp -d)";trap 'rm -rf "$temp"' EXIT
git -C "$temp" init -q
git -C "$temp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$temp" fetch -q --depth=1 origin "$sha"
git -C "$temp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$temp" rev-parse HEAD)" == "$sha" ]] || exit 78
cd "$temp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for c in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do
 if [[ -x "$c" ]]; then chrome="$c"; break; fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
SOURCE="$sha" RID="$id" OUTPUT="$out" CHROME="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { writeFile } from 'node:fs/promises'
import { chromium, expect } from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk'
const root='01m7d2emwsnd0nxw6new8khbje',sha=process.env.SOURCE,id=process.env.RID
const report={source:sha,runId:id,root,result:'UNVERIFIED',checks:[],screenshots:[],errors:[]}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const context=await browser.newContext({viewport:{width:1440,height:940}})
const page=await context.newPage()
const pass=x=>{report.checks.push(x);console.log('RAT170_ISSUE8_PASS='+x)}
const shot=async x=>{await page.locator('.rat-workspace-profile .entity-about').screenshot({path:process.env.OUTPUT+'/'+x+'.jpg',type:'jpeg',quality:48,animations:'disabled'});report.screenshots.push(x)}
const responseAt=path=>page.waitForResponse(r=>r.request().method()==='POST'&&new URL(r.url()).pathname===path,{timeout:25000})
let stage='version'
try{
 const v=await context.request.get(RAT+'/VERSION',{headers:{'Cache-Control':'no-cache'}})
 assert.equal(v.status(),200);assert.equal((await v.text()).trim(),'rat-candidate+'+sha)
 const health=await context.request.get(DEV+'/api/v1/health')
 assert.equal(health.status(),200);assert.ok(String((await health.json()).version).includes(sha))
 const original=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+root)
 assert.equal(original.status(),200);assert.equal((await original.json()).data.active.name,'Carmine Nebula Archive rat170-community-a01')
 pass('exact-served-candidate-and-disposable-thing')
 stage='register'
 await page.goto(RAT+'/e/qa~'+root,{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toContainText('Carmine Nebula Archive',{timeout:25000})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const about=page.locator('.rat-workspace-profile .entity-about')
 await about.getByRole('button',{name:'Create account',exact:true}).click()
 const username='ratpending'+id.replace(/[^a-z0-9]/g,'').slice(-12),password=randomBytes(20).toString('base64url')+'Aa1!'
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
 pass('unrelated-contributor-authenticated')
 stage='pending'
 await form.getByRole('button',{name:'Image'}).click()
 const input=form.getByLabel('Suggested Thing image URL')
 const url='/api/v1/rate-anything/things/'+root+'/edits'
 await input.fill('https://example.org/rat170-issue8-first-'+id+'.jpg')
 await shot('01-image-form')
 let wait=responseAt(url)
 await form.getByRole('button',{name:'Record edit'}).click()
 let resp=await wait,body=await resp.json()
 assert.equal(resp.status(),201);assert.equal(body.meta.applied,false)
 assert.equal(body.data.outcome,'pending');assert.equal(body.data.editorUserId,report.memberId)
 report.firstId=body.data.id
 const item=about.locator('.rat-thing-revision').filter({hasText:'Your submission'})
 await expect(item.first()).toContainText('Awaiting review',{timeout:15000})
 await expect(item.first().getByRole('button',{name:'Withdraw'})).toBeVisible()
 await shot('02-pending-revision')
 pass('pending-image-with-status-and-author-controls')
 stage='duplicate'
 wait=responseAt(url)
 await form.getByRole('button',{name:'Record edit'}).click()
 resp=await wait;body=await resp.json()
 assert.equal(resp.status(),200);assert.equal(body.meta.duplicate,true);assert.equal(body.data.id,report.firstId)
 const history=await context.request.get(RAT+url)
 assert.equal(history.status(),200)
 const revisions=(await history.json()).data
 assert.equal(revisions.filter(x=>x.editorUserId===report.memberId&&x.field==='image_url'&&x.outcome==='pending').length,1)
 assert.ok(!revisions.some(x=>['01m7dxvjchbmnjjnatw0av6a7m','01m7dxwv5cnczfsjset8emjvap'].includes(x.id)))
 pass('duplicate-idempotent-and-others-private-suggestions-hidden')
 stage='replace'
 await item.first().getByRole('button',{name:'Revise suggestion'}).click()
 await expect(form.getByRole('button',{name:'Record replacement'})).toBeVisible()
 await input.fill('https://example.org/rat170-issue8-revised-'+id+'.jpg')
 await page.setViewportSize({width:390,height:880})
 await shot('03-mobile-replacement')
 wait=responseAt(url)
 await form.getByRole('button',{name:'Record replacement'}).click()
 resp=await wait;body=await resp.json()
 assert.equal(resp.status(),201);assert.equal(body.data.outcome,'pending')
 report.revisedId=body.data.id;assert.notEqual(report.firstId,report.revisedId)
 await expect(about.locator('.rat-thing-revision').filter({hasText:'Superseded'})).toBeVisible({timeout:15000})
 await shot('04-superseded')
 pass('explicit-replacement-retains-old-revision')
 stage='withdraw'
 const latest=about.locator('.rat-thing-revision').filter({hasText:'Awaiting review'})
 wait=responseAt(url+'/'+report.revisedId+'/withdraw')
 await latest.getByRole('button',{name:'Withdraw'}).click()
 resp=await wait;body=await resp.json()
 assert.equal(resp.status(),200);assert.equal(body.data.outcome,'withdrawn')
 await expect(about.locator('.rat-thing-revision').filter({hasText:'Withdrawn'})).toBeVisible({timeout:15000})
 await shot('05-withdrawn')
 pass('author-withdraws-without-image-publication')
 stage='reload'
 await page.reload({waitUntil:'domcontentloaded'})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 await expect(page.locator('.rat-workspace-profile .rat-thing-revision').filter({hasText:'Withdrawn'})).toBeVisible({timeout:24000})
 await shot('06-after-refresh')
 pass('revision-states-persist-after-refresh')
 report.result='PASS'
}catch(error){
 report.result='FAIL';report.failedStage=stage
 report.errors.push(String(error instanceof Error?error.stack||error.message:error).slice(0,1500))
 console.error('RAT170_ISSUE8_FAILED_STAGE='+stage)
 console.error('RAT170_ISSUE8_ERROR='+String(error).slice(0,900))
 try{await page.screenshot({path:process.env.OUTPUT+'/99-failure.jpg',type:'jpeg',quality:40});report.screenshots.push('99-failure')}catch{}
}finally{
 await writeFile(process.env.OUTPUT+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT170_ISSUE8_REPORT='+JSON.stringify(report))
 await browser.close()
}
if(report.result!=='PASS')process.exit(1)
NODE
