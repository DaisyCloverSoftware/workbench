#!/usr/bin/env bash
set -euo pipefail
# Exact PR170 source, existing synthetic DEV Things only. Negative requests, duplicate reuse,
# and scoped read-only before/after database checks. Never mutate LIVE, private Things, claims or ratings.
[[ $# -eq 2 ]] || exit 64
sha="$1"; rid="$2"
[[ "$sha" == "e3bb818f1825d778f5084460057ba7ea8bfb422b" ]] || exit 78
[[ "$rid" =~ ^[a-z0-9-]{8,42}$ ]] || exit 64
root="01m7d2emwsnd0nxw6new8khbje"
target="01m7bt378nne388rnrrhtzsrgn"
pending="01m7dbsjhn6a9j373va2yz42yx"
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get ingress -o jsonpath='{.items[0].spec.rules[0].host}')" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get ingress -o jsonpath='{.items[0].spec.rules[0].host}')" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
snapshot(){
  kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r '
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$root="01m7d2emwsnd0nxw6new8khbje";$target="01m7bt378nne388rnrrhtzsrgn";$pending="01m7dbsjhn6a9j373va2yz42yx";
echo json_encode([
  "rootRevisions"=>$db::table("entity_profile_revisions")->where("entity_id",$root)->count(),
  "rootActiveEdges"=>$db::table("entity_relationships")->whereNull("valid_to")->where(fn($q)=>$q->where("source_entity_id",$root)->orWhere("target_entity_id",$root))->count(),
  "originalPairActiveEdges"=>$db::table("entity_relationships")->whereNull("valid_to")->where(fn($q)=>$q->where(fn($a)=>$a->where("source_entity_id",$root)->where("target_entity_id",$target))->orWhere(fn($a)=>$a->where("source_entity_id",$target)->where("target_entity_id",$root)))->count(),
  "pendingPairActiveEdges"=>$db::table("entity_relationships")->whereNull("valid_to")->where(fn($q)=>$q->where(fn($a)=>$a->where("source_entity_id",$root)->where("target_entity_id",$pending))->orWhere(fn($a)=>$a->where("source_entity_id",$pending)->where("target_entity_id",$root)))->count(),
  "pendingThingState"=>$db::table("entities")->where("id",$pending)->value("publication_state"),
], JSON_THROW_ON_ERROR)."\n";
'
}
before="$(snapshot)"
dir="/tmp/rat170-connection-security-$rid"
[[ ! -e "$dir" ]] || { echo 'Evidence exists; do not repeat'; exit 65; }
mkdir -m 700 "$dir"
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$sha"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$sha" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=""
for c in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do if [[ -x "$c" ]]; then chrome="$c";break;fi;done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
set +e
SOURCE="$sha" RID="$rid" ROOT="$root" TARGET="$target" PENDING="$pending" OUT="$dir" CHROME_PATH="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { writeFile } from 'node:fs/promises'
import { chromium, expect } from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk'
const root=process.env.ROOT,target=process.env.TARGET,pending=process.env.PENDING
const sha=process.env.SOURCE,id=process.env.RID
const report={source:sha,runId:id,root,target,pending,status:'UNVERIFIED',checks:[],responses:{},errors:[]}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const context=await browser.newContext({viewport:{width:1280,height:940}})
const page=await context.newPage()
const note=x=>{report.checks.push(x);console.log('RAT170_SECURITY_PASS='+x)}
let stage='versions'
await context.route('**/*',route=>{
 const u=new URL(route.request().url()),m=route.request().method()
 if(u.hostname==='rateurmate.online'||(!['GET','HEAD','OPTIONS'].includes(m)&&![RAT,DEV].includes(u.origin)))return route.abort('blockedbyclient')
 return route.continue()
})
try{
 const version=await context.request.get(RAT+'/VERSION',{headers:{'Cache-Control':'no-cache'}})
 assert.equal(version.status(),200);assert.equal((await version.text()).trim(),'rat-candidate+'+sha)
 const health=await context.request.get(DEV+'/api/v1/health')
 assert.equal(health.status(),200);assert.ok(String((await health.json()).version).includes(sha))
 note('exact-served-api-and-frontend')
 stage='register-distinct-dev-test-account'
 await page.goto(RAT+'/e/community~'+root,{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toBeVisible({timeout:28000})
 await page.getByRole('button',{name:'Add a connection'}).click()
 const panel=page.locator('.rat-community-panel')
 await panel.getByRole('button',{name:'Create account',exact:true}).click()
 const username='ratneg'+id.replace(/[^a-z0-9]/g,'').slice(-12)
 const password=randomBytes(20).toString('base64url')+'Aa1!'
 await panel.getByLabel('Email',{exact:true}).fill(username+'@example.com')
 await panel.getByLabel('Username',{exact:true}).fill(username)
 await panel.getByLabel('Password',{exact:true}).fill(password)
 await panel.getByLabel('Confirm password').fill(password)
 await panel.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await panel.getByRole('checkbox',{name:/Privacy Notice/}).check()
 await panel.getByRole('button',{name:'Create RAT account and continue'}).click()
 await expect(panel.getByLabel('Search Things to connect')).toBeVisible({timeout:21000})
 const me=await context.request.get(RAT+'/api/v1/rate-anything/me')
 assert.equal(me.status(),200)
 report.contributorId=(await me.json()).data.id
 note('unrelated-contributor-authenticated')
 // Use the real browser CSRF cookie. Anonymous probes below deliberately have none.
 const post=async(data)=>page.evaluate(async({root,data})=>{
   const token=document.cookie.split('; ').find(x=>x.startsWith('XSRF-TOKEN='))
   const headers={'Accept':'application/json','Content-Type':'application/json'}
   if(token)headers['X-XSRF-TOKEN']=decodeURIComponent(token.slice('XSRF-TOKEN='.length))
   const response=await fetch('/api/v1/rate-anything/things/'+root+'/connections',{
     method:'POST',credentials:'same-origin',headers,body:JSON.stringify(data)
   })
   return {status:response.status,body:await response.json().catch(()=>({}))}
 },{root,data})
 stage='search-private-pending'
 const privateName='Iridescent Monocle Transit rat170-e3bb-missing-link-m03'
 const lookup=await page.evaluate(async({root,q})=>{
   const r=await fetch('/api/v1/rate-anything/things/'+root+'/connections/search?q='+encodeURIComponent(q),{
     credentials:'same-origin',headers:{Accept:'application/json'}
   })
   return {status:r.status,body:await r.json()}
 },{root,q:privateName})
 assert.equal(lookup.status,200)
 assert.deepEqual(lookup.body.data,[])
 note('private-pending-thing-excluded-from-search')
 stage='private-target-refused'
 const hidden=await post({targetEntityId:pending,relationshipType:'related_to',direction:'outgoing'})
 report.responses.privateTarget=hidden.status
 assert.ok([400,403,404,409,422].includes(hidden.status),'private target must be rejected')
 assert.ok(!JSON.stringify(hidden.body).includes(privateName),'response must not disclose private name')
 note('cannot-connect-private-pending-thing')
 stage='duplicate-existing-edge'
 const dupe=await post({targetEntityId:target,relationshipType:'related_to',direction:'outgoing'})
 report.responses.duplicate=dupe.status
 assert.ok([200,409,422].includes(dupe.status),'duplicate must not create another relationship')
 if(dupe.status===200){
   assert.equal(dupe.body.meta.created,false)
   assert.equal(dupe.body.data.id,'01m7dbp2tf3xzt8wj2jdcfa7tm')
   assert.equal(dupe.body.data.sourceEntityId,root)
   assert.equal(dupe.body.data.targetEntityId,target)
   assert.equal(dupe.body.data.verificationState,'unverified')
   assert.notEqual(dupe.body.data.createdByUserId,report.contributorId,'reused edge must preserve original contributor')
 }
 note('duplicate-safely-refused-or-reused-with-attribution')
 stage='reverse-direction-symmetric'
 const reverse=await post({targetEntityId:target,relationshipType:'related_to',direction:'incoming'})
 report.responses.reverse=reverse.status
 assert.ok([200,409,422].includes(reverse.status),'symmetric reverse cannot create a second edge')
 if(reverse.status===200){
  assert.equal(reverse.body.meta.created,false)
  assert.equal(reverse.body.data.id,'01m7dbp2tf3xzt8wj2jdcfa7tm')
  assert.equal(reverse.body.data.sourceEntityId,root)
  assert.equal(reverse.body.data.targetEntityId,target)
 }
 note('reverse-symmetric-request-does-not-invert-or-duplicate')
 stage='invalid-direction'
 const invalid=await post({targetEntityId:target,relationshipType:'related_to',direction:'sideways'})
 report.responses.invalidDirection=invalid.status
 assert.equal(invalid.status,422,'unknown direction must reject rather than silently reverse')
 note('invalid-direction-rejected')
 stage='unauthenticated-requests'
 const anonymous=await browser.newContext()
 const anonConnection=await anonymous.request.post(RAT+'/api/v1/rate-anything/things/'+root+'/connections',{data:{targetEntityId:target},headers:{Accept:'application/json'}})
 const anonEdit=await anonymous.request.post(RAT+'/api/v1/rate-anything/things/'+root+'/edits',{data:{field:'description',value:'Anonymous change must not apply'},headers:{Accept:'application/json'}})
 report.responses.anonymousConnect=anonConnection.status()
 report.responses.anonymousEdit=anonEdit.status()
 assert.ok([401,419].includes(anonConnection.status()))
 assert.ok([401,419].includes(anonEdit.status()))
 await anonymous.close()
 note('anonymous-connection-and-edit-denied-auth-or-csrf')
 report.status='PASS'
}catch(error){
 report.status='FAIL';report.failedStage=stage
 report.errors.push(String(error instanceof Error?error.stack||error.message:error).slice(0,2000))
 console.error('RAT170_SECURITY_FAILED_STAGE='+stage)
 console.error('RAT170_SECURITY_ERROR='+String(error).slice(0,1400))
}finally{
 await writeFile(process.env.OUT+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT170_SECURITY_REPORT='+JSON.stringify(report))
 await browser.close()
}
if(report.status!=='PASS')process.exit(1)
NODE
browser_status=$?
set -e
after="$(snapshot)"
printf 'RAT170_SECURITY_SQL_BEFORE=%s\nRAT170_SECURITY_SQL_AFTER=%s\n' "$before" "$after"
[[ "$before" == "$after" ]] || { echo 'RAT170_SECURITY_SQL_CHANGED_UNEXPECTEDLY' >&2; exit 2; }
echo 'RAT170_SECURITY_SQL_UNCHANGED=true'
[[ "$browser_status" -eq 0 ]] || exit "$browser_status"
