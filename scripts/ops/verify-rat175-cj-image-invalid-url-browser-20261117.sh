#!/usr/bin/env bash
set -Eeuo pipefail
# PR175 authentic RAT browser check with a disposable member.
# Only permitted write: create one new QA RAT account; deliberate invalid CJ URL
# is rejected before network/storage, no CJ image/revision/rating/identity mutation.
[[ $# -eq 4 ]] || { echo 'usage: source api-image rat-image unique-run-id' >&2; exit 64; }
src="$1";api="$2";rat="$3";rid="$4"
[[ "$src" == "a8aa3a5dcd8a60c154ff8e7c0b4055b50ecede18" ]] || exit 78
[[ "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$rid" =~ ^[a-z0-9-]{15,44}$ ]] || exit 64
out="/tmp/rat175-cj-browser-$rid"
[[ ! -e "$out" ]] || { echo 'Duplicate run id, inspect existing receipt' >&2; exit 65; }
mkdir -m 700 "$out"
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --retry 3 --retry-delay 2 --max-time 20 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$src" ]] || exit 78
health="$(curl -fsS --retry 3 --retry-delay 2 --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$health" "$src" <<'PY'
import json,sys
data=json.loads(sys.argv[1])
assert data.get("status")=="ok" and data.get("service")=="rum-api"
assert sys.argv[2] in str(data.get("version",""))
PY
# Read-only owner fingerprint, counts and canonical media state.
baseline='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;$id="01m07d2f9fj05f4ypdbwp087qj";
$e=$db::table("entities")->where("id",$id)->first();
if(!$e||$e->canonical_name!=="CJ Investigates"||$e->status!=="unverified"
 ||$e->visibility!=="public"||$e->publication_state!=="published"
 ||$e->edit_policy!=="claimant_controlled"||$e->edit_locked_at!==null)exit(78);
if($db::table("entity_types")->where("id",$e->entity_type_id)->value("key")!=="persona")exit(78);
if($db::table("entity_profile_revisions")->where("entity_id",$id)->whereIn("field_key",["profile_image","cover_image"])->count()!==0)exit(78);
if($db::table("entity_claims")->where("entity_id",$id)->where("status","approved")->whereNull("revoked_at")->exists())exit(78);
if($db::table("entity_managers")->where("entity_id",$id)->where("status","active")->whereNull("revoked_at")->exists())exit(78);
print "CJ_OWNER_READONLY_FINGERPRINT=".hash("sha256",json_encode(["id"=>$e->id,"status"=>$e->status,"publication"=>$e->publication_state,"version"=>$e->version,"ratingCount"=>$db::table("rating_events")->where("entity_id",$id)->count(),"revisions"=>0]))."\n";
'
before="$(kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$baseline")"
echo "$before"
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
for p in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do
 if [[ -x "$p" ]]; then chrome="$p"; break; fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
CHROME="$chrome" CJ_RUN_ID="$rid" CJ_OUTPUT="$out" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import {randomBytes} from 'node:crypto'
import {writeFile,stat,readFile} from 'node:fs/promises'
import {chromium,expect} from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk'
const CJ='01m07d2f9fj05f4ypdbwp087qj'
const INVALID='https://unsafe-cj-image.invalid/not-a-real-image.png'
const dir=process.env.CJ_OUTPUT
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const context=await browser.newContext({viewport:{width:1366,height:950}})
const page=await context.newPage()
const report={status:'NOT_ASSESSED',run:process.env.CJ_RUN_ID,errors:[],checked:[]}
const pass=(name)=>{report.checked.push(name);console.log('RAT175_BROWSER_PASS='+name)}
const allow=(req)=>{
 const u=new URL(req.url()),m=req.method()
 if(u.hostname==='rateurmate.online')return false
 if(['GET','HEAD','OPTIONS'].includes(m))return true
 if(u.origin!==RAT||m!=='POST')return false
 if(u.pathname==='/api/v1/rate-anything/auth/register')return true
 if(u.pathname!=='/api/v1/rate-anything/things/'+CJ+'/images/link')return false
 try{const body=req.postDataJSON();return body.slot==='profile'&&body.url===INVALID&&Array.isArray(body.evidence)&&body.evidence.length===0}catch{return false}
}
await page.route('**/*',route=>allow(route.request())?route.continue():route.abort('blockedbyclient'))
let stage='page'
try{
 await page.goto(RAT+'/e/cj-investigates~'+CJ,{waitUntil:'domcontentloaded',timeout:40000})
 await expect(page.locator('.rat-workspace-header h1')).toContainText('CJ Investigates',{timeout:30000})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const panel=page.locator('.rat-workspace-profile .entity-about')
 await panel.getByRole('button',{name:'Create account',exact:true}).click()
 stage='disposable-auth'
 const username='ratcj'+process.env.CJ_RUN_ID.replace(/[^a-z0-9]/g,'').slice(-14)
 const password=randomBytes(22).toString('base64url')+'aA1!'
 await panel.getByLabel('Email',{exact:true}).fill(username+'@example.com')
 await panel.getByLabel('Username',{exact:true}).fill(username)
 await panel.getByLabel('Password',{exact:true}).fill(password)
 await panel.getByLabel('Confirm password').fill(password)
 await panel.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await panel.getByRole('checkbox',{name:/Privacy Notice/}).check()
 const registration=page.waitForResponse(r=>r.request().method()==='POST'&&new URL(r.url()).pathname==='/api/v1/rate-anything/auth/register',{timeout:30000})
 await panel.getByRole('button',{name:'Create RAT account and continue'}).click()
 const reg=await registration
 assert.ok([200,201].includes(reg.status()),'Disposable registration status '+reg.status())
 pass('new-normal-qa-member-authenticated-no-claim')
 stage='image-editor'
 const edit=page.locator('.rat-workspace-profile .entity-about')
 await expect(edit.getByRole('button',{name:'Image',exact:true})).toBeVisible({timeout:20000})
 await edit.getByRole('button',{name:'Image',exact:true}).click()
 await edit.getByRole('button',{name:'Profile picture',exact:true}).click()
 await edit.getByRole('button',{name:'Use image URL',exact:true}).click()
 await edit.getByLabel('Thing image URL').fill(INVALID)
 const responseWait=page.waitForResponse(r=>r.request().method()==='POST'&&new URL(r.url()).pathname.endsWith('/things/'+CJ+'/images/link'),{timeout:25000})
 await edit.getByRole('button',{name:'Save image',exact:true}).click()
 const r=await responseWait
 assert.equal(r.status(),422,'Invalid public image URL must be rejected before fetch, not 404/500')
 const body=await r.json()
 assert.ok(typeof body.detail==='string'&&body.detail.includes('public image hosts'),JSON.stringify({status:r.status(),detail:body.detail}))
 assert.ok(/^[A-Za-z0-9._:-]{8,128}$/.test(body.requestId),'Missing stable requestId')
 assert.equal(body.requestId,r.headers()['x-request-id'])
 const alert=edit.getByRole('alert')
 await expect(alert).toContainText('Only public image hosts can be imported.')
 await expect(alert).toContainText('Request ID: '+body.requestId)
 const errorScreenshot=dir+'/cj-20261117-specific-image-error.jpg'
 await edit.screenshot({path:errorScreenshot,type:'jpeg',quality:65,animations:'disabled'})
 console.log('RAT175_DIAGNOSTIC_ERROR_SCREENSHOT='+errorScreenshot+' bytes='+(await stat(errorScreenshot)).size)
 pass('real-browser-422-with-actionable-link-validation-and-correlation-id')
 stage='reload'
 await page.reload({waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toContainText('CJ Investigates',{timeout:30000})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const panel2=page.locator('.rat-workspace-profile .entity-about')
 const history=await context.request.get(RAT+'/api/v1/rate-anything/things/'+CJ+'/edits')
 assert.equal(history.status(),200,'Public unverified persona should now allow authorised member history')
 const revisions=(await history.json()).data
 assert.ok(Array.isArray(revisions)&&revisions.every(rev=>!['profile_image','cover_image'].includes(rev.field)),'No new CJ image changes from invalid link')
 pass('CJ-image-history-readable-and-still-empty-after-refresh')
 const filename=dir+'/cj-20261117-image-editor-page.jpg'
 await page.screenshot({path:filename,type:'jpeg',quality:60,fullPage:true,animations:'disabled'})
 console.log('RAT175_DIAGNOSTIC_SCREENSHOT='+filename+' bytes='+(await stat(filename)).size)
 report.status='PASS'
 console.log('RAT175_CJ_REAL_BROWSER_DIAGNOSTIC_PASS=true')
}catch(e){
 report.status='FAIL';report.errors.push({stage,message:String(e).slice(0,1200)})
 console.log('RAT175_CJ_REAL_BROWSER_FAIL='+JSON.stringify(report.errors[0]))
 process.exitCode=1
}finally{
 await writeFile(dir+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT175_CJ_BROWSER_REPORT='+JSON.stringify(report))
 await browser.close()
}
NODE
after="$(kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$baseline")"
echo "$after"
[[ "$before" == "$after" ]] || { echo 'CJ data changed during rejected-link test' >&2; exit 78; }
echo RAT175_CJ_CANONICAL_RATING_AND_IMAGE_INTACT=true
