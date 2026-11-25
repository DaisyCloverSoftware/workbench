#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 1 && $1 =~ ^(recover|fix-test)$ ]] || exit 64
mode=$1;ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);umask 077;trap 'rm -rf "$t"' EXIT
if [[ $mode == recover ]];then
 mkdir "$t/relay";git -C "$t/relay" init -q;git -C "$t/relay" remote add origin https://github.com/DaisyCloverSoftware/workbench-relay-private.git;git -C "$t/relay" fetch -q --depth=1 --filter=blob:none origin main
 python3 - "$t/relay" <<'PY'
import json,subprocess,sys
r=sys.argv[1];paths=subprocess.check_output(['git','-C',r,'ls-tree','-r','--name-only','FETCH_HEAD','--','relay/control','relay/control-outbox'],text=True).splitlines();paths=[p for p in paths if '/rat183-' in p];pending=[];applies=[]
for p in paths:
 j=json.loads(subprocess.check_output(['git','-C',r,'show','FETCH_HEAD:'+p],text=True));op=j.get('result',{}).get('operation_script',{});args=j.get('args',{});print('PR183_OPERATION',json.dumps({'path':p,'id':j.get('id'),'status':j.get('status'),'action':j.get('action'),'script':op.get('path',args.get('path')),'args':op.get('args',args.get('args')),'commit':op.get('commit',args.get('commit')),'exit':op.get('exit_code'),'started':j.get('operation_status',{}).get('started')}))
 if p.startswith('relay/control/') and args.get('path')=='scripts/ops/rum-dev-delete-rating.sh' and 'apply' in args.get('args',[]):
  out=p.replace('/control/','/control-outbox/');applies.append(j['id']);pending.extend([j['id']] if out not in paths else [])
print('PR183_PRIOR_DELETE_APPLIES',json.dumps(applies));print('PR183_PENDING_DELETE_APPLIES',json.dumps(pending))
PY
 git -C "$ops" ls-tree -r --name-only HEAD -- scripts/ops | grep -E '(rat183|rum-dev-delete-rating|purge.*cj|cj.*purge)' || true
 sha256sum "$ops/rum-dev-delete-rating.sh"
 bash "$ops/check-rat181-protected-baseline.sh"
 echo PR183_READONLY_RECOVERY_COMPLETE
 exit
fi
sha=349d8178c4da2daa82434a3e3c37dad38c5aebeb;branch=fix/rat-connected-profile-thumbnails-20261125
# No application implementation is changed. Stop if the owner's checkpoint has advanced.
gh api repos/DaisyCloverSoftware/rum/pulls/183 | python3 -c 'import json,sys;p=json.load(sys.stdin);assert p["head"]["sha"]==sys.argv[1] and p["head"]["ref"]==sys.argv[2] and p["base"]["sha"]=="2359178af19fdd9b5b45d103c892cdc36c2ad086" and p["state"]=="open" and p["draft"] and not p["merged"]' "$sha" "$branch"
git -C "$t" init -q;git -C "$t" remote add origin https://github.com/DaisyCloverSoftware/rum.git;git -C "$t" fetch -q --depth=1 origin "$branch";git -C "$t" checkout -q --detach FETCH_HEAD;[[ $(git -C "$t" rev-parse HEAD) == "$sha" ]]||exit 78
python3 - "$t" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1])/'apps/rate-anything/src/ratExplorer.test.ts';s=p.read_text();start=s.index("  it('shows only published profile photos");end=s.index("\n  it('keeps Rate",start)
old=s[start:end];assert "setFailedUrl(profileUrl)" in old
new='''  it('shares published-profile thumbnails across search and direct and indirect connections, with existing fallbacks and dimensions', () => {
    const experience = read('./SearchExperience.tsx')
    const thumbnail = read('./PublishedProfileThumbnail.tsx')
    const workspace = read('./RatEntityWorkspace.tsx')
    const related = read('./RatRelatedConnections.tsx')
    const css = read('./rat-explorer.css')

    expect(thumbnail).toContain("entity.images?.profile?.status === 'published'")
    expect(thumbnail).toContain('onError={() => setFailedImage(imageKey)}')
    expect(thumbnail).toContain('failedImage !== imageKey')
    expect(thumbnail).toContain('`${entity.id}:${profileUrl ?? \\\'\\\'}`')
    expect(thumbnail).toContain('fallback ?? entity.name.slice(0, 2).toUpperCase()')
    for (const surface of [experience, workspace, related]) {
      expect(surface).toContain("import { PublishedProfileThumbnail } from './PublishedProfileThumbnail'")
      expect(surface).not.toContain('function PublishedProfileThumbnail')
      expect(surface).toContain('<PublishedProfileThumbnail entity={item}')
    }
    expect(experience).toContain('<PublishedProfileThumbnail entity={active} className="rat-active-card__avatar"/>')
    expect(experience).toContain('<PublishedProfileThumbnail entity={item} className="rat-linked-card__media"/>')
    expect(workspace).toContain('className="rat-workspace-connection__mark" fallback={initials(item.name)}')
    expect(related).toContain('className="rat-workspace-connection__mark" fallback={item.name.split(')
    expect(css).toContain('.rat-active-card__avatar > img, .rat-linked-card__media > img')
    expect(css).toMatch(/\\.rat-workspace-connection__mark \\{[^}]*width: 42px; height: 42px;/)
    expect(css).toContain('.rat-workspace-connection__mark img { display: block; width: 100%; height: 100%; object-fit: cover; }')
  })
'''
p.write_text(s[:start]+new+s[end:])
PY
cd "$t/apps/rate-anything";npm ci --ignore-scripts --no-audit --no-fund >"$t/npm-install.log" 2>&1;npx playwright install chromium >"$t/browser-install.log" 2>&1
npm run lint;npm run typecheck;npm test;npm run build
cd "$t";git diff --check;[[ $(git diff --name-only) == apps/rate-anything/src/ratExplorer.test.ts ]]||exit 78
[[ $(git ls-remote origin "refs/heads/$branch" | cut -f1) == "$sha" ]]||exit 78
git add apps/rate-anything/src/ratExplorer.test.ts;git -c user.name=Workbench -c user.email=workbench@users.noreply.github.com commit -m 'Test shared published thumbnails after extracting search component';git push origin HEAD:"$branch"
echo "PR183_FULL_FRONTEND_VALIDATION_PASS source=$(git rev-parse HEAD) previous=$sha branch=$branch production_changes=0"
