#!/usr/bin/env bash
set -Eeuo pipefail
# PR #173: bounded, isolated RAT browser acceptance of canonical Judge.
# The only rating/vote mutations target the fixed disposable public QA Thing.
# Author/voter fixture eligibility changes apply only to accounts created here.
# Owner's canonical rating and LIVE are read-only.
[[ $# -eq 4 ]] || { echo 'usage: exact-source-sha api@sha256 rat@sha256 unique-run-id' >&2;exit 64; }
src="$1";api="$2";rat="$3";rid="$4"
[[ "$src" =~ ^[0-9a-f]{40}$ ]] || exit 64
[[ "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$rid" =~ ^[a-z0-9-]{10,44}$ ]] || exit 64
out="/tmp/rat171-judge-browser-$rid"
[[ ! -e "$out" ]] || { echo 'Duplicate run id: inspect original receipt';exit 65; }
mkdir -m 700 "$out"
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
[[ "$(kctl -n rum-dev get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'rateurmate.online' ]] || exit 78
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --max-time 20 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$src" ]] || exit 78
health="$(curl -fsS --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$health" "$src" <<'PY'
import json,sys
j=json.loads(sys.argv[1])
assert j.get("status")=="ok" and j.get("service")=="rum-api"
assert sys.argv[2] in str(j.get("version",""))
PY
# Refuse unless QA Thing still matches known disposable identity and eligibility.
read_only_guard='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;$id="01m7fw9yb4684hygrm14tmm3jr";
$e=$db::table("entities")->where("id",$id)->first();
if(!$e||$e->canonical_name!=="Green Tempest Artwork Catalogue rat170img4way-20261115-r15"
 ||$e->status!=="active"||$e->visibility!=="public"||$e->publication_state!=="published"
 ||$e->edit_policy!=="community_governed"||$e->edit_locked_at!==null||$e->sensitivity!=="normal")exit(78);
if($db::table("entity_types")->where("id",$e->entity_type_id)->value("key")!=="product")exit(78);
if($db::table("entity_claims")->where("entity_id",$id)->where("status","approved")->where("verification_level",">=",1)->whereNull("revoked_at")->exists())exit(78);
if($db::table("entity_managers")->where("entity_id",$id)->where("status","active")->whereNull("revoked_at")->exists())exit(78);
if((bool)config("age_assurance.enabled",false)||(bool)config("features.nsfw",false))exit(78);
echo "RAT171_DISPOSABLE_PUBLIC_QA_AND_GENERAL_AUDIENCE_GUARD_PASS=true\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$read_only_guard"
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q;git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$src"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$src" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for c in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser;do
  if [[ -x "$c" ]];then chrome="$c";break;fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
RAT171_RUN="$rid" RAT171_OUT="$out" CHROME="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { randomUUID,randomBytes } from 'node:crypto'
import { execFileSync } from 'node:child_process'
import { writeFile,stat } from 'node:fs/promises'
import { chromium,expect } from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk'
const QA='01m7fw9yb4684hygrm14tmm3jr',OWNER='01m7bswfh0a3evf271ye4taaf0'
const run=process.env.RAT171_RUN,dir=process.env.RAT171_OUT
const report={run,status:'NOT_ASSESSED',steps:[],accounts:[],ratingId:null,voteResults:[],screenshots:[],errors:[]}
const pass=(name)=>{report.steps.push(name);console.log('RAT171_JUDGE_BROWSER_PASS='+name)}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const authorCtx=await browser.newContext({viewport:{width:1440,height:960}})
const voterCtx=await browser.newContext({viewport:{width:1440,height:960}})
const anonCtx=await browser.newContext({viewport:{width:390,height:844}})
const author=await authorCtx.newPage(),voter=await voterCtx.newPage()
let stage='initial'
const screenshot=async(p,name)=>{const path=dir+'/'+name+'.jpg';await p.screenshot({path,type:'jpeg',quality:45,animations:'disabled',fullPage:true});const f=await stat(path);report.screenshots.push({name,bytes:f.size});console.log('RAT171_JUDGE_SCREENSHOT='+name+' bytes='+f.size)}
const isApproved=(r)=>{const u=new URL(r.url()),m=r.method();if(u.hostname==='rateurmate.online')return false
 if(['GET','HEAD','OPTIONS'].includes(m))return true
 return u.origin===RAT && m==='POST'&&u.pathname==='/api/v1/rate-anything/auth/register'
  ||u.origin===RAT && m==='POST'&&u.pathname==='/api/v1/rate-anything/rating'
  ||u.origin===RAT && ['PUT','DELETE'].includes(m) && u.pathname.startsWith('/api/v1/rate-anything/things/'+QA+'/judge/')
}
for(const p of [author,voter])await p.route('**/*',route=>isApproved(route.request())?route.continue():route.abort('blockedbyclient'))
const signup=async(p,role)=>{
 await p.goto(RAT+'/e/qa~'+QA,{waitUntil:'domcontentloaded',timeout:35000})
 await expect(p.locator('.rat-workspace-header h1')).toContainText('Green Tempest',{timeout:30000})
 await p.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const panel=p.locator('.rat-workspace-profile .entity-about')
 await panel.getByRole('button',{name:'Create account',exact:true}).click()
 const username='ratjudge'+role+run.replace(/[^a-z0-9]/g,'').slice(-10)
 const password=randomBytes(22).toString('base64url')+'aA1!'
 await panel.getByLabel('Email',{exact:true}).fill(username+'@example.com')
 await panel.getByLabel('Username',{exact:true}).fill(username)
 await panel.getByLabel('Password',{exact:true}).fill(password)
 await panel.getByLabel('Confirm password').fill(password)
 await panel.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await panel.getByRole('checkbox',{name:/Privacy Notice/}).check()
 const wait=p.waitForResponse(x=>x.request().method()==='POST'&&new URL(x.url()).pathname==='/api/v1/rate-anything/auth/register',{timeout:30000})
 await panel.getByRole('button',{name:'Create RAT account and continue'}).click()
 const r=await wait;assert.ok([200,201].includes(r.status()),'Disposable registration rejected '+r.status())
 const me=await p.request.get(RAT+'/api/v1/rate-anything/me');assert.equal(me.status(),200)
 const id=(await me.json()).data.id;assert.match(id,/^[0-9a-z]{26}$/)
 report.accounts.push({role,id})
 return {username,id}
}
const postJson=async(p,path,body,method='POST')=>p.evaluate(async ({path,body,method})=>{
 const xsrf=document.cookie.split('; ').find(x=>x.startsWith('XSRF-TOKEN='))
 const token=xsrf?decodeURIComponent(xsrf.slice('XSRF-TOKEN='.length)):''
 const r=await fetch(path,{method,credentials:'same-origin',headers:{Accept:'application/json','Content-Type':'application/json',...(token?{'X-XSRF-TOKEN':token}:{})},body:JSON.stringify(body)})
 return {status:r.status,body:await r.json().catch(()=>({}))}
},{path,body,method})
const judgePage=async(ctx)=>{const r=await ctx.request.get(RAT+'/api/v1/rate-anything/things/'+QA+'/judge?limit=10');return{status:r.status,data:await r.json().catch(()=>({}))}}
const makeEligible=({username,id})=>{
 assert.match(id,/^[0-9a-z]{26}$/);assert.match(username,/^ratjudge[a-z0-9]{6,40}$/)
 const code=[
 'require "vendor/autoload.php";',
 '$app=require "bootstrap/app.php";$app->make(Illuminate\\Contracts\\Console\\Kernel::class)->bootstrap();',
 '$db=Illuminate\\Support\\Facades\\DB::class;',
 '$id="'+id+'";$username="'+username+'";',
 '$u=$db::table("users")->where("id",$id)->first();',
 'if(!$u||$u->username!==$username||$u->email!==$username."@example.com"',
 '||$u->email_verified_at!==null||$u->status!=="active"',
 '||strtotime((string)$u->created_at)<time()-3600)exit(78);',
 'if((bool)config("age_assurance.enabled",false)||(bool)config("features.nsfw",false))exit(78);',
 'if($db::table("entity_claims")->where("claimant_user_id",$id)->exists())exit(78);',
 '$n=$db::table("users")->where("id",$id)->whereNull("email_verified_at")',
 '->update(["email_verified_at"=>now(),"created_at"=>now()->subDays(3)]);',
 'if($n!==1)exit(78);echo "RAT171_NEW_DISPOSABLE_MEMBER_JUDGE_ELIGIBILITY_SET=true\n";'
 ].join('\n')
 const out=execFileSync('sudo',['-n','k3s','kubectl','-n','rum-dev-isolated','exec','deployment/rum-api','-c','php-fpm','--','php','-r',code],{encoding:'utf8',timeout:25000})
 assert.ok(out.includes('RAT171_NEW_DISPOSABLE_MEMBER_JUDGE_ELIGIBILITY_SET=true'))
}
try{
 stage='create-disposable-members'
 const a=await signup(author,'author')
 const b=await signup(voter,'voter')
 assert.notEqual(a.id,b.id)
 pass('two-new-ordinary-disposable-members')
 stage='submit-disposable-rating'
 await author.goto(RAT+'/e/qa~'+QA,{waitUntil:'domcontentloaded'})
 const reason='Disposable RAT Judge browser regression '+run
 const submission=await postJson(author,'/api/v1/rate-anything/rating',{
  entityId:QA,value:5,categoryKey:'quality',reasonKeys:['quality_or_result'],reasonText:reason,audience:'public',idempotencyKey:randomUUID()
 })
 assert.equal(submission.status,201,'new QA rating submission failed '+JSON.stringify(submission).slice(0,450))
 const ratingId=submission.body?.data?.id
 assert.match(ratingId,/^[0-9a-z]{26}$/)
 report.ratingId=ratingId
 pass('disposable-public-rating-created-without-touching-owner')
 stage='author-read'
 const result=await judgePage(authorCtx)
 assert.equal(result.status,200,'author Judge read HTTP '+result.status)
 const item=result.data?.data?.find(x=>x.id===ratingId)
 assert.ok(item,'author cannot find their own new rating '+JSON.stringify(result.data).slice(0,450))
 assert.equal(item.state,'needs_verdict')
 assert.equal(item.ownRating,true)
 assert.equal(item.canVote,false)
 assert.equal(item.countsTowardAggregate,false)
 assert.ok(result.data.meta.accessNotice,'Expected truthful existing Judge account eligibility message')
 await author.reload({waitUntil:'domcontentloaded'})
 await expect(author.locator('.rat-community-judge__item').filter({hasText:'Your rating'}).first()).toContainText('Awaiting community verdict',{timeout:20000})
 await expect(author.locator('.rat-workspace-score')).toContainText('awaiting community verdict')
 await expect(author.locator('.rat-community-judge')).toContainText('cannot vote on your own submission')
 pass('unverified-author-can-see-own-pending-rating-but-not-vote')
 await screenshot(author,'01-author-pending-desktop')
 stage='unverified-unrelated'
 const hidden=await judgePage(voterCtx)
 assert.equal(hidden.status,200)
 assert.ok(!hidden.data.data.some(x=>x.id===ratingId),'Unverified unrelated viewer saw other rating')
 assert.ok(hidden.data.meta.accessNotice)
 pass('unverified-unrelated-member-has-no-new-Judge-vote-rights')
 stage='fixture-eligibility'
 makeEligible(a);makeEligible(b)
 pass('existing-Judge-requirements-satisfied-on-new-QA-accounts-only')
 stage='self-vote-prohibited'
 await author.reload({waitUntil:'domcontentloaded'})
 const ownVisible=await judgePage(authorCtx);assert.equal(ownVisible.status,200)
 assert.equal(ownVisible.data.data.find(x=>x.id===ratingId).canVote,false)
 const self=await postJson(author,'/api/v1/rate-anything/things/'+QA+'/judge/'+ratingId+'/vote',{vote:1},'PUT')
 assert.equal(self.status,403,'Author self-vote should be rejected: '+JSON.stringify(self).slice(0,450))
 pass('eligible-author-cannot-vote-own-rating')
 stage='independent-vote'
 await voter.reload({waitUntil:'domcontentloaded'})
 const others=await judgePage(voterCtx)
 assert.equal(others.status,200)
 const first=others.data.data.find(x=>x.id===ratingId)
 assert.ok(first,'Eligible voter cannot see rating')
 assert.equal(first.canVote,true)
 const row=voter.locator('.rat-community-judge__item').filter({hasText:reason}).first()
 await expect(row).toBeVisible({timeout:22000})
 await expect(row).toContainText('Awaiting community verdict')
 const waitVote=voter.waitForResponse(x=>x.request().method()==='PUT'&&new URL(x.url()).pathname.endsWith('/judge/'+ratingId+'/vote'),{timeout:20000})
 await row.getByRole('button',{name:'Keep'}).click()
 assert.equal((await waitVote).status(),200)
 await expect(row).toContainText('Admission votes: 1 Keep',{timeout:22000})
 const voted=await judgePage(voterCtx)
 assert.equal(voted.data.data.find(x=>x.id===ratingId).viewerVote,1)
 assert.equal(voted.data.data.find(x=>x.id===ratingId).keepVotes,1)
 report.voteResults.push({vote:'Keep',count:1})
 pass('canonical-Judge-vote-recorded-and-visible-in-RAT')
 await screenshot(voter,'02-independent-keep-vote')
 stage='withdraw-vote'
 const waitRemove=voter.waitForResponse(x=>x.request().method()==='DELETE'&&new URL(x.url()).pathname.endsWith('/judge/'+ratingId+'/vote'),{timeout:20000})
 await row.getByRole('button',{name:'Remove my vote'}).click()
 assert.equal((await waitRemove).status(),200)
 await expect(row).toContainText('Admission votes: 0 Keep',{timeout:20000})
 const withdrawn=await judgePage(voterCtx)
 assert.equal(withdrawn.data.data.find(x=>x.id===ratingId).viewerVote,null)
 report.voteResults.push({vote:'removed',count:0})
 pass('canonical-Judge-vote-withdrawal-recorded-and-visible')
 stage='owner-read-only'
 const before=await voterCtx.request.get(RAT+'/api/v1/rate-anything/things/'+OWNER+'/judge?limit=10')
 assert.equal(before.status(),200,'owner Thing Judge view not accessible to eligible viewer')
 const rows=(await before.json()).data
 assert.ok(rows.some(x=>x.state==='needs_verdict'&&x.countsTowardAggregate===false),'owner pending rating absent')
 await voter.goto(RAT+'/e/the-professional-idiots~'+OWNER,{waitUntil:'domcontentloaded'})
 await expect(voter.locator('.rat-workspace-header h1')).toContainText('The Professional Idiots',{timeout:30000})
 await expect(voter.locator('.rat-community-judge__item').filter({hasText:'Awaiting community verdict'}).first()).toBeVisible({timeout:20000})
 const facts=voter.locator('.rat-workspace-profile > .rat-workspace-facts')
 assert.equal(await facts.locator(':scope > div').count(),3)
 const columns=await facts.evaluate(el=>getComputedStyle(el).gridTemplateColumns.split(' ').filter(Boolean).length)
 assert.equal(columns,3,'Expected three balanced desktop columns')
 pass('owner-public-rating-visible-as-pending-and-profile-three-facts-desktop')
 await screenshot(voter,'03-owner-judge-desktop-readonly')
 await voter.setViewportSize({width:390,height:844})
 const mobile=await facts.evaluate(el=>getComputedStyle(el).gridTemplateColumns.split(' ').filter(Boolean).length)
 assert.equal(mobile,1,'Expected one-column facts on mobile')
 const overflow=await voter.evaluate(()=>document.documentElement.scrollWidth-innerWidth)
 assert.ok(overflow<=2,'Mobile horizontal overflow '+overflow)
 pass('three-fact-profile-mobile-no-extra-cell')
 await screenshot(voter,'04-owner-mobile-readonly')
 report.status='PASS'
 console.log('RAT171_CANONICAL_JUDGE_BROWSER_PASS=true')
}catch(e){
 report.status='FAILED';report.errors.push({stage,error:String(e).slice(0,1700)})
 console.error('RAT171_JUDGE_BROWSER_FAILED_STAGE='+stage)
 console.error('RAT171_JUDGE_BROWSER_ERROR='+String(e).slice(0,1700))
 try{await screenshot(voter,'99-failure')}catch{}
 process.exitCode=1
}finally{
 await writeFile(dir+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT171_JUDGE_BROWSER_REPORT='+JSON.stringify(report))
 await browser.close()
}
NODE
