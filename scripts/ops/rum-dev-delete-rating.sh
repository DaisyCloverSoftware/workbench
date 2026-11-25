#!/usr/bin/env bash
set -Eeuo pipefail
# Owner-authorised DEV maintenance only. No environment, host, namespace or SQL selector input.
[[ $# == 3 || $# == 5 ]] || exit 64
[[ $1 =~ ^(plan|apply|verify)$ && $2 =~ ^[0-9a-z]{26}$ && $3 =~ ^[0-9a-z]{26}$ ]] || exit 64
if [[ $1 == apply ]];then [[ $# == 5 && $4 =~ ^[a-f0-9]{64}$ && $5 =~ ^[a-z0-9-]{8,100}$ ]]||exit 64;else [[ $# == 3 ]]||exit 64;fi
k(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
# Deployment-bound identity, deliberately fail closed after infrastructure re-housing.
[[ $(k get ns rum-dev-isolated -o 'jsonpath={.metadata.uid}') == 6804e79b-3000-42c1-b37e-f07a57252425 ]]||exit 78
[[ $(k -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.metadata.uid}') == b2b84090-8585-4d4d-9dc1-0b4a4c5b3138 ]]||exit 78
[[ $(k -n rum-dev-isolated get svc rum-data-postgres -o 'jsonpath={.metadata.uid}') == b5b97fc9-9c44-44b0-b325-ac6973453abd ]]||exit 78
[[ $(k -n rum-dev-isolated get pod rum-data-postgres-0 -o 'jsonpath={.metadata.uid}:{.status.podIP}') == 3ef7dbbb-5610-4405-a56f-85618070df4e:10.42.1.177 ]]||exit 78
[[ $(k -n rum-dev-isolated get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}'|sort -u) == dev-rum.daisycloversoftware.uk ]]||exit 78
k -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- timeout 150s php /dev/stdin "$@" <<'PHP'
<?php
require 'vendor/autoload.php';$app=require 'bootstrap/app.php';$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
use Illuminate\Support\Facades\DB;use Illuminate\Support\Facades\Schema;
[$file,$mode,$entity,$id]=$argv;$expected=$argv[4]??null;$operation=$argv[5]??null;
if(PHP_SAPI!=='cli'||!in_array($mode,['plan','apply','verify'],true)||!preg_match('/^[0-9a-z]{26}$/',$entity)||!preg_match('/^[0-9a-z]{26}$/',$id))throw new RuntimeException('Invalid exact-ID maintenance invocation');
if(DB::getDriverName()!=='pgsql'||config('database.connections.pgsql.host')!=='rum-data-postgres'||config('database.connections.pgsql.database')!=='rum'||config('database.connections.pgsql.username')!=='rum_app')throw new RuntimeException('Not the authorised isolated DEV database configuration');
$identity=DB::selectOne('SELECT current_database() AS db,current_user AS role,inet_server_addr() AS server,inet_server_port() AS port');
if($identity->db!=='rum'||$identity->role!=='rum_app'||$identity->server!=='10.42.1.177'||(int)$identity->port!==5432)throw new RuntimeException('Actual PostgreSQL server is not the pinned isolated DEV instance');
function canonicalHash($rows):string{$a=array_map(static fn($r)=>json_encode($r,JSON_UNESCAPED_SLASHES|JSON_THROW_ON_ERROR),$rows->all());sort($a,SORT_STRING);return hash('sha256',implode("\n",$a));}
function selected(string $table,array $scope){return DB::table($table)->whereIn($scope[0],$scope[1]);}
function guardState():array{return array_map(static fn($r)=>(array)$r,DB::select("SELECT c.relname,t.tgname,t.tgenabled,p.proname,pg_get_userbyid(p.proowner) AS owner,md5(pg_get_functiondef(p.oid)) AS hash FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid JOIN pg_proc p ON p.oid=t.tgfoid WHERE NOT t.tgisinternal AND (c.relname LIKE '%rating%' OR c.relname LIKE '%reputation%') ORDER BY c.relname,t.tgname"));}
$result=DB::transaction(function()use($mode,$entity,$id,$expected,$operation,$identity){
 DB::statement($mode==='apply'?'SET TRANSACTION ISOLATION LEVEL SERIALIZABLE':'SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY');DB::statement("SET LOCAL lock_timeout='8s'");DB::statement("SET LOCAL statement_timeout='60s'");
 if($mode==='apply')DB::select("SELECT pg_advisory_xact_lock(hashtext('rum-dev-owner-rating-delete'))");
 $guards=guardState();foreach($guards as $g)if($g['tgenabled']!=='O')throw new RuntimeException('Evidence guard unexpectedly disabled');
 $known=['rum_rating_event_evidence_guard'=>'53fa59f03e3059cfda3f1d23da7b8053','rum_rating_review_evidence_guard'=>'6caf05605fa6d3f8664d32ea69da501e','rum_rating_event_revision_guard'=>'03d579a7aca171564e54c0b876f06cb2','rum_rating_dimension_value_guard'=>'1af6ea39d64e27ca81e044ceedc10b81'];
 foreach($known as $name=>$hash){$g=array_values(array_filter($guards,fn($g)=>$g['proname']===$name));if(count($g)!==1||$g[0]['hash']!==$hash||$g[0]['owner']!=='rum_app')throw new RuntimeException('Unknown evidence guard contract: '.$name);}
 $root=DB::table('rating_events')->where('id',$id);if($mode==='apply')$root->lockForUpdate();$event=$root->first();
 if($mode!=='verify'&&(!$event||$event->target_entity_id!==$entity))throw new RuntimeException('Exact event/entity pair missing or mismatched; no deletion');
 if($event&&$event->target_entity_id!==$entity)throw new RuntimeException('Wrong event identity');
 $reviewIds=DB::table('rating_reviews')->where('rating_event_id',$id)->pluck('id')->all();$responseIds=DB::table('rating_review_responses')->whereIn('rating_review_id',$reviewIds)->pluck('id')->all();
 $snapshotIds=DB::table('entity_reputation_snapshots')->where('entity_id',$entity)->pluck('id')->all();
 if(DB::table('rating_weight_snapshots')->where('rating_event_id',$id)->whereNotIn('reputation_snapshot_id',$snapshotIds)->exists())throw new RuntimeException('Target affects another entity projection; adapter needs explicit shared-projection handling');
 if(DB::table('rating_journey_entries')->where('parent_rating_event_id',$id)->where(fn($q)=>$q->whereNull('rating_event_id')->orWhere('rating_event_id','!=',$id))->exists())throw new RuntimeException('Shared parent journey dependency; refuse deleting another submission');
 if(DB::table('rating_journey_entries')->where('rating_event_id',$id)->where(fn($q)=>$q->where('entity_id','!=',$entity)->orWhereNotNull('relationship_id'))->exists())throw new RuntimeException('Journey row also preserves graph provenance; requires shared dependency adapter');
 if(DB::table('rating_evidence')->where('rating_event_id',$id)->whereNotNull('media_asset_id')->exists())throw new RuntimeException('Media-backed rating needs exclusive/shared asset dependency inventory');
 foreach(['profile_help_me_out_pins'=>['rating_kind','rating_event','rating_id'],'moderation_cases'=>['object_type','rating_event','object_id'],'moderation_actions'=>['target_type','rating_event','target_id']] as $table=>$v)if(DB::table($table)->where($v[0],$v[1])->where($v[2],$id)->exists())throw new RuntimeException('Uninventoried shared moderation/pin dependency: '.$table);
 $fk=DB::select("SELECT c.relname AS t,a.attname AS col FROM pg_constraint co JOIN pg_class c ON c.oid=co.conrelid JOIN pg_attribute a ON a.attrelid=c.oid AND a.attnum=co.conkey[1] WHERE co.contype='f' AND co.confrelid=to_regclass('rating_events') ORDER BY c.relname,a.attname");
 $allowed=['rating_challenges.rating_event_id','rating_dimension_values.rating_event_id','rating_event_admission_states.rating_event_id','rating_event_admission_votes.rating_event_id','rating_event_reasons.rating_event_id','rating_event_revisions.rating_event_id','rating_evidence.rating_event_id','rating_journey_entries.parent_rating_event_id','rating_journey_entries.rating_event_id','rating_reviews.rating_event_id','rating_weight_snapshots.rating_event_id'];$seen=array_map(fn($f)=>$f->t.'.'.$f->col,$fk);sort($allowed);sort($seen);if($seen!==$allowed)throw new RuntimeException('Rating dependency schema drift; inventory before deleting');
 $outboxIds=DB::table('outbox_events')->where(fn($q)=>$q->where('aggregate_type','rating_event')->where('aggregate_id',$id))->orWhere(fn($q)=>$q->where('aggregate_type','entity_reputation_snapshot')->whereIn('aggregate_id',$snapshotIds))->pluck('id')->all();
 $receiptIds=DB::table('reputation_event_receipts')->whereIn('outbox_event_id',$outboxIds)->orWhereIn('reputation_snapshot_id',$snapshotIds)->pluck('id')->all();
 $weightIds=DB::table('rating_weight_snapshots')->where('rating_event_id',$id)->orWhereIn('reputation_snapshot_id',$snapshotIds)->pluck('id')->all();
 $scopes=['reputation_event_receipts'=>['id',$receiptIds],'outbox_events'=>['id',$outboxIds],'rating_challenges'=>['rating_event_id',[$id]],'rating_review_response_revisions'=>['rating_review_response_id',$responseIds],'rating_review_responses'=>['id',$responseIds],'rating_review_revisions'=>['rating_review_id',$reviewIds],'rating_journey_entries'=>['rating_event_id',[$id]],'rating_weight_snapshots'=>['id',$weightIds],'entity_reputation_snapshots'=>['id',$snapshotIds],'rating_reason_votes'=>['rating_event_id',[$id]],'rating_dimension_values'=>['rating_event_id',[$id]],'rating_event_admission_votes'=>['rating_event_id',[$id]],'rating_event_admission_states'=>['rating_event_id',[$id]],'rating_event_reasons'=>['rating_event_id',[$id]],'rating_event_revisions'=>['rating_event_id',[$id]],'rating_evidence'=>['rating_event_id',[$id]],'rating_reviews'=>['rating_event_id',[$id]],'rating_events'=>['id',[$id]]];
 $inventory=[];foreach($scopes as $t=>$scope){$rows=selected($t,$scope)->get();$inventory[$t]=['count'=>$rows->count(),'ids'=>$rows->map(fn($r)=>$r->id??(($r->rating_event_id??'').(isset($r->rating_reason_definition_id)?':'.$r->rating_reason_definition_id:'')))->all(),'sha256'=>canonicalHash($rows)];}
 $fingerprint=hash('sha256',json_encode([$entity,$id,$inventory,$guards,$seen],JSON_THROW_ON_ERROR));
 $protected=[];foreach(['entities','entity_identifiers','entity_relationships','entity_profile_revisions','users','media_assets','rating_journeys'] as $t)if(Schema::hasTable($t))$protected[$t]=['count'=>DB::table($t)->count(),'sha256'=>canonicalHash(DB::table($t)->get())];
 foreach($scopes as $t=>$scope){if(in_array($t,['outbox_events','reputation_event_receipts'],true))continue;$rows=DB::table($t)->where(fn($q)=>$q->whereNull($scope[0])->orWhereNotIn($scope[0],$scope[1]))->get();$protected['unrelated_'.$t]=['count'=>$rows->count(),'sha256'=>canonicalHash($rows)];}
 $cacheKeys=[];if(config('cache.default')==='database'&&Schema::hasTable('cache'))$cacheKeys=DB::table('cache')->where('key','like','%'.$entity.'%')->orWhere('key','like','%'.$id.'%')->pluck('key')->all();
 if($mode==='verify'){
  foreach($inventory as $t=>$v)if($v['count']!==0)throw new RuntimeException('Residual event/dependent/projection rows: '.$t);
  return ['mode'=>'verify','database'=>$identity,'eventId'=>$id,'entityId'=>$entity,'remaining'=>$inventory,'protected'=>$protected,'guards'=>$guards,'cacheKeysRemaining'=>count($cacheKeys),'physicalAbsence'=>true];
 }
 $plan=['mode'=>$mode,'database'=>$identity,'eventId'=>$id,'entityId'=>$entity,'eventValue'=>(int)$event->value,'submittedAt'=>$event->submitted_at,'source'=>$event->source,'reviewIds'=>$reviewIds,'inventory'=>$inventory,'fingerprint'=>$fingerprint,'protected'=>$protected,'guards'=>$guards,'affectedCacheKeys'=>count($cacheKeys)];
 if($mode==='plan')return $plan;
 if(!hash_equals($fingerprint,(string)$expected))throw new RuntimeException('Deletion scope fingerprint changed; no write');
 $totalBefore=DB::table('rating_events')->count();$definitions=[];
 DB::statement('CREATE TEMPORARY TABLE rum_dev_delete_scope (rating_event_id varchar(26) PRIMARY KEY, entity_id varchar(26) NOT NULL) ON COMMIT DROP');DB::table('rum_dev_delete_scope')->insert(['rating_event_id'=>$id,'entity_id'=>$entity]);
 DB::selectOne("SELECT set_config('rum.dev_owner_delete_operation', ?, true)",[$operation]);
 foreach($known as $name=>$hash){
  $row=DB::selectOne('SELECT pg_get_functiondef(to_regprocedure(?)) AS definition',['public.'.$name.'()']);if(!$row||md5($row->definition)!==$hash)throw new RuntimeException('Guard changed before write');$definitions[$name]=$row->definition;
  $column=$name==='rum_rating_event_evidence_guard'?'id':'rating_event_id';
  $allow="IF TG_OP = 'DELETE' AND current_setting('rum.dev_owner_delete_operation', true) = '".$operation."' AND EXISTS (SELECT 1 FROM pg_temp.rum_dev_delete_scope s WHERE s.rating_event_id = OLD.".$column.($column==='id'?' AND s.entity_id = OLD.target_entity_id':'').") THEN RETURN OLD; END IF;";
  if(substr_count($row->definition,"BEGIN\n")!==1)throw new RuntimeException('Unrecognised trigger body');DB::unprepared(str_replace("BEGIN\n","BEGIN\n    ".$allow."\n",$row->definition));
 }
 $removed=[];foreach($scopes as $t=>$scope){$removed[$t]=selected($t,$scope)->delete();if($removed[$t]!==$inventory[$t]['count'])throw new RuntimeException('Delete count mismatch: '.$t);}
 foreach($definitions as $definition)DB::unprepared($definition);if(guardState()!==$guards)throw new RuntimeException('Immutability restoration mismatch; roll back everything');
 foreach($scopes as $t=>$scope)if(selected($t,$scope)->exists())throw new RuntimeException('Physical deletion incomplete: '.$t);
 foreach($protected as $key=>$v){if(str_starts_with($key,'unrelated_')){$t=substr($key,10);$scope=$scopes[$t];$rows=DB::table($t)->where(fn($q)=>$q->whereNull($scope[0])->orWhereNotIn($scope[0],$scope[1]))->get();}else{$rows=DB::table($key)->get();}if($rows->count()!==$v['count']||canonicalHash($rows)!==$v['sha256'])throw new RuntimeException('Protected records changed: '.$key);}
 // Only ID-addressed cache entries on this already-verified DEV connection. Never flush a shared cache.
 $cacheRemoved=0;if($cacheKeys)$cacheRemoved=DB::table('cache')->whereIn('key',$cacheKeys)->delete();
 $after=DB::table('rating_events')->count();if($after!==$totalBefore-1)throw new RuntimeException('Canonical total mismatch');
 return ['mode'=>'apply','operationId'=>$operation,'database'=>$identity,'deletedEventId'=>$id,'entityId'=>$entity,'planFingerprint'=>$fingerprint,'removedRows'=>$removed,'removedRowIds'=>array_map(fn($x)=>$x['ids'],$inventory),'ratingEventsBefore'=>$totalBefore,'ratingEventsAfter'=>$after,'protected'=>$protected,'guardsRestored'=>true,'guardDefinitions'=>$known,'cacheRowsInvalidated'=>$cacheRemoved,'canonicalReadServicesUseFreshQueries'=>true,'deletedApplicationContentRetained'=>false];
},1);
echo 'DEV_RATING_DELETE_RECEIPT '.json_encode($result,JSON_UNESCAPED_SLASHES|JSON_THROW_ON_ERROR).PHP_EOL;
PHP
