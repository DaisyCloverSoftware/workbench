#!/usr/bin/env bash
set -Eeuo pipefail
# Single authorised ordinary-member canonical CJ classification edit and read-only
# Chromium acceptance. No direct DB updates, no LIVE, no unrelated new Things.
[[ $# -eq 4 ]] || { echo 'usage: source api-image rat-image unique-run-id' >&2; exit 64; }
src="$1";api="$2";rat="$3";rid="$4"
[[ "$src" == '049c9e304823e040a087ab4760bd932a98e93217' ]] || exit 78
[[ "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$api" == '__RAT177_API_DIGEST_PENDING__' ]] || exit 78
[[ "$rat" == '__RAT177_RAT_DIGEST_PENDING__' ]] || exit 78
[[ "$rid" =~ ^[a-z0-9-]{18,44}$ ]] || exit 64
out="/tmp/rat177-cj-type-browser-$rid"
[[ ! -e "$out" ]] || { echo 'Duplicate browser acceptance run ID'; exit 65; }
mkdir -m 700 "$out"
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(kctl -n rum-dev get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'rateurmate.online' ]] || exit 78
[[ "$(kctl -n rum-dev-isolated get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
[[ "$(curl -fsS --retry 3 --retry-delay 2 --max-time 20 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$src" ]] || exit 78
health="$(curl -fsS --retry 3 --retry-delay 2 --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$health" "$src" <<'PY'
import json,sys
h=json.loads(sys.argv[1])
assert h.get('status')=='ok' and h.get('service')=='rum-api' and sys.argv[2] in str(h.get('version',''))
PY
# No mutation before proving this is the intended existing unclaimed public
# persona and the recorded original YouTube creator class remains current.
preflight='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$id="01m07d2f9fj05f4ypdbwp087qj";$e=App\Models\Entity::query()->with("type")->findOrFail($id);
$p=app(App\Services\RatCommunityContributionPolicy::class);
$primary=$db::table("entity_classifications as a")->join("classifications as c","c.id","=","a.classification_id")
 ->where("a.entity_id",$id)->whereNull("a.valid_to")->where("a.is_primary",true)->value("c.key");
$channel=$db::table("classifications as c")->join("entity_types as t","t.id","=","c.entity_type_id")
 ->where("c.key","community_788e6ae3bfcd391d111e3ca31a1332ebf92906d0d0868e7cf7bb")
 ->where("c.status","active")->where("t.key","digital_resource")->value("c.canonical_name");
if($e->canonical_name!=="CJ Investigates"||$e->type?->key!=="persona"||$e->status!=="unverified"
 ||$e->visibility!=="public"||$e->publication_state!=="published"||$e->rateability!=="open"
 ||$e->edit_locked_at!==null||$p->isClaimed($e)||$primary!=="youtube_creator"
 ||$channel!=="YouTube Channel"||$db::table("rating_events")->where("target_entity_id",$id)->count()!==54
 ||$db::table("entity_profile_revisions")->where("entity_id",$id)->where("field_key","thing_type")->exists()
 ||$db::table("entities")->where("canonical_name","CJ Investigates")->count()!==1)exit(78);
$photos=app(App\Services\ThingImageService::class)->publicImages($e);
if($photos["profile"]===null||$photos["cover"]===null)exit(78);
if($db::table("account_entity_links")->where("entity_id",$id)->exists())exit(78);
echo "RAT177_PRE_EDIT_CJ_CANONICAL_GUARD_PASS=true\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$preflight"
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
 if [[ -x "$p" ]];then chrome="$p";break;fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
CHROME="$chrome" RAT177_CJ_RUN="$rid" RAT177_CJ_OUT="$out" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import {randomBytes,createHash} from 'node:crypto'
import {writeFile,stat} from 'node:fs/promises'
import {chromium,expect} from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk'
const DEV='https://dev-rum.daisycloversoftware.uk'
const CJ='01m07d2f9fj05f4ypdbwp087qj'
const KEY='community_788e6ae3bfcd391d111e3ca31a1332ebf92906d0d0868e7cf7bb'
const dir=process.env.RAT177_CJ_OUT
const run=process.env.RAT177_CJ_RUN
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const ctx=await browser.newContext({viewport:{width:1440,height:940}})
const page=await ctx.newPage()
const report={status:'NOT_ASSESSED',run,checks:[],screenshots:[],errors:[],revisionId:null}
const pass=(name)=>{report.checks.push(name);console.log('RAT177_CJ_BROWSER_PASS='+name)}
let stage='baseline'
const getNode=async()=>{
 const r=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+CJ)
 assert.equal(r.status(),200,'Canonical public node unavailable')
 const b=await r.json()
 assert.equal(b.meta?.privacy,'private-rum-people-excluded')
 assert.equal(b.data?.active?.id,CJ)
 return b.data.active
}
const capture=async(p,name,locator)=>{
 const ext=locator?'png':'jpg',file=name+'.'+ext,path=dir+'/'+file
 if(locator)await locator.screenshot({path,animations:'disabled'})
 else await p.screenshot({path,type:'jpeg',quality:64,fullPage:true,animations:'disabled'})
 const bytes=(await stat(path)).size
 assert.ok(bytes>1300&&bytes<800000,'Screenshot size outside bound')
 report.screenshots.push({name:file,bytes})
 console.log('RAT177_REAL_CJ_SCREENSHOT='+file+' bytes='+bytes)
}
const approved=(request)=>{
 const u=new URL(request.url()),method=request.method()
 if(u.hostname==='rateurmate.online')return false
 if(['GET','HEAD','OPTIONS'].includes(method))return true
 if(u.origin!==RAT||method!=='POST')return false
 if(u.pathname==='/api/v1/rate-anything/auth/register')return true
 if(u.pathname!=='/api/v1/rate-anything/things/'+CJ+'/edits')return false
 try{const d=request.postDataJSON()
  return d.field==='thing_type'&&d.value===KEY&&!d.supersedesRevisionId
 }catch{return false}
}
await page.route('**/*',route=>approved(route.request())?route.continue():route.abort('blockedbyclient'))
try{
 const initial=await getNode()
 assert.equal(initial.name,'CJ Investigates')
 assert.equal(initial.entityType,'persona')
 assert.equal(initial.classifications.find(c=>c.primary)?.key,'youtube_creator')
 assert.ok(initial.images?.profile?.url&&initial.images?.cover?.url)
 const original={
  profile:initial.images.profile.url,cover:initial.images.cover.url,
  reputation:JSON.stringify(initial.reputation)
 }
 const beforePhotos=[]
 for(const url of [original.profile,original.cover]){
  const r=await ctx.request.get(url);assert.equal(r.status(),200)
  beforePhotos.push(createHash('sha256').update(await r.body()).digest('hex'))
 }
 pass('CJ-canonical-persona-and-profile-cover-image-checksums-protected')
 stage='register'
 await page.goto(RAT+'/e/cj-investigates~'+CJ,{waitUntil:'domcontentloaded',timeout:40000})
 await expect(page.locator('.rat-workspace-header h1')).toContainText('CJ Investigates',{timeout:30000})
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const panel=page.locator('.rat-workspace-profile .entity-about')
 await panel.getByRole('button',{name:'Create account',exact:true}).click()
 const username='rattyp'+run.replace(/[^a-z0-9]/g,'').slice(-13)
 const password=randomBytes(24).toString('base64url')+'aA1!'
 await panel.getByLabel('Email',{exact:true}).fill(username+'@example.com')
 await panel.getByLabel('Username',{exact:true}).fill(username)
 await panel.getByLabel('Password',{exact:true}).fill(password)
 await panel.getByLabel('Confirm password').fill(password)
 await panel.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await panel.getByRole('checkbox',{name:/Privacy Notice/}).check()
 const register=page.waitForResponse(r=>r.request().method()==='POST'&&new URL(r.url()).pathname==='/api/v1/rate-anything/auth/register',{timeout:30000})
 await panel.getByRole('button',{name:'Create RAT account and continue'}).click()
 const registered=await register
 assert.ok([200,201].includes(registered.status()),'QA member registration status '+registered.status())
 pass('ordinary-authenticated-community-member-no-claim')
 stage='select-type'
 const editor=page.locator('.rat-workspace-profile .entity-about')
 await expect(editor.getByRole('button',{name:'Thing type',exact:true})).toBeVisible({timeout:25000})
 await editor.getByRole('button',{name:'Thing type',exact:true}).click()
 await expect(editor).toContainText('Current Thing type:')
 await expect(editor).toContainText('YouTube creator')
 const typesResponse=await ctx.request.get(RAT+'/api/v1/rate-anything/thing-types?q=YouTube%20Channel')
 assert.equal(typesResponse.status(),200)
 const types=(await typesResponse.json()).data
 assert.ok(types.some(t=>t.label==='YouTube Channel'&&t.entityType==='digital_resource'&&t.classificationKey===KEY))
 const combo=editor.getByRole('combobox',{name:'Choose Thing type'})
 await combo.fill('YouTube Channel')
 const choices=editor.getByRole('listbox',{name:'Available Thing types'})
 await expect(choices.getByRole('button',{name:/YouTube Channel/}).first()).toBeVisible({timeout:25000})
 await choices.getByRole('button',{name:/YouTube Channel/}).first().click()
 await expect(editor).toContainText('Selected: YouTube Channel')
 pass('existing-searchable-canonical-youtube-channel-type-selected')
 stage='submit-thing-type'
 const wait=page.waitForResponse(r=>r.request().method()==='POST'
  && new URL(r.url()).pathname==='/api/v1/rate-anything/things/'+CJ+'/edits',{timeout:40000})
 await editor.getByRole('button',{name:'Save Thing type',exact:true}).click()
 const saved=await wait
 const body=await saved.json()
 assert.equal(saved.status(),201,'Type POST rejected: '+JSON.stringify(body).slice(0,550))
 assert.equal(body.meta?.applied,true)
 assert.equal(body.data?.field,'thing_type')
 assert.equal(body.data?.outcome,'applied')
 assert.equal(body.data?.source,'community')
 assert.equal(body.data?.previousTypeLabel,'YouTube creator')
 assert.equal(body.data?.proposedTypeLabel,'YouTube Channel')
 assert.match(body.data?.id,/^[0-9a-z]{26}$/)
 report.revisionId=body.data.id
 await expect(editor.getByRole('status')).toContainText('Edit applied and recorded',{timeout:15000})
 pass('actual-CJ-type-edit-directly-applied-as-one-attributable-reversible-revision')
 const changed=await getNode()
 assert.equal(changed.id,CJ)
 assert.equal(changed.entityType,'digital_resource')
 assert.equal(changed.classifications.find(c=>c.primary)?.name,'YouTube Channel')
 assert.equal(changed.classifications.find(c=>c.primary)?.key,KEY)
 assert.equal(changed.images.profile.url,original.profile)
 assert.equal(changed.images.cover.url,original.cover)
 assert.equal(JSON.stringify(changed.reputation),original.reputation)
 pass('canonical-structural-type-and-primary-classification-updated-images-reputation-same')
 stage='screenshots'
 for(const width of [1440,768,390]){
  await page.setViewportSize({width,height:width===390?850:940})
  await page.goto(RAT+'/e/cj-investigates~'+CJ,{waitUntil:'domcontentloaded',timeout:40000})
  await expect(page.locator('.rat-workspace-header h1')).toContainText('CJ Investigates',{timeout:30000})
  await expect(page.locator('.rat-workspace-kickers')).toContainText('YouTube Channel',{timeout:25000})
  const typeCell=page.locator('.rat-workspace-facts > div').filter({has:page.locator('dt', {hasText:'Type of Thing'})}).first()
  await expect(typeCell.locator('dd')).toHaveText('YouTube Channel')
  const avatar=page.locator('.rat-workspace-avatar > img')
  const cover=page.locator('.rat-workspace-cover > img')
  await expect(avatar).toHaveAttribute('src',original.profile)
  await expect(cover).toHaveAttribute('src',original.cover)
  await expect.poll(()=>avatar.evaluate(el=>el.complete&&el.naturalWidth>0),{timeout:20000}).toBe(true)
  await expect.poll(()=>cover.evaluate(el=>el.complete&&el.naturalWidth>0),{timeout:20000}).toBe(true)
  const score=page.locator('.rat-workspace-score')
  await expect(score.getByRole('button',{name:'Rate'})).toBeVisible()
  await expect(score.locator('a[href="#ratings"]')).toBeVisible()
  await expect(score.locator('img[alt="RUM Thumb"]')).toBeVisible()
  await expect(score).toContainText('awaiting community verdict',{timeout:25000})
  const labels=await score.locator('.rum-rating__empty').allTextContents()
  assert.equal(labels.length,1,'Missing branded status line')
  assert.match(labels[0],/At least \d+ ratings? awaiting community verdict/)
  const number=await score.evaluate(el=>(el.innerText.match(/At least \d+ ratings? awaiting community verdict/g)||[]).length)
  assert.equal(number,1,'Duplicate unbranded status line in main Thing card')
  const overflow=await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)
  assert.ok(overflow<=2,'Thing page horizontal overflow at '+width+': '+overflow)
  await capture(page,'cj-'+width+'-thing-full')
  await capture(page,'cj-'+width+'-main-card',page.locator('.rat-workspace-header'))
  await page.reload({waitUntil:'domcontentloaded'})
  await expect(page.locator('.rat-workspace-kickers')).toContainText('YouTube Channel',{timeout:20000})
  pass('CJ-'+width+'-thing-one-RUM-status-type-and-media-persist-after-refresh')
  await page.goto(RAT+'/search?q=cj&active='+CJ,{waitUntil:'domcontentloaded',timeout:40000})
  await expect(page.locator('.rat-active-card h1')).toContainText('CJ Investigates',{timeout:30000})
  await expect(page.locator('.rat-active-card__type')).toHaveText('YouTube Channel')
  const searchPicture=page.locator('.rat-active-card__avatar img')
  await expect(searchPicture).toHaveAttribute('src',original.profile)
  await expect.poll(()=>searchPicture.evaluate(el=>el.complete&&el.naturalWidth>0)).toBe(true)
  const searchOverflow=await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)
  assert.ok(searchOverflow<=2,'Search page overflow at '+width+': '+searchOverflow)
  await capture(page,'cj-'+width+'-search-full')
  await page.reload({waitUntil:'domcontentloaded'})
  await expect(page.locator('.rat-active-card__type')).toHaveText('YouTube Channel',{timeout:20000})
  pass('CJ-'+width+'-search-shows-same-channel-without-overflow')
 }
 stage='read-only-navigation'
 await page.setViewportSize({width:1440,height:940})
 await page.goto(RAT+'/e/cj-investigates~'+CJ,{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-kickers')).toContainText('YouTube Channel')
 const rate=page.locator('.rat-workspace-score').getByRole('button',{name:'Rate'})
 await rate.click()
 await expect(page.getByRole('dialog',{name:'Rate CJ Investigates'})).toBeVisible()
 await page.getByRole('button',{name:'Close rating'}).click()
 await expect(page.getByRole('dialog',{name:'Rate CJ Investigates'})).toHaveCount(0)
 await page.locator('.rat-workspace-score a[href="#ratings"]').click()
 await expect(page.locator('#ratings')).toBeVisible()
 pass('existing-Rate-and-See-ratings-navigation-still-works-without-voting')
 stage='history'
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const history=page.locator('.rat-thing-revisions')
 await expect(history).toContainText('Thing type')
 await expect(history).toContainText('Applied')
 await expect(history).toContainText('Community contribution')
 const histRequest=await ctx.request.get(RAT+'/api/v1/rate-anything/things/'+CJ+'/edits')
 assert.equal(histRequest.status(),200)
 const hist=(await histRequest.json()).data
 const typeRevs=hist.filter(r=>r.field==='thing_type')
 assert.equal(typeRevs.length,1)
 assert.equal(typeRevs[0].id,report.revisionId)
 assert.equal(typeRevs[0].proposedTypeLabel,'YouTube Channel')
 pass('canonical-attributable-type-edit-history-revision-persisted')
 stage='new-browser-context'
 const fresh=await browser.newContext({viewport:{width:390,height:850}})
 const freshPage=await fresh.newPage()
 await freshPage.goto(RAT+'/e/cj-investigates~'+CJ,{waitUntil:'domcontentloaded',timeout:40000})
 await expect(freshPage.locator('.rat-workspace-kickers')).toContainText('YouTube Channel',{timeout:30000})
 await expect(freshPage.locator('.rat-workspace-facts dd').first()).toHaveText('YouTube Channel')
 await freshPage.goto(RAT+'/search?q=cj&active='+CJ,{waitUntil:'domcontentloaded'})
 await expect(freshPage.locator('.rat-active-card__type')).toHaveText('YouTube Channel',{timeout:30000})
 await fresh.close()
 pass('new-independent-browser-session-sees-published-channel-type')
 const final=await getNode()
 assert.equal(final.entityType,'digital_resource')
 assert.equal(final.images.profile.url,original.profile)
 assert.equal(final.images.cover.url,original.cover)
 assert.equal(JSON.stringify(final.reputation),original.reputation)
 const afterPhotos=[]
 for(const url of [original.profile,original.cover]){
  const r=await ctx.request.get(url);assert.equal(r.status(),200)
  afterPhotos.push(createHash('sha256').update(await r.body()).digest('hex'))
 }
 assert.deepEqual(afterPhotos,beforePhotos,'Published profile or cover bytes changed')
 pass('both-original-media-contents-bit-identical-and-existing-reputation-intact')
 report.status='PASS'
 console.log('RAT177_REAL_CJ_THING_TYPE_BROWSER_PASS=true')
}catch(error){
 report.status='FAIL'
 report.errors.push({stage,message:String(error).slice(0,1650)})
 console.log('RAT177_REAL_CJ_THING_TYPE_BROWSER_FAILURE='+JSON.stringify(report.errors[0]))
 process.exitCode=1
}finally{
 await writeFile(dir+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT177_REAL_CJ_BROWSER_REPORT='+JSON.stringify(report))
 await browser.close()
}
NODE
# Reassert rating counts, one type edit, no duplicate entity and image preservation.
after='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;$id="01m07d2f9fj05f4ypdbwp087qj";
$e=App\Models\Entity::query()->with("type")->findOrFail($id);
$p=app(App\Services\RatCommunityContributionPolicy::class);
$primary=$db::table("entity_classifications as a")->join("classifications as c","c.id","=","a.classification_id")
 ->where("a.entity_id",$id)->whereNull("a.valid_to")->where("a.is_primary",true)->value("c.key");
$prior=$db::table("entity_classifications as a")->join("classifications as c","c.id","=","a.classification_id")
 ->where("a.entity_id",$id)->whereNotNull("a.valid_to")->where("c.key","youtube_creator")->count();
$revs=$db::table("entity_profile_revisions")->where("entity_id",$id)->where("field_key","thing_type")->get();
$photos=app(App\Services\ThingImageService::class)->publicImages($e);
if($e->canonical_name!=="CJ Investigates"||$e->type?->key!=="digital_resource"
 ||$e->status!=="unverified"||$e->visibility!=="public"||$e->rateability!=="open"
 ||$e->publication_state!=="published"||$p->isClaimed($e)
 ||$primary!=="community_788e6ae3bfcd391d111e3ca31a1332ebf92906d0d0868e7cf7bb"
 ||$prior<1||$revs->count()!==1||$revs[0]->outcome!=="applied"
 ||$db::table("rating_events")->where("target_entity_id",$id)->count()!==54
 ||$db::table("entities")->where("canonical_name","CJ Investigates")->count()!==1
 ||$photos["profile"]===null||$photos["cover"]===null)exit(78);
echo "RAT177_PERSISTED_CJ_TYPE=".json_encode(["id"=>$id,"type"=>$e->type?->key,
 "classification"=>"YouTube Channel","ratingEvents"=>54,"typeRevisions"=>1,
 "oldYoutubeCreatorAssignmentEnded"=>true,"profileAndCoverPublished"=>true,"claimed"=>false,"duplicateEntities"=>false])."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$after"
echo RAT177_CJ_TYPE_AND_REPUTATION_PERSISTED=true