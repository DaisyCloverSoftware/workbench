#!/usr/bin/env bash
set -Eeuo pipefail
# Focused, one-contributor replacement/rollback on a fixed previously-vetted
# disposable public unclaimed RAT QA product; never the owner's Thing or LIVE.
[[ $# -eq 4 ]] || { echo 'usage: source-sha api@sha256 rat@sha256 run-id' >&2; exit 64; }
src="$1";api="$2";rat="$3";rid="$4"
[[ "$src" =~ ^[0-9a-f]{40}$ && "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ && "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ && "$rid" =~ ^[a-z0-9-]{10,44}$ ]] || exit 64
out="/tmp/rat170-replace-rollback-$rid"; [[ ! -e "$out" ]] || { echo 'Duplicate acceptance run ID' >&2; exit 65; };mkdir -m 700 "$out"
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
[[ "$(kctl -n rum-dev get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'rateurmate.online' ]] || exit 78
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --max-time 20 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$src" ]] || exit 78
read_only_qa='
require "vendor/autoload.php";
$app=require "bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;$id="01m7fvzcen292rvqeqj03evtxe";
$e=$db::table("entities")->where("id",$id)->first();
if(!$e||$e->canonical_name!=="RAT170 Image QA profile-url rat170img4way-20261115-r13"||$e->visibility!=="public"||$e->publication_state!=="published"||$e->edit_policy!=="community_governed"||$e->status!=="active"||$e->sensitivity!=="normal"||$e->edit_locked_at!==null)exit(78);
if($db::table("entity_claims")->where("entity_id",$id)->where("status","approved")->where("verification_level",">=",1)->whereNull("revoked_at")->exists())exit(78);
if($db::table("entity_managers")->where("entity_id",$id)->where("status","active")->whereNull("revoked_at")->exists())exit(78);
echo "RAT170_FOCUSED_QA_GUARD_PASS=true\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$read_only_qa"
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q;git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$src"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$src" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for b in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do [[ ! -x "$b" ]] || { chrome="$b"; break; }; done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
RAT170_RID="$rid" RAT170_OUT="$out" CHROME="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { deflateSync } from 'node:zlib'
import { writeFile, stat } from 'node:fs/promises'
import { chromium, expect } from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk'
const id='01m7fvzcen292rvqeqj03evtxe',name='RAT170 Image QA profile-url rat170img4way-20261115-r13'
const rid=process.env.RAT170_RID,dir=process.env.RAT170_OUT
const report={rid,status:'NOT_STARTED',steps:[],errors:[],revisions:[],images:[],responses:[],screenshots:[]}
const pass=x=>{report.steps.push(x);console.log('RAT170_FOCUSED_PASS='+x)}
const crc=(b)=>{let c=0xffffffff;for(const v of b){c^=v;for(let i=0;i<8;i++)c=(c>>>1)^((c&1)?0xedb88320:0)}return(c^0xffffffff)>>>0}
const chunk=(n,d)=>{const a=Buffer.from(n),l=Buffer.alloc(4),c=Buffer.alloc(4);l.writeUInt32BE(d.length);c.writeUInt32BE(crc(Buffer.concat([a,d])));return Buffer.concat([l,a,d,c])}
const png=()=>{const w=256,h=160,ih=Buffer.alloc(13);ih.writeUInt32BE(w,0);ih.writeUInt32BE(h,4);ih[8]=8;ih[9]=2;const scan=Buffer.alloc(h*(w*3+1));for(let y=0;y<h;y++)for(let x=0;x<w;x++){let z=y*(w*3+1)+1+x*3;scan[z]=39;scan[z+1]=96;scan[z+2]=186}return Buffer.concat([Buffer.from([137,80,78,71,13,10,26,10]),chunk('IHDR',ih),chunk('IDAT',deflateSync(scan)),chunk('IEND',Buffer.alloc(0))])}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const ctx=await browser.newContext({viewport:{width:1440,height:960}}),anon=await browser.newContext({viewport:{width:390,height:844}})
const page=await ctx.newPage()
const approved=(m,path)=>m==='POST'&&(path==='/api/v1/rate-anything/auth/register'||path==='/api/v1/rate-anything/things/'+id+'/images/upload'||/^\/api\/v1\/rate-anything\/things\/[0-9a-z]{26}\/edits\/[0-9a-z]{26}\/rollback$/.test(path))
await page.route('**/*',r=>{const q=r.request(),u=new URL(q.url());if(u.hostname==='rateurmate.online'||(!['GET','HEAD','OPTIONS'].includes(q.method())&&!approved(q.method(),u.pathname)))return r.abort('blockedbyclient');return r.continue()})
page.on('response',r=>{if(r.url().includes('/rat-catalogue/nodes/'+id)||r.url().includes('/images/upload'))report.responses.push({url:r.url().split('?')[0],status:r.status()})})
let stage='first-public-image'
const snap=async(label)=>{const path=dir+'/'+label+'.jpg';await page.screenshot({path,type:'jpeg',quality:45,fullPage:true,animations:'disabled'});const s=await stat(path);report.screenshots.push({label,bytes:s.size});console.log('RAT170_FOCUSED_SHOT='+label+' '+s.size)}
const node=async()=>{const r=await anon.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+id);assert.equal(r.status(),200);return (await r.json()).data.active}
try{
 const original=await node()
 assert.equal(original.name,name)
 const originalPath=new URL(original.images.profile.url).pathname
 assert.ok(originalPath.includes('/rat-image/'))
 report.images.push({step:'previous',path:originalPath,revisionId:original.images.profile.revisionId})
 pass('preexisting-public-avatar-confirmed')
 stage='registration'
 await page.goto(RAT+'/e/qa~'+id,{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toContainText(name,{timeout:30000})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const panel=page.locator('.rat-workspace-profile .entity-about')
 await panel.getByRole('button',{name:'Create account',exact:true}).click()
 const username='ratrep'+rid.replace(/[^a-z0-9]/g,'').slice(-14)
 const password=randomBytes(22).toString('base64url')+'Aa1!'
 await panel.getByLabel('Email',{exact:true}).fill(username+'@example.com')
 await panel.getByLabel('Username',{exact:true}).fill(username)
 await panel.getByLabel('Password',{exact:true}).fill(password)
 await panel.getByLabel('Confirm password').fill(password)
 await panel.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await panel.getByRole('checkbox',{name:/Privacy Notice/}).check()
 const reg=page.waitForResponse(r=>r.request().method()==='POST'&&new URL(r.url()).pathname.endsWith('/auth/register'),{timeout:30000})
 await panel.getByRole('button',{name:'Create RAT account and continue'}).click()
 assert.ok([200,201].includes((await reg).status()))
 await expect(panel.locator('.rat-thing-edit-form')).toBeVisible({timeout:30000})
 const memberResp=await ctx.request.get(RAT+'/api/v1/rate-anything/me')
 assert.equal(memberResp.status(),200)
 const memberId=(await memberResp.json()).data.id
 pass('fresh-unverified-community-contributor')
 stage='submit-replacement'
 const form=panel.locator('.rat-thing-edit-form')
 await form.getByRole('button',{name:'Image',exact:true}).click()
 await form.getByRole('group',{name:'Image field'}).getByRole('button',{name:'Profile picture'}).click()
 await form.getByRole('group',{name:'Image source'}).getByRole('button',{name:'Upload image'}).click()
 await form.getByLabel('Upload Thing image').setInputFiles({name:'rat170-focus-replacement.png',mimeType:'image/png',buffer:png()})
 const wait=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().includes('/things/'+id+'/images/upload'),{timeout:60000})
 await form.getByRole('button',{name:'Save image'}).click()
 const resp=await wait
 const payload=await resp.json().catch(()=>({}))
 assert.equal(resp.status(),201,'replacement response '+JSON.stringify(payload).slice(0,400))
 const rev=payload.data
 assert.equal(rev.outcome,'applied');assert.equal(rev.moderationStatus,'approved');assert.equal(payload.meta.applied,true)
 const current=await node()
 const newPath=new URL(current.images.profile.url).pathname
 assert.notEqual(originalPath,newPath)
 assert.equal(current.images.profile.revisionId,rev.id)
 report.images.push({step:'replacement',path:newPath,revisionId:rev.id})
 pass('canonical-replacement-applied-and-visible-from-public-api')
 stage='immediate-avatar'
 const selector='.rat-workspace-avatar > img'
 let immediate=true
 try{
  await expect(page.locator(selector)).toBeVisible({timeout:9000})
  await expect.poll(async()=>page.locator(selector).getAttribute('src'),{timeout:7000}).toContain(newPath)
 }catch(e){
  immediate=false
  console.log('RAT170_FOCUSED_IMMEDIATE_RENDER_FAILURE='+String(e).slice(0,450))
  console.log('RAT170_FOCUSED_RESPONSE_LOG='+JSON.stringify(report.responses).slice(-1800))
 }
 if(immediate)pass('immediate-profile-replacement-renders')
 await page.reload({waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toContainText(name,{timeout:25000})
 await expect(page.locator(selector)).toBeVisible({timeout:20000})
 await expect.poll(async()=>page.locator(selector).getAttribute('src'),{timeout:10000}).toContain(newPath)
 await expect.poll(async()=>page.locator(selector).evaluate(el=>el.complete&&el.naturalWidth>0),{timeout:10000}).toBe(true)
 pass('replacement-public-avatar-persists-after-refresh')
 await snap('01-replacement')
 const history=await ctx.request.get(RAT+'/api/v1/rate-anything/things/'+id+'/edits')
 assert.equal(history.status(),200)
 const rows=(await history.json()).data
 const ours=rows.find(x=>x.id===rev.id)
 const earlier=rows.find(x=>x.id===original.images.profile.revisionId)
 assert.ok(ours&&earlier)
 assert.equal(ours.editorUserId,memberId)
 assert.notEqual(ours.editorUserId,earlier.editorUserId)
 assert.equal(ours.previousValue,earlier.id)
 report.revisions=[{id:ours.id,editor:ours.editorUserId,outcome:ours.outcome},{id:earlier.id,editor:earlier.editorUserId,outcome:earlier.outcome}]
 pass('cross-contributor-revision-attribution-and-prior-link')
 stage='rollback'
 const rollbackPath='/api/v1/rate-anything/things/'+id+'/edits/'+rev.id+'/rollback'
 const res=await page.evaluate(async path=>{const cookie=document.cookie.split('; ').find(x=>x.startsWith('XSRF-TOKEN='));const token=cookie?decodeURIComponent(cookie.slice('XSRF-TOKEN='.length)):'';const r=await fetch(path,{method:'POST',credentials:'same-origin',headers:{Accept:'application/json',...(token?{'X-XSRF-TOKEN':token}:{})}});return{status:r.status,data:await r.json().catch(()=>({}))}},rollbackPath)
 assert.equal(res.status,200,'rollback '+JSON.stringify(res).slice(0,550))
 assert.equal(res.data.data.source,'rollback')
 const restored=await node()
 assert.equal(new URL(restored.images.profile.url).pathname,originalPath)
 const restoredResp=await anon.request.get(DEV+originalPath)
 assert.equal(restoredResp.status(),200)
 assert.ok((restoredResp.headers()['content-type']||'').includes('image/webp'))
 const replacedResp=await anon.request.get(DEV+newPath)
 assert.equal(replacedResp.status(),404)
 await page.reload({waitUntil:'domcontentloaded'})
 await expect(page.locator(selector)).toBeVisible({timeout:25000})
 assert.ok((await page.locator(selector).getAttribute('src')).includes(originalPath))
 pass('attributable-rollback-restores-original-public-image-after-refresh')
 await snap('02-rollback')
 report.status=immediate?'PASS':'FAILED_IMMEDIATE_RENDER'
 if(!immediate)throw Error('Replacement was applied publicly and persists after refresh, but immediate browser avatar did not update')
 console.log('RAT170_FOCUSED_REPLACE_ROLLBACK_BROWSER_PASS=true')
}catch(e){if(report.status==='NOT_STARTED')report.status='FAILED';report.errors.push({stage,error:String(e).slice(0,1800)});console.error('RAT170_FOCUSED_FAILED_STAGE='+stage);console.error('RAT170_FOCUSED_ERROR='+String(e).slice(0,1800));try{await snap('99-failure')}catch{}process.exitCode=1}
finally{await writeFile(dir+'/report.json',JSON.stringify(report,null,2));console.log('RAT170_FOCUSED_REPORT='+JSON.stringify(report));await browser.close()}
NODE
