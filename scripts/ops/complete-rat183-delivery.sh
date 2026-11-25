#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 1 && $1 =~ ^(cleanup|release|thumbnails)$ ]]||exit 64
mode=$1;sha=bb7dc6f8659b21fe612c1a003b8054b29a8317e9;base=2359178af19fdd9b5b45d103c892cdc36c2ad086;ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);umask 077;trap 'rm -rf "$t"' EXIT
if [[ $mode != cleanup ]];then
 for f in release-rat183-thumbnails-isolated.sh accept-rat183-connected-thumbnails.sh check-rat181-protected-baseline.sh;do cp "$ops/$f" "$t/$f";done
 python3 - "$t" "$sha" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]);sha=sys.argv[2]
for filename in ['release-rat183-thumbnails-isolated.sh','accept-rat183-connected-thumbnails.sh']:
 f=p/filename;s=f.read_text();assert s.count('349d8178c4da2daa82434a3e3c37dad38c5aebeb')==1;s=s.replace('349d8178c4da2daa82434a3e3c37dad38c5aebeb',sha)
 if filename.startswith('release'):
  assert s.count('45614394445')==1;s=s.replace('45614394445','45620813540');s=s.replace('--depth=2 origin','--depth=3 origin')
 else:s=s.replace('evidence.ratingCleanupNotPerformed=','evidence.identifiedRatingStillPresent=')
 f.write_text(s)
PY
 if [[ $mode == release ]];then bash -n "$t/release-rat183-thumbnails-isolated.sh";bash "$t/release-rat183-thumbnails-isolated.sh";else bash -n "$t/accept-rat183-connected-thumbnails.sh";bash "$t/accept-rat183-connected-thumbnails.sh";fi
 exit
fi
# Independent read-only verification: never executes apply and never opens a LIVE DB connection.
bash "$ops/rum-dev-delete-rating.sh" verify 01m7x6ez57zpsgbv486y9t0y9h 01m7x799g070af1g4je8hh468v >"$t/verify"
mkdir "$t/relay";git -C "$t/relay" init -q;git -C "$t/relay" remote add origin https://github.com/DaisyCloverSoftware/workbench-relay-private.git;git -C "$t/relay" fetch -q --depth=1 --filter=blob:none origin main;git -C "$t/relay" show FETCH_HEAD:relay/control-outbox/rat183-20261125-delete-rating-apply-r1.json >"$t/apply"
python3 - "$t" <<'PY'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);r=json.loads((p/'apply').read_text());assert r['status']=='completed' and r['operation_status']['process_exit_code']==0;op=r['result']['operation_script'];assert op['exit_code']==0
prefix='DEV_RATING_DELETE_RECEIPT ';a=json.loads(op['output'].split(prefix,1)[1]);v=json.loads((p/'verify').read_text().split(prefix,1)[1]);assert a['deletedEventId']==v['eventId']=='01m7x799g070af1g4je8hh468v';assert v['physicalAbsence'] and v['cacheKeysRemaining']==0 and all(x['count']==0 for x in v['remaining'].values());assert a['protected']==v['protected'];assert a['guardsRestored'];assert all(g['tgenabled']=='O' for g in v['guards'])
print('PR183_PHYSICAL_DELETION_AND_PROTECTED_HASHES_PASS',json.dumps({'eventId':v['eventId'],'entityId':v['entityId'],'ratingEventsBefore':a['ratingEventsBefore'],'ratingEventsAfter':a['ratingEventsAfter'],'removedRows':a['removedRows'],'remainingScopedRows':sum(x['count'] for x in v['remaining'].values()),'protectedTablesMatched':len(v['protected']),'guardsEnabled':len(v['guards']),'cacheKeysRemaining':v['cacheKeysRemaining']}))
PY
k(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
k -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php <<'PHP'
<?php
require 'vendor/autoload.php';$app=require 'bootstrap/app.php';$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();use Illuminate\Support\Facades\DB;
DB::transaction(function(){DB::statement('SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY');$i=DB::selectOne('SELECT current_database() AS db,inet_server_addr() AS server');if($i->db!=='rum'||$i->server!=='10.42.1.177')throw new RuntimeException('Not pinned isolated DEV');
foreach(['rating_events'=>'01m7x799g070af1g4je8hh468v','rating_reviews'=>'01m7x799gn390a9k1j5kqv90q8','rating_journey_entries'=>'01m7x6ez8tty0499rb6bbe7e6n','outbox_events'=>'01m7x799gsy8dgqefzcmyhhp97'] as $t=>$id)if(DB::table($t)->where('id',$id)->exists())throw new RuntimeException('Residual exact removed row '.$t);
$main='01m7x6ez57zpsgbv486y9t0y9h';$person='01m0mw9hkbynssr7kbc0gk534j';$second='01m81mbmqvwr3d1nvfp9gkp3xf';$rating='01m81mdfc3af7x8xxn0mrvz1fq';$r=DB::table('rating_events')->where('id',$rating)->first();if(!$r||$r->target_entity_id!==$second)throw new RuntimeException('Protected second rating missing');$counts=[];foreach([$person,$main,$second] as $id)$counts[$id]=DB::table('rating_events')->where('target_entity_id',$id)->count();if($counts[$person]!==0||$counts[$main]!==0||$counts[$second]!==1)throw new RuntimeException('Unexpected per-entity rating counts');
$edges=DB::table('entity_relationships')->whereIn('source_entity_id',[$person,$main,$second])->orWhereIn('target_entity_id',[$person,$main,$second])->orderBy('id')->get();if($edges->pluck('id')->all()!==['01m7x7kn332fqjxyj9b78zrmh0','01m81n8fsgxf1p52bw62e9rzyp'])throw new RuntimeException('Protected edges changed');echo 'PR183_EXACT_IDS_ABSENT_OTHER_RATING_AND_EDGES_PRESERVED '.json_encode(['ratingCounts'=>$counts,'protectedRatingId'=>$rating,'relationshipIds'=>$edges->pluck('id')->all()]).PHP_EOL;
});
PHP
bash "$ops/check-rat181-protected-baseline.sh" >"$t/before"
mkdir "$t/source";git -C "$t/source" init -q;git -C "$t/source" remote add origin https://github.com/DaisyCloverSoftware/rum.git;git -C "$t/source" fetch -q --depth=1 origin "$sha";git -C "$t/source" checkout -q --detach FETCH_HEAD;cd "$t/source/apps/rate-anything";npm ci --ignore-scripts --no-audit --no-fund >"$t/npm" 2>&1;npx playwright install chromium >"$t/pw" 2>&1
node --input-type=module - "$sha" "$base" <<'JS'
import {chromium,expect} from '@playwright/test';import assert from 'node:assert/strict';
const [sha,base]=process.argv.slice(2),rat='https://dev-rum-ra.daisycloversoftware.uk',dev='https://dev-rum.daisycloversoftware.uk';const person='01m0mw9hkbynssr7kbc0gk534j',main='01m7x6ez57zpsgbv486y9t0y9h',second='01m81mbmqvwr3d1nvfp9gkp3xf',deleted='01m7x799g070af1g4je8hh468v',preserved='01m81mdfc3af7x8xxn0mrvz1fq';
const browser=await chromium.launch({headless:true,args:['--no-sandbox','--disable-dev-shm-usage']});const page=await browser.newPage({viewport:{width:1440,height:1000},serviceWorkers:'block'});page.setDefaultTimeout(30000);page.setDefaultNavigationTimeout(45000);const writes=[];
await page.route('**/*',r=>{const q=r.request(),u=new URL(q.url());if(!['GET','HEAD','OPTIONS'].includes(q.method())){writes.push({method:q.method(),path:u.pathname});return r.abort();}return [rat,dev].includes(u.origin)?r.continue():r.abort();});
async function get(path){const r=await page.request.get(dev+'/api/v1/public/entities/rat-catalogue/nodes/'+path);assert.equal(r.status(),200);return (await r.json()).data;}
try{
const version=(await (await page.request.get(rat+'/VERSION')).text()).trim();assert.ok([sha,base].some(s=>version==='rat-candidate+'+s));
const a=await get(main+'/ratings?offset=0&limit=20'),b=await get(second+'/ratings?offset=0&limit=20');assert.equal(a.items.length,0);for(const key of ['publicSubmitted','accepted','awaitingVerdict','inReview'])assert.equal(a.counts[key],0,key);assert.ok(b.items.some(x=>x.id===preserved));assert.ok(!JSON.stringify([a,b]).includes(deleted));const n=await get(main);assert.equal(n.active.id,main);assert.ok(n.active.reputation==null||n.active.reputation.ratingCount===0);assert.equal(n.categories.reduce((s,c)=>s+c.total,0),1);assert.equal(n.relatedThrough[0].items[0].id,second);
const url=rat+'/e/thing~'+main+'#ratings';await page.goto(url);await expect(page.locator('.rat-workspace-id')).toHaveText(main);const ratings=page.getByLabel('Public rating details');await expect(ratings.getByText('No public rating submissions for this Thing yet.',{exact:true})).toBeVisible();await expect(ratings.locator('.rat-public-rating-card')).toHaveCount(0);await expect(ratings.locator('.rat-public-ratings__summary')).toContainText('0 accepted');await expect(ratings.locator('.rat-public-ratings__summary')).toContainText('0 awaiting community verdict');await page.reload();await expect(ratings.getByText('No public rating submissions for this Thing yet.',{exact:true})).toBeVisible();await expect(ratings.locator('.rat-public-rating-card')).toHaveCount(0);
await page.goto(rat+'/e/thing~'+second+'#ratings');await expect(page.locator('.rat-workspace-id')).toHaveText(second);await expect(page.locator('.rat-public-rating-card')).toHaveCount(1);await page.goto(url);await expect(ratings.getByText('No public rating submissions for this Thing yet.',{exact:true})).toBeVisible();assert.deepEqual(writes,[]);
console.log('PR183_DELETED_RATING_API_AND_PUBLIC_CHROMIUM_PASS',JSON.stringify({deletedEventId:deleted,preservedEventId:preserved,chromium:browser.version(),frontendVersion:version,mainCounts:a.counts,mainCards:0,secondCards:1,reputation:n.active.reputation,refresh:true,navigation:true,httpWrites:0}));
}finally{await browser.close();}
JS
bash "$ops/check-rat181-protected-baseline.sh" >"$t/after";for f in before after;do grep -E '^((entities|entity_identifiers|rating_events|relationships) rows=|AUDIENCE |ROUTING )' "$t/$f" >"$t/$f.compare";done;diff -u "$t/before.compare" "$t/after.compare";cat "$t/after.compare"
echo PR183_PERMANENT_DELETION_ACCEPTED_DB_API_AND_BROWSER
