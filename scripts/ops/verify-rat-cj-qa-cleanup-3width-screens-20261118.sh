#!/usr/bin/env bash
set -Eeuo pipefail
# READ ONLY: final DEV CJ cleanup acceptance at 1440 / 768 / 390, including reload and actual screenshots.
[[ $# -eq 0 ]] || exit 64
run='ratcj-clean-3width-20261118-r1'
dir="/tmp/$run"
[[ ! -e "$dir" ]] || { echo "Browser evidence already exists: $dir"; exit 65; }
mkdir -m 700 "$dir"
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
api='ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430'
rat='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:ec11bfec432ff10d508850a0a071b3af3787bff656d2decca90eed5951cd0003'
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$id="01m07d2f9fj05f4ypdbwp087qj";
$cj=App\Models\Entity::query()->with(["type","classificationAssignments.classification"])->findOrFail($id);
$rel=$db::table("entity_relationships")->where(function($q)use($id){$q->where("source_entity_id",$id)->orWhere("target_entity_id",$id);})->get(["id","valid_to"]);
$historical=$rel->whereNotNull("valid_to")->count();
$active=$rel->whereNull("valid_to")->count();
$ratings=$db::table("rating_events")->where("target_entity_id",$id)->count();
$primary=$cj->classificationAssignments->first(fn($a)=>(bool)$a->is_primary&&$a->valid_to===null);
$revisions=$db::table("entity_profile_revisions")->where("entity_id",$id)->where("field_key","thing_type")->get(["id","outcome"]);
$approvedClaims=$db::table("entity_claims")->where("entity_id",$id)->where("status","approved")->whereNull("revoked_at")->count();
$images=app(App\Services\ThingImageService::class)->publicImages($cj);
if($cj->canonical_name!=="CJ Investigates"||$cj->type?->key!=="digital_resource"||
 $cj->visibility!=="public"||$cj->publication_state!=="published"||$cj->rateability!=="open"||
 $cj->deleted_at!==null||$approvedClaims!==0||$active!==0||$historical!==22||
 $ratings!==54||$revisions->count()!==1||$revisions[0]->id!=="01m7qsxdjs1tg298py83xk2y49"||
 $primary?->classification?->canonical_name!=="YouTube Channel"||
 $images["profile"]===null||$images["cover"]===null)throw new RuntimeException("CJ canonical preservation guard failed");
$endedEvents=$db::table("outbox_events")->where("aggregate_type","entity_relationship")->whereIn("aggregate_id",$rel->pluck("id")->all())->where("event_type","entity.relationship_ended.v1")->count();
if($endedEvents!==22)throw new RuntimeException("Missing closure audit events");
echo "RATCJ_FINAL_CANONICAL_STATE=".json_encode(["id"=>$id,"name"=>$cj->canonical_name,"originalThingType"=>$cj->type?->key,"primaryClassification"=>$primary->classification->canonical_name,"published"=>$cj->publication_state,"rateability"=>$cj->rateability,"visibility"=>$cj->visibility,"editPolicy"=>$cj->edit_policy,"approvedActiveClaims"=>$approvedClaims,"relationsHistoricalEnded"=>$historical,"relationsActive"=>$active,"relationshipEndedEvents"=>$endedEvents,"ratingEvents"=>$ratings,"thingTypeRevision"=>$revisions[0]->id,"profileAndCoverPublished"=>true],JSON_UNESCAPED_SLASHES)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
[[ "$(curl -fsS --retry 3 --max-time 20 https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == 'rat-candidate+887b5e02f4be6ec7e52371666effa9eb47892d6e' ]] || exit 78
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin 887b5e02f4be6ec7e52371666effa9eb47892d6e
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == '887b5e02f4be6ec7e52371666effa9eb47892d6e' ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for p in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser; do if [[ -x "$p" ]]; then chrome="$p"; break; fi; done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
CHROME="$chrome" REPORT_DIR="$dir" REPORT_RUN="$run" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import {createHash} from 'node:crypto'
import {stat,readFile,writeFile} from 'node:fs/promises'
import {chromium,expect} from '@playwright/test'
const CJ='01m07d2f9fj05f4ypdbwp087qj'
const DEV='https://dev-rum.daisycloversoftware.uk'
const RAT='https://dev-rum-ra.daisycloversoftware.uk'
const dir=process.env.REPORT_DIR
const report={status:'NOT_ASSESSED',run:process.env.REPORT_RUN,source:'887b5e02f4be6ec7e52371666effa9eb47892d6e',id:CJ,checks:[],screenshots:[],errors:[]}
const pass=msg=>{report.checks.push(msg);console.log('RATCJ_3WIDTH_PASS='+msg)}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
try{
 const ctx=await browser.newContext({viewport:{width:1440,height:940}})
 const page=await ctx.newPage()
 const errors=[],statuses=[],unauthed=[]
 page.on('pageerror',e=>errors.push(e.message))
 page.on('console',m=>{if(m.type()==='error')errors.push(m.text())})
 page.on('response',r=>{
  const status=r.status()
  if(status===401)unauthed.push({method:r.request().method(),url:r.url()})
  if(status>=400 && status!==401)statuses.push({status,url:r.url()})
 })
 await page.route('**/*',route=>['GET','HEAD','OPTIONS'].includes(route.request().method())?route.continue():route.abort('blockedbyclient'))
 const response=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+CJ,{headers:{'Cache-Control':'no-cache'}})
 assert.equal(response.status(),200,'CJ canonical public graph')
 const root=(await response.json()).data
 assert.equal(root.active.id,CJ)
 assert.equal(root.active.name,'CJ Investigates')
 assert.equal(root.active.rateability,'open')
 assert.equal(root.active.entityType,'digital_resource')
 assert.deepEqual(root.topLinked,[])
 assert.deepEqual(root.categories,[])
 const typeText=root.active.classifications.find(x=>x.primary)?.name??''
 assert.equal(typeText,'YouTube Channel')
 const profile=root.active.images?.profile?.url
 const cover=root.active.images?.cover?.url
 assert.ok(profile && cover,'Published profile/cover URLs required')
 pass('canonical-public-API-no-linked-items-preserved-youtube-channel-and-media')
 const slug=root.active.slug?.trim()||'thing'
 const detail=RAT+'/e/'+encodeURIComponent(slug)+'~'+encodeURIComponent(CJ)
 const search=RAT+'/search?q=CJ%20Investigates&active='+CJ
 const checkNoOverflow=async width=>{
  const px=await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)
  assert.ok(px<=2,'Horizontal overflow at '+width+': '+px)
 }
 const imageDecoded=async selector=>{
  const el=page.locator(selector).first()
  await expect(el).toBeVisible({timeout:20000})
  await expect.poll(()=>el.evaluate(img=>img.complete && img.naturalWidth>0 && img.naturalHeight>0),{timeout:25000}).toBe(true)
  await el.evaluate(img=>img.decode())
 }
 const capture=async (width,view)=>{
  const name='cj-investigates-'+width+'-'+view+'-after-refresh.jpg'
  const path=dir+'/'+name
  await page.screenshot({path,type:'jpeg',quality:68,animations:'disabled',fullPage:true})
  const bytes=(await stat(path)).size
  assert.ok(bytes>12000 && bytes<900000,'Unexpected screenshot size: '+name+': '+bytes)
  const sha256=createHash('sha256').update(await readFile(path)).digest('hex')
  report.screenshots.push({name,view,width,bytes,sha256})
  console.log('RATCJ_SCREENSHOT='+JSON.stringify({name,bytes,sha256}))
 }
 const searchCheck=async width=>{
  await expect(page.locator('.rat-active-card h1')).toHaveText('CJ Investigates',{timeout:30000})
  await expect(page.locator('.rat-active-card__type')).toHaveText('YouTube Channel')
  await expect(page.locator('.rat-top-linked')).toHaveCount(0)
  await expect(page.locator('.rat-linked-card')).toHaveCount(0)
  await expect(page.locator('section.rat-category:not(.rat-category--empty)')).toHaveCount(0)
  await expect(page.locator('section.rat-category--empty h2')).toHaveText('No public linked Things yet')
  await expect(page.locator('.rat-active-card__actions').getByRole('button',{name:'Rate'})).toBeVisible()
  await expect(page.locator('.rat-active-card__actions').getByRole('button',{name:'See ratings'})).toBeVisible()
  const text=await page.locator('body').innerText()
  assert.ok(!/Quillstone|RUM DEV linked check|Top linked things|Products/.test(text),'QA linked Products still visible in search')
  await imageDecoded('.rat-active-card__avatar img')
  await checkNoOverflow(width)
 }
 const workspaceCheck=async width=>{
  await expect(page.locator('.rat-workspace-identity h1')).toHaveText('CJ Investigates',{timeout:30000})
  await expect(page.locator('.rat-workspace-kickers')).toContainText('YouTube Channel')
  await expect(page.locator('#connections h2')).toHaveText('Connected Things')
  await expect(page.locator('.rat-workspace-connection-group')).toHaveCount(0)
  await expect(page.locator('.rat-workspace-connection')).toHaveCount(0)
  await expect(page.locator('.rat-workspace-empty')).toContainText('No public connections are attached to this Thing yet.')
  await expect(page.locator('.rat-workspace-profile .rat-workspace-facts')).toContainText('Rateability')
  await expect(page.locator('.rat-workspace-profile .rat-workspace-facts')).toContainText('open')
  await expect(page.locator('.rat-workspace-score .rat-primary-action')).toBeVisible()
  const text=await page.locator('body').innerText()
  assert.ok(!/Quillstone|RUM DEV linked check/.test(text),'QA linked Products still visible in workspace')
  await imageDecoded('.rat-workspace-avatar img')
  await imageDecoded('.rat-workspace-cover img')
  await checkNoOverflow(width)
 }
 for(const width of [1440,768,390]){
  await page.setViewportSize({width,height:width===390?850:940})
  await page.goto(search,{waitUntil:'domcontentloaded',timeout:45000})
  await searchCheck(width)
  await page.reload({waitUntil:'domcontentloaded',timeout:45000})
  await searchCheck(width)
  await capture(width,'search')
  pass('search-'+width+'-no-fixtures-and-stable-after-reload')
  await page.goto(detail,{waitUntil:'domcontentloaded',timeout:45000})
  await workspaceCheck(width)
  await page.reload({waitUntil:'domcontentloaded',timeout:45000})
  await workspaceCheck(width)
  await capture(width,'workspace')
  pass('workspace-'+width+'-zero-public-connections-published-images-and-stable-after-reload')
 }
 console.log('RATCJ_EXPECTED_UNAUTHENTICATED_REQUESTS='+JSON.stringify(unauthed))
 console.log('RATCJ_OTHER_HTTP_FAILURES='+JSON.stringify(statuses))
 console.log('RATCJ_BROWSER_ERRORS='+JSON.stringify(errors))
 assert.deepEqual(statuses,[],'Unrelated HTTP 4xx/5xx')
 assert.ok(unauthed.every(r=>r.method==='GET' && r.url===RAT+'/api/v1/rate-anything/me'),'Unexpected 401')
 const expectedConsole='Failed to load resource: the server responded with a status of 401 ()'
 assert.equal(errors.filter(x=>x===expectedConsole).length,unauthed.length,'Expected anonymous me 401 console mismatch')
 assert.deepEqual(errors.filter(x=>x!==expectedConsole),[],'Unexpected browser console/page errors')
 report.status='PASS'
 await writeFile(dir+'/report.json',JSON.stringify(report,null,2))
 await ctx.close()
 console.log('RATCJ_FINAL_THREE_WIDTH_BROWSER_ACCEPTANCE_PASS='+JSON.stringify({checks:report.checks.length,screenshots:report.screenshots.length,widths:[1440,768,390],views:['search','workspace'],reloadVerified:true}))
}catch(e){report.status='FAIL';report.errors.push(String(e));await writeFile(dir+'/report.json',JSON.stringify(report,null,2));throw e}
finally{await browser.close()}
NODE
echo RATCJ_FINAL_3WIDTH_READONLY_ACCEPTANCE_COMPLETE=true
