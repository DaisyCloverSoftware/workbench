#!/usr/bin/env bash
set -euo pipefail
# Exact deployed PR170 frontend layout acceptance. Mutates only an existing disposable DEV test Thing.
if [[ $# -gt 0 && "$1" == image ]]; then
  [[ $# -eq 3 ]] || exit 64
  rid="$2"; stage="$3"
  [[ "$rid" =~ ^[a-z0-9-]{8,42}$ && "$stage" =~ ^[0-9]{2}-[a-z0-9-]+$ ]] || exit 64
  file="/tmp/rat170-edit-form-$rid/$stage.jpg"
  [[ -f "$file" && ! -L "$file" ]] || exit 66
  echo "RAT170_EDIT_FORM_SCREENSHOT_STAGE=$stage"
  echo "RAT170_EDIT_FORM_SCREENSHOT_BYTES=$(wc -c < "$file")"
  printf 'RAT170_EDIT_FORM_SCREENSHOT_BASE64='; base64 -w0 "$file"; printf '\n'
  exit 0
fi
[[ $# -eq 2 ]] || exit 64
sha="$1"; rid="$2"
[[ "$sha" == 'e7234490ffec27e0d645911bd62a18d746d3758e' && "$rid" =~ ^[a-z0-9-]{8,42}$ ]] || exit 78
out="/tmp/rat170-edit-form-$rid"
[[ ! -e "$out" ]] || { echo 'Existing mutation evidence; recover instead'; exit 65; }
mkdir -m 700 "$out"
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
  if [[ -x "$p" ]]; then chrome="$p";break;fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
SOURCE="$sha" RID="$rid" OUT="$out" CHROMIUM_PATH="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { writeFile } from 'node:fs/promises'
import { chromium, expect } from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk'
const root='01m7d2emwsnd0nxw6new8khbje', rootName='Carmine Nebula Archive rat170-community-a01'
const sha=process.env.SOURCE, rid=process.env.RID, out=process.env.OUT
const report={source:sha,root,runId:rid,result:'UNVERIFIED',screenshots:[],checks:[],viewports:[],errors:[],requests:[]}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROMIUM_PATH||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const context=await browser.newContext({viewport:{width:1440,height:1000}})
const page=await context.newPage()
let stage='versions'
const mark=x=>{report.checks.push(x);console.log('RAT170_EDIT_FORM_PASS='+x)}
const shot=async x=>{await page.locator('.rat-thing-edit-form').screenshot({path:out+'/'+x+'.jpg',type:'jpeg',quality:48,animations:'disabled'});report.screenshots.push(x)}
page.on('pageerror',e=>report.errors.push('pageerror '+e.message))
page.on('response',r=>{const u=new URL(r.url());if([RAT,DEV].includes(u.origin)&&u.pathname.startsWith('/api/')&&(r.request().method()!=='GET'||r.status()>=400)) report.requests.push({path:u.pathname,method:r.request().method(),status:r.status()})})
await context.route('**/*',route=>{
 const r=route.request(),u=new URL(r.url())
 if(u.hostname==='rateurmate.online'||(!['GET','HEAD','OPTIONS'].includes(r.method())&&![RAT,DEV].includes(u.origin)))return route.abort('blockedbyclient')
 return route.continue()
})
try {
 const ver=await context.request.get(RAT+'/VERSION',{headers:{'Cache-Control':'no-cache'}})
 assert.equal(ver.status(),200);assert.equal((await ver.text()).trim(),'rat-candidate+'+sha)
 const health=await context.request.get(DEV+'/api/v1/health')
 assert.equal(health.status(),200);assert.ok(String((await health.json()).version).includes('e3bb818f1825d778f5084460057ba7ea8bfb422b'))
 const rootRes=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+root)
 assert.equal(rootRes.status(),200)
 const catalogue=await rootRes.json()
 const rootEntity=catalogue.data.active
 assert.equal(rootEntity.id,root)
 assert.equal(rootEntity.name,rootName)
 assert.equal(catalogue.meta.privacy,'private-rum-people-excluded')
 assert.equal(catalogue.meta.audience,'rat-public')
 mark('exact-served-frontend-and-compatible-canonical-api')
 stage='register-unrelated-user'
 await page.goto(RAT+'/e/test~'+root,{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toHaveText(rootName,{timeout:28000})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const about=page.locator('.rat-workspace-profile .entity-about')
 await expect(about.getByRole('button',{name:'Create account',exact:true})).toBeVisible({timeout:20000})
 await about.getByRole('button',{name:'Create account',exact:true}).click()
 const username='ratlayout'+rid.replace(/[^a-z0-9]/g,'').slice(-12)
 const password=randomBytes(20).toString('base64url')+'Aa1!'
 await about.getByLabel('Email',{exact:true}).fill(username+'@example.com')
 await about.getByLabel('Username',{exact:true}).fill(username)
 await about.getByLabel('Password',{exact:true}).fill(password)
 await about.getByLabel('Confirm password').fill(password)
 await about.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await about.getByRole('checkbox',{name:/Privacy Notice/}).check()
 await about.getByRole('button',{name:'Create RAT account and continue'}).click()
 const form=about.locator('.rat-thing-edit-form')
 await expect(form).toBeVisible({timeout:21000})
 const me=await context.request.get(RAT+'/api/v1/rate-anything/me')
 assert.equal(me.status(),200)
 report.memberId=(await me.json()).data.id
 report.testUsername=username
 mark('unrelated-authenticated-test-member')
 const tabs=form.getByRole('group',{name:'Edit type'})
 const desc=form.getByLabel('Proposed Thing description')
 const evidenceUrl=form.getByLabel('Edit evidence URL')
 const evidenceNote=form.getByLabel('Edit evidence note')
 const button=form.getByRole('button',{name:'Record edit'})
 await expect(tabs.getByRole('button',{name:'Description'})).toHaveAttribute('aria-pressed','true')
 await expect(desc).toHaveAttribute('rows','6')
 stage='responsive'
 for(const [width,name] of [[1440,'01-1440-description'],[900,'02-900-description'],[390,'03-390-description']]){
  await page.setViewportSize({width,height:1000})
  await expect(desc).toBeVisible();await expect(evidenceUrl).toBeVisible();await expect(evidenceNote).toBeVisible();await expect(button).toBeVisible()
  const d=await form.evaluate(el=>{
   const rect=n=>{const r=n.getBoundingClientRect();return {x:r.x,y:r.y,right:r.right,bottom:r.bottom,height:r.height}}
   return {form:rect(el),desc:rect(el.querySelector('[aria-label="Proposed Thing description"]')),
    url:rect(el.querySelector('[aria-label="Edit evidence URL"]')),note:rect(el.querySelector('[aria-label="Edit evidence note"]')),
    button:rect(el.querySelector('.rat-thing-edit-form__actions .primary')),
    tabs:Array.from(el.querySelectorAll('.rat-thing-edit-form__types button')).map(rect),
    scrollWidth:document.documentElement.scrollWidth,viewport:window.innerWidth}
  })
  assert.ok(d.form.x>=-2 && d.form.right<=width+2,'form leaves viewport')
  assert.ok(d.desc.height>=150 && d.desc.x>=d.form.x && d.desc.right<=d.form.right+1,'description too narrow or short')
  assert.ok(d.url.right<=d.form.right+1 && d.note.right<=d.form.right+1,'evidence clipped')
  assert.ok(d.button.y>=Math.max(d.url.bottom,d.note.bottom)-2,'submit overlaps evidence')
  assert.ok(d.tabs.every(x=>x.x>=d.form.x-1&&x.right<=d.form.right+1),'tabs clipped')
  if(width===1440) assert.ok(Math.abs(d.url.y-d.note.y)<9,'wide evidence should be two-column')
  if(width===390) assert.ok(d.note.y>=d.url.bottom-2,'mobile evidence should stack')
  assert.ok(d.scrollWidth<=d.viewport+2,'horizontal scrolling')
  await shot(name)
  report.viewports.push({width,descriptionHeight:Math.round(d.desc.height),columns:Math.abs(d.url.y-d.note.y)<9?2:1,horizontalOverflow:false})
  mark('responsive-'+width)
 }
 stage='focused-field'
 await desc.focus()
 await expect(desc).toBeFocused()
 await shot('04-390-focused-description')
 mark('focused-description-state-visible')
 stage='draft-preservation'
 await desc.fill('Draft description '+rid)
 await tabs.getByRole('button',{name:'Name'}).click()
 await expect(tabs.getByRole('button',{name:'Name'})).toHaveAttribute('aria-pressed','true')
 await form.getByLabel('Proposed Thing name').fill('Draft name '+rid)
 await tabs.getByRole('button',{name:'Image'}).click()
 await expect(form.getByText('Direct cover-photo uploading is not available here.')).toBeVisible()
 await form.getByLabel('Suggested Thing image URL').fill('https://example.org/qa-image.jpg')
 await shot('05-390-image')
 await tabs.getByRole('button',{name:'Description'}).click()
 await expect(form.getByLabel('Proposed Thing description')).toHaveValue('Draft description '+rid)
 await tabs.getByRole('button',{name:'Name'}).click()
 await expect(form.getByLabel('Proposed Thing name')).toHaveValue('Draft name '+rid)
 await tabs.getByRole('button',{name:'Image'}).click()
 await expect(form.getByLabel('Suggested Thing image URL')).toHaveValue('https://example.org/qa-image.jpg')
 await tabs.getByRole('button',{name:'Description'}).click()
 mark('per-type-drafts-preserved-and-image-moderation-explained')
 stage='save-revision'
 await page.setViewportSize({width:1440,height:1000})
 const proposal='DEV layout acceptance edit on an unrelated synthetic Thing only. '+rid
 await desc.fill(proposal)
 await evidenceUrl.fill('https://example.org/rat-qa-evidence')
 await evidenceNote.fill('Harmless supporting evidence for isolated DEV acceptance. '+rid)
 await shot('06-1440-ready-submit')
 await desc.fill('')
 await button.click()
 await expect(about.getByRole('alert')).toContainText('Enter a meaningful proposed value before submitting an edit.')
 await shot('07-1440-validation-state')
 await desc.fill(proposal)
 mark('empty-edit-validation-shown-without-submission')
 const saved=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/rate-anything/things/'+root+'/edits'&&r.request().method()==='POST',{timeout:30000})
 await button.click()
 const resp=await saved;assert.equal(resp.status(),201)
 const json=await resp.json()
 assert.equal(json.meta.applied,true)
 assert.equal(json.data.editorUserId,report.memberId)
 assert.equal(json.data.field,'description')
 assert.equal(json.data.proposedValue,proposal)
 report.revisionId=json.data.id
 report.savedDescription=proposal
 await expect(about.getByRole('status')).toContainText('Edit applied and recorded as an attributable revision.',{timeout:25000})
 await expect(about.locator('.relationship').first()).toContainText('description')
 await shot('08-1440-saved-revision')
 mark('unclaimed-noncreator-edit-and-attributable-revision-visible')
 stage='reload'
 await page.reload({waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toHaveText(rootName,{timeout:28000})
 await expect(page.locator('.rat-workspace-description')).toHaveText(proposal,{timeout:28000})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 await expect(page.locator('.rat-workspace-profile .entity-about .relationship').first()).toContainText('description',{timeout:28000})
 await shot('09-1440-persisted-history')
 mark('canonical-edit-persists-and-history-survives-refresh')
 report.result='PASS'
}catch(e){
 report.result='FAIL';report.failedStage=stage
 report.errors.push(e instanceof Error?(e.stack||e.message).slice(0,1700):String(e))
 console.error('RAT170_EDIT_FORM_FAILED_STAGE='+stage)
 console.error('RAT170_EDIT_FORM_ERROR='+String(e).slice(0,1500))
 try{await page.screenshot({path:out+'/99-failure.jpg',type:'jpeg',quality:40});report.screenshots.push('99-failure')}catch{}
}finally{
 await writeFile(out+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT170_EDIT_FORM_REPORT='+JSON.stringify(report))
 await browser.close()
}
if(report.result!=='PASS')process.exit(1)
NODE
