#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]] || exit 64
ops=$(cd "$(dirname "$0")" && pwd); t=$(mktemp -d); umask 077; trap 'rm -rf "$t"' EXIT
sha=2359178af19fdd9b5b45d103c892cdc36c2ad086
k(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
gh api repos/DaisyCloverSoftware/rum/pulls/182 | python3 -c 'import json,sys;p=json.load(sys.stdin);assert p["head"]["sha"]==sys.argv[1] and p["draft"] and not p["merged"];print("APPROVED_BASE",p["head"]["sha"])' "$sha"
gh api 'repos/DaisyCloverSoftware/rum/pulls?state=open&per_page=40' | python3 -c 'import json,sys;print("OPEN_PRS",json.dumps([{k:p[k] for k in ("number","title","draft")} | {"head":p["head"]["sha"],"branch":p["head"]["ref"]} for p in json.load(sys.stdin) if p["number"]>=180]))'
bash "$ops/check-rat181-protected-baseline.sh"
for repo in rum workbench-relay-private;do mkdir "$t/$repo";git -C "$t/$repo" init -q;git -C "$t/$repo" remote add origin "https://github.com/DaisyCloverSoftware/$repo.git";ref=main;[[ $repo != rum ]]||ref=$sha;git -C "$t/$repo" fetch -q --depth=1 origin "$ref";git -C "$t/$repo" checkout -q --detach FETCH_HEAD;done
python3 - "$t" "$ops" <<'PY'
from pathlib import Path
import json,subprocess,sys,re
root=Path(sys.argv[1]);ops=Path(sys.argv[2]);repo=root/'rum'
for p in repo.rglob('*'):
 if p.is_file() and '.git' not in p.parts and re.search(r'purge|immutab',str(p),re.I):print('PURGE_SOURCE',str(p.relative_to(repo)))
for p in ops.glob('*'):
 if re.search(r'purge|cj',p.name,re.I): print('PRIOR_OP',p.name)
for p in sorted((root/'workbench-relay-private/relay/control-outbox').glob('*')):
 if re.search(r'(ratcj.*(purge|delete|audit)|rat.*20261125.*(thumbnail|rating))',p.name,re.I):
  d=json.loads(p.read_text());op=d.get('result',{}).get('operation_script',{});print('PRIOR_RECEIPT',p.name,d.get('status'),op.get('path'),op.get('commit'));print(str(op.get('output',d.get('result',{})))[-1800:])
for f in ['apps/rate-anything/src/rat-workspace.css','apps/rate-anything/src/rat-explorer.css','apps/rate-anything/package.json']:
 p=repo/f
 if p.exists():
  s=p.read_text();
  if f.endswith('package.json'):print('PACKAGE',s)
  else:
   for m in re.finditer(r'[^{}]*rat-workspace-connection__mark[^{}]*\{[^}]*\}',s):print('MARK_CSS',m.group(0))
for f in ['apps/api/app/Services/RatExplorerService.php','apps/api/app/Http/Resources/UniversalEntityResource.php']:
 p=repo/f;s=p.read_text().splitlines()
 for i,l in enumerate(s):
  if 'images' in l or 'ThingImageService' in l:print('IMAGE_CONTRACT',f,i+1,'\n'.join(s[max(0,i-2):i+4]))
PY
k -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php <<'PHP'
<?php
require 'vendor/autoload.php';$app=require 'bootstrap/app.php';$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
use Illuminate\Support\Facades\DB;use Illuminate\Support\Facades\Schema;
DB::transaction(function(){
 DB::statement('SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY');
 echo 'DATABASE_IDENTITY '.json_encode(['driver'=>DB::getDriverName(),'host'=>config('database.connections.pgsql.host'),'database'=>DB::selectOne('SELECT current_database() AS name')->name]).PHP_EOL;
 $target='01m7x6ez57zpsgbv486y9t0y9h';$events=DB::table('rating_events')->where('target_entity_id',$target)->orderBy('id')->get();echo 'TARGET_EVENTS '.$events->toJson().PHP_EOL;$ids=$events->pluck('id')->all();
 foreach(['rating_reviews','rating_event_reasons','rating_dimension_values','rating_evidence','rating_event_admission_states','rating_event_admission_votes','rating_event_revisions','rating_journey_entries'] as $table){if(Schema::hasTable($table)&&Schema::hasColumn($table,'rating_event_id'))echo 'TARGET_DEPENDENTS '.$table.' '.DB::table($table)->whereIn('rating_event_id',$ids)->get()->toJson().PHP_EOL;}
 echo 'RATING_FKS '.json_encode(DB::select("SELECT conrelid::regclass::text AS child, confrelid::regclass::text AS parent, conname, pg_get_constraintdef(oid) AS definition FROM pg_constraint WHERE contype='f' AND (conrelid::regclass::text LIKE '%rating%' OR confrelid::regclass::text LIKE '%rating%' OR conrelid::regclass::text LIKE '%reputation%') ORDER BY child,conname")).PHP_EOL;
 echo 'RATING_TRIGGERS '.json_encode(DB::select("SELECT c.relname AS tablename,t.tgname,t.tgenabled,pg_get_triggerdef(t.oid) AS definition,pg_get_functiondef(t.tgfoid) AS function FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid WHERE NOT t.tgisinternal AND (c.relname LIKE '%rating%' OR c.relname LIKE '%reputation%') ORDER BY c.relname,t.tgname")).PHP_EOL;
 echo 'PROJECTION_TABLES '.json_encode(DB::select("SELECT table_name,column_name FROM information_schema.columns WHERE table_schema='public' AND (table_name LIKE '%reputation%' OR table_name LIKE '%projection%' OR table_name LIKE '%aggregate%' OR table_name LIKE '%review%') ORDER BY table_name,ordinal_position")).PHP_EOL;
});
PHP
for id in 01m0mw9hkbynssr7kbc0gk534j 01m7x6ez57zpsgbv486y9t0y9h 01m81mbmqvwr3d1nvfp9gkp3xf;do curl -fsS --retry 2 --max-time 25 "https://dev-rum.daisycloversoftware.uk/api/v1/public/entities/rat-catalogue/nodes/$id" | python3 -c 'import json,sys;n=json.load(sys.stdin)["data"];print("PUBLIC_IMAGE_GRAPH",json.dumps({"active":{k:n["active"].get(k) for k in ["id","images"]},"direct":[{k:i.get(k) for k in ["id","images"]} for c in n["categories"] for i in c["items"]],"indirect":[{k:i.get(k) for k in ["id","images"]} for g in n.get("relatedThrough",[]) for i in g["items"]]}))';done
echo READONLY_THUMBNAIL_AND_RATING_PREFLIGHT_COMPLETE
