#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]]||exit 64
sha=349d8178c4da2daa82434a3e3c37dad38c5aebeb;run=45614394445;repo=DaisyCloverSoftware/rum
base=2359178af19fdd9b5b45d103c892cdc36c2ad086
api=ghcr.io/daisycloversoftware/rum-api@sha256:5e8486292d3f1b0ab0b834cd98028892ccf73a600548234d737ea7402e60d336
old=ghcr.io/daisycloversoftware/rum-rate-anything@sha256:ba60b264d34730fb32a8f20a0ebbd26030dbff5917ad2de1407471f904cb091f
ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
exec 9>"${TMPDIR:-/tmp}/rat183-thumbnails-release.lock";flock -n 9||exit 75
k(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
gh api "repos/$repo/pulls/183" >"$t/pr";gh api "repos/$repo/actions/runs/$run" >"$t/run";gh api "repos/$repo/actions/runs/$run/jobs?per_page=100" >"$t/jobs"
python3 - "$t" "$sha" "$base" <<'PY'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);pr=json.loads((p/'pr').read_text());run=json.loads((p/'run').read_text());jobs=json.loads((p/'jobs').read_text())['jobs']
assert pr['head']['sha']==sys.argv[2] and pr['base']['sha']==sys.argv[3] and pr['draft'] and pr['state']=='open' and not pr['merged'];assert run['head_sha']==sys.argv[2] and run['conclusion']=='success' and run['status']=='completed';assert len(jobs)==9 and all(j['conclusion']=='success' for j in jobs);print('EXACT_HEAD_NINE_JOBS_GREEN',run['id'],run['run_attempt'],sys.argv[2])
for c in ('api','rate-anything'):(p/(c+'.job')).write_text(str(next(j for j in jobs if j['name']==f'RAT candidate image ({c})')['id']))
(p/'validation.job').write_text(str(next(j for j in jobs if j['name']=='api')['id']))
PY
for c in api rate-anything;do gh api "repos/$repo/actions/jobs/$(cat "$t/$c.job")/logs" >"$t/log";python3 - "$t" "$c" "$sha" <<'PY'
from pathlib import Path
import re,sys
p=Path(sys.argv[1]);c,sha=sys.argv[2:];s=(p/'log').read_text();assert re.search(r'CANDIDATE_SOURCE_SHA='+sha+r'\s',s);v=set(re.findall(r'CANDIDATE_IMAGE=(ghcr\.io/daisycloversoftware/rum-'+c+r'@sha256:[a-f0-9]{64})\s',s));assert len(v)==1;image=v.pop();(p/c).write_text(image);print('PUBLISHED',c,(p/(c+'.job')).read_text(),sha,image)
PY
done
gh api "repos/$repo/actions/jobs/$(cat "$t/validation.job")/logs" >"$t/validation"
python3 - "$t/validation" <<'PY'
import re,sys
s=re.sub(r'\x1b\[[0-9;]*m','',open(sys.argv[1]).read());assert '[OK] No errors' in s;print('PHPSTAN_ZERO_ERRORS');a=re.findall(r'OK \(\d+ tests, \d+ assertions\)',s);assert a;print('PHPUNIT',a[-1]);print('\n'.join(x for x in s.splitlines() if 'PASS' in x and 'files' in x))
PY
rat=$(cat "$t/rate-anything")
mkdir "$t/source";git -C "$t/source" init -q;git -C "$t/source" remote add origin "https://github.com/$repo.git";git -C "$t/source" fetch -q --depth=2 origin "$sha";git -C "$t/source" checkout -q --detach FETCH_HEAD
git -C "$t/source" diff --exit-code "$base" "$sha" -- apps/api apps/web infra apps/rate-anything/public/brand apps/rate-anything/src/RatBrand.tsx apps/rate-anything/src/brandAssets.ts apps/rate-anything/src/RumThumbSelector.tsx
for n in rum-dev rum-dev-isolated rum-rate-anything-preview;do case $n in rum-dev) h=rateurmate.online;;rum-dev-isolated) h=dev-rum.daisycloversoftware.uk;;*)h=dev-rum-ra.daisycloversoftware.uk;;esac;[[ $(k -n "$n" get ingress -o 'jsonpath={range .items[*].spec.rules[*]}{.host}{"\n"}{end}'|sort -u) == "$h" ]]||exit 78;done
[[ $(k -n rum-dev-isolated get deploy rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}') == "$api" ]]||exit 78
current=$(k -n rum-rate-anything-preview get deploy rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')
[[ $current == "$old" || $current == "$rat" ]]||exit 78
snapshot(){ for n in rum-dev rum-dev-isolated rum-rate-anything-preview;do k -n "$n" get deployment,statefulset,pvc,cronjob,ingress,configmap -o json|python3 -c 'import json,sys
rows=[]
for x in json.load(sys.stdin)["items"]:
 m=x["metadata"];key=(m["namespace"],x["kind"],m["name"]);s=x.get("spec")
 if key==("rum-rate-anything-preview","Deployment","rum-rate-anything"):
  for c in s["template"]["spec"]["containers"]:
   if c["name"]=="rate-anything":c["image"]="<approved-frontend-image-only>"
 rows.append((key,m["uid"],s,x.get("data")))
print(json.dumps(sorted(rows),sort_keys=True,separators=(",",":")))';done|sha256sum|cut -d' ' -f1; }
protected=$(snapshot);bash "$ops/check-rat181-protected-baseline.sh" >"$t/before";echo "PR183_PREFLIGHT api=$api rat=$current protected=$protected"
changed=0
rollback(){ rc=$?;trap - ERR;set +e;if [[ $changed == 1 ]];then k -n rum-rate-anything-preview set image deployment/rum-rate-anything "rate-anything=$current";k -n rum-rate-anything-preview rollout status deployment/rum-rate-anything --timeout=240s;fi;echo ROLLOUT_FAILED;exit "$rc"; };trap rollback ERR
if [[ $current != "$rat" ]];then changed=1;k -n rum-rate-anything-preview set image deployment/rum-rate-anything "rate-anything=$rat";fi
k -n rum-rate-anything-preview rollout status deployment/rum-rate-anything --timeout=300s
selector=$(k -n rum-rate-anything-preview get deploy rum-rate-anything -o json|python3 -c 'import json,sys;print(",".join(k+"="+v for k,v in json.load(sys.stdin)["spec"]["selector"]["matchLabels"].items()))')
k -n rum-rate-anything-preview get pods -l "$selector" -o json|python3 -c 'import json,sys;ps=[p for p in json.load(sys.stdin)["items"] if not p["metadata"].get("deletionTimestamp")];assert ps
for p in ps:
 c=next(c for c in p["status"]["containerStatuses"] if c["name"]=="rate-anything");assert c["ready"] and c["imageID"].endswith(sys.argv[1].split("@")[1]);print("READY_FRONTEND",p["metadata"]["name"],c["imageID"])' "$rat"
[[ $(curl -fsS --retry 3 --max-time 25 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION) == "rat-candidate+$sha" ]]
[[ $(curl -fsS --retry 2 --max-time 25 https://dev-rum-ra.daisycloversoftware.uk/healthz) == ok ]]
curl -fsS --retry 2 --max-time 25 https://dev-rum.daisycloversoftware.uk/api/v1/health|python3 -c 'import json,sys;d=json.load(sys.stdin);assert d["status"]=="ok" and sys.argv[1] in d["version"];print("UNCHANGED_API_HEALTH",d["version"])' "$base"
[[ $(snapshot) == "$protected" ]];trap - ERR
bash "$ops/check-rat181-protected-baseline.sh" >"$t/after";for f in before after;do grep -E '^((entities|entity_identifiers|rating_events|relationships) rows=|AUDIENCE |ROUTING )' "$t/$f" >"$t/$f.compare";done;diff -u "$t/before.compare" "$t/after.compare";cat "$t/after.compare"
echo "PR183_FRONTEND_DEPLOYED_AND_PROTECTED sha=$sha rat=$rat unchanged_api=$api protected=$protected"
