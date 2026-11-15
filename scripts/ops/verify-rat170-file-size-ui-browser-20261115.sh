#!/usr/bin/env bash
set -Eeuo pipefail
# Focused RAT upload-size validation UI test. Uses a new disposable member
# on an existing synthetic Thing. The only attempted image is oversized:
# it MUST be rejected without creating any media/revision.
if [[ $# -gt 0 && "$1" == image ]]; then
  [[ $# -eq 3 ]] || exit 64
  id="$2"; name="$3"
  [[ "$id" =~ ^[a-z0-9-]{8,42}$ && "$name" =~ ^[0-9]{2}-[a-z0-9-]+$ ]] || exit 64
  file="/tmp/rat170-size-ui-$id/$name.jpg"
  [[ -f "$file" && ! -L "$file" ]] || exit 66
  echo "RAT170_SIZE_UI_SCREENSHOT_STAGE=$name"
  printf 'RAT170_SIZE_UI_SCREENSHOT_BASE64='; base64 -w0 "$file"; printf '\n'
  exit 0
fi
[[ $# -eq 3 ]] || { echo 'usage: frontend-source-sha rat@sha256 run-id' >&2; exit 64; }
sha="$1"; digest="$2"; rid="$3"
[[ "$sha" =~ ^[0-9a-f]{40}$ && "$digest" =~ ^ghcr[.]io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ && "$rid" =~ ^[a-z0-9-]{8,42}$ ]] || exit 64
out="/tmp/rat170-size-ui-$rid"
[[ ! -e "$out" ]] || { echo 'Run already exists, do not repeat account registration';exit 65; }
mkdir -m 700 "$out"
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$digest" ]] || exit 78
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:0fc1525d9184bdb140bb543baae817fc54bf3237629ceec38429fd3e2301f521' ]] || exit 78
config="$(kctl -n rum-rate-anything-preview exec deployment/rum-rate-anything -c rate-anything -- nginx -T 2>&1)"
grep -q 'client_max_body_size 16m;' <<<"$config" || exit 78
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$sha"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$sha" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for b in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser;do
 if [[ -x "$b" ]];then chrome="$b";break;fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
RAT170_UI_SHA="$sha" RAT170_UI_RID="$rid" RAT170_UI_OUT="$out" CHROME="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { deflateSync } from 'node:zlib'
import { writeFile,stat } from 'node:fs/promises'
import { chromium,expect } from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk'
const sha=process.env.RAT170_UI_SHA,run=process.env.RAT170_UI_RID,out=process.env.RAT170_UI_OUT
const id='01m7fvzcen292rvqeqj03evtxe', existingMedia='01m7g0sjg1re0m8y1cg1neey7y'
const report={source:sha,runId:run,thingId:id,status:'UNVERIFIED',checks:[],screenshots:[],errors:[],createdAccounts:[],newMedia:0,oversizeResponse:null,blocked:[]}
const b=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const ctx=await b.newContext({viewport:{width:1440,height:930}}),anon=await b.newContext({viewport:{width:390,height:844}})
const page=await ctx.newPage()
let stage='version'
const pass=s=>{report.checks.push(s);console.log('RAT170_FILE_SIZE_UI_PASS='+s)}
const shot=async name=>{
 const path=out+'/'+name+'.jpg';await page.screenshot({path,type:'jpeg',quality:44,animations:'disabled',fullPage:true,timeout:16000})
 const f=await stat(path);report.screenshots.push({stage:name,path,bytes:f.size});console.log('RAT170_FILE_SIZE_UI_SCREENSHOT='+name+' bytes='+f.size)
}
await page.route('**/*',route=>{
 const req=route.request(),u=new URL(req.url()),method=req.method()
 const allowed=method==='POST'&&u.origin===RAT&&(u.pathname==='/api/v1/rate-anything/auth/register'||u.pathname==='/api/v1/rate-anything/things/'+id+'/images/upload')
 if(u.hostname==='rateurmate.online'||(!['GET','HEAD','OPTIONS'].includes(method)&&!allowed)){
  report.blocked.push({method,host:u.hostname,path:u.pathname});return route.abort('blockedbyclient')
 }
 return route.continue()
})
const crc=buf=>{let c=0xffffffff;for(const byte of buf){c^=byte;for(let j=0;j<8;j++)c=(c>>>1)^((c&1)?0xedb88320:0)}return(c^0xffffffff)>>>0}
const chunk=(type,data)=>{
 const name=Buffer.from(type,'ascii'),size=Buffer.alloc(4),checksum=Buffer.alloc(4)
 size.writeUInt32BE(data.length);checksum.writeUInt32BE(crc(Buffer.concat([name,data])))
 return Buffer.concat([size,name,data,checksum])
}
const validNoisePng=(w,h)=>{
 const ihdr=Buffer.alloc(13);ihdr.writeUInt32BE(w,0);ihdr.writeUInt32BE(h,4);ihdr[8]=8;ihdr[9]=2
 const random=randomBytes(w*h*3),scan=Buffer.alloc((w*3+1)*h)
 for(let row=0;row<h;row++){let offset=row*(w*3+1);scan[offset]=0;random.copy(scan,offset+1,row*w*3,(row+1)*w*3)}
 return Buffer.concat([Buffer.from([137,80,78,71,13,10,26,10]),chunk('IHDR',ihdr),chunk('IDAT',deflateSync(scan,{level:1})),chunk('IEND',Buffer.alloc(0))])
}
try{
 const version=await anon.request.get(RAT+'/VERSION',{headers:{'Cache-Control':'no-cache'}})
 assert.equal(version.status(),200);assert.equal((await version.text()).trim(),'rat-candidate+'+sha)
 const health=await anon.request.get(DEV+'/api/v1/health')
 assert.equal(health.status(),200);assert.ok(String((await health.json()).version).includes('17cef49b962943095df5374f65de7c74cda640dd'))
 const publicNode=await anon.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+id)
 assert.equal(publicNode.status(),200)
 assert.equal((await publicNode.json()).data.active.name,'RAT170 Image QA profile-url rat170img4way-20261115-r13')
 const media=await anon.request.get(DEV+'/api/v1/public/entities/rat-image/'+existingMedia+'/content')
 assert.equal(media.status(),404,'Previously processed 2MiB image was inadvertently published')
 pass('new-frontend-and-unchanged-api-preserve-original-large-image-private')
 stage='account'
 await page.goto(RAT+'/e/qa~'+id,{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toBeVisible({timeout:22000})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const editor=page.locator('.rat-workspace-profile .entity-about')
 await editor.getByRole('button',{name:'Create account',exact:true}).click()
 const username='ratlimit'+run.replace(/[^a-z0-9]/g,'').slice(-13)
 const pwd=randomBytes(22).toString('base64url')+'Aa1!'
 await editor.getByLabel('Email',{exact:true}).fill(username+'@example.com')
 await editor.getByLabel('Username',{exact:true}).fill(username)
 await editor.getByLabel('Password',{exact:true}).fill(pwd)
 await editor.getByLabel('Confirm password').fill(pwd)
 await editor.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await editor.getByRole('checkbox',{name:/Privacy Notice/}).check()
 const waitReg=page.waitForResponse(r=>r.request().method()==='POST'&&new URL(r.url()).pathname==='/api/v1/rate-anything/auth/register',{timeout:30000})
 await editor.getByRole('button',{name:'Create RAT account and continue'}).click()
 const registered=await waitReg;assert.ok([200,201].includes(registered.status()))
 const me=await ctx.request.get(RAT+'/api/v1/rate-anything/me')
 assert.equal(me.status(),200);report.createdAccounts.push((await me.json()).data.id)
 pass('genuine-unclaimed-thing-member-authenticated')
 stage='oversize'
 const form=editor.locator('.rat-thing-edit-form')
 await expect(form).toBeVisible({timeout:24000})
 await form.getByRole('button',{name:'Image',exact:true}).click()
 await form.getByRole('group',{name:'Image field'}).getByRole('button',{name:'Cover photo'}).click()
 await form.getByRole('group',{name:'Image source'}).getByRole('button',{name:'Upload image'}).click()
 const image=validNoisePng(2150,2150)
 assert.ok(image.length>12*1024*1024&&image.length<16*1024*1024)
 console.log('RAT170_FILE_SIZE_UI_VALID_OVER_LIMIT_PNG_BYTES='+image.length)
 await form.getByLabel('Upload Thing image').setInputFiles({name:'valid-but-over-12mib.png',mimeType:'image/png',buffer:image})
 await expect(form.getByText('Local file preview — not yet uploaded or reviewed.')).toBeVisible()
 const target='/api/v1/rate-anything/things/'+id+'/images/upload'
 const wait=page.waitForResponse(r=>r.request().method()==='POST'&&new URL(r.url()).pathname===target,{timeout:65000})
 await form.getByRole('button',{name:'Submit image for review'}).click()
 const resp=await wait
 const body=await resp.json().catch(()=>({}))
 report.oversizeResponse={status:resp.status(),fileError:body.errors?.file?.[0]??null,size:image.length}
 assert.equal(resp.status(),422,'API 12MiB file limit was not enforced (HTTP '+resp.status()+')')
 const fileError=body.errors?.file?.[0]??null
 assert.ok(typeof fileError==='string'&&fileError.includes('12288'),'Missing canonical 12MiB limit validation')
 const alert=page.getByRole('alert').filter({hasText:'Edit needs attention'})
 await expect(alert).toBeVisible({timeout:19000})
 await expect(alert).toContainText(fileError)
 await expect(alert).not.toContainText('One or more fields could not be accepted.')
 pass('over-12MiB-valid-image-rejected-422-and-exact-actionable-file-error-displayed')
 await shot('01-oversize-clear-validation')
 const stillPrivate=await anon.request.get(DEV+'/api/v1/public/entities/rat-image/'+existingMedia+'/content')
 assert.equal(stillPrivate.status(),404)
 pass('original-2MiB-processed-media-remains-private-and-unchanged')
 assert.equal(report.blocked.length,0,'Unexpected mutation or LIVE request')
 report.status='PASS'
 console.log('RAT170_FILE_SIZE_UI_BROWSER_PASS=true')
}catch(e){
 report.status='FAILED'
 report.errors.push({stage,message:String(e).slice(0,1400)})
 console.error('RAT170_FILE_SIZE_UI_FAIL_STAGE='+stage)
 console.error('RAT170_FILE_SIZE_UI_ERROR='+String(e).slice(0,1400))
 try{await shot('99-failure')}catch{}
 process.exitCode=1
}finally{
 await writeFile(out+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT170_FILE_SIZE_UI_REPORT='+JSON.stringify(report))
 await b.close()
}
NODE
