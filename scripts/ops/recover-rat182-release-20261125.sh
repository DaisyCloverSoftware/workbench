#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]] || exit 64
ops=$(cd "$(dirname "$0")" && pwd); t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
# Read only: do not run, re-submit or alter any recovered request.
git -C "$t" init -q; git -C "$t" remote add origin https://github.com/DaisyCloverSoftware/workbench-relay-private.git
git -C "$t" fetch -q --depth=1 --filter=blob:none origin main
python3 - "$t" <<'PY'
import json,subprocess,sys
root=sys.argv[1]
def git(*a):return subprocess.check_output(['git','-C',root,*a],text=True)
paths=git('ls-tree','-r','--name-only','FETCH_HEAD','relay/control','relay/control-outbox').splitlines()
for p in paths:
 if '/rat182-' not in p:continue
 d=json.loads(git('show','FETCH_HEAD:'+p));op=d.get('result',{}).get('operation_script',{});a=d.get('args',{})
 print('RELAY_STATE',json.dumps({'path':p,'id':d.get('id'),'status':d.get('status'),'action':d.get('action'),'script':op.get('path',a.get('path')),'commit':op.get('commit',a.get('commit')),'exit_code':op.get('exit_code'),'updated_at':d.get('updated_at'),'output_tail':op.get('output','')[-500:]}))
PY
bash "$ops/check-rat181-protected-baseline.sh"
for spec in 'rum-dev-isolated rum-api' 'rum-rate-anything-preview rum-rate-anything';do read -r ns dep <<<"$spec"; sudo -n k3s kubectl -n "$ns" get deployment "$dep" -o 'jsonpath={.metadata.namespace}{" "}{.metadata.name}{" uid="}{.metadata.uid}{" generation="}{.metadata.generation}{" observed="}{.status.observedGeneration}{" ready="}{.status.readyReplicas}{" images="}{.spec.template.spec.containers[*].image}{"\n"}';done
curl -fsS --retry 1 --max-time 25 https://dev-rum.daisycloversoftware.uk/api/v1/health;echo
curl -fsS --retry 1 --max-time 25 https://dev-rum-ra.daisycloversoftware.uk/VERSION;echo
echo RAT182_RECOVERY_READONLY_COMPLETE
