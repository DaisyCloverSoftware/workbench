#!/usr/bin/env bash
set -Eeuo pipefail
# Run only on the exact isolated RAT/canonical DEV release, against previously
# vetted disposable QA Things. No owner Thing or LIVE mutation, no reseeding.
[[ $# -eq 4 ]] || { echo "usage: source-sha api@sha256 rat@sha256 unique-run-id" >&2; exit 64; }
src="$1"; api="$2"; rat="$3"; runid="$4"
[[ "$src" =~ ^[0-9a-f]{40}$ ]] || exit 64
[[ "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$runid" =~ ^[a-z0-9-]{10,44}$ ]] || exit 64
out="/tmp/rat170-direct-publication-$runid"
[[ ! -e "$out" ]] || { echo "Duplicate run ID: inspect original receipt" >&2; exit 65; }
mkdir -m 700 "$out"
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == "dev-rum.daisycloversoftware.uk" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == "dev-rum-ra.daisycloversoftware.uk" ]] || exit 78
[[ "$(kctl -n rum-dev get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == "rateurmate.online" ]] || exit 78
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS -H 'Cache-Control: no-cache' --max-time 20 https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$src" ]] || exit 78
health="$(curl -fsS --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$health" "$src" <<'PY'
import json,sys
j=json.loads(sys.argv[1])
assert j["status"]=="ok" and j["service"]=="rum-api" and sys.argv[2] in str(j["version"])
PY
# Refuse any mutation unless all four *named disposable* Things are
# still public, discoverable, unclaimed, unlocked product fixtures.
read_only_fixture_guard='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$ids=[
"01m7fvzcen292rvqeqj03evtxe",
"01m7fw9xhpfmdw0cmpaf3zngkm",
"01m7fw9yb4684hygrm14tmm3jr",
"01m7fw9z1b6n969xbmgdwew45t"
];
foreach($ids as $id){
 $e=$db::table("entities")->where("id",$id)->first();
 if(!$e || $e->status!=="active" || $e->visibility!=="public" || $e->publication_state!=="published" || $e->edit_policy!=="community_governed" || $e->edit_locked_at!==null || $e->sensitivity!=="normal")exit(78);
 $type=$db::table("entity_types")->where("id",$e->entity_type_id)->value("key");
 if($type!=="product")exit(78);
 if($db::table("entity_claims")->where("entity_id",$id)->where("status","approved")->whereNull("revoked_at")->where("verification_level",">=",1)->exists())exit(78);
 if($db::table("entity_managers")->where("entity_id",$id)->where("status","active")->whereNull("revoked_at")->exists())exit(78);
}
echo "RAT170_DISPOSABLE_UNCLAIMED_GUARDS_PASS=true\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$read_only_fixture_guard"
echo RAT170_DIRECT_DEPLOYMENT_PROVENANCE_OK=true
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$src"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$src" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=""
for b in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser;do
  if [[ -x "$b" ]];then chrome="$b";break;fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
RAT170_SRC="$src" RAT170_RID="$runid" RAT170_OUT="$out" RAT170_CHROME="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { deflateSync } from 'node:zlib'
import { writeFile, stat } from 'node:fs/promises'
import { chromium, expect } from '@playwright/test'

const RAT='https://dev-rum-ra.daisycloversoftware.uk'
const DEV='https://dev-rum.daisycloversoftware.uk'
const source=process.env.RAT170_SRC
const dir=process.env.RAT170_OUT
const runid=process.env.RAT170_RID
const qa=[
 {id:'01m7fvzcen292rvqeqj03evtxe',name:'RAT170 Image QA profile-url rat170img4way-20261115-r13',kind:'profile-url',slot:'profile',mode:'url'},
 {id:'01m7fw9xhpfmdw0cmpaf3zngkm',name:'Purple Lantern Product Fixture rat170img4way-20261115-r15',kind:'profile-upload',slot:'profile',mode:'upload'},
 {id:'01m7fw9yb4684hygrm14tmm3jr',name:'Green Tempest Artwork Catalogue rat170img4way-20261115-r15',kind:'cover-url',slot:'cover',mode:'url'},
 {id:'01m7fw9z1b6n969xbmgdwew45t',name:'Amber Prairie Display Collection rat170img4way-20261115-r15',kind:'cover-upload',slot:'cover',mode:'upload'},
]
const url='https://upload.wikimedia.org/wikipedia/commons/a/a9/Example.jpg'
const report={source,runid,status:'NOT_ASSESSED',checks:[],caseResults:[],screenshots:[],errors:[],accounts:[],mutations:[],fixtureIds:qa.map(x=>x.id)}
const mark=(name)=>{report.checks.push(name);console.log('RAT170_DIRECT_PASS='+name)}
const crc32=(b)=>{let c=0xffffffff;for(const v of b){c^=v;for(let i=0;i<8;i++)c=(c>>>1)^((c&1)?0xedb88320:0)}return(c^0xffffffff)>>>0}
const chunk=(name,data)=>{const n=Buffer.from(name,'ascii'),s=Buffer.alloc(4),c=Buffer.alloc(4);s.writeUInt32BE(data.length);c.writeUInt32BE(crc32(Buffer.concat([n,data])));return Buffer.concat([s,n,data,c])}
const png=(r,g,b,w=192,h=128)=>{const info=Buffer.alloc(13);info.writeUInt32BE(w,0);info.writeUInt32BE(h,4);info[8]=8;info[9]=2;const scan=Buffer.alloc((w*3+1)*h);for(let y=0;y<h;y++)for(let x=0;x<w;x++){const i=y*(w*3+1)+1+x*3;scan[i]=r;scan[i+1]=g;scan[i+2]=b}return Buffer.concat([Buffer.from([137,80,78,71,13,10,26,10]),chunk('IHDR',info),chunk('IDAT',deflateSync(scan)),chunk('IEND',Buffer.alloc(0))])}
const browser=await chromium.launch({headless:true,executablePath:process.env.RAT170_CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const ctx=await browser.newContext({viewport:{width:1440,height:960}})
const ctx2=await browser.newContext({viewport:{width:1440,height:960}})
const anon=await browser.newContext({viewport:{width:390,height:844}})
const page=await ctx.newPage(),other=await ctx2.newPage()
let stage='preflight'
const safeRoute=async (route)=>{const r=route.request();const u=new URL(r.url());const m=r.method();const approved=m==='POST'&&u.origin===RAT&&(/\/api\/v1\/rate-anything\/auth\/register$/.test(u.pathname)||/\/api\/v1\/rate-anything\/things\/[0-9a-z]{26}\/images\/(upload|link)$/.test(u.pathname)||/\/api\/v1\/rate-anything\/things\/[0-9a-z]{26}\/edits\/[0-9a-z]{26}\/rollback$/.test(u.pathname));if(u.hostname==='rateurmate.online'||(!['GET','HEAD','OPTIONS'].includes(m)&&!approved)){report.mutations.push({method:m,path:u.pathname,blocked:true});return route.abort('blockedbyclient')}if(!['GET','HEAD','OPTIONS'].includes(m))report.mutations.push({method:m,path:u.pathname,blocked:false});return route.continue()}
await page.route('**/*',safeRoute)
await other.route('**/*',safeRoute)
const shot=async (p,name)=>{const path=dir+'/'+name+'.jpg';await p.screenshot({path,type:'jpeg',quality:45,animations:'disabled',fullPage:true});const s=await stat(path);report.screenshots.push({name,bytes:s.size,path});console.log('RAT170_DIRECT_SHOT='+name+' bytes='+s.size)}
const node=async (id)=>{const r=await anon.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+id);assert.equal(r.status(),200);return (await r.json()).data.active}
const register=async (p,role,id)=>{
 await p.goto(RAT+'/e/qa~'+id,{waitUntil:'domcontentloaded'})
 await expect(p.locator('.rat-workspace-header h1')).toBeVisible({timeout:30000})
 await p.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const editor=p.locator('.rat-workspace-profile .entity-about')
 await editor.getByRole('button',{name:'Create account',exact:true}).click()
 const username='ratdirect'+role+runid.replace(/[^a-z0-9]/g,'').slice(-9)
 const password=randomBytes(24).toString('base64url')+'Aa1!'
 await editor.getByLabel('Email',{exact:true}).fill(username+'@example.com')
 await editor.getByLabel('Username',{exact:true}).fill(username)
 await editor.getByLabel('Password',{exact:true}).fill(password)
 await editor.getByLabel('Confirm password').fill(password)
 await editor.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await editor.getByRole('checkbox',{name:/Privacy Notice/}).check()
 const resp=p.waitForResponse(x=>x.request().method()==='POST'&&new URL(x.url()).pathname==='/api/v1/rate-anything/auth/register',{timeout:30000})
 await editor.getByRole('button',{name:'Create RAT account and continue'}).click()
 const r=await resp
 assert.ok([200,201].includes(r.status()),'Disposable sign-up HTTP '+r.status())
 await expect(editor.locator('.rat-thing-edit-form')).toBeVisible({timeout:25000})
 const me=await p.request.get(RAT+'/api/v1/rate-anything/me')
 assert.equal(me.status(),200)
 const idCreated=(await me.json()).data.id
 report.accounts.push({role,id:idCreated})
 return idCreated
}
const publish=async (p,test,replace=false)=>{
 await p.goto(RAT+'/e/qa~'+test.id,{waitUntil:'domcontentloaded'})
 await expect(p.locator('.rat-workspace-header h1')).toContainText(test.name,{timeout:25000})
 const before=await node(test.id)
 assert.equal(before.name,test.name)
 if(test.slot==='profile')await p.locator('.rat-workspace-avatar-controls .rat-workspace-image-action').click()
 else await p.locator('.rat-workspace-cover .rat-workspace-image-action').click()
 const editor=p.locator('.rat-workspace-profile .entity-about')
 const form=editor.locator('.rat-thing-edit-form')
 await expect(form).toBeVisible({timeout:20000})
 await form.getByRole('button',{name:'Image',exact:true}).click()
 await form.getByRole('group',{name:'Image field'}).getByRole('button',{name:test.slot==='profile'?'Profile picture':'Cover photo'}).click()
 const m=replace?'upload':test.mode
 if(m==='upload'){
  await form.getByRole('group',{name:'Image source'}).getByRole('button',{name:'Upload image'}).click()
  await form.getByLabel('Upload Thing image').setInputFiles({name:'rat170-disposable-'+test.kind+(replace?'-replace':'')+'.png',mimeType:'image/png',buffer:replace?png(35,152,72):png(180,70,140)})
  await expect(form.getByText('Local file preview — not yet saved.')).toBeVisible()
 }else{
  await form.getByRole('group',{name:'Image source'}).getByRole('button',{name:'Use image URL'}).click()
  await form.getByLabel('Thing image URL').fill(url)
 }
 stage=(replace?'replace-':'save-')+test.kind
 const requestPath='/api/v1/rate-anything/things/'+test.id+'/images/'+(m==='upload'?'upload':'link')
 const wait=p.waitForResponse(x=>x.request().method()==='POST'&&new URL(x.url()).pathname===requestPath,{timeout:70000})
 await form.getByRole('button',{name:'Save image'}).click()
 const resp=await wait
 const payload=await resp.json().catch(()=>({}))
 assert.equal(resp.status(),201,'Image save did not create an applied revision '+test.kind+' '+resp.status()+' '+JSON.stringify(payload).slice(0,350))
 const revision=payload.data
 assert.equal(revision.thingId,test.id);assert.equal(revision.field,test.slot==='profile'?'profile_image':'cover_image')
 assert.equal(revision.outcome,'applied');assert.equal(revision.moderationStatus,'approved');assert.equal(payload.meta?.applied,true)
 assert.ok(revision.previewUrl,'processed media reference missing')
 const match=/\/images\/([0-9a-z]{26})\/preview/.exec(new URL(revision.previewUrl,RAT).pathname)
 assert.ok(match,'media id missing from protected preview')
 const mediaId=match[1]
 await expect(editor.getByText('Image published and saved to this Thing.')).toBeVisible({timeout:20000})
 const path='/api/v1/public/entities/rat-image/'+mediaId+'/content'
 const content=await anon.request.get(DEV+path)
 assert.equal(content.status(),200,'Public media not served: '+test.kind)
 assert.ok((content.headers()['content-type']||'').includes('image/webp'))
 assert.ok((await content.body()).length>50)
 await expect.poll(async()=>((await node(test.id)).images?.[test.slot]?.url||''),{timeout:25000}).toContain(path)
 const selector=test.slot==='profile'?'.rat-workspace-avatar > img':'.rat-workspace-cover > img'
 await expect(p.locator(selector)).toBeVisible({timeout:25000})
 await expect.poll(async()=>p.locator(selector).evaluate(el=>el.complete&&el.naturalWidth>0),{timeout:25000}).toBe(true)
 const onPage=await p.locator(selector).getAttribute('src')
 assert.ok(onPage&&onPage.includes(path),'Public image did not render from canonical endpoint')
 await p.reload({waitUntil:'domcontentloaded'})
 await expect(p.locator(selector)).toBeVisible({timeout:25000})
 await expect.poll(async()=>p.locator(selector).evaluate(el=>el.complete&&el.naturalWidth>0),{timeout:25000}).toBe(true)
 assert.ok((await p.locator(selector).getAttribute('src')).includes(path),'image lost after refresh')
 const history=await p.request.get(RAT+'/api/v1/rate-anything/things/'+test.id+'/edits')
 assert.equal(history.status(),200)
 const own=(await history.json()).data.find(x=>x.id===revision.id)
 assert.ok(own&&own.outcome==='applied'&&own.editorUserId, 'Attributable applied revision missing')
 await p.getByRole('button',{name:'Edit or suggest a correction'}).click()
 await expect(p.locator('.rat-thing-revision').filter({hasText:'Your submission'}).first()).toContainText('Applied',{timeout:20000})
 const result={kind:replace?'replacement':test.kind,thingId:test.id,revisionId:revision.id,mediaId,editorId:own.editorUserId,slot:test.slot,publicPath:path,refresh:true}
 report.caseResults.push(result)
 await shot(p,(replace?'05-replacement':'0'+(qa.findIndex(x=>x.id===test.id)+1)+'-'+test.kind)+'-published')
 mark((replace?'replacement':test.kind)+'-public-webp-visible-after-refresh-and-attributed')
 return result
}
try{
 for(const t of qa){const n=await node(t.id);assert.equal(n.name,t.name);assert.equal(n.visibility??'public','public')}
 mark('four-previously-vetted-disposable-public-things-only')
 const a=await register(page,'one',qa[0].id)
 const b=await register(other,'two',qa[0].id)
 assert.notEqual(a,b,'noncreator contributors must be distinct')
 mark('two-disposable-authenticated-contributors')
 for(const t of qa)await publish(page,t)
 const first=report.caseResults[0]
 const second=await publish(other,qa[0],true)
 assert.notEqual(second.mediaId,first.mediaId)
 const prior=await anon.request.get(DEV+first.publicPath)
 assert.equal(prior.status(),404,'Superseded image still accessible as current public avatar')
 stage='rollback'
 const rollbackPath='/api/v1/rate-anything/things/'+qa[0].id+'/edits/'+second.revisionId+'/rollback'
 const resp=await other.evaluate(async (path)=>{
  const c=document.cookie.split('; ').find(i=>i.startsWith('XSRF-TOKEN='))
  const token=c?decodeURIComponent(c.slice('XSRF-TOKEN='.length)):''
  const r=await fetch(path,{method:'POST',credentials:'same-origin',headers:{Accept:'application/json',...(token?{'X-XSRF-TOKEN':token}:{})}})
  return {status:r.status,body:await r.json().catch(()=>({}))}
 },rollbackPath)
 assert.equal(resp.status,200,'rollback returned '+JSON.stringify(resp).slice(0,330))
 assert.equal(resp.body.data.source,'rollback')
 await expect.poll(async()=>(await node(qa[0].id)).images?.profile?.url||'',{timeout:20000}).toContain(first.publicPath)
 await other.reload({waitUntil:'domcontentloaded'})
 await expect(other.locator('.rat-workspace-avatar > img')).toBeVisible()
 assert.ok((await other.locator('.rat-workspace-avatar > img').getAttribute('src')).includes(first.publicPath))
 const restored=await anon.request.get(DEV+first.publicPath)
 assert.equal(restored.status(),200)
 const former=await anon.request.get(DEV+second.publicPath)
 assert.equal(former.status(),404)
 mark('second-member-replacement-and-attributable-image-rollback')
 await shot(other,'06-rollback-restored')
 stage='regressions'
 await page.goto(RAT+'/search?q=DJI',{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-active-card h1',{hasText:'DJI'})).toBeVisible({timeout:30000})
 await page.goto(RAT+'/search?q=CJ%20Investigates',{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-active-card h1',{hasText:'CJ Investigates'})).toBeVisible({timeout:30000})
 const types=await ctx.request.get(RAT+'/api/v1/rate-anything/thing-types?q=YouTube%20Channel')
 assert.equal(types.status(),200);assert.ok(JSON.stringify(await types.json()).toLowerCase().includes('youtube channel'))
 await page.goto(RAT+'/search?q=rat170-direct-no-match-unfindable-1234567',{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-message')).toContainText('No RAT-visible Thing matched',{timeout:30000})
 await expect(page.locator('.rat-message button.rat-primary-action')).toBeVisible({timeout:30000})
 mark('DJI-CJ-search-youtube-type-and-no-match-add-action')
 await page.goto(RAT+'/e/qa~'+qa[0].id,{waitUntil:'domcontentloaded'})
 await expect(page.getByRole('button',{name:'Rate',exact:true})).toBeVisible({timeout:20000})
 await expect(page.getByRole('button',{name:'Add a connection'})).toBeVisible({timeout:20000})
 await page.setViewportSize({width:390,height:844})
 await expect(page.locator('.rat-workspace-avatar > img')).toBeVisible()
 const overflow=await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)
 assert.ok(overflow<=2,'mobile overflow '+overflow)
 mark('mobile-avatar-branding-rating-and-connections-controls')
 await shot(page,'07-mobile-after-rollback')
 assert.equal(report.mutations.filter(m=>m.blocked).length,0,'unexpected blocked mutation attempt')
 report.status='PASS'
 console.log('RAT170_DIRECT_PUBLICATION_BROWSER_PASS=true')
}catch(e){
 report.status='FAILED';report.errors.push({stage,message:String(e).slice(0,1700)})
 console.error('RAT170_DIRECT_FAILED_STAGE='+stage)
 console.error('RAT170_DIRECT_ERROR='+String(e).slice(0,1700))
 try{await shot(page,'99-failure')}catch{}
 process.exitCode=1
}finally{
 await writeFile(dir+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT170_DIRECT_REPORT='+JSON.stringify(report))
 await browser.close()
}
NODE
