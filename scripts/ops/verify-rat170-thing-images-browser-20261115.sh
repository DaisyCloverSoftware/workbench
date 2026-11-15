#!/usr/bin/env bash
set -euo pipefail
# One-time real deployed browser acceptance. Only prior approved synthetic
# Carmine Nebula Archive; NEVER The Professional Idiots or any real entity.
[[ $# -eq 2 ]] || exit 64
sha="$1"; runid="$2"
[[ "$sha" == "56a2fdfe9ae885f2e8889b5251f0b3c598eab27f" && "$runid" =~ ^[a-z0-9-]{8,42}$ ]] || exit 78
out="/tmp/rat170-media-$runid"
[[ ! -e "$out" ]] || { echo 'Previous test evidence exists; do not repeat mutations'; exit 65; }
mkdir -m 700 "$out"
temp="$(mktemp -d)";trap 'rm -rf "$temp"' EXIT
git -C "$temp" init -q
git -C "$temp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$temp" fetch -q --depth=1 origin "$sha"
git -C "$temp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$temp" rev-parse HEAD)" == "$sha" ]] || exit 78
python3 - "$out/synthetic.png" <<'PY'
import struct,sys,zlib
path=sys.argv[1]
w,h=128,96
raw=b''.join(b'\0'+b''.join(bytes([(x*2)%256,(y*3)%256,160]) for x in range(w)) for y in range(h))
def chunk(t,d):return struct.pack('!I',len(d))+t+d+struct.pack('!I',zlib.crc32(t+d)&0xffffffff)
open(path,'wb').write(b'\x89PNG\r\n\x1a\n'+chunk(b'IHDR',struct.pack('!2I5B',w,h,8,2,0,0,0))+chunk(b'IDAT',zlib.compress(raw))+chunk(b'IEND',b''))
PY
cd "$temp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for binary in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do
 if [[ -x "$binary" ]]; then chrome="$binary";break;fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
SOURCE="$sha" RUN="$runid" OUT="$out" CHROME="$chrome" node --input-type=module <<'NODE'
import { chromium, expect } from '@playwright/test'
import { randomBytes } from 'node:crypto'
import { writeFile } from 'node:fs/promises'
import assert from 'node:assert/strict'
const RAT='https://dev-rum-ra.daisycloversoftware.uk'
const DEV='https://dev-rum.daisycloversoftware.uk'
const root='01m7d2emwsnd0nxw6new8khbje'
const owner='01m7bswfh0a3evf271ye4taaf0'
const source=process.env.SOURCE,id=process.env.RUN
const report={id,source,root,result:'UNVERIFIED',checks:[],screenshots:[],revisions:[],errors:[],http:[]}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const ctx=await browser.newContext({viewport:{width:1440,height:1000}})
const page=await ctx.newPage()
let stage='versions'
const pass=x=>{report.checks.push(x);console.log('RAT170_THING_MEDIA_PASS='+x)}
const shot=async name=>{await page.screenshot({path:process.env.OUT+'/'+name+'.jpg',type:'jpeg',quality:46,animations:'disabled'});report.screenshots.push(name)}
page.on('response',r=>{
 try{const u=new URL(r.url());if([RAT,DEV].includes(u.origin)&&u.pathname.includes('/images/')&&r.request().method()!=='GET')report.http.push({method:r.request().method(),status:r.status(),path:u.pathname})}catch{}
})
try{
 const v=await ctx.request.get(RAT+'/VERSION',{headers:{'Cache-Control':'no-cache'}})
 assert.equal(v.status(),200);assert.equal((await v.text()).trim(),'rat-candidate+'+source)
 const api=await ctx.request.get(DEV+'/api/v1/health')
 assert.equal(api.status(),200);assert.ok(String((await api.json()).version).includes(source))
 const orig=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+root)
 assert.equal(orig.status(),200)
 const origJson=await orig.json()
 assert.equal(origJson.data.active.name,'Carmine Nebula Archive rat170-community-a01')
 assert.equal(origJson.meta.privacy,'private-rum-people-excluded')
 pass('exact-running-source-and-synthetic-public-thing')
 stage='profile-ui'
 await page.goto(RAT+'/e/qa~'+root,{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toHaveText('Carmine Nebula Archive rat170-community-a01',{timeout:30000})
 await expect(page.getByRole('button',{name:'Change profile picture'})).toBeVisible()
 await expect(page.getByRole('button',{name:/cover photo/}).first()).toBeVisible()
 await expect(page.locator('.rat-workspace-facts dt').filter({hasText:'Type of Thing'})).toHaveCount(1)
 await expect(page.locator('.rat-workspace-facts dt').filter({hasText:/Public classifications|Entity type/i})).toHaveCount(0)
 await shot('01-desktop-profile')
 pass('profile-cover-actions-and-single-type-label')
 stage='account'
 await page.getByRole('button',{name:'Change profile picture'}).click()
 const about=page.locator('.rat-workspace-profile .entity-about')
 await expect(about.getByRole('button',{name:'Create account',exact:true})).toBeVisible({timeout:18000})
 await about.getByRole('button',{name:'Create account',exact:true}).click()
 const user='ratmedia'+id.replace(/[^a-z0-9]/g,'').slice(-12)
 const pw=randomBytes(21).toString('base64url')+'Aa1!'
 await about.getByLabel('Email',{exact:true}).fill(user+'@example.com')
 await about.getByLabel('Username',{exact:true}).fill(user)
 await about.getByLabel('Password',{exact:true}).fill(pw)
 await about.getByLabel('Confirm password').fill(pw)
 await about.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await about.getByRole('checkbox',{name:/Privacy Notice/}).check()
 await about.getByRole('button',{name:'Create RAT account and continue'}).click()
 const form=about.locator('.rat-thing-edit-form')
 await expect(form).toBeVisible({timeout:30000})
 const me=await ctx.request.get(RAT+'/api/v1/rate-anything/me')
 assert.equal(me.status(),200);report.memberId=(await me.json()).data.id
 pass('authenticated-noncreator-image-editor')
 const imageTab=form.getByRole('button',{name:'Image',exact:true})
 await imageTab.click()
 const controls=form.getByRole('group',{name:'Image field'})
 const modes=form.getByRole('group',{name:'Image source'})
 const submit=form.getByRole('button',{name:'Submit image for review'})
 const support=form.locator('details.rat-thing-edit-form__supporting')
 await expect(support).not.toHaveAttribute('open')
 await support.locator('summary').click()
 await expect(form.getByLabel('Edit evidence URL')).toBeVisible()
 await expect(form.getByLabel('Suggested Thing image URL')).toBeVisible()
 await support.locator('summary').click()
 pass('collapsed-evidence-distinguished-from-image-source')
 const imageUrl=RAT+'/brand/rum-thumb.webp?v=ad234fbada71f3be'
 const apiPath='/api/v1/rate-anything/things/'+root+'/images/'
 const expected=async(slot,mode,fn)=>{
   stage=slot+'-'+mode
   await controls.getByRole('button',{name:slot==='profile'?'Profile picture':'Cover photo'}).click()
   await modes.getByRole('button',{name:mode==='url'?'Use image URL':'Upload image'}).click()
   if(mode==='url') await form.getByLabel('Suggested Thing image URL').fill(imageUrl)
   else {
     const chooser=form.getByLabel('Upload Thing image')
     await chooser.setInputFiles(process.env.OUT+'/synthetic.png')
     await expect(form.locator('.rat-thing-image-editor__preview img')).toBeVisible()
   }
   const wait=page.waitForResponse(r=>r.request().method()==='POST'&&new URL(r.url()).pathname===apiPath+(mode==='url'?'link':'upload'),{timeout:40000})
   await submit.click()
   const response=await wait
   assert.equal(response.status(),201,slot+' '+mode+' expected 201')
   const body=await response.json()
   assert.equal(body.data.outcome,'pending');assert.equal(body.data.moderationStatus,'pending')
   assert.equal(body.data.sourceType,mode==='url'?'linked':'uploaded')
   assert.equal(body.data.sourceSynchronised,false);assert.equal(body.meta.applied,false)
   assert.equal(body.data.field,slot==='profile'?'profile_image':'cover_image')
   assert.ok(body.data.previewUrl)
   report.revisions.push({slot,mode,id:body.data.id,previewUrl:body.data.previewUrl})
   await expect(form.locator('.rat-thing-image-editor__preview img')).toBeVisible({timeout:20000})
   const uri=new URL(body.data.previewUrl)
   const preview=await ctx.request.get(body.data.previewUrl)
   assert.equal(preview.status(),200)
   assert.match(preview.headers()['content-type']||'',/image\/webp/)
   const asset=uri.pathname.match(/\/images\/([0-9a-z]{26})\/preview$/)?.[1]
   assert.ok(asset,'no private preview asset')
   const publicAttempt=await ctx.request.get(DEV+'/api/v1/public/entities/rat-image/'+asset+'/content')
   assert.equal(publicAttempt.status(),404,'pending image must not be public')
   await shot('0'+(report.revisions.length+1)+'-'+slot+'-'+mode+'-pending')
   pass(slot+'-'+mode+'-sanitised-private-pending')
 }
 await expected('profile','url')
 await expected('profile','upload')
 await expected('cover','url')
 await expected('cover','upload')
 stage='mobile'
 for(const width of [900,390]){
   await page.setViewportSize({width,height:900})
   await expect(form).toBeVisible()
   const scroll=await page.evaluate(()=>document.documentElement.scrollWidth)
   assert.ok(scroll<=width+2,'horizontal page overflow at '+width)
   await shot('0'+(report.screenshots.length+1)+'-'+width+'-profile')
   pass('responsive-'+width+'-no-overflow')
 }
 stage='refresh'
 await page.reload({waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toHaveText('Carmine Nebula Archive rat170-community-a01',{timeout:30000})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const history=await ctx.request.get(RAT+'/api/v1/rate-anything/things/'+root+'/edits')
 assert.equal(history.status(),200)
 const list=(await history.json()).data
 for(const recorded of report.revisions){
  const row=list.find(x=>x.id===recorded.id)
  assert.ok(row,'missing revision after refresh')
  assert.equal(row.editorUserId,report.memberId)
  assert.equal(row.outcome,'pending')
  assert.equal(row.imageSource,recorded.mode==='url'?'linked':'uploaded')
 }
 await expect(page.locator('.rat-thing-revision').filter({hasText:'Awaiting review'})).toHaveCount(4)
 await shot('09-refresh-history')
 pass('all-four-image-revisions-persist-after-refresh')
 report.result='PASS'
}catch(e){
 report.result='FAIL';report.failedStage=stage;report.errors.push(String(e instanceof Error?e.stack||e.message:e).slice(0,2200))
 console.error('RAT170_THING_MEDIA_FAILED_STAGE='+stage)
 console.error('RAT170_THING_MEDIA_ERROR='+String(e).slice(0,1300))
 try{await page.screenshot({path:process.env.OUT+'/99-failure.jpg',type:'jpeg',quality:40});report.screenshots.push('99-failure')}catch{}
}finally{
 await writeFile(process.env.OUT+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT170_THING_MEDIA_REPORT='+JSON.stringify(report))
 await browser.close()
}
if(report.result!=='PASS')process.exit(1)
NODE
