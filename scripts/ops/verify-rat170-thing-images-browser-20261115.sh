#!/usr/bin/env bash
set -Eeuo pipefail

# Destructive operations are ONLY creation/edits of freshly named disposable
# isolated-DEV Things by disposable RAT members; never existing owner's Things.
# Test real deployed browser, canonical API, and private moderated media.
if [[ $# -gt 0 && "$1" == image ]]; then
  [[ $# -eq 3 ]] || exit 64
  id="$2"; name="$3"
  [[ "$id" =~ ^[a-z0-9-]{8,42}$ && "$name" =~ ^[0-9]{2}-[a-z0-9-]+$ ]] || exit 64
  file="/tmp/rat170-thing-images-$id/$name.jpg"
  [[ -f "$file" && ! -L "$file" ]] || exit 66
  echo "RAT170_THING_IMAGES_SCREENSHOT_STAGE=$name"
  printf 'RAT170_THING_IMAGES_SCREENSHOT_BASE64='; base64 -w0 "$file"; printf '\n'
  exit 0
fi
[[ $# -eq 2 ]] || { echo 'usage: <exact-40-source-sha> <unique-run-id>' >&2; exit 64; }
source_sha="$1"; run_id="$2"
[[ "$source_sha" =~ ^[0-9a-f]{40}$ && "$run_id" =~ ^[a-z0-9-]{8,42}$ ]] || exit 64
out="/tmp/rat170-thing-images-$run_id"
[[ ! -e "$out" ]] || { echo 'Browser run ID already exists; inspect prior result instead of repeating mutations'; exit 65; }
mkdir -m 700 "$out"
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
  if [[ -x "$b" ]]; then chrome="$b";break;fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
RAT170_SOURCE="$source_sha" RAT170_RUN="$run_id" RAT170_OUTPUT="$out" CHROME="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes, randomUUID } from 'node:crypto'
import { deflateSync } from 'node:zlib'
import { writeFile, stat } from 'node:fs/promises'
import { chromium, expect } from '@playwright/test'

const RAT='https://dev-rum-ra.daisycloversoftware.uk', DEV='https://dev-rum.daisycloversoftware.uk'
const src=process.env.RAT170_SOURCE, rid=process.env.RAT170_RUN, dir=process.env.RAT170_OUTPUT
const anchor='01m7d2emwsnd0nxw6new8khbje' // Previously verified disposable synthetic Thing; authentication only, no edits.
const testUrl='https://upload.wikimedia.org/wikipedia/commons/a/a9/Example.jpg'
const report={source:src,runId:rid,createdThings:[],testAccounts:[],checks:[],screenshots:[],status:'UNVERIFIED',errors:[],publication:'NOT_TESTED_NO_AUTHORISED_MFA_SESSION'}
const b=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const ownerContext=await b.newContext({viewport:{width:1440,height:950}})
const contributorContext=await b.newContext({viewport:{width:1440,height:950}})
const anonymous=await b.newContext({viewport:{width:390,height:844}})
const owner=await ownerContext.newPage(), page=await contributorContext.newPage()
let stage='verify-deploy'
const record=x=>{report.checks.push(x);console.log('RAT170_THING_IMAGE_PASS='+x)}
const snapshot=async (name,viewport=page)=>{
 const file=dir+'/'+name+'.jpg'
 await viewport.screenshot({path:file,type:'jpeg',quality:48,animations:'disabled',fullPage:true})
 const buf=await stat(file)
 report.screenshots.push({stage:name,bytes:buf.size,path:file})
 console.log('RAT170_THING_IMAGE_SCREENSHOT='+name+' bytes='+buf.size)
}
const xsrfFetch=async (p,data)=>{
 return await p.evaluate(async ({path,body})=>{
  const x=document.cookie.split('; ').find(v=>v.startsWith('XSRF-TOKEN='))
  const token=x?decodeURIComponent(x.slice('XSRF-TOKEN='.length)):''
  const r=await fetch(path,{method:'POST',credentials:'same-origin',
    headers:{'Accept':'application/json','Content-Type':'application/json',...(token?{'X-XSRF-TOKEN':token}:{})},body:JSON.stringify(body)})
  return {status:r.status,body:await r.json().catch(()=>({}))}
 },data)
}
const c32=(buf)=>{
 let c=0xffffffff
 for(const byte of buf){c^=byte;for(let i=0;i<8;i++)c=(c>>>1) ^ ((c&1)?0xedb88320:0)}
 return (c^0xffffffff)>>>0
}
const chunk=(type,data)=>{
 const name=Buffer.from(type,'ascii'), size=Buffer.alloc(4), checksum=Buffer.alloc(4)
 size.writeUInt32BE(data.length);checksum.writeUInt32BE(c32(Buffer.concat([name,data])))
 return Buffer.concat([size,name,data,checksum])
}
const makePng=(red,green,blue,w=180,h=120)=>{
 const ihdr=Buffer.alloc(13);ihdr.writeUInt32BE(w,0);ihdr.writeUInt32BE(h,4);ihdr[8]=8;ihdr[9]=2
 const scan=Buffer.alloc((w*3+1)*h,0)
 for(let y=0;y<h;y++)for(let x=0;x<w;x++){let i=y*(w*3+1)+1+x*3;scan[i]=red;scan[i+1]=green;scan[i+2]=blue}
 return Buffer.concat([Buffer.from([137,80,78,71,13,10,26,10]),chunk('IHDR',ihdr),chunk('IDAT',deflateSync(scan)),chunk('IEND',Buffer.alloc(0))])
}
const register=async(p,tag,path)=>{
 await p.goto(RAT+'/e/qa~'+path,{waitUntil:'domcontentloaded'})
 await expect(p.locator('.rat-workspace-header h1')).toBeVisible({timeout:25000})
 await p.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const editor=p.locator('.rat-workspace-profile .entity-about')
 await editor.getByRole('button',{name:'Create account',exact:true}).click()
 const username='ratimg'+tag+rid.replace(/[^a-z0-9]/g,'').slice(-10)
 const password=randomBytes(24).toString('base64url')+'aA1!'
 await editor.getByLabel('Email',{exact:true}).fill(username+'@example.com')
 await editor.getByLabel('Username',{exact:true}).fill(username)
 await editor.getByLabel('Password',{exact:true}).fill(password)
 await editor.getByLabel('Confirm password').fill(password)
 await editor.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await editor.getByRole('checkbox',{name:/Privacy Notice/}).check()
 await editor.getByRole('button',{name:'Create RAT account and continue'}).click()
 await expect(editor.locator('.rat-thing-edit-form')).toBeVisible({timeout:30000})
 const me=await p.request.get(RAT+'/api/v1/rate-anything/me')
 assert.equal(me.status(),200)
 const id=(await me.json()).data.id
 report.testAccounts.push({role:tag,memberId:id})
 return id
}
const publicNode=async(id)=>{
 const r=await anonymous.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+id)
 assert.equal(r.status(),200,'public node lookup')
 return (await r.json()).data.active
}
const appendCase=async (caseInfo)=>{
 stage='image-'+caseInfo.kind
 await page.goto(RAT+'/e/qa~'+caseInfo.id,{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toContainText('RAT170 Image QA',{timeout:25000})
 await expect(page.locator('.rat-workspace-facts dt',{hasText:'Type of Thing'})).toHaveCount(1)
 await expect(page.locator('.rat-workspace-facts dt',{hasText:'Other classifications'})).toHaveCount(0)
 record(caseInfo.kind+'-profile-type-single-and-no-duplicate')
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const editor=page.locator('.rat-workspace-profile .entity-about')
 const form=editor.locator('.rat-thing-edit-form')
 await expect(form).toBeVisible({timeout:25000})
 await form.getByRole('button',{name:'Image',exact:true}).click()
 await form.getByRole('group',{name:'Image field'}).getByRole('button',{name:caseInfo.slot==='profile'?'Profile picture':'Cover photo'}).click()
 const isUpload=caseInfo.mode==='upload'
 if(isUpload){
  await form.getByRole('group',{name:'Image source'}).getByRole('button',{name:'Upload image'}).click()
  await form.getByLabel('Upload Thing image').setInputFiles({name:caseInfo.kind+'.png',mimeType:'image/png',
   buffer:makePng(caseInfo.slot==='profile'?100:20,caseInfo.slot==='cover'?120:70,180)})
  await expect(form.getByText('Local file preview — not yet uploaded or reviewed.')).toBeVisible()
  record(caseInfo.kind+'-local-file-preview')
 }else{
  await form.getByRole('group',{name:'Image source'}).getByRole('button',{name:'Use image URL'}).click()
  await form.getByLabel('Suggested Thing image URL').fill(testUrl)
  await expect(form.getByText(/snapshots, not automatic source synchronisation/)).toBeVisible()
 }
 await snapshot(caseInfo.kind==='profile-url'?'01-profile-url-before':caseInfo.kind==='profile-upload'?'03-profile-upload-before':caseInfo.kind==='cover-url'?'05-cover-url-before':'07-cover-upload-before')
 const reqPath='/api/v1/rate-anything/things/'+caseInfo.id+'/images/'+(isUpload?'upload':'link')
 const wait=page.waitForResponse(r=>new URL(r.url()).pathname===reqPath&&r.request().method()==='POST',{timeout:55000})
 await form.getByRole('button',{name:'Submit image for review'}).click()
 const response=await wait
 const responseBody=await response.json().catch(()=>({}))
 assert.equal(response.status(),201,caseInfo.kind+' submission failed status '+response.status()+' '+JSON.stringify(responseBody).slice(0,350))
 const rev=responseBody.data
 assert.equal(rev.thingId,caseInfo.id)
 assert.equal(rev.field,caseInfo.slot==='profile'?'profile_image':'cover_image')
 assert.equal(rev.sourceType,isUpload?'uploaded':'linked')
 assert.equal(rev.sourceSynchronised,false)
 assert.equal(rev.outcome,'pending')
 assert.equal(rev.moderationStatus,'pending')
 assert.ok(rev.previewUrl,caseInfo.kind+' private preview URL absent')
 caseInfo.revisionId=rev.id
 const m=/\/images\/([0-9a-z]{26})\/preview/.exec(new URL(rev.previewUrl,RAT).pathname)
 assert.ok(m,'private preview asset ID missing')
 caseInfo.assetId=m[1]
 await expect(form.getByText(/Validated, sanitised preview/)).toBeVisible({timeout:15000})
 const img=form.locator('img[alt="Private preview of the proposed Thing image"]')
 await expect(img).toBeVisible({timeout:15000})
 await expect.poll(async()=>await img.evaluate(el=>el.complete&&el.naturalWidth>0),{timeout:20000}).toBe(true)
 const ownPreview=await contributorContext.request.get(RAT+new URL(rev.previewUrl,RAT).pathname)
 assert.equal(ownPreview.status(),200)
 assert.ok((ownPreview.headers()['content-type']||'').includes('image/webp'))
 const publicImage=await anonymous.request.get(DEV+'/api/v1/public/entities/rat-image/'+caseInfo.assetId+'/content')
 assert.equal(publicImage.status(),404,'pending image is publicly retrievable')
 // Generic canonical media metadata/content also must not become anonymous
 // publication channels before staff approval.
 const genericMeta=await anonymous.request.get(DEV+'/api/v1/media/'+caseInfo.assetId)
 const genericContent=await anonymous.request.get(DEV+'/api/v1/media/'+caseInfo.assetId+'/content')
 assert.ok([401,403,404].includes(genericMeta.status()),'generic media metadata leaked publicly')
 assert.ok([401,403,404].includes(genericContent.status()),'generic media bytes leaked publicly')
 const otherMember=await ownerContext.request.get(RAT+new URL(rev.previewUrl,RAT).pathname)
 assert.equal(otherMember.status(),404,'different member can retrieve pending preview')
 const liveNode=await publicNode(caseInfo.id)
 assert.equal(liveNode.images?.[caseInfo.slot]??null,null,'pending image leaked into public Thing profile')
 await snapshot(caseInfo.kind==='profile-url'?'02-profile-url-pending':caseInfo.kind==='profile-upload'?'04-profile-upload-pending':caseInfo.kind==='cover-url'?'06-cover-url-pending':'08-cover-upload-pending')
 record(caseInfo.kind+'-processed-webp-private-pending-and-not-public')
 await page.reload({waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toContainText(caseInfo.name,{timeout:25000})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const revision=page.locator('.rat-workspace-profile .rat-thing-revision').filter({hasText:'Your submission'})
 await expect(revision.first()).toContainText('Awaiting review',{timeout:20000})
 await expect(revision.first()).toContainText(isUpload?'Uploaded file':'Safely imported image URL')
 record(caseInfo.kind+'-persisted-after-refresh')
}
const rejection=async (entityId,kind)=>{
 return await page.evaluate(async ({id,kind})=>{
  const cookie=document.cookie.split('; ').find(i=>i.startsWith('XSRF-TOKEN='))
  const token=cookie?decodeURIComponent(cookie.slice('XSRF-TOKEN='.length)):''
  const isUrl=kind==='private-host'
  const data=new FormData()
  data.append('slot','profile')
  if(!isUrl){
   const big=kind==='oversized'
   const buffer=big?new Uint8Array(12*1024*1024+1000):new TextEncoder().encode('not an image')
   data.append('file',new File([buffer],big?'too-large.png':'invalid-format.png',{type:big?'image/png':'text/plain'}))
  }
  const path='/api/v1/rate-anything/things/'+id+'/images/'+(isUrl?'link':'upload')
  const body=isUrl?JSON.stringify({slot:'profile',url:'https://127.0.0.1/private.png'}):data
  const r=await fetch(path,{method:'POST',credentials:'same-origin',
   headers:{Accept:'application/json',...(isUrl?{'Content-Type':'application/json'}:{}),...(token?{'X-XSRF-TOKEN':token}:{})},body})
  return {status:r.status,text:(await r.text()).slice(0,220)}
 },{id:entityId,kind})
}
try {
 const v=await anonymous.request.get(RAT+'/VERSION',{headers:{'Cache-Control':'no-cache'}})
 assert.equal(v.status(),200);assert.equal((await v.text()).trim(),'rat-candidate+'+src)
 const health=await anonymous.request.get(DEV+'/api/v1/health')
 assert.equal(health.status(),200);assert.ok(String((await health.json()).version).includes(src))
 const a=await anonymous.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+anchor)
 assert.equal(a.status(),200)
 assert.equal((await a.json()).data.active.name,'Carmine Nebula Archive rat170-community-a01')
 record('exact-deployed-api-and-frontend-and-existing-disposable-anchor')
 stage='create-disposable-things'
 const creatorId=await register(owner,'creator',anchor)
 const plans=[
  {kind:'profile-url',slot:'profile',mode:'url'},
  {kind:'profile-upload',slot:'profile',mode:'upload'},
  {kind:'cover-url',slot:'cover',mode:'url'},
  {kind:'cover-upload',slot:'cover',mode:'upload'},
 ]
 for(const item of plans){
  // Canonical Add a Thing uses standalone creation then the first rating.
  // The /entities route targets the preview-local database, not canonical DEV.
  const name='RAT170 Image QA '+item.kind+' '+rid
  const created=await xsrfFetch(owner,{path:'/api/v1/rate-anything/standalone-things',
   body:{entityType:'product',name,description:'Disposable media processing and moderation verification record '+rid,
    idempotencyKey:randomUUID()}})
  assert.equal(created.status,201,'Failed to create '+item.kind+': '+JSON.stringify(created.body).slice(0,400))
  const id=created.body.data?.id, journeyId=created.body.meta?.journey?.id
  assert.ok(/^[0-9a-z]{26}$/.test(id??''),'bad created Thing ID')
  assert.ok(/^[0-9a-z]{26}$/.test(journeyId??''),'missing canonical creation-rating journey')
  assert.equal(created.body.data.publicationState,'pending')
  item.id=id;item.name=name
  report.createdThings.push({id,name,kind:item.kind,journeyId,published:false})
  const rating=await xsrfFetch(owner,{path:'/api/v1/rate-anything/rating',
   body:{entityId:id,journeyId,value:5,categoryKey:'quality',reasonKeys:['quality_or_result'],
    reasonText:'Harmless disposable isolated DEV media journey verification only.',
    idempotencyKey:randomUUID(),audience:'public'}})
  assert.equal(rating.status,201,'Initial rating/publish failed '+item.kind+': '+JSON.stringify(rating.body).slice(0,400))
  assert.equal(rating.body.meta?.thingId,id)
  const node=await publicNode(id)
  assert.equal(node.name,name)
  report.createdThings[report.createdThings.length-1].published=true
 }
 record('four-new-independent-public-disposable-things-via-canonical-initial-rating')
 stage='register-noncreator'
 const editorId=await register(page,'editor',plans[0].id)
 assert.notEqual(creatorId,editorId)
 record('separate-noncreator-member-authenticated')
 const youtube=await contributorContext.request.get(RAT+'/api/v1/rate-anything/thing-types?q=YouTube%20Channel')
 assert.equal(youtube.status(),200)
 assert.ok(JSON.stringify(await youtube.json()).toLowerCase().includes('youtube channel'))
 record('youtube-channel-type-catalogue-still-available')
 for(const item of plans)await appendCase(item)
 stage='security-negatives'
 for(const kind of ['invalid-format','oversized','private-host']){
  const result=await rejection(plans[0].id,kind)
  assert.ok(result.status>=400&&result.status<500,kind+' unexpectedly accepted: '+JSON.stringify(result))
  record('reject-'+kind+'-http-'+result.status)
 }
 stage='connection-and-rating-regression'
 await page.goto(RAT+'/e/qa~'+plans[0].id,{waitUntil:'domcontentloaded'})
 await expect(page.getByRole('button',{name:'Add a connection'})).toBeVisible({timeout:22000})
 await page.getByRole('button',{name:'Add a connection'}).click()
 const panel=page.locator('.rat-community-panel')
 await expect(panel.getByLabel('Search Things to connect')).toBeVisible({timeout:14000})
 await panel.getByLabel('Search Things to connect').fill(plans[1].name)
 await panel.getByRole('button',{name:'Search existing Things'}).click()
 await expect(panel.locator('.rat-community-choices button').filter({hasText:plans[1].name}).first()).toBeVisible({timeout:15000})
 await panel.locator('.rat-community-choices button').filter({hasText:plans[1].name}).first().click()
 await panel.getByRole('button',{name:'Confirm connection'}).click()
 await expect(page.locator('.rat-workspace-connection').filter({hasText:plans[1].name}).first()).toBeVisible({timeout:22000})
 await page.reload({waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-connection').filter({hasText:plans[1].name}).first()).toBeVisible({timeout:22000})
 record('community-connection-persists')
 await page.setViewportSize({width:390,height:844})
 await snapshot('09-mobile-pending-and-connections')
 const overflow=await page.evaluate(()=>document.documentElement.scrollWidth-window.innerWidth)
 assert.ok(overflow<=2,'mobile horizontal overflow: '+overflow)
 record('mobile-profile-and-connections-no-horizontal-overflow')
 for(const [query,expected] of [['DJI','DJI'],['CJ','CJ Investigates']]){
  const r=await anonymous.request.get(DEV+'/api/v1/public/entities/rat-catalogue/search?q='+encodeURIComponent(query))
  assert.equal(r.status(),200)
  const d=await r.json()
  assert.equal(d.data.active.name,expected)
 }
 record('DJI-and-CJ-catalogue-search-unchanged')
 await page.goto(RAT+'/search?q=DJI',{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-active-card h1',{hasText:'DJI'})).toBeVisible({timeout:23000})
 await page.getByRole('button',{name:'Rate',exact:true}).click()
 await expect.poll(()=>page.url(),{timeout:11000}).toContain('#rate')
 record('canonical-rating-flow-opens')
 await page.goto(RAT+'/search?q=zzzz-rat170-qa-no-match-'+rid,{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-message')).toBeVisible({timeout:22000})
 await expect(page.getByRole('button',{name:/Add a Thing/})).toBeVisible({timeout:10000})
 record('search-no-match-add-a-thing-offered')
 report.status='PASS_PENDING_PUBLICATION_GATE'
 console.log('RAT170_THING_IMAGES_BROWSER_PASS=true')
} catch(error){
 report.status='FAILED'
 const message=String(error).slice(0,1100)
 report.errors.push({stage,message})
 console.error('RAT170_THING_IMAGE_FAILED_STAGE='+stage)
 console.error('RAT170_THING_IMAGE_ERROR='+message)
 try{await snapshot('99-failure')}catch{}
 process.exitCode=1
} finally {
 await writeFile(dir+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT170_THING_IMAGE_REPORT='+JSON.stringify(report))
 await b.close()
}
NODE
