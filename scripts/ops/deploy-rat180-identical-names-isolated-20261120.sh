#!/usr/bin/env bash
set -Eeuo pipefail
# PR180 guarded two-workload candidate rollout. Refuse until pinned immutable
# candidate digests are populated from exact-head GitHub publication receipts.
[[ $# -eq 3 ]] || { echo 'usage: <source-sha> <api-image@sha256> <rat-image@sha256>' >&2; exit 64; }
source_sha="$1"; api_candidate="$2"; rat_candidate="$3"
expected_source='061beefd23bd71cf5902ff2a7724ff78acb515f6'
expected_api='ghcr.io/daisycloversoftware/rum-api@sha256:7411cc02eb6568e6c6d228b551683294bfac49a6f89c320078af5550adce5d56'
expected_rat='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:9a33ea315f52d5e14a5ece0e57895aae8d4eb4d7d7e9955110d49fc9676c64d9'
[[ "$source_sha" == "$expected_source" ]] || exit 78
[[ "$api_candidate" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$rat_candidate" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$api_candidate" == "$expected_api" && "$rat_candidate" == "$expected_rat" ]] || { echo 'PR180_IMAGE_PROVENANCE_NOT_PINNED' >&2; exit 78; }
api_baseline='ghcr.io/daisycloversoftware/rum-api@sha256:56b4af9e34067b93a552c90d212f9c4b35a909507be56e606146bcca9c19e588'
rat_baseline='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:5435f2cc40608d5ba09007b0c356565670d796e288946ffc0d1d91671259a69f'
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
image(){ kctl -n "$1" get deployment "$2" -o "jsonpath={.spec.template.spec.containers[?(@.name==\"$3\")].image}"; }
hosts(){ kctl -n "$1" get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u; }
for ns in rum-dev rum-dev-isolated rum-rate-anything-preview;do kctl get namespace "$ns" >/dev/null;done
[[ "$(hosts rum-dev)" == 'rateurmate.online' ]] || exit 78
[[ "$(hosts rum-dev-isolated)" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ "$(hosts rum-rate-anything-preview)" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
api_before="$(image rum-dev-isolated rum-api php-fpm)"
rat_before="$(image rum-rate-anything-preview rum-rate-anything rate-anything)"
[[ "$api_before" == "$api_baseline" || "$api_before" == "$api_candidate" ]] || { echo "API_BASELINE_DRIFT=$api_before"; exit 78; }
[[ "$rat_before" == "$rat_baseline" || "$rat_before" == "$rat_candidate" ]] || { echo "RAT_BASELINE_DRIFT=$rat_before"; exit 78; }
protected_snapshot(){
 for ns in rum-dev rum-dev-isolated rum-rate-anything-preview;do
  kctl -n "$ns" get deployment,statefulset,cronjob,configmap,persistentvolumeclaim,ingress -o json |
  python3 -c '
import json,sys
rows=[]
for item in json.load(sys.stdin)["items"]:
 m=item["metadata"]
 if (m["namespace"],item["kind"],m["name"]) in [
  ("rum-dev-isolated","Deployment","rum-api"),
  ("rum-rate-anything-preview","Deployment","rum-rate-anything")]:continue
 rows.append({"ns":m["namespace"],"kind":item["kind"],"name":m["name"],"spec":item.get("spec"),"data":item.get("data")})
print(json.dumps(sorted(rows,key=lambda x:(x["ns"],x["kind"],x["name"])),sort_keys=True,separators=(",",":")))'
 done | sha256sum | cut -d' ' -f1
}
protected_before="$(protected_snapshot)"
changed_api=false; changed_rat=false
rollback(){
 rc=$?
 trap - ERR
 if [[ "$changed_rat" == true ]];then
  kctl -n rum-rate-anything-preview set image deployment/rum-rate-anything "rate-anything=$rat_before" >/dev/null || true
  kctl -n rum-rate-anything-preview rollout status deployment/rum-rate-anything --timeout=240s || true
 fi
 if [[ "$changed_api" == true ]];then
  kctl -n rum-dev-isolated set image deployment/rum-api "php-fpm=$api_before" >/dev/null || true
  kctl -n rum-dev-isolated rollout status deployment/rum-api --timeout=240s || true
 fi
 echo "RAT180_ISOLATED_ROLLOUT_FAILED exit=$rc api_rollback=$changed_api rat_rollback=$changed_rat" >&2
 exit "$rc"
}
trap rollback ERR
if [[ "$api_before" != "$api_candidate" ]];then
 changed_api=true
 kctl -n rum-dev-isolated set image deployment/rum-api "php-fpm=$api_candidate" >/dev/null
fi
kctl -n rum-dev-isolated rollout status deployment/rum-api --timeout=300s
[[ "$(image rum-dev-isolated rum-api php-fpm)" == "$api_candidate" ]] || { echo 'API_IMAGE_MISMATCH' >&2; false; }
api_health="$(curl -fsS --retry 5 --retry-delay 2 --max-time 25 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$api_health" "$source_sha" <<'PY'
import json,sys
a=json.loads(sys.argv[1])
assert a.get('status')=='ok' and a.get('service')=='rum-api'
assert sys.argv[2] in str(a.get('version',''))
PY
if [[ "$rat_before" != "$rat_candidate" ]];then
 changed_rat=true
 kctl -n rum-rate-anything-preview set image deployment/rum-rate-anything "rate-anything=$rat_candidate" >/dev/null
fi
kctl -n rum-rate-anything-preview rollout status deployment/rum-rate-anything --timeout=300s
[[ "$(image rum-rate-anything-preview rum-rate-anything rate-anything)" == "$rat_candidate" ]] || { echo 'RAT_IMAGE_MISMATCH' >&2; false; }
served="$(curl -fsS --retry 5 --retry-delay 2 --max-time 25 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')"
[[ "$served" == "rat-candidate+$source_sha" ]] || { echo 'RAT_SERVED_SOURCE_MISMATCH' >&2; false; }
protected_after="$(protected_snapshot)"
[[ "$protected_before" == "$protected_after" ]] || { echo 'PROTECTED_RESOURCE_DRIFT' >&2; false; }
trap - ERR
printf 'RAT180_ISOLATED_DEPLOYED=true\nsource=%s\napi=%s\nrat=%s\napi_before=%s\nrat_before=%s\nprotected_before=%s\nprotected_after=%s\nlive_untouched=true\nno_database_mutations=true\n' "$source_sha" "$api_candidate" "$rat_candidate" "$api_before" "$rat_before" "$protected_before" "$protected_after"
