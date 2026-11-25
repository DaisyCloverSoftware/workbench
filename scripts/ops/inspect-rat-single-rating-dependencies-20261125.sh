#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]] || exit 64
k(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
for ns in rum-dev-isolated rum-dev;do
 k -n "$ns" get namespace "$ns" -o 'jsonpath={.metadata.name}{" uid="}{.metadata.uid}{"\n"}'
 k -n "$ns" get svc rum-data-postgres -o 'jsonpath={.metadata.namespace}{" serviceUID="}{.metadata.uid}{" clusterIP="}{.spec.clusterIP}{" type="}{.spec.type}{"\n"}'
 k -n "$ns" get endpoints rum-data-postgres -o json|python3 -c 'import json,sys;j=json.load(sys.stdin);print("DB_ENDPOINT",j["metadata"]["namespace"],json.dumps(j["subsets"]))'
done
k -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php <<'PHP'
<?php
require 'vendor/autoload.php';$app=require 'bootstrap/app.php';$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
use Illuminate\Support\Facades\DB;use Illuminate\Support\Facades\Schema;
DB::transaction(function(){DB::statement('SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY');
$id='01m7x799g070af1g4je8hh468v';$entity='01m7x6ez57zpsgbv486y9t0y9h';$review='01m7x799gn390a9k1j5kqv90q8';$journey='01m7x6ez8pj1k53cj4y8nnw4q6';
echo 'DB_ID '.json_encode(DB::selectOne('SELECT current_database() AS db,current_user AS role,inet_server_addr() AS server,inet_server_port() AS port')).PHP_EOL;
foreach(['rating_challenges','rating_dimension_values','rating_event_admission_states','rating_event_admission_votes','rating_event_reasons','rating_event_revisions','rating_evidence','rating_journey_entries','rating_weight_snapshots','rating_reason_votes'] as $t){$q=DB::table($t)->where('rating_event_id',$id);$rows=$q->get();echo 'DEPENDENCY '.json_encode(['table'=>$t,'count'=>$rows->count(),'ids'=>$rows->map(fn($r)=>$r->id??null),'sha256'=>hash('sha256',$rows->toJson())]).PHP_EOL;}
foreach(['rating_review_revisions','rating_review_responses','rating_challenges'] as $t){$rows=DB::table($t)->where('rating_review_id',$review)->get();echo 'REVIEW_DEPENDENCY '.json_encode(['table'=>$t,'count'=>$rows->count(),'ids'=>$rows->map(fn($r)=>$r->id??null)]).PHP_EOL;}
echo 'JOURNEY '.DB::table('rating_journeys')->where('id',$journey)->get()->toJson().PHP_EOL;echo 'JOURNEY_ALL_ENTRIES '.DB::table('rating_journey_entries')->where('rating_journey_id',$journey)->get()->toJson().PHP_EOL;
echo 'PARENT_REFERENCES '.DB::table('rating_journey_entries')->where('parent_rating_event_id',$id)->count().PHP_EOL;
foreach(['profile_help_me_out_pins'=>['rating_kind','rating_event','rating_id'],'moderation_cases'=>['object_type','rating_event','object_id'],'moderation_actions'=>['target_type','rating_event','target_id']] as $t=>$v)echo 'POLYMORPHIC '.$t.' '.DB::table($t)->where($v[0],$v[1])->where($v[2],$id)->count().PHP_EOL;
$snap=DB::table('entity_reputation_snapshots')->where('entity_id',$entity)->orderBy('id')->get();echo 'TARGET_SNAPSHOTS '.$snap->toJson().PHP_EOL;
$out=DB::table('outbox_events')->where(fn($q)=>$q->where('aggregate_type','rating_event')->where('aggregate_id',$id))->orWhere(fn($q)=>$q->where('aggregate_type','entity_reputation_snapshot')->whereIn('aggregate_id',$snap->pluck('id')))->orderBy('id')->get(['id','aggregate_type','aggregate_id','event_type']);echo 'OUTBOX '.$out->toJson().PHP_EOL;
echo 'REPUTATION_RECEIPTS '.DB::table('reputation_event_receipts')->whereIn('outbox_event_id',$out->pluck('id'))->get()->toJson().PHP_EOL;
echo 'OTHER_RATING_IDS '.DB::table('rating_events')->where('id','!=',$id)->orderBy('id')->pluck('id')->toJson().PHP_EOL;
echo 'SECOND_RATING '.DB::table('rating_events')->where('target_entity_id','01m81mbmqvwr3d1nvfp9gkp3xf')->get(['id','target_entity_id','value','status'])->toJson().PHP_EOL;
echo 'GUARDS '.json_encode(DB::select("SELECT c.relname,t.tgname,t.tgenabled,p.proname,pg_get_userbyid(p.proowner) AS owner,md5(pg_get_functiondef(p.oid)) AS hash FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid JOIN pg_proc p ON p.oid=t.tgfoid WHERE NOT t.tgisinternal AND (c.relname LIKE '%rating%' OR c.relname LIKE '%reputation%') ORDER BY c.relname,t.tgname")).PHP_EOL;
foreach(['entities','entity_identifiers','entity_relationships','users','thing_images','entity_profile_revisions'] as $t)if(Schema::hasTable($t)){echo 'PROTECTED_HASH '.$t.' '.DB::table($t)->count().' '.hash('sha256',DB::table($t)->orderBy('id')->get()->toJson()).PHP_EOL;}
echo 'UNRELATED_RATINGS_HASH '.hash('sha256',DB::table('rating_events')->where('id','!=',$id)->orderBy('id')->get()->toJson()).PHP_EOL;
});
PHP
k -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- sh -c 'grep -R -n -E "Cache::|cache\\(|function.*(invalidate|forget)|reputation:" app/Services/*Reputation* app/Services/Rat* app/Services/UniversalEntityReadService.php | head -70'
echo SINGLE_RATING_DEPENDENCY_PREFLIGHT_COMPLETE_NO_WRITES
