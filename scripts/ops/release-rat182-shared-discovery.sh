#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 3 && $1 =~ ^[0-9]+$ && $2 =~ ^[a-f0-9]{40}$ && $3 =~ ^[0-9]+$ ]] || exit 64
pr=$1;sha=$2;run=$3;repo=DaisyCloverSoftware/rum;ops=$(cd "$(dirname "$0")" && pwd)
t=$(mktemp -d);umask 077;trap 'rm -rf "$t"' EXIT
exec 9>"${TMPDIR:-/tmp}/rat182-shared-discovery-release.lock";flock -n 9 || { echo CONCURRENT_RELEASE_NO_WRITE;exit 75; }
# Ordinary configured GitHub access only. No credential retrieval or logging.
gh api "repos/$repo/pulls/$pr" >"$t/pr";gh api "repos/$repo/actions/runs/$run" >"$t/run";gh api "repos/$repo/actions/runs/$run/jobs?per_page=100" >"$t/jobs"
python3 - "$t" "$pr" "$sha" <<'PY'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);pr=json.loads((p/'pr').read_text());r=json.loads((p/'run').read_text());jobs=json.loads((p/'jobs').read_text())['jobs'];sha=sys.argv[3]
assert pr['number']==int(sys.argv[2]) and pr['state']=='open' and pr['draft'] and not pr['merged']
assert pr['head']['sha']==sha and pr['head']['ref']=='feat/rat-shared-persona-discovery-20261124'
assert pr['base']['sha']=='b135949c8d97072cca3b24c2611984ae6d6fc30c' and pr['base']['ref']=='fix/rat-persist-explicit-linked-things-20261122'
assert r['head_sha']==sha and r['status']=='completed' and r['conclusion']=='success'
assert r['head_branch']==pr['head']['ref'] and any(x['number']==pr['number'] for x in r['pull_requests'])
assert len(jobs)==9 and all(x['conclusion']=='success' for x in jobs)
print('EXACT_HEAD_NINE_JOBS_GREEN',r['id'],r['run_attempt'],sha)
for component in ('api','rate-anything'):
 j=next(x for x in jobs if x['name']==f'RAT candidate image ({component})');(p/(component+'.job')).write_text(str(j['id']))
j=next(x for x in jobs if x['name']=='api');assert any(s['name']=='Run vendor/bin/phpunit' and s['conclusion']=='success' for s in j['steps']);(p/'api-check.job').write_text(str(j['id']))
PY
gh api "repos/$repo/actions/jobs/$(cat "$t/api-check.job")/logs" >"$t/api-check.log"
python3 - "$t/api-check.log" <<'PY'
import re,sys
s=re.sub(r'\x1b\[[0-9;]*m','',open(sys.argv[1],errors='replace').read());matches=re.findall(r'OK \(\d+ tests?, \d+ assertions?\)',s);assert matches,'No successful PHPUnit lifecycle summary';print('PHPUNIT_RESULT',matches[-1]);assert '[OK] No errors' in s;print('PHPSTAN_RESULT zero errors')
for line in s.splitlines():
 if 'PASS' in line and 'files' in line:print('PINT_RESULT',line)
PY
for c in api rate-anything;do id=$(cat "$t/$c.job");gh api "repos/$repo/actions/jobs/$id/logs" >"$t/publication.log";python3 - "$t" "$c" "$sha" "$id" <<'PY'
from pathlib import Path
import re,sys
p=Path(sys.argv[1]);c,sha,job=sys.argv[2:];s=(p/'publication.log').read_text(errors='replace');assert re.search(r'CANDIDATE_SOURCE_SHA='+sha+r'\s',s);images=set(re.findall(r'CANDIDATE_IMAGE=(ghcr\.io/daisycloversoftware/rum-'+re.escape(c)+r'@sha256:[a-f0-9]{64})\s',s));assert len(images)==1;image=images.pop();(p/c).write_text(image);print('PUBLISHED',c,job,sha,image)
PY
done
bash "$ops/check-rat181-protected-baseline.sh" >"$t/before"
grep -Fxq 'entities rows=3 sha256=c1090b07abd3511fb5b4c40862025ca1b69e12c0742739a303d52b7f02563e7e' "$t/before"
grep -Fxq 'entity_identifiers rows=1 sha256=db111e02ca58748931dd5252ff6fc4ab03332bcf37678297ff241c3ca821f791' "$t/before"
grep -Fxq 'rating_events rows=2 sha256=51fd8c48c16b3ec3ad33a5d7a70031c9f0122a51bad8bbb8cd4b8d45c25336e4' "$t/before"
grep -Fxq 'relationships rows=2 sha256=688a57ffb9558a55e5098d3a04006abdb733446d15bd6828077263538b088c7c' "$t/before"
# Reuse the reviewed image-only rollout, rollback and protected-workload snapshot checks.
python3 - "$ops/deploy-rat181-linked-things-isolated.sh" "$t/deploy.sh" <<'PY'
from pathlib import Path
import sys
s=Path(sys.argv[1]).read_text()
for old,new in [
 ('7411cc02eb6568e6c6d228b551683294bfac49a6f89c320078af5550adce5d56','8f22ae6cbe62f06e3363cf3ec897d769c59e4e6522d87f919a62f443943ac894'),
 ('9a33ea315f52d5e14a5ece0e57895aae8d4eb4d7d7e9955110d49fc9676c64d9','6d6253ee6ab8f1709ad7f013a83943c3c6674621a53f7a7c3cdf18c26c09ebc2'),
 ('refs/heads/fix/rat-persist-explicit-linked-things-20261122','refs/heads/feat/rat-shared-persona-discovery-20261124'),
 ('PR181_DEPLOYED','PR182_DEPLOYED')]:
 assert s.count(old)==1,old;s=s.replace(old,new)
Path(sys.argv[2]).write_text(s)
PY
bash "$t/deploy.sh" "$sha" "$(cat "$t/api")" "$(cat "$t/rate-anything")"
bash "$ops/check-rat181-protected-baseline.sh" >"$t/after"
for f in before after;do grep -E '^((entities|entity_identifiers|rating_events|relationships) rows=|AUDIENCE |ROUTING )' "$t/$f" >"$t/$f.compare";done
diff -u "$t/before.compare" "$t/after.compare";cat "$t/after.compare"
echo "RAT182_ROLLOUT_AND_PROTECTED_AUDIT_PASS pr=$pr sha=$sha run=$run"
