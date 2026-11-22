#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]] || exit 64
repo=DaisyCloverSoftware/rum;run=45032334541;sha=b135949c8d97072cca3b24c2611984ae6d6fc30c
command -v gh >/dev/null || { echo NO_CONFIGURED_GH_CLI;exit 78; };t=$(mktemp -d);umask 077;trap 'rm -rf "$t"' EXIT
# Only use ordinary configured gh read access. Never acquire or print credentials.
gh api "repos/$repo/actions/runs/$run" >"$t/run" || { echo USE_GITHUB_CONNECTOR;exit 78; }
python3 - "$t/run" "$sha" <<'PY'
import sys,json
j=json.load(open(sys.argv[1]));assert j['head_sha']==sys.argv[2] and j['head_branch']=='fix/rat-persist-explicit-linked-things-20261122' and any(p['number']==181 for p in j['pull_requests']);print('CI_STATUS',j['status'],j['conclusion'],j['run_attempt'])
PY
if [[ $(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["status"])' "$t/run") != completed ]];then timeout 240s gh run watch "$run" --repo "$repo" --interval 40 --exit-status >"$t/watch" 2>&1 || { tail -5 "$t/watch";echo 'CI_NOT_CONFIRMED';exit 75; };fi
gh api "repos/$repo/actions/runs/$run" >"$t/run";gh api "repos/$repo/actions/runs/$run/jobs?per_page=100" >"$t/jobs"
python3 - "$t" "$sha" <<'PY'
import json,sys
from pathlib import Path
t=Path(sys.argv[1]);j=json.loads((t/'run').read_text());assert j['head_sha']==sys.argv[2] and j['conclusion']=='success';jobs=json.loads((t/'jobs').read_text())['jobs'];assert len(jobs)==9 and all(x['conclusion']=='success' for x in jobs);print('NINE_JOBS_GREEN',j['run_attempt']);
for k in ['api','rate-anything']:
 a=[x for x in jobs if x['name']==f'RAT candidate image ({k})'];assert len(a)==1;(t/(k+'.id')).write_text(str(a[0]['id']));
PY
for c in api rate-anything;do id=$(cat "$t/$c.id"); gh api "repos/$repo/actions/jobs/$id/logs" >"$t/log";python3 - "$t/log" "$c" "$sha" "$id" <<'PY'
import re,sys
s=open(sys.argv[1],errors='replace').read(); c,sha,job=sys.argv[2:];assert re.search(r'CANDIDATE_SOURCE_SHA='+sha+r'\s',s); dig=set(re.findall(r'CANDIDATE_IMAGE=(ghcr\.io/daisycloversoftware/rum-'+re.escape(c)+r'@sha256:[0-9a-f]{64})\s',s));assert len(dig)==1;print('PUBLISHED',c,job,sha,dig.pop())
PY
done
