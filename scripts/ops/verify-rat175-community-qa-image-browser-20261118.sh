#!/usr/bin/env bash
set -Eeuo pipefail
# Isolated DEV QA: one existing synthetic UNCLAIMED persona. Never alter CJ,
# claims, ratings or LIVE. Upload a generated QA graphic and roll it back.
[[ $# -eq 4 ]] || { echo 'usage: exact-source api-digest rat-digest unique-run-id' >&2; exit 64; }
src="$1";api="$2";rat="$3";rid="$4"
[[ "$src" == '9c8af9fe07e9a0c4b0b1896f4be1903719b781e3' ]] || exit 78
[[ "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$api" == '__CJ_COMMUNITY_API_DIGEST_UNPUBLISHED__' ]] || exit 78
[[ "$rat" == '__CJ_COMMUNITY_RAT_DIGEST_UNPUBLISHED__' ]] || exit 78
[[ "$rid" =~ ^[a-z0-9-]{15,44}$ ]] || exit 64
dir="/tmp/rat175-community-qa-browser-$rid"
[[ ! -e "$dir" ]] || { echo 'QA browser run already exists' >&2; exit 65; }
mkdir -m 700 "$dir"
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --retry 3 --retry-delay 2 --max-time 20 https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$src" ]] || exit 78
health="$(curl -fsS --retry 3 --retry-delay 2 --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$health" "$src" <<'PY'
import json,sys
h=json.loads(sys.argv[1])
assert h.get("status")=="ok" and h.get("service")=="rum-api" and sys.argv[2] in str(h.get("version",""))
PY
# Fail closed unless the original demo identity is eligible, unclaimed, image-free,
# published and unlocked. Do not create/seed/rename any Thing.
guard='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$id="01m784s54kww2vqw891mnmce71";$e=App\Models\Entity::query()->findOrFail($id);$db=Illuminate\Support\Facades\DB::class;
$policy=app(App\Services\RatCommunityContributionPolicy::class);
if($e->canonical_name!=="RAT Demo Drone Creator"||$e->type?->key!=="persona"||$e->status!=="active"
 ||$e->visibility!=="public"||$e->publication_state!=="published"||$e->edit_policy!=="claimant_controlled"
 ||$e->edit_locked_at!==null||$policy->isClaimed($e))exit(78);
if($db::table("entity_profile_revisions")->where("entity_id",$id)->where("field_key","profile_image")->exists())exit(78);
if(app(App\Services\ThingImageService::class)->publicImages($e)["profile"]!==null)exit(78);
echo "RAT175_SYNTHETIC_QA_PERSONA_SAFE=true\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$guard"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$src"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$src" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for p in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser;do
 if [[ -x "$p" ]];then chrome="$p";break;fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
CHROME="$chrome" QA_RUN_ID="$rid" QA_DIR="$dir" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import {randomBytes} from 'node:crypto'
import {writeFile,stat} from 'node:fs/promises'
import {chromium,expect} from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk'
const DEV='https://dev-rum.daisycloversoftware.uk'
const QA='01m784s54kww2vqw891mnmce71'
const pagePath='/e/rat-demo-drone-creator~'+QA
const OUT=process.env.QA_DIR
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const ctx=await browser.newContext({viewport:{width:1440,height:940}})
const page=await ctx.newPage()
let savedRevision=null,rolledBack=false,publishedUrl=null,stage='initial'
const report={run:process.env.QA_RUN_ID,status:'NOT_ASSESSED',steps:[],screenshots:[],errors:[]}
const pass=(k)=>{report.steps.push(k);console.log('RAT175_COMMUNITY_QA_PASS='+k)}
const allowed=(req)=>{
 const u=new URL(req.url()),m=req.method()
 if(u.hostname==='rateurmate.online')return false
 if(['GET','HEAD','OPTIONS'].includes(m))return true
 if(u.origin!==RAT||m!=='POST')return false
 return u.pathname==='/api/v1/rate-anything/auth/register'
  ||u.pathname==='/api/v1/rate-anything/things/'+QA+'/images/upload'
  || /^\/api\/v1\/rate-anything\/things\/01m784s54kww2vqw891mnmce71\/edits\/[0-9a-z]+\/rollback$/.test(u.pathname)
}
await page.route('**/*',route=>allowed(route.request())?route.continue():route.abort('blockedbyclient'))
const postJson=async(path,body={})=>page.evaluate(async ({path,body})=>{
 const xsrf=document.cookie.split('; ').find(x=>x.startsWith('XSRF-TOKEN='))
 const token=xsrf?decodeURIComponent(xsrf.slice('XSRF-TOKEN='.length)):''
 const r=await fetch(path,{method:'POST',credentials:'same-origin',headers:{Accept:'application/json','Content-Type':'application/json',...(token?{'X-XSRF-TOKEN':token}:{})},body:JSON.stringify(body)})
 return {status:r.status,data:await r.json().catch(()=>({}))}
},{path,body})
const publicNode=async()=>{const r=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+QA);assert.equal(r.status(),200);return (await r.json()).data.active}
const screenshot=async(stem)=>{
 const name=stem+'.jpg',path=OUT+'/'+name
 await page.screenshot({path,type:'jpeg',quality:63,fullPage:true,animations:'disabled'})
 const bytes=(await stat(path)).size
 assert.ok(bytes>1500 && bytes<650000)
 report.screenshots.push({name,bytes});console.log('RAT175_COMMUNITY_QA_SCREENSHOT='+name+' bytes='+bytes)
}
try{
 stage='before'
 const before=await publicNode()
 assert.equal(before.name,'RAT Demo Drone Creator')
 assert.equal(before.images?.profile??null,null)
 await page.goto(RAT+pagePath,{waitUntil:'domcontentloaded',timeout:40000})
 await expect(page.locator('.rat-workspace-header h1')).toContainText('RAT Demo Drone Creator',{timeout:30000})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const panel=page.locator('.rat-workspace-profile .entity-about')
 await panel.getByRole('button',{name:'Create account',exact:true}).click()
 stage='account'
 const username='ratqa'+process.env.QA_RUN_ID.replace(/[^a-z0-9]/g,'').slice(-13)
 const password=randomBytes(22).toString('base64url')+'aA1!'
 await panel.getByLabel('Email',{exact:true}).fill(username+'@example.com')
 await panel.getByLabel('Username',{exact:true}).fill(username)
 await panel.getByLabel('Password',{exact:true}).fill(password)
 await panel.getByLabel('Confirm password').fill(password)
 await panel.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await panel.getByRole('checkbox',{name:/Privacy Notice/}).check()
 const registered=page.waitForResponse(r=>r.request().method()==='POST'&&new URL(r.url()).pathname==='/api/v1/rate-anything/auth/register',{timeout:30000})
 await panel.getByRole('button',{name:'Create RAT account and continue'}).click()
 assert.ok([200,201].includes((await registered).status()))
 pass('ordinary-community-member-registered-no-identity-claim')
 stage='profile-upload'
 const editor=page.locator('.rat-workspace-profile .entity-about')
 await expect(editor.getByRole('button',{name:'Image',exact:true})).toBeVisible({timeout:25000})
 await editor.getByRole('button',{name:'Image',exact:true}).click()
 await editor.getByRole('button',{name:'Profile picture',exact:true}).click()
 await editor.getByRole('button',{name:'Upload image',exact:true}).click()
 // Create an unmistakably disposable QA graphic in the browser: no external or owner image.
 const png=await page.evaluate(()=>{
  const c=document.createElement('canvas');c.width=160;c.height=160;const g=c.getContext('2d')
  g.fillStyle='#f4f6f8';g.fillRect(0,0,160,160)
  g.fillStyle='#234c8e';g.fillRect(13,13,134,134)
  g.font='bold 58px sans-serif';g.fillStyle='white';g.textAlign='center';g.fillText('QA',80,99)
  return c.toDataURL('image/png').split(',')[1]
 })
 await editor.getByLabel('Upload Thing image').setInputFiles({name:'temporary-qa-profile.png',mimeType:'image/png',buffer:Buffer.from(png,'base64')})
 const uploaded=page.waitForResponse(r=>r.request().method()==='POST'&&new URL(r.url()).pathname.endsWith('/things/'+QA+'/images/upload'),{timeout:35000})
 await editor.getByRole('button',{name:'Save image',exact:true}).click()
 const response=await uploaded,body=await response.json()
 assert.equal(response.status(),201,JSON.stringify(body).slice(0,500))
 assert.equal(body.meta?.applied,true,'Unclaimed public persona must publish directly')
 assert.equal(body.data?.outcome,'applied')
 assert.equal(body.data?.sourceType,'uploaded')
 savedRevision=body.data.id
 assert.match(savedRevision,/^[0-9a-z]{26}$/)
 pass('unclaimed-persona-profile-upload-directly-published-with-canonical-revision')
 const after=await publicNode()
 publishedUrl=after.images?.profile?.url
 assert.ok(publishedUrl?.includes('/api/v1/public/entities/rat-image/'))
 assert.equal(after.images.profile.status,'published')
 assert.equal(after.id,QA)
 await page.reload({waitUntil:'domcontentloaded'})
 const avatar=page.locator('.rat-workspace-avatar > img')
 await expect(avatar).toHaveAttribute('src',publishedUrl,{timeout:30000})
 await expect.poll(()=>avatar.evaluate(el=>el.complete&&el.naturalWidth>0),{timeout:25000}).toBe(true)
 pass('profile-image-persisted-after-reload-and-rendered-in-canonical-thing')
 await screenshot('01-qa-thing-profile-published')
 stage='search'
 await page.goto(RAT+'/search?q=demo+drone&active='+QA,{waitUntil:'domcontentloaded',timeout:40000})
 await expect(page.locator('.rat-active-card h1')).toContainText('RAT Demo Drone Creator',{timeout:30000})
 const searchAvatar=page.locator('.rat-active-card__avatar > img')
 await expect(searchAvatar).toHaveAttribute('src',publishedUrl)
 await expect.poll(()=>searchAvatar.evaluate(el=>el.complete&&el.naturalWidth>0)).toBe(true)
 await screenshot('02-qa-search-published-desktop')
 await page.setViewportSize({width:390,height:860})
 await expect.poll(()=>searchAvatar.evaluate(el=>el.complete&&el.naturalWidth>0)).toBe(true)
 assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)<=2)
 await screenshot('03-qa-search-published-mobile')
 pass('same-published-canonical-image-in-rum-rat-search-desktop-mobile')
 stage='rollback'
 const rollback=await postJson('/api/v1/rate-anything/things/'+QA+'/edits/'+savedRevision+'/rollback')
 assert.equal(rollback.status,200,JSON.stringify(rollback.data).slice(0,600))
 assert.equal(rollback.data.data?.source,'rollback')
 rolledBack=true
 const restored=await publicNode()
 assert.equal(restored.images?.profile??null,null,'QA image not cleared after rollback')
 assert.equal(restored.id,QA)
 const old=await ctx.request.get(publishedUrl)
 assert.equal(old.status(),404,'Rolled-back QA image still publicly visible')
 pass('qa-image-rollback-restores-original-blank-profile-and-hides-old-media')
 report.status='PASS'
 console.log('RAT175_COMMUNITY_QA_REAL_BROWSER_PASS=true')
}catch(e){
 report.status='FAIL';report.errors.push({stage,message:String(e).slice(0,1300)})
 console.log('RAT175_COMMUNITY_QA_BROWSER_FAIL='+JSON.stringify(report.errors[0]))
 process.exitCode=1
}finally{
 if(savedRevision&&!rolledBack){
  try{
   await page.goto(RAT+pagePath,{waitUntil:'domcontentloaded',timeout:40000})
   const rollback=await postJson('/api/v1/rate-anything/things/'+QA+'/edits/'+savedRevision+'/rollback')
   if(rollback.status===200){rolledBack=true;pass('emergency-qa-profile-rollback-success')}
   else{report.errors.push({stage:'emergency-rollback',message:'HTTP '+rollback.status});process.exitCode=1}
  }catch(e){report.errors.push({stage:'emergency-rollback',message:String(e).slice(0,400)});process.exitCode=1}
 }
 report.qaRolledBack=rolledBack
 await writeFile(OUT+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT175_COMMUNITY_QA_REPORT='+JSON.stringify(report))
 await browser.close()
}
NODE
echo RAT175_QA_BROWSER_RUN_RECORDED=true
