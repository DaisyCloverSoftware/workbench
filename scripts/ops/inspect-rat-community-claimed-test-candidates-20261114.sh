#!/usr/bin/env bash
set -euo pipefail
# READ ONLY: discover safe, explicitly DEV-labelled verified claimed Things.
# No claimant names, accounts, evidence, credentials, or private identities.
[[ $# -eq 0 ]] || exit 64
kctl() { if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
hosts="$(kctl -n rum-dev-isolated get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u)"
[[ "$hosts" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
probe='
require "vendor/autoload.php";
$app=require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$rows=$db::table("entities as e")
->join("entity_claims as c","c.entity_id","=","e.id")
->join("entity_types as t","t.id","=","e.entity_type_id")
->where("e.visibility","public")->where("e.publication_state","published")
->where("e.status","active")->whereNull("e.deleted_at")
->where("c.status","approved")->whereNull("c.revoked_at")
->where("c.verification_level",">=",2)
->whereNotIn("t.key",["person","persona","digital_identity"])
->where(function($q){$q->where("e.canonical_name","like","DEV %")->orWhere("e.canonical_name","like","RAT %")->orWhere("e.canonical_name","like","% QA %")->orWhere("e.canonical_name","like","% DEV %");})
->orderBy("e.canonical_name")->limit(20)
->get(["e.id","e.canonical_name","t.key as kind","e.edit_policy"]);
echo "RAT_COMMUNITY_CLAIMED_TEST_CANDIDATES=".json_encode([
 "canonicalDev"=>true,
 "verifiedClaimedPublicDevTestCount"=>$rows->count(),
 "testOnlyCandidates"=>$rows->map(fn($r)=>["id"=>$r->id,"name"=>$r->canonical_name,"kind"=>$r->kind,"policy"=>$r->edit_policy])->all(),
 "readOnly"=>true
],JSON_THROW_ON_ERROR)."\n";
'
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$probe"
echo RAT_CLAIMED_TEST_READ_ONLY=true
