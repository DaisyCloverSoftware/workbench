#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only authentic browser verification of the EXISTING published CJ photo.
[[ $# -eq 4 ]] || exit 64
source_sha="$1";api="$2";rat="$3";rid="$4"
[[ "$source_sha" == '9c8af9fe07e9a0c4b0b1896f4be1903719b781e3' ]] || exit 78
[[ "$api" == 'ghcr.io/daisycloversoftware/rum-api@sha256:76731a87574a11547938a398dc15ecf87ced56d63b6622cd50bab5836dfeee71' ]] || exit 78
[[ "$rat" == 'ghcr.io/daisycloversoftware/rum-rate-anything@sha256:3938ed73de97546fccc527f9025cd5b7b21fb56d69bd4eb9be3a7ebabffb3919' ]] || exit 78
[[ "$rid" =~ ^[a-z0-9-]{18,44}$ ]] || exit 64
out="/tmp/rat175-cj-published-browser-$rid"
[[ ! -e "$out" ]] || { echo 'Refusing duplicate browser run id'; exit 65; }
mkdir -m 700 "$out"
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --retry 3 --retry-delay 2 --max-time 20 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$source_sha" ]] || exit 78
health="$(curl -fsS --retry 3 --retry-delay 2 --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$health" "$source_sha" <<'PY'
import json,sys
j=json.loads(sys.argv[1])
assert j.get("status")=="ok" and j.get("service")=="rum-api" and sys.argv[2] in str(j.get("version",""))
PY
# Read-only canonical guards prevent confusing a different identity/image with CJ.
code='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;$id="01m07d2f9fj05f4ypdbwp087qj";
$e=App\Models\Entity::query()->findOrFail($id);
$media=App\Models\MediaAsset::query()->findOrFail("01m7q2fbv84465vcd0jsng1vbp");
$rev=App\Models\EntityProfileRevision::query()->findOrFail("01m7q2fdfz4kb104tbgbjr6veh");
$p=app(App\Services\RatCommunityContributionPolicy::class);
if($e->canonical_name!=="CJ Investigates"||$e->type?->key!=="persona"||$e->status!=="unverified"
 ||$e->visibility!=="public"||$e->publication_state!=="published"||$p->isClaimed($e))exit(78);
if($db::table("entities")->where("canonical_name","CJ Investigates")->count()!==1)exit(78);
if($db::table("entity_profile_revisions")->where("entity_id",$id)->where("field_key","profile_image")->count()!==1)exit(78);
if($rev->outcome!=="applied"||$rev->moderation_status!=="approved"||$rev->field_key!=="profile_image"
 ||$media->status!=="ready"||$media->mime_type!=="image/webp"||$media->sha256!=="55124575ce8c175d9def68e6918a269b876f5b88e4ceff7498b902011684e1a9")exit(78);
$pic=app(App\Services\ThingImageService::class)->publicImages($e)["profile"];
if(!is_array($pic)||!str_contains((string)$pic["url"],(string)$media->id))exit(78);
echo "CJ_PUBLISHED_CANONICAL_READONLY=".json_encode(["entity"=>$id,"profileImageRevision"=>$rev->id,"media"=>$media->id,"published"=>true,"duplicateEntities"=>false,"profileImageRevisionCount"=>1,"ratingEventCount"=>$db::table("rating_events")->where("target_entity_id",$id)->count()])."\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$code"
scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT
git -C "$scratch" init -q
git -C "$scratch" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$scratch" fetch -q --depth=1 origin "$source_sha"
git -C "$scratch" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$scratch" rev-parse HEAD)" == "$source_sha" ]] || exit 78
cd "$scratch/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for p in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser;do
 if [[ -x "$p" ]];then chrome="$p";break;fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
CHROME="$chrome" CJ_DIR="$out" CJ_RUN_ID="$rid" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import {createHash} from 'node:crypto'
import {writeFile,stat} from 'node:fs/promises'
import {chromium,expect} from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk'
const DEV='https://dev-rum.daisycloversoftware.uk'
const CJ='01m07d2f9fj05f4ypdbwp087qj'
const MEDIA='01m7q2fbv84465vcd0jsng1vbp'
const REV='01m7q2fdfz4kb104tbgbjr6veh'
const expectedSha='55124575ce8c175d9def68e6918a269b876f5b88e4ceff7498b902011684e1a9'
const dir=process.env.CJ_DIR
const report={status:'NOT_ASSESSED',run:process.env.CJ_RUN_ID,checks:[],screenshots:[],errors:[]}
const pass=s=>{report.checks.push(s);console.log('CJ_PUBLISHED_BROWSER_PASS='+s)}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const take=async(page,name,locator)=>{
 const filename=name+'.jpg',path=dir+'/'+filename
 if(locator)await locator.screenshot({path,type:'jpeg',quality:72,animations:'disabled'})
 else await page.screenshot({path,type:'jpeg',quality:68,fullPage:true,animations:'disabled'})
 let bytes=(await stat(path)).size
 assert.ok(bytes>1200&&bytes<750000,'Unexpected screenshot dimensions')
 report.screenshots.push({name:filename,bytes})
 console.log('CJ_REAL_PAGE_SCREENSHOT='+filename+' bytes='+bytes)
}
let stage='canonical'
try{
 const ctx=await browser.newContext({viewport:{width:1440,height:940}})
 const nodeResponse=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+CJ)
 assert.equal(nodeResponse.status(),200)
 const body=await nodeResponse.json()
 assert.equal(body.meta?.privacy,'private-rum-people-excluded')
 assert.equal(body.data?.active?.id,CJ)
 assert.equal(body.data?.active?.name,'CJ Investigates')
 const picture=body.data.active.images?.profile
 assert.equal(picture?.status,'published')
 assert.equal(picture?.revisionId,REV)
 assert.ok(picture.url.includes('/api/v1/public/entities/rat-image/'+MEDIA+'/content'))
 const imageResponse=await ctx.request.get(picture.url)
 assert.equal(imageResponse.status(),200)
 assert.ok((imageResponse.headers()['content-type']||'').startsWith('image/webp'))
 const imageBytes=await imageResponse.body()
 assert.equal(createHash('sha256').update(imageBytes).digest('hex'),expectedSha)
 pass('original-sanitised-CJ-public-160px-webp-loaded-from-canonical-media')

 for(const width of [1440,390]){
  const page=await ctx.newPage()
  await page.setViewportSize({width,height:width===390?844:940})
  const pageErrors=[]
  page.on('pageerror',e=>pageErrors.push(String(e).slice(0,300)))
  stage='thing-'+width
  await page.goto(RAT+'/e/cj-investigates~'+CJ,{waitUntil:'domcontentloaded',timeout:40000})
  await expect(page.locator('.rat-workspace-header h1')).toContainText('CJ Investigates',{timeout:30000})
  const hero=page.locator('.rat-workspace-avatar > img')
  await expect(hero).toHaveAttribute('src',picture.url,{timeout:30000})
  await expect.poll(()=>hero.evaluate(el=>el.complete&&el.naturalWidth>0&&el.naturalHeight>0),{timeout:20000}).toBe(true)
  assert.equal(await hero.evaluate(el=>el.naturalWidth),160)
  await take(page,'cj-'+width+'-thing-full')
  await take(page,'cj-'+width+'-thing-avatar',page.locator('.rat-workspace-avatar'))
  await page.reload({waitUntil:'domcontentloaded'})
  await expect(page.locator('.rat-workspace-avatar > img')).toHaveAttribute('src',picture.url,{timeout:25000})
  await expect.poll(()=>page.locator('.rat-workspace-avatar > img').evaluate(el=>el.complete&&el.naturalWidth===160)).toBe(true)
  pass('cj-'+width+'-thing-photo-persists-after-refresh')
  stage='search-'+width
  await page.goto(RAT+'/search?q=cj&active='+CJ,{waitUntil:'domcontentloaded',timeout:40000})
  await expect(page.locator('.rat-active-card h1')).toContainText('CJ Investigates',{timeout:30000})
  const search=page.locator('.rat-active-card__avatar > img')
  await expect(search).toHaveAttribute('src',picture.url)
  await expect.poll(()=>search.evaluate(el=>el.complete&&el.naturalWidth===160),{timeout:20000}).toBe(true)
  await take(page,'cj-'+width+'-search-full')
  await take(page,'cj-'+width+'-search-avatar',page.locator('.rat-active-card__avatar'))
  await page.reload({waitUntil:'domcontentloaded'})
  await expect(page.locator('.rat-active-card__avatar > img')).toHaveAttribute('src',picture.url,{timeout:30000})
  await expect.poll(()=>page.locator('.rat-active-card__avatar > img').evaluate(el=>el.complete&&el.naturalWidth===160)).toBe(true)
  assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)<=2,'Horizontal overflow')
  assert.deepEqual(pageErrors,[])
  pass('cj-'+width+'-search-photo-persists-without-initials-or-broken-images')
  await page.close()
 }
 await ctx.close()
 stage='new-session'
 const fresh=await browser.newContext({viewport:{width:390,height:844}})
 const freshPage=await fresh.newPage()
 await freshPage.goto(RAT+'/search?q=cj&active='+CJ,{waitUntil:'domcontentloaded',timeout:40000})
 await expect(freshPage.locator('.rat-active-card h1')).toContainText('CJ Investigates',{timeout:30000})
 await expect(freshPage.locator('.rat-active-card__avatar > img')).toHaveAttribute('src',picture.url)
 await expect.poll(()=>freshPage.locator('.rat-active-card__avatar > img').evaluate(el=>el.complete&&el.naturalWidth===160)).toBe(true)
 pass('independent-new-browser-session-uses-same-real-CJ-photo')
 await fresh.close()
 report.status='PASS'
 console.log('CJ_PUBLISHED_IMAGE_REAL_BROWSER_ACCEPTANCE_PASS=true')
}catch(e){
 report.status='FAIL';report.errors.push({stage,message:String(e).slice(0,1500)})
 console.log('CJ_PUBLISHED_IMAGE_BROWSER_FAILURE='+JSON.stringify(report.errors[0]))
 process.exitCode=1
}finally{
 await writeFile(dir+'/report.json',JSON.stringify(report,null,2))
 console.log('CJ_PUBLISHED_BROWSER_REPORT='+JSON.stringify(report))
 await browser.close()
}
NODE
echo CJ_PUBLISHED_IMAGE_BROWSER_RECEIPT_READY=true
