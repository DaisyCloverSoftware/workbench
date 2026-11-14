#!/usr/bin/env bash
set -euo pipefail
# Exact-source served isolated-RAT browser acceptance of YouTube Channel.
# No owner Thing write. YouTube-type writes are intercepted browser fixtures.
# No LIVE, migrations, fixture seeds/resets or deployment operations.
[[ $# -ge 1 ]] || exit 64
mode="$1"; shift
root='/tmp/rum-rat169-youtube-browser-20261114'
case "$mode" in
image)
  [[ $# -eq 2 ]] || exit 64
  id="$1"; stage="$2"
  [[ "$id" =~ ^[a-z0-9-]{8,42}$ && "$stage" =~ ^[0-9]{2}-[a-z-]+$ ]] || exit 64
  file="$root/$id/$stage.jpg"
  [[ -f "$file" && ! -L "$file" ]] || exit 66
  echo "RAT169_YOUTUBE_IMAGE_STAGE=$stage"
  echo "RAT169_YOUTUBE_IMAGE_BYTES=$(wc -c < "$file")"
  printf 'RAT169_YOUTUBE_IMAGE_BASE64='
  base64 -w0 "$file"
  printf '\n'
  exit 0
  ;;
run)
  [[ $# -eq 2 ]] || exit 64
  sha="$1"; id="$2"
  [[ "$sha" =~ ^[0-9a-f]{40}$ && "$id" =~ ^[a-z0-9-]{8,42}$ ]] || exit 64
  mkdir -p "$root"; chmod 700 "$root"
  dir="$root/$id"
  [[ ! -e "$dir" ]] || { echo 'Existing attempt: reuse evidence instead of rerunning'; exit 65; }
  mkdir -m 700 "$dir"
  temp="$(mktemp -d)"
  trap 'rm -rf "$temp"' EXIT
  git -C "$temp" init -q
  git -C "$temp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
  git -C "$temp" fetch -q --depth=1 origin "$sha"
  git -C "$temp" checkout -q --detach FETCH_HEAD
  [[ "$(git -C "$temp" rev-parse HEAD)" == "$sha" ]] || exit 78
  cd "$temp/apps/rate-anything"
  npm ci --ignore-scripts --no-audit --no-fund >/dev/null
  browser_path=""
  for p in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do
    if [[ -x "$p" ]]; then browser_path="$p"; break; fi
  done
  [[ -n "$browser_path" ]] || npx playwright install chromium >/dev/null
  RAT_SOURCE_SHA="$sha" RAT_CAPTURE_DIR="$dir" RAT_CAPTURE_ID="$id" CHROMIUM_PATH="$browser_path" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { writeFile } from 'node:fs/promises'
import { chromium, expect } from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk'
const DEV='https://dev-rum.daisycloversoftware.uk'
const sha=process.env.RAT_SOURCE_SHA, dir=process.env.RAT_CAPTURE_DIR, id=process.env.RAT_CAPTURE_ID
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROMIUM_PATH||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const context=await browser.newContext({viewport:{width:1440,height:1000}})
const page=await context.newPage()
const report={sourceSha:sha,runId:id,checks:[],screenshots:[],requests:[],errors:[],testThingWrites:0,youtubeTypeRealWrites:0}
const note=(name,detail='')=>{report.checks.push({name,detail});console.log('RAT169_YOUTUBE_CHECK='+name+(detail?' '+detail:''))}
const shot=async(stage,locator=page)=>{
  await locator.screenshot({path:dir+'/'+stage+'.jpg',type:'jpeg',quality:45,animations:'disabled',timeout:15000})
  report.screenshots.push(stage)
  console.log('RAT169_YOUTUBE_SCREENSHOT='+stage)
}
const normalize=s=>s.normalize('NFC').toLowerCase().replace(/\s+/g,' ').trim()
const label='YouTube Channel'
let stage='preflight',forcedMissing=false,postBehavior='none',typePosts=0
page.on('pageerror',e=>report.errors.push('page error '+e.message))
page.on('response',r=>{
 const u=new URL(r.url())
 if([RAT,DEV].includes(u.origin)&&u.pathname.startsWith('/api/')) {
   const method=r.request().method()
   if(method!=='GET'||r.status()>=400)report.requests.push({method,path:u.pathname,status:r.status()})
 }
})
await context.route('**/*',route=>{
 const u=new URL(route.request().url())
 const method=route.request().method()
 // A visual owner-form check must never send a real Thing or rating write,
 // even if a later UI change accidentally submits the outer form.
 if(method==='POST'&&u.pathname==='/api/v1/rate-anything/standalone-things'){
   report.testThingWrites+=1
   return route.abort('blockedbyclient')
 }
 if(method==='POST'&&u.pathname==='/api/v1/rate-anything/thing-types'){
   // The page-scoped fixture must intercept both explicit test POSTs first.
   report.youtubeTypeRealWrites+=1
   return route.abort('blockedbyclient')
 }
 const permittedAuthWrite=method==='POST'&&u.origin===RAT&&['/api/v1/rate-anything/auth/register','/api/v1/rate-anything/auth/login'].includes(u.pathname)
 if(u.hostname==='rateurmate.online'||(!['GET','HEAD','OPTIONS'].includes(method)&&!permittedAuthWrite))return route.abort('blockedbyclient')
 return route.continue()
})
try{
 stage='served-identities'
 const version=await context.request.get(RAT+'/VERSION',{headers:{'Cache-Control':'no-cache'}})
 assert.equal(version.status(),200)
 report.ratVersion=(await version.text()).trim()
 assert.equal(report.ratVersion,'rat-candidate+'+sha)
 const health=await context.request.get(DEV+'/api/v1/health')
 assert.equal(health.status(),200)
 const data=await health.json()
 assert.ok(String(data.version).includes(sha))
 note('served-exact-candidate',report.ratVersion)
 stage='no-match'
 let name='The Professional Idiots'
 await page.goto(RAT+'/search?q='+encodeURIComponent(name),{waitUntil:'domcontentloaded'})
 const add=page.getByRole('button',{name:'Add a Thing',exact:true})
 try { await expect(add).toBeVisible({timeout:14000}) } catch {
   name='RATDEV safe UI probe '+id
   await page.goto(RAT+'/search?q='+encodeURIComponent(name),{waitUntil:'domcontentloaded'})
   await expect(add).toBeVisible({timeout:14000})
 }
 report.prefilledName=name
 await add.click()
 const dialog=page.getByRole('dialog',{name:'Add a Thing',exact:true})
 await expect(dialog).toBeVisible()
 stage='create-preview-test-account'
 await dialog.getByRole('button',{name:'Create account',exact:true}).click()
 const suffix=id.replace(/[^a-z0-9]/g,'').slice(-12)
 const username='ratytqa'+suffix
 const password=randomBytes(20).toString('base64url')+'Aa1!'
 await dialog.getByLabel('Email',{exact:true}).fill(username+'@example.com')
 await dialog.getByLabel('Username',{exact:true}).fill(username)
 await dialog.getByLabel('Password',{exact:true}).fill(password)
 await dialog.getByLabel('Confirm password').fill(password)
 await dialog.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await dialog.getByRole('checkbox',{name:/Privacy Notice/}).check()
 await dialog.getByRole('button',{name:'Create RAT account and continue'}).click()
 await expect(dialog.locator('.contribution-member')).toBeVisible({timeout:20000})
 await expect(dialog.getByLabel('Thing name',{exact:true})).toHaveValue(name)
 report.testUsername=username
 note('authenticated-test-only-form',name)
 const desc='Owner form values must survive without creating an owner Thing.'
 const url='https://example.org/youtube-acceptance-no-thing-write'
 await dialog.getByLabel('Thing description').fill(desc)
 await dialog.getByLabel('Optional supporting media URL').fill(url)
 const checkFields=async()=>{
   await expect(dialog.getByLabel('Thing name',{exact:true})).toHaveValue(name)
   await expect(dialog.getByLabel('Thing description')).toHaveValue(desc)
   await expect(dialog.getByLabel('Optional supporting media URL')).toHaveValue(url)
 }
 stage='server-recognition'
 const response=await context.request.get(RAT+'/api/v1/rate-anything/thing-types?q='+encodeURIComponent(label),{headers:{'Cache-Control':'no-cache'}})
 assert.equal(response.status(),200)
 const liveTypes=await response.json()
 assert.equal(liveTypes.meta.creation.mode,'recognised')
 report.existingType=liveTypes.data.find(x=>[x.label,...x.aliases].some(y=>normalize(y)===normalize(label)))??null
 const alt=await context.request.get(RAT+'/api/v1/rate-anything/thing-types?q='+encodeURIComponent('   yOUTUBE    CHANNEL  '))
 assert.equal(alt.status(),200)
 assert.equal((await alt.json()).meta.creation.mode,'recognised')
 const generic=await context.request.get(RAT+'/api/v1/rate-anything/thing-types?q=River%20Channel')
 assert.equal(generic.status(),200)
 assert.equal((await generic.json()).meta.creation.mode,'clarify')
 note('canonical-server-normalised-recognition-and-no-channel-wildcard',report.existingType?'existing-type-reuse-available':'missing-type')
 stage='controlled-missing-path'
 // A deliberately missing catalogue response tests the deployed frontend without
 // creating "YouTube Channel" ahead of its owner's walkthrough.
 // Use the REAL server catalogue when the label is absent; apply the controlled
 // missing-data fixture only if a future deployment already has this type.
 forcedMissing=Boolean(report.existingType)
 const handler=async(route)=>{
   const req=route.request(),u=new URL(req.url())
   if(u.pathname!=='/api/v1/rate-anything/thing-types')return route.continue()
   if(req.method()==='GET'&&normalize(u.searchParams.get('q')||'')===normalize(label)&&forcedMissing){
      const upstream=await route.fetch()
      const body=await upstream.json()
      assert.equal(body.meta.creation.mode,'recognised')
      body.data=body.data.filter(x=>![x.label,...x.aliases].some(y=>normalize(y)===normalize(label)))
      await route.fulfill({response:upstream,contentType:'application/json',body:JSON.stringify(body)})
      return
   }
   if(req.method()==='POST'&&normalize(req.postDataJSON()?.label||'')===normalize(label)){
      typePosts+=1
      assert.deepEqual(req.postDataJSON(),{label})
      if(postBehavior==='fail'){
         await route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({message:'Injected DEV test failure; nothing was saved'})})
         return
      }
      if(postBehavior==='saved'){
         await route.fulfill({status:201,contentType:'application/json',body:JSON.stringify({
           data:{id:'browser-fixture-only',label,aliases:[],entityType:'digital_resource',classificationKey:'browser_fixture_not_persisted',source:'classification'},meta:{created:true}
         })})
         return
      }
      throw Error('Unexpected YouTube type POST: blocked from reaching canonical DB')
   }
   return route.continue()
 }
 await page.route('**/api/v1/rate-anything/thing-types*',handler)
 const picker=dialog.getByRole('combobox',{name:'What kind of thing is it?'})
 await picker.fill(label)
 await expect(dialog.getByRole('button',{name:/^Add “YouTube Channel” as a new type/})).toBeVisible({timeout:14000})
 await dialog.getByRole('button',{name:/^Add “YouTube Channel” as a new type/}).click()
 await expect(dialog.getByText(/A YouTube channel is a page people can watch or follow online/)).toBeVisible()
 await expect(dialog.getByText(/What is it generally/)).toHaveCount(0)
 await expect(dialog.getByRole('radio')).toHaveCount(0)
 const save=dialog.getByRole('button',{name:'Save new type'})
 await expect(save).toBeEnabled()
 await expect(save).toBeFocused()
 await checkFields()
 await shot('01-youtube-save-enabled',dialog)
 note('recognised-type-save-enabled-without-general-kind')
 stage='cancel-preserves'
 await dialog.getByRole('button',{name:'Back to types'}).click()
 await expect(picker).toHaveValue(label)
 await checkFields()
 await dialog.getByRole('button',{name:/^Add “YouTube Channel” as a new type/}).click()
 stage='failure-preserves'
 postBehavior='fail'
 await save.click()
 await expect(dialog.getByText(/Injected DEV test failure; nothing was saved/)).toBeVisible({timeout:12000})
 await expect(picker).toHaveValue(label)
 await checkFields()
 await shot('02-youtube-failure-preserves',dialog)
 note('field-level-creation-failure-retains-all-input')
 stage='fixture-success'
 postBehavior='saved'
 await save.click()
 await expect(dialog.getByText('Selected: YouTube Channel')).toBeVisible()
 await expect(picker).toHaveValue(label)
 await checkFields()
 await shot('03-youtube-selected-label',dialog)
 assert.equal(typePosts,2)
 assert.equal(report.youtubeTypeRealWrites,0)
 note('explicit-fixture-type-save-selects-facing-label-no-canonical-youtube-write')
 stage='ambiguity'
 await picker.fill('Community channel')
 await expect(dialog.getByRole('button',{name:/^Add “Community channel” as a new type/})).toBeVisible({timeout:14000})
 await dialog.getByRole('button',{name:/^Add “Community channel” as a new type/}).click()
 await expect(dialog.getByText('Is this a channel people follow, or the group running it?')).toBeVisible()
 await expect(dialog.getByRole('radio',{name:/An online page or channel/})).toBeVisible()
 await expect(dialog.getByRole('radio',{name:/A group or organisation/})).toBeVisible()
 await expect(dialog.getByRole('button',{name:'Save new type'})).toBeDisabled()
 await checkFields()
 await shot('04-ambiguous-channel-question',dialog)
 await dialog.getByRole('radio',{name:/A group or organisation/}).check()
 await expect(dialog.getByRole('button',{name:'Save new type'})).toBeEnabled()
 await checkFields()
 await dialog.getByRole('button',{name:'Back to types'}).click()
 await expect(picker).toHaveValue('Community channel')
 note('ambiguous-channel-clarified-not-defaulted')
 await dialog.getByRole('button',{name:'Close',exact:true}).click()
 report.result='PASS'
}catch(e){
 report.result='FAIL'
 report.failedStage=stage
 report.errors.push(e instanceof Error?e.stack||e.message:String(e))
 console.error('RAT169_YOUTUBE_FAILED_STAGE='+stage)
 console.error('RAT169_YOUTUBE_ERROR='+String(e).slice(0,1500))
 try{await shot('99-failure')}catch{}
}finally{
 await writeFile(dir+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT169_YOUTUBE_REPORT='+JSON.stringify(report))
 await browser.close()
}
if(report.result!=='PASS')process.exit(1)
NODE
  ;;
*) exit 64;;
esac
