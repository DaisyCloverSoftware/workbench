#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 0 ]] || exit 64
k(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ $(k -n rum-dev-isolated get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u) == dev-rum.daisycloversoftware.uk ]] || exit 78
k -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php <<'PHP'
<?php
use App\Models\Entity;use App\Models\EntityRelationship;use App\Services\RumEntityAudiencePolicy;use Illuminate\Support\Facades\DB;
require 'vendor/autoload.php';$app=require 'bootstrap/app.php';$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$p='01m0mw9hkbynssr7kbc0gk534j';$m='01m7x6ez57zpsgbv486y9t0y9h';$c='01m81mbmqvwr3d1nvfp9gkp3xf';$policy=app(RumEntityAudiencePolicy::class);$evidence=[];
foreach ([[$p,true,'4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945'],[$m,false,'c0d71c0cdeb103562b52798f03fe3ae433c42a7663d121137b30cce98805bb4d'],[$c,false,'ca91bf9718ce128f21a1d468137211cfd14e5ab2a8804fd364faea67510b9428']] as [$id,$allowed,$hash]){$row=Entity::query()->findOrFail($id);$got=$policy->allows($row);$events=DB::table('rating_events')->where('target_entity_id',$id)->orderBy('id')->pluck('id');if($got!==$allowed||hash('sha256',$events->toJson())!==$hash)throw new RuntimeException('Audience or original rating set mismatch for '.$id);$evidence[]=['id'=>$id,'name'=>$row->canonical_name,'rumAllowed'=>$got,'ratingEvents'=>$events->count()];}
$expected=['01m7x7kn332fqjxyj9b78zrmh0'=>$m,'01m81n8fsgxf1p52bw62e9rzyp'=>$c];foreach($expected as $id=>$other){$r=EntityRelationship::query()->findOrFail($id);$pair=[$r->source_entity_id,$r->target_entity_id];if(!in_array($p,$pair,true)||!in_array($other,$pair,true)||$r->visibility!=='public'||$r->verification_state!=='unverified'||$r->valid_to!==null)throw new RuntimeException('Edge mismatch');}echo 'DEV_PERSISTED_AUDIT_PASS='.json_encode(['entities'=>$evidence,'twoPublicUnverifiedEdges'=>$expected,'originalRatingIdSetsUnchanged'=>true]).PHP_EOL;
PHP