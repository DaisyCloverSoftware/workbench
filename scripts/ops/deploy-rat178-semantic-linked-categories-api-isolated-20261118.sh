#!/usr/bin/env bash
set -Eeuo pipefail
# Guarded ONE-API-only PR178 promotion. Never touches LIVE/RAT frontend/data.
[[ $# -eq 2 ]] || exit 64
src="$1"; candidate="$2"
[[ "$src" == 'acebcc208f19348ab9eb01d05434b0e20a5e256c' ]] || exit 78
[[ "$candidate" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$candidate" == '__RAT178_API_IMAGE_PENDING__' ]] || exit 78
baseline='ghcr.io/daisycloversoftware/rum-api@sha256:c68f9e4902b7ab71abab2cf0ea50244bf80dd93809385add84b764ff1fc34d0c'
frontend='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:ec11bfec432ff10d508850a0a071b3af3787bff656d2decca90eed5951cd0003'
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
image(){ kctl -n "$1" get deployment "$2" -o "jsonpath={.spec.template.spec.containers[?(@.name==\"$3\")].image}"; }
hosts(){ kctl -n "$1" get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u; }
for ns in rum-dev rum-dev-isolated rum-rate-anything-preview;do kctl get namespace "$ns" >/dev/null;done
[[ "$(hosts rum-dev)" == 'rateurmate.online' ]] || exit 78
[[ "$(hosts rum-dev-isolated)" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ "$(hosts rum-rate-anything-preview)" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
[[ "$(image rum-rate-anything-preview rum-rate-anything rate-anything)" == "$frontend" ]] || exit 78
before="$(image rum-dev-isolated rum-api php-fpm)"
[[ "$before" == "$baseline" || "$before" == "$candidate" ]] || { echo "RAT178_API_BASELINE_DRIFT=$before"; exit 78; }
protected_snapshot(){
 for ns in rum-dev rum-dev-isolated rum-rate-anything-preview;do
  kctl -n "$ns" get deployment,statefulset,cronjob,configmap,persistentvolumeclaim,ingress -o json |
  python3 -c '
import json,sys
rows=[]
for x in json.load(sys.stdin)["items"]:
 m=x["metadata"]
 if m["namespace"]=="rum-dev-isolated" and x["kind"]=="Deployment" and m["name"]=="rum-api":continue
 rows.append({"ns":m["namespace"],"kind":x["kind"],"name":m["name"],"spec":x.get("spec"),"data":x.get("data")})
print(json.dumps(sorted(rows,key=lambda x:(x["ns"],x["kind"],x["name"])),sort_keys=True,separators=(",",":")))'
 done | sha256sum | cut -d' ' -f1
}
protected_before="$(protected_snapshot)"
changed=false
rollback(){
 rc=$?
 trap - ERR
 if [[ "$changed" == true ]];then
  kctl -n rum-dev-isolated set image deployment/rum-api "php-fpm=$before" >/dev/null || true
  kctl -n rum-dev-isolated rollout status deployment/rum-api --timeout=240s || true
 fi
 echo "RAT178_GUARDED_API_ROLLOUT_FAILED rollback_attempted=$changed" >&2
 exit "$rc"
}
trap rollback ERR
if [[ "$before" != "$candidate" ]];then
 changed=true
 kctl -n rum-dev-isolated set image deployment/rum-api "php-fpm=$candidate" >/dev/null
fi
kctl -n rum-dev-isolated rollout status deployment/rum-api --timeout=300s
[[ "$(image rum-dev-isolated rum-api php-fpm)" == "$candidate" ]]
[[ "$(image rum-rate-anything-preview rum-rate-anything rate-anything)" == "$frontend" ]]
health="$(curl -fsS --retry 6 --retry-delay 2 --max-time 25 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$health" "$src" <<'PY'
import json,sys
health=json.loads(sys.argv[1])
assert health.get('status')=='ok' and health.get('service')=='rum-api'
assert sys.argv[2] in str(health.get('version',''))
PY
[[ "$(curl -fsS --retry 6 --retry-delay 2 --max-time 25 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == 'rat-candidate+887b5e02f4be6ec7e52371666effa9eb47892d6e' ]]
node="$(curl -fsS --retry 6 --retry-delay 2 --max-time 25 -H 'Origin: https://dev-rum-ra.daisycloversoftware.uk' -H 'Accept: application/json' https://dev-rum.daisycloversoftware.uk/api/v1/public/entities/rat-catalogue/nodes/01m7bswfh0a3evf271ye4taaf0)"
python3 - "$node" <<'PY'
import json,sys
b=json.loads(sys.argv[1]);a=b['data']['active']
src='01m7bswfh0a3evf271ye4taaf0';linked='01m7jss48cpff8c59j6mdbbb7f';rel='01m7jswzw09b94n00am5003dej'
assert b['meta']['privacy']=='private-rum-people-excluded'
assert a['id']==src and a['name']=='The Professional Idiots' and a['entityType']=='digital_resource'
assert any(c['primary'] and c['name']=='YouTube Channel' for c in a['classifications'])
channels=[c for c in b['data']['categories'] if c['key']=='channels']
assert len(channels)==1 and channels[0]['label']=='Channels'
items=[i for c in channels for i in c['items'] if i['id']==linked]
assert len(items)==1 and items[0]['relationship']['id']==rel
assert any(c['primary'] and c['name']=='YouTube Channel' for c in items[0]['classifications'])
assert not any(i['id']==linked for c in b['data']['categories'] if c['key']=='projects-tools' for i in c['items'])
assert any(i['id']==linked and i['relationship']['id']==rel for i in b['data']['topLinked'])
print('RAT178_CANONICAL_CHANNEL_CATEGORY_API_PASS=true')
PY
protected_after="$(protected_snapshot)"
[[ "$protected_after" == "$protected_before" ]] || { echo "Protected resource changed" >&2; exit 78; }
trap - ERR
printf 'RAT178_LINKED_CHANNEL_API_DEPLOYED\nsource_sha=%s\napi_image=%s\nprevious_api=%s\nrat_frontend_preserved=%s\nprotected_before=%s\nprotected_after=%s\nno_database_reset_or_reseed=true\nlive_unchanged=true\n' "$src" "$candidate" "$before" "$frontend" "$protected_before" "$protected_after"
