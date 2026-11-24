#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]] || exit 64
ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
python3 - "$ops/prepare-rat182-shared-discovery.sh" "$t/prepare.sh" <<'PYMAKE'
from pathlib import Path
import sys
s=Path(sys.argv[1]).read_text();assert s.count('pod=rat182-source-preflight-r1')==1;s=s.replace('pod=rat182-source-preflight-r1','pod=rat182-source-preflight-r2')
extra=r"""
p='apps/api/app/Services/RatSharedConnectionDiscovery.php'
patch(p,'    {\n        return EntityRelationship::query()',"    {\n        $visibleEntities = Entity::query()->select('entities.id');\n        $this->visible($visibleEntities);\n\n        return EntityRelationship::query()")
patch(p,"            ->whereHas('sourceEntity', fn (Builder $query) => $this->visible($query))\n            ->whereHas('targetEntity', fn (Builder $query) => $this->visible($query))", "            ->whereIn('source_entity_id', clone $visibleEntities)\n            ->whereIn('target_entity_id', clone $visibleEntities)")
p='apps/rate-anything/src/SearchExperience.tsx'
patch(p,'function LinkedCard({ item, onSelect }: { item: RatLinkedItem; onSelect: (item: RatLinkedItem) => void })','function LinkedCard<T extends RatEntity>({ item, onSelect }: { item: T; onSelect: (item: T) => void })')
patch(p,'className="rat-category" aria-labelledby={`category-${category.key}`}','className="rat-category" data-connection-kind="direct" aria-labelledby={`category-${category.key}`}')
a='      {node.categories.map((category) => <CategorySection key={category.key} category={category} onSelect={rebase}/>)}'
b=a+'''
      {(node.relatedThrough ?? []).map((group) => <section className="rat-category" data-connection-kind="indirect" aria-label={group.label} key={group.key}>
        <header className="rat-category__head"><h2>{group.label}</h2><span>{group.total} indirect</span></header>
        <p className="rat-muted">Indirect public connections, not direct links or verified shared ownership.</p>
        <button type="button" className="rat-secondary-action" onClick={() => rebase(group.via)}>View connecting Thing: {group.via.name}</button>
        <div className="rat-category__row">{group.items.map((item) => <LinkedCard key={item.id} item={item} onSelect={rebase}/>)}</div>
      </section>)}'''
patch(p,a,b)
p='apps/rate-anything/src/ratRelatedConnections.test.ts'
a="    expect(component).not.toContain('ratCommunityConnectionApi')"
b=a+'''
    const search = readFileSync(new URL('./SearchExperience.tsx', import.meta.url), 'utf8')
    expect(search).toContain('(node.relatedThrough ?? []).map((group)')
    expect(search).toContain('<LinkedCard key={item.id} item={item} onSelect={rebase}/>')
    expect(search).toContain('onClick={() => rebase(group.via)}')
    expect(search).toContain('data-connection-kind="indirect"')'''
patch(p,a,b)
"""
marker='\nPY\n(cd apps/rate-anything;';assert s.count(marker)==1;s=s.replace(marker,'\n'+extra+marker)
a='npm test -- src/ratRelatedConnections.test.ts >';assert s.count(a)==1;s=s.replace(a,'npm test -- src/ratRelatedConnections.test.ts src/ratExplorer.test.ts >')
a='git add apps/api/app/Services/RatExplorerService.php apps/api/app/Services/RatSharedConnectionDiscovery.php apps/api/tests/Feature/Entity/RatSharedConnectionDiscoveryTest.php apps/rate-anything/src/ratExplorerApi.ts apps/rate-anything/src/RatEntityWorkspace.tsx';assert s.count(a)==1;s=s.replace(a,a+' apps/rate-anything/src/SearchExperience.tsx apps/rate-anything/src/ratRelatedConnections.test.ts')
Path(sys.argv[2]).write_text(s)
PYMAKE
bash -n "$t/prepare.sh";bash "$t/prepare.sh"
