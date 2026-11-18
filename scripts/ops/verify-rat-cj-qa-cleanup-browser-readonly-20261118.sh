#!/usr/bin/env bash
set -Eeuo pipefail
# Independent read-only post-cleanup audit: exact DEV API/RAT images, canonical DB, HTTP and real Chromium.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
api='ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430'
rat='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:ec11bfec432ff10d508850a0a071b3af3787bff656d2decca90eed5951cd0003'
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;$id="01m07d2f9fj05f4ypdbwp087qj";
$cj=App\Models\Entity::query()->with("type")->findOrFail($id);
$rows=$db::table("entity_relationships")->where(function($q)use($id){$q->where("source_entity_id",$id)->orWhere("target_entity_id",$id);})->get(["id","valid_to"]);
$ended=$rows->whereNotNull("valid_to")->count();$active=$rows->whereNull("valid_to")->count();
$ratings=$db::table("rating_events")->where("target_entity_id",$id)->count();
$revisions=$db::table("entity_profile_revisions")->where("entity_id",$id)->where("field_key","thing_type")->get(["id","outcome"]);
$photos=app(App\Services\ThingImageService::class)->publicImages($cj);
if($cj->canonical_name!=="CJ Investigates"||$cj->type?->key!=="digital_resource"||$active!==0||$ended!==22||$ratings!==54||
 $revisions->count()!==1||$revisions[0]->id!=="01m7qsxdjs1tg298py83xk2y49"||
 $photos["profile"]===null||$photos["cover"]===null)exit(78);
echo "CJ_QA_POST_DB=".json_encode(["historicalEnded"=>$ended,"active"=>$active,"ratingEvents"=>$ratings,"thingTypeRevision"=>$revisions[0]->id,"imagesPreserved"=>true],JSON_UNESCAPED_SLASHES)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
[[ "$(curl -fsS --retry 3 --max-time 20 https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == 'rat-candidate+887b5e02f4be6ec7e52371666effa9eb47892d6e' ]] || exit 78
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin 887b5e02f4be6ec7e52371666effa9eb47892d6e
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == '887b5e02f4be6ec7e52371666effa9eb47892d6e' ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for p in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser;do if [[ -x "$p" ]];then chrome="$p";break;fi;done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
CHROME="$chrome" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import {chromium,expect} from '@playwright/test'
const CJ='01m07d2f9fj05f4ypdbwp087qj'
const DEV='https://dev-rum.daisycloversoftware.uk'
const RAT='https://dev-rum-ra.daisycloversoftware.uk'
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
try{
 const ctx=await browser.newContext({viewport:{width:1440,height:940}})
 const errors=[]
 const page=await ctx.newPage()
 page.on('pageerror',error=>errors.push(error.message))
 page.on('console',message=>{if(message.type()==='error')errors.push(message.text())})
 await page.route('**/*',r=>['GET','HEAD','OPTIONS'].includes(r.request().method())?r.continue():r.abort('blockedbyclient'))
 const response=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+CJ,{headers:{'Cache-Control':'no-cache'}})
 assert.equal(response.status(),200,'Public canonical CJ API')
 const data=(await response.json()).data
 assert.equal(data.active.id,CJ)
 assert.equal(data.active.name,'CJ Investigates')
 assert.deepEqual(data.topLinked,[],'Public API topLinked is not empty')
 assert.deepEqual(data.categories,[],'Public API categories are not empty')
 console.log('CJ_QA_PUBLIC_API_PASS='+JSON.stringify({topLinked:data.topLinked.length,categories:data.categories.length,entity:data.active.name}))
 for(const width of [1440,390]){
  await page.setViewportSize({width,height:width===390?850:940})
  await page.goto(RAT+'/search?q=CJ%20Investigates&active='+CJ,{waitUntil:'domcontentloaded',timeout:45000})
  await expect(page.locator('.rat-active-card h1')).toHaveText('CJ Investigates',{timeout:30000})
  await expect(page.locator('.rat-top-linked')).toHaveCount(0,{timeout:30000})
  await expect(page.locator('section.rat-category:not(.rat-category--empty)')).toHaveCount(0)
  await expect(page.locator('section.rat-category--empty h2')).toHaveText('No public linked Things yet')
  await expect(page.locator('.rat-linked-card')).toHaveCount(0)
  const body=await page.locator('body').innerText()
  assert.ok(!/Quillstone|RUM DEV linked check|Top linked things|Products/.test(body),'QA fixture or category still visible')
  assert.ok((await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth))<=2,'Horizontal overflow')
  console.log('CJ_QA_RAT_BROWSER_PASS='+JSON.stringify({width,active:'CJ Investigates',linkedCards:0,topLinkedSection:false,productsSection:false,emptyState:true}))
 }
 assert.deepEqual(errors,[],'Browser console/page errors')
 await ctx.close()
 console.log('CJ_QA_BROWSER_ACCEPTANCE_COMPLETE=true')
}finally{await browser.close()}
NODE
echo CJ_QA_READONLY_POST_VERIFICATION_PASS=true
