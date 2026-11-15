#!/usr/bin/env bash
set -Eeuo pipefail
# One bounded, real-brower upload using an unused cover slot of the existing
# disposable Thing. No owner Things; no new Thing; no approval; no LIVE.
# Also exercises a >12 MiB valid PNG to confirm canonical API validation.
[[ $# -eq 3 ]] || { echo 'usage: new-frontend-source-sha rat@sha256 unique-run-id' >&2;exit 64; }
source_sha="$1";rat="$2";rid="$3"
[[ "$source_sha" =~ ^[0-9a-f]{40}$ && "$rat" =~ ^ghcr[.]io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ && "$rid" =~ ^[a-z0-9-]{8,42}$ ]] || exit 64
out="/tmp/rat170-upload-2mib-$rid"
[[ ! -e "$out" ]] || { echo "Duplicate run ID: read original receipts instead" >&2;exit 65; }
mkdir -m 700 "$out"
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-rate-anything-preview get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
[[ "$(kctl -n rum-dev-isolated get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ "$(kctl -n rum-dev get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'rateurmate.online' ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:0fc1525d9184bdb140bb543baae817fc54bf3237629ceec38429fd3e2301f521' ]] || exit 78
nginx_conf="$(kctl -n rum-rate-anything-preview exec deployment/rum-rate-anything -c rate-anything -- nginx -T 2>&1)"
grep -q 'client_max_body_size 16m;' <<<"$nginx_conf" || exit 78
echo RAT170_REAL_UPLOAD_PRECHECK_BOUND_16M_AND_UNCHANGED_API=true
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$source_sha"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$source_sha" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for b in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser;do
 [[ ! -x "$b" ]] || { chrome="$b";break; }
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
RAT170_UPLOAD_SOURCE="$source_sha" RAT170_UPLOAD_RUN="$rid" RAT170_UPLOAD_OUT="$out" CHROME="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { deflateSync } from 'node:zlib'
import { writeFile, stat } from 'node:fs/promises'
import { chromium, expect } from '@playwright/test'

const RAT='https://dev-rum-ra.daisycloversoftware.uk'
const DEV='https://dev-rum.daisycloversoftware.uk'
const src=process.env.RAT170_UPLOAD_SOURCE, rid=process.env.RAT170_UPLOAD_RUN, dir=process.env.RAT170_UPLOAD_OUT
const id='01m7fvzcen292rvqeqj03evtxe'
const expected='RAT170 Image QA profile-url rat170img4way-20261115-r13'
const report={source:src,runId:rid,thingId:id,slot:'cover',status:'UNVERIFIED',checks:[],screenshots:[],errors:[],newMedia:[],createdAccounts:[],negativeRequests:[],blockedMutations:[],noNewThings:true}
const mark=x=>{report.checks.push(x); console.log('RAT170_UPLOAD2MIB_PASS='+x)}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const ctx=await browser.newContext({viewport:{width:1440,height:930}})
const anon=await browser.newContext({viewport:{width:390,height:844}})
const page=await ctx.newPage()
let stage='deployed-versions'
const shot=async name=>{
 const filepath=dir+'/'+name+'.jpg'
 await page.screenshot({path:filepath,type:'jpeg',quality:46,animations:'disabled',fullPage:true,timeout:16000})
 const st=await stat(filepath)
 report.screenshots.push({stage:name,bytes:st.size,path:filepath})
 console.log('RAT170_UPLOAD2MIB_SCREENSHOT='+name+' bytes='+st.size)
}
await page.route('**/*',route=>{
 const r=route.request(),u=new URL(r.url()),method=r.method()
 const approved=method==='POST'&&u.origin===RAT&&(u.pathname==='/api/v1/rate-anything/auth/register'||u.pathname==='/api/v1/rate-anything/things/'+id+'/images/upload')
 if(u.hostname==='rateurmate.online'||(!['GET','HEAD','OPTIONS'].includes(method)&&!approved)){
  report.blockedMutations.push({method,host:u.hostname,path:u.pathname})
  return route.abort('blockedbyclient')
 }
 return route.continue()
})
const c32=buf=>{let c=0xffffffff;for(const byte of buf){c^=byte;for(let i=0;i<8;i++)c=(c>>>1)^((c&1)?0xedb88320:0)}return(c^0xffffffff)>>>0}
const chunk=(type,data)=>{
 const name=Buffer.from(type,'ascii'),size=Buffer.alloc(4),checksum=Buffer.alloc(4)
 size.writeUInt32BE(data.length);checksum.writeUInt32BE(c32(Buffer.concat([name,data])))
 return Buffer.concat([size,name,data,checksum])
}
const noisePng=(w,h)=>{
 const ihdr=Buffer.alloc(13);ihdr.writeUInt32BE(w,0);ihdr.writeUInt32BE(h,4);ihdr[8]=8;ihdr[9]=2
 const noise=randomBytes(w*h*3),scan=Buffer.alloc((w*3+1)*h)
 for(let y=0;y<h;y++){const stride=y*(w*3+1);scan[stride]=0;noise.copy(scan,stride+1,y*w*3,(y+1)*w*3)}
 return Buffer.concat([Buffer.from([137,80,78,71,13,10,26,10]),chunk('IHDR',ihdr),chunk('IDAT',deflateSync(scan,{level:1})),chunk('IEND',Buffer.alloc(0))])
}
try{
 const v=await anon.request.get(RAT+'/VERSION',{headers:{'Cache-Control':'no-cache'}})
 assert.equal(v.status(),200);assert.equal((await v.text()).trim(),'rat-candidate+'+src)
 const health=await anon.request.get(DEV+'/api/v1/health')
 assert.equal(health.status(),200);assert.ok(String((await health.json()).version).includes('17cef49b962943095df5374f65de7c74cda640dd'))
 const existing=await anon.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+id)
 assert.equal(existing.status(),200)
 const node=(await existing.json()).data.active
 assert.equal(node.name,expected)
 assert.equal(node.images?.cover??null,null,'Test Thing already has a publicly applied cover; protect from overwrite')
 mark('new-rat-source-unchanged-api-and-existing-unclaimed-disposable-thing')
 stage='contributor-registration'
 await page.goto(RAT+'/e/qa~'+id,{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toContainText(expected,{timeout:25000})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const editor=page.locator('.rat-workspace-profile .entity-about')
 await editor.getByRole('button',{name:'Create account',exact:true}).click()
 const username='ratlarge'+rid.replace(/[^a-z0-9]/g,'').slice(-13)
 const password=randomBytes(23).toString('base64url')+'Aa1!'
 await editor.getByLabel('Email',{exact:true}).fill(username+'@example.com')
 await editor.getByLabel('Username',{exact:true}).fill(username)
 await editor.getByLabel('Password',{exact:true}).fill(password)
 await editor.getByLabel('Confirm password').fill(password)
 await editor.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await editor.getByRole('checkbox',{name:/Privacy Notice/}).check()
 const registered=page.waitForResponse(r=>r.request().method()==='POST'&&new URL(r.url()).pathname==='/api/v1/rate-anything/auth/register',{timeout:30000})
 await editor.getByRole('button',{name:'Create RAT account and continue'}).click()
 const reg=await registered; assert.ok([200,201].includes(reg.status()),'Disposable member registration failed '+reg.status())
 const form=editor.locator('.rat-thing-edit-form')
 await expect(form).toBeVisible({timeout:25000})
 const me=await ctx.request.get(RAT+'/api/v1/rate-anything/me')
 assert.equal(me.status(),200)
 report.createdAccounts.push({memberId:(await me.json()).data.id})
 mark('new-disposable-member-contribution-authority')
 stage='2mib-valid-input'
 await form.getByRole('button',{name:'Image',exact:true}).click()
 await form.getByRole('group',{name:'Image field'}).getByRole('button',{name:'Cover photo'}).click()
 await form.getByRole('group',{name:'Image source'}).getByRole('button',{name:'Upload image'}).click()
 const valid=noisePng(850,850)
 assert.ok(valid.length>2*1024*1024&&valid.length<3*1024*1024,'PNG fixture not ~2MiB '+valid.length)
 console.log('RAT170_UPLOAD2MIB_VALID_PNG_BYTES='+valid.length)
 await form.getByLabel('Upload Thing image').setInputFiles({name:'synthetic-850x850-public-noise.png',mimeType:'image/png',buffer:valid})
 await expect(form.getByText('Local file preview — not yet uploaded or reviewed.')).toBeVisible()
 const input=form.getByLabel('Upload Thing image')
 const size=await input.evaluate(el=>el.files?.[0]?.size??null)
 assert.equal(size,valid.length)
 mark('real-valid-2MiB-png-selected-and-browser-previews')
 await shot('01-large-valid-local-preview')
 stage='2mib-real-submit'
 const uploadPath='/api/v1/rate-anything/things/'+id+'/images/upload'
 const waiting=page.waitForResponse(r=>r.request().method()==='POST'&&new URL(r.url()).pathname===uploadPath,{timeout:60000})
 await form.getByRole('button',{name:'Submit image for review'}).click()
 const response=await waiting
 const payload=await response.json().catch(()=>({}))
 assert.equal(response.status(),201,'Valid ~2MiB image must reach canonical processing (HTTP '+response.status()+') '+JSON.stringify(payload).slice(0,350))
 const rev=payload.data
 assert.equal(rev.thingId,id);assert.equal(rev.field,'cover_image')
 assert.equal(rev.sourceType,'uploaded')
 assert.equal(rev.outcome,'pending');assert.equal(rev.moderationStatus,'pending');assert.equal(rev.sourceSynchronised,false)
 assert.ok(rev.previewUrl,'No private review preview')
 const match=/\/images\/([0-9a-z]{26})\/preview/.exec(new URL(rev.previewUrl,RAT).pathname)
 assert.ok(match,'Private processed media ID missing from preview URL')
 const assetId=match[1]
 report.newMedia.push({revisionId:rev.id,mediaAssetId:assetId,thingId:id,slot:'cover',inputBytes:valid.length})
 await expect(form.getByText(/Validated, sanitised preview/)).toBeVisible({timeout:22000})
 const reviewImage=form.locator('img[alt="Private preview of the proposed Thing image"]')
 await expect(reviewImage).toBeVisible({timeout:18000})
 await expect.poll(async()=>reviewImage.evaluate(el=>el.complete&&el.naturalWidth>0),{timeout:20000}).toBe(true)
 const own=await ctx.request.get(RAT+new URL(rev.previewUrl,RAT).pathname)
 assert.equal(own.status(),200)
 assert.ok((own.headers()['content-type']||'').includes('image/webp'))
 const bytes=await own.body();assert.ok(bytes.length>50)
 const publicContent=await anon.request.get(DEV+'/api/v1/public/entities/rat-image/'+assetId+'/content')
 assert.equal(publicContent.status(),404)
 const generic=await anon.request.get(DEV+'/api/v1/media/'+assetId+'/content')
 assert.ok([401,403,404].includes(generic.status()))
 const pubNode=await anon.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+id)
 assert.equal(pubNode.status(),200)
 assert.equal((await pubNode.json()).data.active.images?.cover??null,null)
 mark('actual-2MiB-image-accepted-201-sanitised-webp-pending-canonical-cover-private')
 await shot('02-large-valid-private-pending')
 stage='refresh'
 await page.reload({waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toContainText(expected,{timeout:25000})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const refreshed=page.locator('.rat-workspace-profile .entity-about')
 const revItem=refreshed.locator('.rat-thing-revision').filter({hasText:'Your submission'}).first()
 await expect(revItem).toContainText('Awaiting review',{timeout:20000})
 await expect(revItem).toContainText('Uploaded file')
 mark('actual-2MiB-canonical-revision-persisted-after-refresh')
 stage='12mib-oversized-input'
 const editForm=refreshed.locator('.rat-thing-edit-form')
 await editForm.getByRole('button',{name:'Image',exact:true}).click()
 await editForm.getByRole('group',{name:'Image field'}).getByRole('button',{name:'Cover photo'}).click()
 await editForm.getByRole('group',{name:'Image source'}).getByRole('button',{name:'Upload image'}).click()
 const oversized=noisePng(2150,2150)
 assert.ok(oversized.length>12*1024*1024 && oversized.length<16*1024*1024,'Oversized fixture not between 12 and 16 MiB '+oversized.length)
 console.log('RAT170_UPLOAD2MIB_OVER_LIMIT_VALID_PNG_BYTES='+oversized.length)
 await editForm.getByLabel('Upload Thing image').setInputFiles({name:'over-limit-synthetic-valid.png',mimeType:'image/png',buffer:oversized})
 await expect(editForm.getByText('Local file preview — not yet uploaded or reviewed.')).toBeVisible()
 const overWait=page.waitForResponse(r=>r.request().method()==='POST'&&new URL(r.url()).pathname===uploadPath,{timeout:65000})
 await editForm.getByRole('button',{name:'Submit image for review'}).click()
 const denied=await overWait
 const deniedText=(await denied.text()).slice(0,1000)
 report.negativeRequests.push({name:'valid-png-over-12MiB',size:oversized.length,http:denied.status(),bodyPreview:deniedText.slice(0,260)})
 assert.equal(denied.status(),422,'Expected canonical 12 MiB validation HTTP 422, got '+denied.status()+' '+deniedText.slice(0,400))
 await expect(page.getByRole('alert').filter({hasText:'Edit needs attention'})).toBeVisible({timeout:16000})
 const alert=await page.getByRole('alert').filter({hasText:'Edit needs attention'}).innerText()
 assert.ok(alert.length>25,'Oversize validation message not understandable')
 assert.ok(/file|image|size|large|exceed|kilobyte/i.test(alert),'No meaningful over-limit feedback '+alert)
 mark('valid-image-exceeding-12MiB-rejected-422-with-visible-validation-message')
 await shot('03-overlimit-validation-error')
 assert.equal(report.blockedMutations.length,0,'Unexpected frontend mutation attempt')
 report.status='PASS'
 console.log('RAT170_UPLOAD2MIB_REAL_BROWSER_PASS=true')
}catch(e){
 report.status='FAILED'
 report.errors.push({stage,message:String(e).slice(0,1500)})
 console.error('RAT170_UPLOAD2MIB_FAILED_STAGE='+stage)
 console.error('RAT170_UPLOAD2MIB_ERROR='+String(e).slice(0,1500))
 try{await shot('99-failure')}catch{}
 process.exitCode=1
}finally{
 await writeFile(dir+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT170_UPLOAD2MIB_REPORT='+JSON.stringify(report))
 await browser.close()
}
NODE
