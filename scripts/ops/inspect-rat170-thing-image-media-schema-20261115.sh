#!/usr/bin/env bash
set -euo pipefail
# Read-only production-safe preflight, executed ONLY on canonical isolated DEV.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
host="$(kctl -n rum-dev-isolated get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)"
[[ "$host" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
php='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$c=$db::select("SELECT conname,pg_get_constraintdef(oid) AS definition FROM pg_constraint WHERE conrelid = '\''media_assets'\''::regclass AND conname IN ('\''media_assets_purpose_allowed'\'','\''media_assets_attachment_type_allowed'\'') ORDER BY conname");
if(count($c)!==2)exit(78);
$one=$db::table("entities")->where("id","01m7bswfh0a3evf271ye4taaf0")->first(["canonical_name"]);
if(!$one||$one->canonical_name!=="The Professional Idiots")exit(78);
$rows=$db::table("entity_profile_revisions")->whereIn("id",["01m7dxvjchbmnjjnatw0av6a7m","01m7dxwv5cnczfsjset8emjvap"])->orderBy("id")->get(["id","proposed_value","outcome","moderation_status"]);
if(count($rows)!==2)exit(78);
$owner=[];
foreach($rows as $r)$owner[]=["id"=>$r->id,"hash12"=>substr(hash("sha256",(string)$r->proposed_value),0,12),"outcome"=>$r->outcome,"moderation"=>$r->moderation_status];
echo "RAT170_ISOLATED_MEDIA_SCHEMA_READONLY=".json_encode(["constraints"=>$c,"mediaAssetsCount"=>$db::table("media_assets")->count(),"namedMigrationRan"=>$db::table("migrations")->where("migration","2026_11_15_000001_allow_thing_image_media_purpose")->exists(),"protectedOwnerRevisionSnapshot"=>$owner,"noMutation"=>true],JSON_THROW_ON_ERROR)."\n";
';
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$php"
echo 'RAT170_THING_IMAGE_MEDIA_PREFLIGHT_READONLY=true'
