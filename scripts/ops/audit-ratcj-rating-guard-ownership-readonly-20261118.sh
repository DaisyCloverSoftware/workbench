#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only database immutability trigger ownership check before considering owner-approved scoped maintenance capability.
[[ $# -eq 0 ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == 'ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430' ]] || exit 78
php="$(cat <<'RATCJ_TRIGGER_READONLY_PHP'

require "vendor/autoload.php";$app=require "bootstrap/app.php";$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$db=Illuminate\Support\Facades\DB::class;
$who=$db::selectOne("SELECT current_user AS username,session_user AS session_name,current_database() AS database_name");
$names=["rum_rating_event_evidence_guard","rum_rating_review_evidence_guard","rum_rating_event_revision_guard","rum_rating_dimension_value_guard"];
$rows=$db::select("SELECT p.proname name,n.nspname schema_name,pg_get_userbyid(p.proowner) owner_name,
 md5(pg_get_functiondef(p.oid)) hash,has_function_privilege(current_user,p.oid,'EXECUTE') as can_execute,
 pg_has_role(current_user,p.proowner,'MEMBER') as is_owner_role,
 p.prosecdef as security_definer
 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE p.proname=ANY(?::text[]) ORDER BY p.proname",
 ['{'.implode(",",$names).'}']);
$triggers=$db::select("SELECT tgname,tgenabled,pg_get_triggerdef(oid) description FROM pg_trigger
 WHERE tgname LIKE 'rum_rating_%_guard_trigger' ORDER BY tgname");
echo "RATCJ_GUARD_OWNERSHIP=".json_encode(["identity"=>$who,"functions"=>$rows,"triggers"=>$triggers],JSON_UNESCAPED_SLASHES)."\n";

RATCJ_TRIGGER_READONLY_PHP
)"
kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- timeout 50s php -r "$php"
echo RATCJ_GUARD_OWNERSHIP_AUDIT_PASS=true
