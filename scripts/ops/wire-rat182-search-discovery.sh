#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 1 && $1 =~ ^[a-f0-9]{40}$ ]] || exit 64
expected=$1;branch=feat/rat-shared-persona-discovery-20261124;t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
git -C "$t" init -q;git -C "$t" remote add origin https://github.com/DaisyCloverSoftware/rum.git;git -C "$t" fetch -q --depth=1 origin "$branch";git -C "$t" checkout -q --detach FETCH_HEAD;cd "$t";[[ $(git rev-parse HEAD) == "$expected" ]] || exit 78
# Require the completed API/workspace integration, never a partially prepared branch.
grep -q "'relatedThrough' => \$this->relatedThrough" apps/api/app/Services/RatExplorerService.php
grep -q '<RatRelatedConnections groups={node.relatedThrough' apps/rate-anything/src/RatEntityWorkspace.tsx
python3 - <<'PY'
from pathlib import Path
p=Path('apps/rate-anything/src/SearchExperience.tsx');s=p.read_text()
a='function LinkedCard({ item, onSelect }: { item: RatLinkedItem; onSelect: (item: RatLinkedItem) => void })';b="function LinkedCard<T extends RatEntity>({ item, onSelect }: { item: T; onSelect: (item: T) => void })";assert s.count(a)==1;s=s.replace(a,b)
a='className="rat-category" aria-labelledby={`category-${category.key}`}';b='className="rat-category" data-connection-kind="direct" aria-labelledby={`category-${category.key}`}';assert s.count(a)==1;s=s.replace(a,b)
a='      {node.categories.map((category) => <CategorySection key={category.key} category={category} onSelect={rebase}/>)}'
b=a+'''
      {(node.relatedThrough ?? []).map((group) => <section className="rat-category" data-connection-kind="indirect" aria-label={group.label} key={group.key}>
        <header className="rat-category__head"><h2>{group.label}</h2><span>{group.total} indirect</span></header>
        <p className="rat-muted">Indirect public connections, not direct links or verified shared ownership.</p>
        <button type="button" className="rat-secondary-action" onClick={() => rebase(group.via)}>View connecting Thing: {group.via.name}</button>
        <div className="rat-category__row">{group.items.map((item) => <LinkedCard key={item.id} item={item} onSelect={rebase}/>)}</div>
      </section>)}'''
assert s.count(a)==1;s=s.replace(a,b);p.write_text(s)
p=Path('apps/rate-anything/src/ratRelatedConnections.test.ts');s=p.read_text();a="    expect(component).not.toContain('ratCommunityConnectionApi')";b=a+'''
    const search = readFileSync(new URL('./SearchExperience.tsx', import.meta.url), 'utf8')
    expect(search).toContain('(node.relatedThrough ?? []).map((group)')
    expect(search).toContain('<LinkedCard key={item.id} item={item} onSelect={rebase}/>')
    expect(search).toContain('onClick={() => rebase(group.via)}')
    expect(search).toContain('data-connection-kind="indirect"')''';assert s.count(a)==1;p.write_text(s.replace(a,b))
PY
(cd apps/rate-anything;npm ci --no-audit --no-fund --ignore-scripts >"$t/npm.log" 2>&1||{ tail -35 "$t/npm.log";exit 1; };npm run build >"$t/build.log" 2>&1||{ tail -40 "$t/build.log";exit 1; };npm run lint >"$t/lint.log" 2>&1||{ tail -40 "$t/lint.log";exit 1; };npm test -- src/ratRelatedConnections.test.ts src/ratExplorer.test.ts >"$t/tests.log" 2>&1||{ tail -70 "$t/tests.log";exit 1; };tail -15 "$t/tests.log")
git diff --check;git add apps/rate-anything/src/SearchExperience.tsx apps/rate-anything/src/ratRelatedConnections.test.ts;git -c user.name=Workbench -c user.email=workbench@users.noreply.github.com commit -m 'Show shared-connection discovery in RAT search without changing direct cards';git push origin HEAD:"$branch";echo "RAT182_SEARCH_PRESENTATION_PASS source=$(git rev-parse HEAD)"
