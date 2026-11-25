#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 1 && $1 =~ ^(browser|audit)$ ]]||exit 64
ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
if [[ $1 == browser ]];then
 cp "$ops/check-rat181-protected-baseline.sh" "$t/check-rat181-protected-baseline.sh"
 python3 - "$ops/accept-rat183-connected-thumbnails.sh" "$t/accept.sh" <<'PY'
from pathlib import Path
import sys
s=Path(sys.argv[1]).read_text();assert s.count('349d8178c4da2daa82434a3e3c37dad38c5aebeb')==1;s=s.replace('349d8178c4da2daa82434a3e3c37dad38c5aebeb','bb7dc6f8659b21fe612c1a003b8054b29a8317e9');s=s.replace('const page=await browser.newPage','let page=await browser.newPage');s=s.replace('evidence.ratingCleanupNotPerformed=','evidence.identifiedRatingStillPresent=')
start=s.index("await page.route('**/*'");end=s.index('\nconst direct=',start);routing=s[start:end];assert routing.endswith('});')
s=s[:start]+'async function readOnlyRoutes(){'+routing+'}\nawait readOnlyRoutes();'+s[end:]
old="const imageRequest=u=>u.href===photo;await page.route(imageRequest,r=>r.abort());await workspace(second);await expect(indirect().locator('.rat-workspace-connection__mark')).toHaveText('C');"
new="""await page.close();page=await browser.newPage({viewport:{width:1440,height:1000},serviceWorkers:'block'});page.setDefaultTimeout(30000);page.setDefaultNavigationTimeout(45000);await readOnlyRoutes();
let blockedProfileRequests=0;const imageRequest=u=>u.href===photo;await page.route(imageRequest,r=>{blockedProfileRequests++;return r.abort('failed');});await workspace(second);await indirect().scrollIntoViewIfNeeded();await expect.poll(()=>blockedProfileRequests,{timeout:30000}).toBeGreaterThan(0);await expect(indirect().locator('.rat-workspace-connection__mark')).toHaveText('C',{timeout:30000});evidence.blockedProfileRequests=blockedProfileRequests;"""
assert s.count(old)==1;s=s.replace(old,new)
s=s.replace('evidence.directPublishedPhotoRefreshNavigation=true;',"evidence.directPublishedPhotoRefreshNavigation=true;console.log('DIRECT_PUBLISHED_PHOTO_PASS');").replace('evidence.indirectPublishedPhotoRefreshNavigation=true;',"evidence.indirectPublishedPhotoRefreshNavigation=true;console.log('INDIRECT_PUBLISHED_PHOTO_PASS');").replace('evidence.searchAndLinkedCategoriesPreserved=true;',"evidence.searchAndLinkedCategoriesPreserved=true;console.log('SEARCH_PUBLISHED_PHOTO_PASS');")
Path(sys.argv[2]).write_text(s)
PY
 bash -n "$t/accept.sh";bash "$t/accept.sh";exit
fi
# No application-data writes. Explain any change between independently guarded operations.
bash "$ops/check-rat181-protected-baseline.sh"
k(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ $(k -n rum-dev-isolated get namespace rum-dev-isolated -o 'jsonpath={.metadata.uid}') == 6804e79b-3000-42c1-b37e-f07a57252425 ]]||exit 78
k -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php <<'PHP'
<?php
require 'vendor/autoload.php';$app=require 'bootstrap/app.php';$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();use Illuminate\Support\Facades\DB;
DB::transaction(function(){DB::statement('SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY');$i=DB::selectOne('SELECT current_database() AS db,inet_server_addr() AS server');if($i->db!=='rum'||$i->server!=='10.42.1.177')throw new RuntimeException('Not pinned DEV');
$ids=['01m0mw9hkbynssr7kbc0gk534j','01m7x6ez57zpsgbv486y9t0y9h','01m81mbmqvwr3d1nvfp9gkp3xf'];$rows=DB::table('entities')->whereIn('id',$ids)->orderBy('id')->get(['id','status','visibility','publication_state','version','updated_at','entity_type_id']);echo 'PROTECTED_ENTITY_CURRENT_METADATA '.$rows->toJson().PHP_EOL;
$r=DB::table('entity_profile_revisions')->whereIn('entity_id',$ids)->where('occurred_at','>=','2026-11-25 13:12:00+00')->orderBy('occurred_at')->get(['id','entity_id','field_key','source','moderation_status','outcome','occurred_at','metadata']);echo 'INTERVENING_PROFILE_REVISION_METADATA '.$r->toJson().PHP_EOL;
$event='01m7x799g070af1g4je8hh468v';if(DB::table('rating_events')->where('id',$event)->exists())throw new RuntimeException('Deleted event unexpectedly present');if(DB::table('rating_events')->where('id','01m81mdfc3af7x8xxn0mrvz1fq')->where('target_entity_id',$ids[2])->count()!==1)throw new RuntimeException('Protected rating missing');echo 'EXACT_DELETION_AND_SECOND_RATING_STILL_CORRECT'.PHP_EOL;
});
PHP
