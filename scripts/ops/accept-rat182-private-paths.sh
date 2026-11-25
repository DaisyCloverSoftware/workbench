#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 4 && $1 == 2359178af19fdd9b5b45d103c892cdc36c2ad086 && $2 =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[a-f0-9]{64}$ && $3 =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[a-f0-9]{64}$ && $4 =~ ^qa181[a-z0-9]{8}$ ]] || exit 64
ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);umask 077;trap 'rm -rf "$t"' EXIT
# Reuse the reviewed fresh loopback PostgreSQL pod and exact-image/static-asset topology.
# No ingress, service, existing secret, canonical database or shared volume is changed.
cp "$ops/rat181-fixture-loopback.mjs" "$t/rat181-fixture-loopback.mjs"
cat >"$t/rat181-fixture-probe-20261122.sh" <<'PROBE'
#!/usr/bin/env bash
# PHP is extracted into a pod-UID-bound probe by the reviewed disposable runner.
exit 64
: <<'PHP'
<?php
require 'vendor/autoload.php';$app=require 'bootstrap/app.php';$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
use App\Models\Entity;use App\Models\EntityType;use App\Models\EntityRelationship;use App\Models\RelationshipType;use App\Models\Classification;use App\Services\EntityClassificationService;use App\Services\EntityNameNormalizer;use Illuminate\Support\Facades\DB;
[$file,$action,$tag]=$argv;
if(!preg_match('/^qa181[a-z0-9]{8}$/',$tag)||!app()->environment('testing')||config('database.connections.pgsql.host')!=='127.0.0.1'||DB::selectOne('SELECT current_database() AS db')->db!=='rat181_browser')throw new RuntimeException('Not isolated fixture DB');
$stateFile='/tmp/rat182-private-paths-'.$tag.'.json';
if($action==='seed'){
 if(is_file($stateFile)||Entity::query()->exists()||DB::table('rating_events')->exists())throw new RuntimeException('Fixture DB not empty');
 $make=function(string $key,string $name,bool $publicPersona=false)use($tag):Entity{return Entity::query()->create(['entity_type_id'=>EntityType::query()->where('key',$key)->firstOrFail()->id,'canonical_name'=>$name.' '.$tag,'normalised_name'=>app(EntityNameNormalizer::class)->normalise($name.' '.$tag),'canonical_slug'=>str($name.' '.$tag)->slug()->toString(),'status'=>'active','visibility'=>'public','publication_state'=>'published','rateability'=>$publicPersona||$key!=='person'?'open':'closed','reviewability'=>'closed','sensitivity'=>$publicPersona?'elevated':'normal','verification_state'=>'unverified']);};
 $channel=function(string $name)use($make):Entity{$e=$make('digital_resource',$name);$key='community_'.substr(hash('sha256',app(EntityNameNormalizer::class)->normalise('YouTube Channel')),0,52);Classification::query()->firstOrCreate(['key'=>$key],['canonical_name'=>'YouTube Channel','entity_type_id'=>$e->entity_type_id,'status'=>'active']);app(EntityClassificationService::class)->assign($e,[$key]);return $e;};
 $edge=function(Entity $a,Entity $b,string $visibility='public'):EntityRelationship{return EntityRelationship::query()->create(['source_entity_id'=>$a->id,'target_entity_id'=>$b->id,'relationship_type_id'=>RelationshipType::query()->where('key','related_to')->firstOrFail()->id,'valid_from'=>now(),'visibility'=>$visibility,'verification_state'=>'unverified','source'=>'user_submitted','provenance'=>'user_submitted']);};
 $a=$channel('Fixture amber broadcast');$b=$channel('Fixture blue broadcast');$p=$make('person','Fixture public creator',true);$p->forceFill(['status'=>'unverified'])->save();$private=$make('person','PRIVATE RUM IDENTITY');$private->forceFill(['visibility'=>'private','publication_state'=>'hidden'])->save();$ordinary=$make('person','ORDINARY RUM MEMBER');$d=$channel('PRIVATE PATH DESTINATION');$hidden=$channel('HIDDEN DESTINATION');$hidden->forceFill(['publication_state'=>'hidden'])->save();$memberEdge=$channel('MEMBERS EDGE DESTINATION');$generic=$make('organisation','GENERIC NEIGHBOUR');
 $first=$edge($a,$p);$second=$edge($p,$b);$edge($a,$private);$edge($private,$d);$edge($a,$ordinary);$edge($ordinary,$d);$edge($p,$hidden);$edge($p,$memberEdge,'members');$edge($a,$generic);$edge($generic,$d);
 $state=['source'=>(string)$a->id,'target'=>(string)$b->id,'via'=>(string)$p->id,'viaName'=>(string)$p->canonical_name,'excluded'=>array_map(static fn(Entity $e):string=>(string)$e->id,[$private,$ordinary,$d,$hidden,$memberEdge]),'path'=>[(string)$first->id,(string)$second->id],'edgeHash'=>hash('sha256',DB::table('entity_relationships')->orderBy('id')->get()->toJson())];file_put_contents($stateFile,json_encode($state,JSON_THROW_ON_ERROR));echo json_encode($state,JSON_THROW_ON_ERROR);exit;
}
if(!is_file($stateFile))throw new RuntimeException('No fixture state');$s=json_decode(file_get_contents($stateFile),true,512,JSON_THROW_ON_ERROR);
if($action==='hide'||$action==='restore')Entity::query()->findOrFail($s['via'])->forceFill(['publication_state'=>$action==='hide'?'hidden':'published'])->save();
elseif($action==='hide-target')Entity::query()->findOrFail($s['target'])->forceFill(['visibility'=>'private'])->save();
elseif($action!=='inspect')throw new RuntimeException('Unknown test action');
$edges=DB::table('entity_relationships')->orderBy('id')->get();$ratings=DB::table('rating_events')->count();$direct=EntityRelationship::query()->whereIn('source_entity_id',[$s['source'],$s['target']])->whereIn('target_entity_id',[$s['source'],$s['target']])->count();
if($ratings!==0||$direct!==0||hash('sha256',$edges->toJson())!==$s['edgeHash'])throw new RuntimeException('Unexpected fixture ratings or graph write');echo json_encode(['action'=>$action,'edgeCount'=>$edges->count(),'ratings'=>$ratings,'artificialDirectEdges'=>$direct,'edgeHash'=>$s['edgeHash']],JSON_THROW_ON_ERROR);
PHP
PROBE
cat >"$t/rat181-acceptance-20261122.mjs" <<'JS'
import {chromium,expect} from '@playwright/test';import assert from 'node:assert/strict';import {execFileSync} from 'node:child_process';
const [sha,tag,probePath]=process.argv.slice(2),rat='https://dev-rum-ra.daisycloversoftware.uk',dev='https://dev-rum.daisycloversoftware.uk';
const db=action=>JSON.parse(execFileSync('bash',[probePath,sha,action,tag],{encoding:'utf8',timeout:60000}));
const browser=await chromium.launch({headless:true,args:['--no-sandbox','--disable-dev-shm-usage']});const page=await browser.newPage({viewport:{width:1440,height:1000},serviceWorkers:'block'});page.setDefaultTimeout(30000);const writes=[];
await page.route('**/*',r=>{const q=r.request();if(!['GET','HEAD','OPTIONS'].includes(q.method())){writes.push(new URL(q.url()).pathname);return r.abort();}return [rat,dev].includes(new URL(q.url()).origin)?r.continue():r.abort();});
async function node(id){const r=await page.evaluate(async url=>{const response=await fetch(url,{credentials:'omit',headers:{Accept:'application/json'}});return {status:response.status,body:await response.json()};},dev+'/api/v1/public/entities/rat-catalogue/nodes/'+id);assert.equal(r.status,200);assert.equal(r.body.meta.privacy,'private-rum-people-excluded');return r.body;}
try{await page.goto(rat);const s=db('seed');await page.goto(rat+'/e/thing~'+s.source);await expect(page.locator('.rat-workspace-id')).toHaveText(s.source);let j=await node(s.source);for(const id of s.excluded)assert.ok(!JSON.stringify(j).includes(id),'private/generic path leaked '+id);assert.equal(j.data.relatedThrough.length,1);assert.equal(j.data.relatedThrough[0].via.id,s.via);assert.equal(j.data.relatedThrough[0].items.length,1);assert.equal(j.data.relatedThrough[0].items[0].id,s.target);assert.deepEqual(j.data.relatedThrough[0].items[0].path.map(e=>e.id),s.path);assert.equal(j.data.categories.reduce((n,c)=>n+c.total,0),2);
const derived=page.getByRole('region',{name:'Indirect connected Things'});await expect(derived.locator('[data-entity-id]')).toHaveCount(1);await expect(derived.getByText(/not direct links or verified shared ownership/)).toBeVisible();await derived.locator(`[data-entity-id="${s.target}"]`).click();await expect(page.locator('.rat-workspace-id')).toHaveText(s.target);await page.reload();await expect(derived.locator(`[data-entity-id="${s.source}"]`)).toBeVisible();
db('hide');await page.goto(rat+'/e/thing~'+s.source);await expect(page.locator('.rat-workspace-id')).toHaveText(s.source);await expect(derived).toHaveCount(0);j=await node(s.source);assert.equal(j.data.relatedThrough.length,0);assert.ok(!JSON.stringify(j).includes(s.via));for(const id of s.excluded)assert.ok(!JSON.stringify(j).includes(id));
db('restore');await page.reload();await expect(derived.locator(`[data-entity-id="${s.target}"]`)).toBeVisible();db('hide-target');await page.reload();await expect(page.locator('.rat-workspace-id')).toHaveText(s.source);await expect(derived).toHaveCount(0);j=await node(s.source);assert.equal(j.data.relatedThrough.length,0);assert.ok(!JSON.stringify(j).includes(s.target));
assert.deepEqual(writes,[]);const snapshot=db('inspect');assert.equal(snapshot.artificialDirectEdges,0);assert.equal(snapshot.ratings,0);console.log('RAT182_PRIVATE_PATH_CHROMIUM_PASS '+JSON.stringify({sha,chromium:browser.version(),tag,source:s.source,target:s.target,via:s.via,persistedPath:s.path,privateAndOrdinaryPeopleExcluded:true,privatePathsExcluded:true,hiddenIntermediateRemovedOnRefresh:true,privateTargetRemovedOnRefresh:true,genericNeighbourNotExpanded:true,actualReciprocalNavigation:true,artificialDirectEdges:snapshot.artificialDirectEdges,ratings:snapshot.ratings,httpWrites:writes.length,canonicalWrites:0}));
}catch(e){console.log('PRIVATE_PATH_BROWSER_FAILURE',new URL(page.url()).pathname,(await page.locator('body').innerText()).slice(-1600));throw e;}finally{await browser.close();}
JS
python3 - "$ops/verify-rat181-disposable-image-browser.sh" "$t/run.sh" "$1" <<'PY'
from pathlib import Path
import sys
s=Path(sys.argv[1]).read_text();assert s.count('b135949c8d97072cca3b24c2611984ae6d6fc30c')==1;s=s.replace('b135949c8d97072cca3b24c2611984ae6d6fc30c',sys.argv[3])
for old,new in [
 ('pod=rat181-cell-$tag','pod=rat182-privacy-$tag'),
 ('"$tag" "$t" &tls=$!', '"$tag" "$t" </dev/null >"$t/tls.log" 2>&1 &tls=$!'),
 (':18081 >"$t/pf"',':18081 </dev/null >"$t/pf"'),
 ('^(seed|hide|restore|inspect)$','^(seed|hide|restore|hide-target|inspect)$'),
 ('"SESSION_DRIVER":"file"','"SESSION_DRIVER":"file","MODERATION_EVIDENCE_KEY":"base64:"+base64.b64encode(secrets.token_bytes(32)).decode(),"MODERATION_AUDIT_IP_HASH_KEY":"base64:"+base64.b64encode(secrets.token_bytes(32)).decode()')]:
 assert s.count(old)==1,(old,s.count(old));s=s.replace(old,new)
Path(sys.argv[2]).write_text(s)
PY
node --check "$t/rat181-acceptance-20261122.mjs";bash -n "$t/run.sh";bash "$t/run.sh" "$@"
echo RAT182_PRIVATE_DEPLOYED_IMAGE_ACCEPTANCE_PASS
