#!/usr/bin/env bash
set -Eeuo pipefail
# Guarded immutable PR179 API + RAT-only promotion. Database, worker, RUM web, and LIVE are untouched.
[[ $# -eq 3 ]] || exit 64
src="$1"; api="$2"; rat="$3"
[[ "$src" =~ ^[0-9a-f]{40}$ ]] || exit 64
[[ "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
old_api='ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430'
old_rat='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:ec11bfec432ff10d508850a0a071b3af3787bff656d2decca90eed5951cd0003'
dev='rum-dev-isolated'; preview='rum-rate-anything-preview'; live='rum-dev'
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
image(){ kctl -n "$1" get deployment "$2" -o "jsonpath={.spec.template.spec.containers[?(@.name==\"$3\")].image}"; }
hosts(){ kctl -n "$1" get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u; }
for ns in "$dev" "$preview" "$live"; do kctl get namespace "$ns" >/dev/null; done
[[ "$(hosts "$dev")" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ "$(hosts "$preview")" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
[[ "$(hosts "$live")" == 'rateurmate.online' ]] || exit 78
api_before="$(image "$dev" rum-api php-fpm)"
rat_before="$(image "$preview" rum-rate-anything rate-anything)"
[[ "$api_before" == "$old_api" || "$api_before" == "$api" ]] || { echo "Unexpected canonical API baseline: $api_before"; exit 78; }
[[ "$rat_before" == "$old_rat" || "$rat_before" == "$rat" ]] || { echo "Unexpected RAT baseline: $rat_before"; exit 78; }
protected_snapshot(){
 for ns in "$dev" "$preview" "$live";do
  kctl -n "$ns" get deployment,statefulset,cronjob,configmap,persistentvolumeclaim,ingress -o json |
   python3 -c '
import json,sys
items=[]
for x in json.load(sys.stdin)["items"]:
 m=x["metadata"]
 if (m["namespace"],x["kind"],m["name"]) in [
   ("rum-dev-isolated","Deployment","rum-api"),
   ("rum-rate-anything-preview","Deployment","rum-rate-anything")
 ]:continue
 items.append({"namespace":m["namespace"],"kind":x["kind"],"name":m["name"],"spec":x.get("spec"),"data":x.get("data")})
print(json.dumps(sorted(items,key=lambda x:(x["namespace"],x["kind"],x["name"])),sort_keys=True,separators=(",",":")))'
 done | sha256sum | cut -d' ' -f1
}
protected_before="$(protected_snapshot)"
changed_api=false;changed_rat=false
rollback(){
 rc=$?;trap - ERR
 if [[ "$changed_rat" == true ]];then
  kctl -n "$preview" set image deployment/rum-rate-anything "rate-anything=$rat_before" >/dev/null || true
  kctl -n "$preview" rollout status deployment/rum-rate-anything --timeout=240s || true
 fi
 if [[ "$changed_api" == true ]];then
  kctl -n "$dev" set image deployment/rum-api "php-fpm=$api_before" >/dev/null || true
  kctl -n "$dev" rollout status deployment/rum-api --timeout=240s || true
 fi
 echo "RATCJ_RELEASE_ROLLBACK attempted api=$changed_api rat=$changed_rat" >&2
 exit "$rc"
}
trap rollback ERR
if [[ "$api_before" != "$api" ]];then
 changed_api=true
 kctl -n "$dev" set image deployment/rum-api "php-fpm=$api" >/dev/null
fi
kctl -n "$dev" rollout status deployment/rum-api --timeout=300s
[[ "$(image "$dev" rum-api php-fpm)" == "$api" ]]
health="$(curl -fsS --retry 5 --retry-delay 2 --max-time 25 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$health" "$src" <<'PY'
import json,sys
h=json.loads(sys.argv[1])
assert h.get("status")=="ok" and h.get("service")=="rum-api"
assert sys.argv[2] in str(h.get("version",""))
PY
ratings="$(curl -fsS --retry 5 --retry-delay 2 --max-time 25 -H 'Origin: https://dev-rum-ra.daisycloversoftware.uk' -H 'Cache-Control: no-cache' "https://dev-rum.daisycloversoftware.uk/api/v1/public/entities/rat-catalogue/nodes/01m07d2f9fj05f4ypdbwp087qj/ratings")"
python3 - "$ratings" <<'PY'
import json,sys
r=json.loads(sys.argv[1])
m=r['meta'];d=r['data']; c=d['counts'];a=d['items']
assert m['source']=='rum-dev-canonical' and m['privacy']=='private-rum-people-excluded'
assert c=={'publicSubmitted':1,'accepted':0,'awaitingVerdict':1,'inReview':0},c
assert len(a)==1 and a[0]['id']=='01m0mvrxv8c3s12q0w7gwmzt03'
assert a[0]['value']==5 and a[0]['verdict']=={'state':'needs_verdict','keepVotes':0,'sinBinVotes':0,'score':0,'countsTowardAccepted':False}
assert a[0]['review']=="I have been watching CJ's progress over the years and he is starting to make some great content."
print("RATCJ_CANONICAL_PUBLIC_RATINGS_API_PASS=true")
PY
node="$(curl -fsS --retry 5 --retry-delay 2 --max-time 25 "https://dev-rum.daisycloversoftware.uk/api/v1/public/entities/rat-catalogue/nodes/01m07d2f9fj05f4ypdbwp087qj")"
python3 - "$node" <<'PY'
import json,sys
n=json.loads(sys.argv[1])['data']
a=n['active']
assert a['id']=='01m07d2f9fj05f4ypdbwp087qj' and a['name']=='CJ Investigates'
assert a['reputation'] is None and n['topLinked']==[] and n['categories']==[]
assert any(c['primary'] and c['name']=='YouTube Channel' for c in a['classifications'])
assert a['images']['profile'] is not None and a['images']['cover'] is not None
print('RATCJ_PENDING_PUBLIC_REPUTATION_NOT_COUNTED_PASS=true')
PY
if [[ "$rat_before" != "$rat" ]];then
 changed_rat=true
 kctl -n "$preview" set image deployment/rum-rate-anything "rate-anything=$rat" >/dev/null
fi
kctl -n "$preview" rollout status deployment/rum-rate-anything --timeout=300s
[[ "$(image "$preview" rum-rate-anything rate-anything)" == "$rat" ]]
version="$(curl -fsS --retry 5 --retry-delay 2 --max-time 25 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')"
[[ "$version" == "rat-candidate+$src" ]]
[[ "$(image "$dev" rum-api php-fpm)" == "$api" ]]
[[ "$(protected_snapshot)" == "$protected_before" ]] || { echo "Protected resources changed"; exit 78; }
trap - ERR
printf 'RATCJ_GUARDED_ISOLATED_RELEASE_PASS=true\nsource=%s\napi=%s\nrat=%s\nprior_api=%s\nprior_rat=%s\nprotected_hash=%s\nno_database_reset_or_reseed=true\nlive_untouched=true\n' "$src" "$api" "$rat" "$api_before" "$rat_before" "$protected_before"
