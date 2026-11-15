#!/usr/bin/env bash
set -Eeuo pipefail
# Fixed disposable QA Thing only. Temporarily set "restricted" to create an
# author-owned private processed image, then restore community policy and use
# existing processed derivative. No owner Thing, seed, reset, claim or LIVE.
[[ $# -eq 4 ]] || exit 64
src="$1";api="$2";rat="$3";rid="$4"
[[ "$src" =~ ^[0-9a-f]{40}$ && "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ && "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ && "$rid" =~ ^[a-z0-9-]{10,44}$ ]] || exit 64
out="/tmp/rat170-processed-reuse-$rid"; [[ ! -e "$out" ]] || { echo 'Duplicate run id';exit 65; }; mkdir -m 700 "$out"
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --max-time 20 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$src" ]] || exit 78
php_boot='require "vendor/autoload.php";$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();$db=Illuminate\Support\Facades\DB::class;$id="01m7fw9xhpfmdw0cmpaf3zngkm";'
preflight="$php_boot"'
$e=$db::table("entities")->where("id",$id)->first();
if(!$e||$e->canonical_name!=="Purple Lantern Product Fixture rat170img4way-20261115-r15"||$e->edit_policy!=="community_governed"||$e->status!=="active"||$e->visibility!=="public"||$e->publication_state!=="published"||$e->sensitivity!=="normal"||$e->edit_locked_at!==null)exit(78);
$type=$db::table("entity_types")->where("id",$e->entity_type_id)->value("key");if($type!=="product")exit(78);
if($db::table("entity_claims")->where("entity_id",$id)->where("status","approved")->whereNull("revoked_at")->where("verification_level",">=",1)->exists())exit(78);
if($db::table("entity_managers")->where("entity_id",$id)->where("status","active")->whereNull("revoked_at")->exists())exit(78);
if($db::table("entity_profile_revisions")->where("entity_id",$id)->where("field_key","cover_image")->where("outcome","applied")->exists())exit(78);
echo "RAT170_REUSE_PRECHECK_DISPOSABLE_ONLY=true\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$preflight"
lock_php="$php_boot"'$n=$db::table("entities")->where("id",$id)->where("edit_policy","community_governed")->update(["edit_policy"=>"restricted"]);if($n!==1)exit(78);echo "RAT170_REUSE_TEMP_RESTRICTED=true\n";'
unlock_php="$php_boot"'$e=$db::table("entities")->where("id",$id)->first();if(!$e)exit(78);if($e->edit_policy==="restricted"){$n=$db::table("entities")->where("id",$id)->where("edit_policy","restricted")->update(["edit_policy"=>"community_governed"]);if($n!==1)exit(78);}elseif($e->edit_policy!=="community_governed")exit(78);echo "RAT170_REUSE_POLICY_RESTORED=true\n";'
restricted=false
restore(){
 if [[ "$restricted" == true ]];then
  kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$unlock_php" || true
 fi
}
trap restore EXIT
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$lock_php"
restricted=true
tmp="$(mktemp -d)";trap 'rm -rf "$tmp";restore' EXIT
git -C "$tmp" init -q;git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$src"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$src" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for x in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser;do
 if [[ -x "$x" ]];then chrome="$x";break;fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
RAT170_OUT="$out" RAT170_RUN="$rid" CHROME="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { deflateSync } from 'node:zlib'
import { execFileSync } from 'node:child_process'
import { writeFile, stat } from 'node:fs/promises'
import { chromium,expect } from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk'
const id='01m7fw9xhpfmdw0cmpaf3zngkm',name='Purple Lantern Product Fixture rat170img4way-20261115-r15'
const report={runid:process.env.RAT170_RUN,status:'NOT_STARTED',steps:[],revisionId:null,mediaId:null,errors:[]}
const pass=x=>{report.steps.push(x);console.log('RAT170_REUSE_PASS='+x)}
const crc=(b)=>{let c=0xffffffff;for(const v of b){c^=v;for(let i=0;i<8;i++)c=(c>>>1)^((c&1)?0xedb88320:0)}return(c^0xffffffff)>>>0}
const chunk=(n,d)=>{const a=Buffer.from(n),l=Buffer.alloc(4),c=Buffer.alloc(4);l.writeUInt32BE(d.length);c.writeUInt32BE(crc(Buffer.concat([a,d])));return Buffer.concat([l,a,d,c])}
const png=()=>{const w=330,h=200,info=Buffer.alloc(13);info.writeUInt32BE(w,0);info.writeUInt32BE(h,4);info[8]=8;info[9]=2;const pixels=Buffer.alloc(h*(w*3+1));for(let y=0;y<h;y++)for(let x=0;x<w;x++){const i=y*(w*3+1)+1+x*3;pixels[i]=95;pixels[i+1]=180;pixels[i+2]=80}return Buffer.concat([Buffer.from([137,80,78,71,13,10,26,10]),chunk('IHDR',info),chunk('IDAT',deflateSync(pixels)),chunk('IEND',Buffer.alloc(0))])}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const ctx=await browser.newContext({viewport:{width:1440,height:940}}),anon=await browser.newContext({viewport:{width:390,height:844}})
const page=await ctx.newPage()
await page.route('**/*',route=>{const r=route.request(),u=new URL(r.url()),m=r.method();const allowed=m==='POST'&&u.origin===RAT&&(u.pathname==='/api/v1/rate-anything/auth/register'||u.pathname==='/api/v1/rate-anything/things/'+id+'/images/upload'||/\/api\/v1\/rate-anything\/things\/[0-9a-z]{26}\/edits\/[0-9a-z]{26}\/publish-image$/.test(u.pathname));if(u.hostname==='rateurmate.online'||(!['GET','HEAD','OPTIONS'].includes(m)&&!allowed))return route.abort('blockedbyclient');return route.continue()})
const php='require "vendor/autoload.php";$app=require "bootstrap/app.php";$app->make(Illuminate\\Contracts\\Console\\Kernel::class)->bootstrap();$db=Illuminate\\Support\\Facades\\DB::class;$id="01m7fw9xhpfmdw0cmpaf3zngkm";$e=$db::table("entities")->where("id",$id)->first();if(!$e||$e->canonical_name!=="Purple Lantern Product Fixture rat170img4way-20261115-r15"||$e->edit_policy!=="restricted")exit(78);$n=$db::table("entities")->where("id",$id)->where("edit_policy","restricted")->update(["edit_policy"=>"community_governed"]);if($n!==1)exit(78);echo "RAT170_REUSE_POLICY_RESTORED_FOR_AUTHOR=true\\n";'
const restore=()=>{
 const r=execFileSync('sudo',['-n','k3s','kubectl','-n','rum-dev-isolated','exec','deployment/rum-api','-c','php-fpm','--','php','-r',php],{encoding:'utf8',timeout:20000})
 console.log(r.trim())
}
let stage='register'
try{
 await page.goto(RAT+'/e/qa~'+id,{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toContainText(name,{timeout:25000})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const editor=page.locator('.rat-workspace-profile .entity-about')
 await editor.getByRole('button',{name:'Create account',exact:true}).click()
 const username='ratupload'+report.runid.replace(/[^a-z0-9]/g,'').slice(-12)
 const password=randomBytes(24).toString('base64url')+'Aa1!'
 await editor.getByLabel('Email',{exact:true}).fill(username+'@example.com')
 await editor.getByLabel('Username',{exact:true}).fill(username)
 await editor.getByLabel('Password',{exact:true}).fill(password)
 await editor.getByLabel('Confirm password').fill(password)
 await editor.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await editor.getByRole('checkbox',{name:/Privacy Notice/}).check()
 const reg=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().includes('/auth/register'),{timeout:30000})
 await editor.getByRole('button',{name:'Create RAT account and continue'}).click()
 assert.ok([200,201].includes((await reg).status()))
 const form=editor.locator('.rat-thing-edit-form')
 await expect(form).toBeVisible({timeout:25000})
 pass('disposable-contributor-registered')
 stage='private-processed-upload'
 await form.getByRole('button',{name:'Image',exact:true}).click()
 await form.getByRole('group',{name:'Image field'}).getByRole('button',{name:'Cover photo'}).click()
 await form.getByRole('group',{name:'Image source'}).getByRole('button',{name:'Upload image'}).click()
 await form.getByLabel('Upload Thing image').setInputFiles({name:'rat170-private-pending-qa.png',mimeType:'image/png',buffer:png()})
 const wait=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().includes('/things/'+id+'/images/upload'),{timeout:60000})
 await form.getByRole('button',{name:'Save image'}).click()
 const resp=await wait,payload=await resp.json().catch(()=>({}))
 assert.equal(resp.status(),201)
 assert.equal(payload.data.outcome,'pending')
 assert.equal(payload.meta.applied,false)
 const rev=payload.data
 report.revisionId=rev.id
 assert.ok(rev.previewUrl)
 const assetId=/\/images\/([0-9a-z]{26})\/preview/.exec(new URL(rev.previewUrl,RAT).pathname)?.[1]
 assert.ok(assetId)
 report.mediaId=assetId
 const before=await anon.request.get(DEV+'/api/v1/public/entities/rat-image/'+assetId+'/content')
 assert.equal(before.status(),404)
 pass('private-processed-webp-held-on-restricted-qa-thing')
 stage='restore-policy'
 restore()
 pass('community-policy-restored-before-publication')
 stage='reuse-button'
 await expect(editor.getByRole('button',{name:'Use this processed image'})).toBeVisible({timeout:16000})
 const saved=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/edits/'+rev.id+'/publish-image'),{timeout:30000})
 await editor.getByRole('button',{name:'Use this processed image'}).click()
 const published=await saved, out=await published.json().catch(()=>({}))
 assert.equal(published.status(),200,'publish existing HTTP '+published.status()+' '+JSON.stringify(out).slice(0,350))
 assert.equal(out.data.outcome,'applied')
 assert.equal(out.meta.applied,true)
 const publicUrl=DEV+'/api/v1/public/entities/rat-image/'+assetId+'/content'
 const fetched=await anon.request.get(publicUrl)
 assert.equal(fetched.status(),200)
 assert.ok((fetched.headers()['content-type']||'').includes('image/webp'))
 await page.reload({waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-cover > img')).toBeVisible({timeout:22000})
 assert.ok((await page.locator('.rat-workspace-cover > img').getAttribute('src')).includes(assetId))
 await expect.poll(async()=>page.locator('.rat-workspace-cover > img').evaluate(el=>el.complete&&el.naturalWidth>0),{timeout:14000}).toBe(true)
 const history=await ctx.request.get(RAT+'/api/v1/rate-anything/things/'+id+'/edits')
 assert.equal(history.status(),200)
 const item=(await history.json()).data.find(x=>x.id===rev.id)
 assert.ok(item&&item.outcome==='applied')
 pass('same-original-processed-image-published-by-author-without-reupload-after-refresh')
 report.status='PASS'
 console.log('RAT170_PROCESSED_IMAGE_REUSE_BROWSER_PASS=true')
}catch(e){report.status='FAILED';report.errors.push({stage,message:String(e).slice(0,1700)});console.error('RAT170_REUSE_ERROR_STAGE='+stage);console.error('RAT170_REUSE_ERROR='+String(e).slice(0,1700));process.exitCode=1}
finally{await writeFile(process.env.RAT170_OUT+'/report.json',JSON.stringify(report,null,2));console.log('RAT170_REUSE_REPORT='+JSON.stringify(report));await browser.close()}
NODE
