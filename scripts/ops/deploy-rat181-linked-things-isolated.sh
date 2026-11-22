#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 3 && $1 =~ ^[a-f0-9]{40}$ && $2 =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[a-f0-9]{64}$ && $3 =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[a-f0-9]{64}$ ]] || exit 64
sha=$1; api=$2; rat=$3; readonly baseAPI=ghcr.io/daisycloversoftware/rum-api@sha256:7411cc02eb6568e6c6d228b551683294bfac49a6f89c320078af5550adce5d56 baseRAT=ghcr.io/daisycloversoftware/rum-rate-anything@sha256:9a33ea315f52d5e14a5ece0e57895aae8d4eb4d7d7e9955110d49fc9676c64d9
[[ $(git ls-remote https://github.com/DaisyCloverSoftware/rum.git refs/heads/fix/rat-persist-explicit-linked-things-20261122 | cut -f1) == "$sha" && $api != "$baseAPI" && $rat != "$baseRAT" ]] || exit 78
k(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
for n in rum-dev rum-dev-isolated rum-rate-anything-preview;do case $n in rum-dev) expected=rateurmate.online;; rum-dev-isolated) expected=dev-rum.daisycloversoftware.uk;; *) expected=dev-rum-ra.daisycloversoftware.uk;;esac; [[ $(k -n "$n" get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}'|sort -u) == "$expected" ]] || exit 78;done
image(){ k -n "$1" get deployment "$2" -o "jsonpath={.spec.template.spec.containers[?(@.name==\"$3\")].image}"; }
oldAPI=$(image rum-dev-isolated rum-api php-fpm);oldRAT=$(image rum-rate-anything-preview rum-rate-anything rate-anything)
[[ $oldAPI == "$baseAPI" || $oldAPI == "$api" ]] && [[ $oldRAT == "$baseRAT" || $oldRAT == "$rat" ]] || { echo 'BASELINE_DRIFT';exit 78; }
snapshot(){ for n in rum-dev rum-dev-isolated rum-rate-anything-preview;do k -n "$n" get deployment,statefulset,pvc,cronjob,ingress,configmap -o json | python3 -c 'import json,sys
rows=[]
for x in json.load(sys.stdin)["items"]:
 m=x["metadata"];key=(m["namespace"],x["kind"],m["name"]);spec=x.get("spec"); allowed={("rum-dev-isolated","Deployment","rum-api"):"php-fpm",("rum-rate-anything-preview","Deployment","rum-rate-anything"):"rate-anything"}.get(key)
 if allowed:
  for c in spec["template"]["spec"]["containers"]:
   if c["name"]==allowed:c["image"]="<only-approved-image-field>"
 rows.append((key,spec,x.get("data")))
print(json.dumps(sorted(rows,key=lambda x:x[0]),sort_keys=True,separators=(",",":")))';done|sha256sum|cut -d' ' -f1; }
pods(){ local selector;selector=$(k -n "$1" get deploy "$2" -o json|python3 -c 'import json,sys;print(",".join(k+"="+v for k,v in json.load(sys.stdin)["spec"]["selector"]["matchLabels"].items()))');k -n "$1" get pods -l "$selector" -o json | python3 -c 'import json,sys
ps=[p for p in json.load(sys.stdin)["items"] if not p["metadata"].get("deletionTimestamp")];assert ps
for p in ps:
 c=next(c for c in p["status"]["containerStatuses"] if c["name"]==sys.argv[1]); assert c["ready"] and c["imageID"].endswith(sys.argv[2].split("@")[1]),c;print("READY_POD",p["metadata"]["name"],c["imageID"])' "$3" "$4"; }
s0=$(snapshot); echo "BEFORE protected=$s0 api=$oldAPI rat=$oldRAT"; changedAPI=0; changedRAT=0
rollback(){ local rc=$?; trap - ERR; set +e; if [[ $changedRAT == 1 && $(image rum-rate-anything-preview rum-rate-anything rate-anything) == "$rat" ]];then k -n rum-rate-anything-preview set image deployment/rum-rate-anything "rate-anything=$oldRAT"; k -n rum-rate-anything-preview rollout status deployment/rum-rate-anything --timeout=240s;fi; if [[ $changedAPI == 1 && $(image rum-dev-isolated rum-api php-fpm) == "$api" ]];then k -n rum-dev-isolated set image deployment/rum-api "php-fpm=$oldAPI";k -n rum-dev-isolated rollout status deployment/rum-api --timeout=240s;fi; echo "FAILED rc=$rc";exit "$rc"; }
trap rollback ERR
if [[ $oldAPI != "$api" ]];then changedAPI=1;k -n rum-dev-isolated set image deployment/rum-api "php-fpm=$api";fi
k -n rum-dev-isolated rollout status deployment/rum-api --timeout=300s;pods rum-dev-isolated rum-api php-fpm "$api";curl -fsS --retry 5 --max-time 25 https://dev-rum.daisycloversoftware.uk/api/v1/health|python3 -c 'import json,sys;d=json.load(sys.stdin);assert d.get("status")=="ok" and d.get("service")=="rum-api" and sys.argv[1] in str(d.get("version"));print("API_HEALTH",d["version"])' "$sha"
if [[ $oldRAT != "$rat" ]];then changedRAT=1;k -n rum-rate-anything-preview set image deployment/rum-rate-anything "rate-anything=$rat";fi
k -n rum-rate-anything-preview rollout status deployment/rum-rate-anything --timeout=300s;pods rum-rate-anything-preview rum-rate-anything rate-anything "$rat";served=$(curl -fsS --retry 4 --max-time 25 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION|tr -d '\r\n'); [[ $served == "rat-candidate+$sha" ]]; [[ $(curl -fsS --retry 3 --max-time 25 https://dev-rum-ra.daisycloversoftware.uk/healthz) == 'ok' ]];s1=$(snapshot);[[ "$s0" == "$s1" ]];trap - ERR;printf 'PR181_DEPLOYED sha=%s api=%s rat=%s served=%s protected_before=%s protected_after=%s\n' "$sha" "$api" "$rat" "$served" "$s0" "$s1"
