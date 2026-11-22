#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 1 && $1 =~ ^qa181[a-z0-9]{8}$ ]] || exit 64
tag=$1;sha=b135949c8d97072cca3b24c2611984ae6d6fc30c;ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
id=rat181-20261122-guarded-isolated-promote-r1
# Read only the exact known Workbench receipt via the runner's usual Git transport.
git -C "$t" init -q;git -C "$t" remote add origin https://github.com/DaisyCloverSoftware/workbench-relay-private.git;git -C "$t" fetch -q --depth=1 --filter=blob:none origin main; git -C "$t" show "FETCH_HEAD:relay/control-outbox/$id.json" >"$t/release" || { echo 'NO_ROLLOUT_RECEIPT_NO_FIXTURES'; exit 75; }
python3 - "$t" "$id" "$sha" <<'PY'
import sys,json,re
from pathlib import Path
t=Path(sys.argv[1]);r=json.loads((t/'release').read_text());op=r.get('result',{}).get('operation_script',{});status=r.get('operation_status',{});assert r['id']==sys.argv[2] and r['status']=='completed' and op['exit_code']==0 and status['process_exit_code']==0 and status['process_result_observed'] and op['commit']=='1989970f11c4a2ec33a579933a8608891caa19ae';o=op['output'];assert 'PR181_ROLLOUT_AND_PROTECTED_AUDIT_PASS' in o and 'PR181_DEPLOYED sha='+sys.argv[3] in o
for k in ('api','rate-anything'):
 d=re.findall(r'(?m)^PUBLISHED '+k+r' [0-9]+ '+sys.argv[3]+r' (ghcr\.io/daisycloversoftware/rum-'+k+r'@sha256:[0-9a-f]{64})$',o);assert len(d)==1;(t/k).write_text(d[0]);print('VERIFIED_ROLLOUT_IMAGE',k,d[0]);
PY
[[ $(curl -fsS --max-time 20 https://dev-rum-ra.daisycloversoftware.uk/VERSION) == "rat-candidate+$sha" ]] || exit 78
curl -fsS --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health|python3 -c 'import json,sys;d=json.load(sys.stdin);assert d.get("status")=="ok" and sys.argv[1] in d["version"];print("CANONICAL_API_UP",d["version"])' "$sha"
# Browser 1 drives actual deployed ingress (GET/HEAD/OPTIONS only). Browser 2 writes
# through the deployed-image test cell with its own DB, never by weakening staging fixture controls.
bash "$ops/verify-rat-second-channel-chromium-20261122.sh"; echo CANONICAL_DEV_READONLY_CHROMIUM_PASS
set +e; bash "$ops/verify-rat181-workflow.sh" images "$sha" "$(cat "$t/api")" "$(cat "$t/rate-anything")" "$tag" >"$t/browser" 2>&1;rc=$?;set -e;tail -55 "$t/browser"
bash "$ops/check-rat181-protected-baseline.sh" >"$t/after";grep -E '^((entities|entity_identifiers|rating_events|relationships) rows=|AUDIENCE |ROUTING )' "$t/after";grep -Fxq 'entities rows=3 sha256=5112dc1f68d8b48f6c5c41968619ff9d362de16d66b637b2dd6c60271ff9e443' "$t/after";grep -Fxq 'entity_identifiers rows=1 sha256=db111e02ca58748931dd5252ff6fc4ab03332bcf37678297ff241c3ca821f791' "$t/after";grep -Fxq 'rating_events rows=2 sha256=51fd8c48c16b3ec3ad33a5d7a70031c9f0122a51bad8bbb8cd4b8d45c25336e4' "$t/after";grep -Fxq 'relationships rows=2 sha256=688a57ffb9558a55e5098d3a04006abdb733446d15bd6828077263538b088c7c' "$t/after";echo PROTECTED_RECORDS_UNCHANGED_AFTER_BROWSER
[[ $rc == 0 ]] || exit "$rc";grep -q '^FULL_ACCEPTANCE_PASS ' "$t/browser";echo VERIFIED_PR181_IMAGE_WORKFLOW_ACCEPTANCE
