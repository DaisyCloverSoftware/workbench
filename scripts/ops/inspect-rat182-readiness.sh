#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]] || exit 64
ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
for f in release-rat182-shared-discovery.sh accept-rat182-shared-discovery.sh accept-rat182-private-paths.sh;do bash -n "$ops/$f";done
python3 - "$ops" "$t" <<'PY'
from pathlib import Path
import sys,subprocess
ops,t=map(Path,sys.argv[1:]);s=(ops/'accept-rat182-private-paths.sh').read_text();js=s.split("<<'JS'\n",1)[1].split('\nJS\n',1)[0];p=t/'privacy.mjs';p.write_text(js);subprocess.run(['node','--check',str(p)],check=True);compile(s.split("<<'PY'\n",1)[1].rsplit('\nPY\n',1)[0],'<image-harness>','exec');s=(ops/'accept-rat182-shared-discovery.sh').read_text();js=s.split("<<'JS'\n",1)[1].split('\nJS\n',1)[0];p=t/'public.mjs';p.write_text(js);subprocess.run(['node','--check',str(p)],check=True);print('ACCEPTANCE_SCRIPT_SYNTAX_PASS')
PY
for id in 01m0mw9hkbynssr7kbc0gk534j 01m7x6ez57zpsgbv486y9t0y9h 01m81mbmqvwr3d1nvfp9gkp3xf;do curl -fsS --retry 2 --max-time 25 "https://dev-rum.daisycloversoftware.uk/api/v1/public/entities/rat-catalogue/nodes/$id" | python3 -c 'import json,sys;d=json.load(sys.stdin);n=d["data"];assert n["active"]["id"]==sys.argv[1];print("CURRENT_CANONICAL_GRAPH",json.dumps({"id":n["active"]["id"],"name":n["active"]["name"],"type":n["active"]["entityType"],"classifications":n["active"]["classifications"],"direct":[{"id":i["id"],"relationship":i["relationship"]} for c in n["categories"] for i in c["items"]],"derived":n.get("relatedThrough")}))' "$id";done
gh api repos/DaisyCloverSoftware/rum/actions/runs/45375632818/jobs?per_page=100 | python3 -c 'import json,sys;print("CI_JOB_STATE",json.dumps([{k:x[k] for k in ("id","name","status","conclusion","head_sha")} for x in json.load(sys.stdin)["jobs"]]))'
echo RAT182_READONLY_READINESS_PASS
