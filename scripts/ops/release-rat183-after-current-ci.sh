#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]]||exit 64
sha=bb7dc6f8659b21fe612c1a003b8054b29a8317e9;run=45620813540;repo=DaisyCloverSoftware/rum;ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
# Observe the existing run only. No workflow retries, dispatch, builds or source edits.
gh api "repos/$repo/actions/runs/$run" >"$t/run"
python3 - "$t/run" "$sha" <<'PY'
import json,sys
r=json.load(open(sys.argv[1]));assert r['head_sha']==sys.argv[2] and r['head_branch']=='fix/rat-connected-profile-thumbnails-20261125' and any(p['number']==183 for p in r['pull_requests']);print('OBSERVING_EXISTING_EXACT_HEAD_CI',r['id'],r['status'],r['conclusion'])
PY
if [[ $(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["status"])' "$t/run") != completed ]];then
 timeout 600s gh run watch "$run" --repo "$repo" --interval 60 --exit-status >"$t/watch" 2>&1 || { tail -25 "$t/watch";echo CI_NOT_SUCCESSFUL_NO_DEPLOYMENT;exit 75; }
fi
# The existing guarded release revalidates PR, all nine jobs, publication provenance,
# approved API baseline, routing and non-target workloads immediately before mutation.
bash "$ops/complete-rat183-delivery.sh" release
