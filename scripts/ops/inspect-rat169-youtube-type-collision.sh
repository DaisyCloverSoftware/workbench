#!/usr/bin/env bash
set -euo pipefail
# Fixed read-only canonical DEV duplicate/synonym inspection for the exact
# owner-requested type label. No identity/person records or secrets are read.
[[ $# -eq 0 ]] || exit 64
kctl() { if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get ingress -o 'jsonpath={.items[0].spec.rules[0].host}')" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
probe='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$n=app(App\Services\EntityNameNormalizer::class);
$term="youtube channel";
$classes=$db::table("classifications")->get(["id","key","canonical_name","status","entity_type_id"]);
$exact=$classes->filter(fn($c)=>$n->normalise((string)$c->canonical_name)===$term)->values();
$synonyms=$db::table("classification_synonyms")->where("normalised_term",$term)->get(["classification_id"]);
$matched=$classes->filter(fn($c)=>$synonyms->contains("classification_id",$c->id))->values();
$kindKeys=$db::table("entity_types")->pluck("key","id");
$result=[
 "target"=>"youtube channel",
 "canonical_matches"=>$exact->map(fn($c)=>["key"=>$c->key,"status"=>$c->status,"kind"=>$kindKeys[$c->entity_type_id] ?? null])->all(),
 "synonym_matches"=>$matched->map(fn($c)=>["key"=>$c->key,"status"=>$c->status,"kind"=>$kindKeys[$c->entity_type_id] ?? null])->all(),
 "total_classifications"=>$classes->count(),
];
echo "RAT169_YOUTUBE_EXISTING_CANONICAL_MATCHES=".json_encode($result,JSON_THROW_ON_ERROR)."\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$probe"
echo 'RAT169_YOUTUBE_COLLISION_INSPECTION_READ_ONLY=true'
