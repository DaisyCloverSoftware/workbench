#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 1 && $1 =~ ^[a-f0-9]{40}$ && $1 != b8127ac50dce516379c6a188403d2a99277cfce6 ]] || exit 64
sha=$1;ops=$(cd "$(dirname "$0")" && pwd);base=$(git -C "$ops" rev-parse HEAD);t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
gh api repos/DaisyCloverSoftware/rum/pulls/182 | python3 -c 'import json,sys;p=json.load(sys.stdin);assert p["head"]["sha"]==sys.argv[1] and p["head"]["ref"]=="feat/rat-shared-persona-discovery-20261124" and p["base"]["sha"]=="b135949c8d97072cca3b24c2611984ae6d6fc30c" and p["state"]=="open" and p["draft"] and not p["merged"]' "$sha"
git -C "$t" init -q;git -C "$t" remote add origin https://github.com/DaisyCloverSoftware/workbench.git;git -C "$t" fetch -q --depth=1 origin main;git -C "$t" checkout -q --detach FETCH_HEAD;[[ $(git -C "$t" rev-parse HEAD) == "$base" ]] || { echo WORKBENCH_HEAD_MOVED_NO_WRITE;exit 78; }
python3 - "$t" "$sha" <<'PY'
from pathlib import Path
import sys
root=Path(sys.argv[1]);sha=sys.argv[2]
p=root/'scripts/ops/release-rat182-shared-discovery.sh';s=p.read_text()
for old,new in [('8f22ae6cbe62f06e3363cf3ec897d769c59e4e6522d87f919a62f443943ac894','6ad2db06b44acca08d26ac4e062bc77d279513e45a62e862d4bb85e10a21d450'),('6d6253ee6ab8f1709ad7f013a83943c3c6674621a53f7a7c3cdf18c26c09ebc2','672f775608943603548c7b3269e41c68ba31a806070705e224a1b1f073950a54')]:
 assert s.count(old)==1,(old,s.count(old));s=s.replace(old,new)
p.write_text(s)
p=root/'scripts/ops/accept-rat182-private-paths.sh';s=p.read_text();old='$1 == b8127ac50dce516379c6a188403d2a99277cfce6';assert s.count(old)==1;s=s.replace(old,'$1 == '+sha)
old="$p=$make('persona','Fixture public creator',true);";new="$p=$make('person','Fixture public creator',true);$p->forceFill(['status'=>'unverified'])->save();";assert s.count(old)==1;s=s.replace(old,new);p.write_text(s)
PY
cd "$t";bash -n scripts/ops/release-rat182-shared-discovery.sh;bash -n scripts/ops/accept-rat182-private-paths.sh;git diff --check
git add scripts/ops/release-rat182-shared-discovery.sh scripts/ops/accept-rat182-private-paths.sh
git -c user.name=Workbench -c user.email=workbench@users.noreply.github.com commit -m 'Gate corrected RAT candidate from current DEV images and test unverified public people'
git push origin HEAD:main
echo "RAT182_CORRECTED_DELIVERY_READY app=$sha workbench=$(git rev-parse HEAD) deployment_not_submitted=true"
