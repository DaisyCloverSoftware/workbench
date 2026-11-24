#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 2 && $1 =~ ^[a-f0-9]{40}$ && $2 =~ ^rat182-20261124-isolated-release-r[1-5]$ ]] || exit 64
sha=$1;release=$2;ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);umask 077;trap 'rm -rf "$t"' EXIT
mkdir "$t/relay";git -C "$t/relay" init -q;git -C "$t/relay" remote add origin https://github.com/DaisyCloverSoftware/workbench-relay-private.git;git -C "$t/relay" fetch -q --depth=1 --filter=blob:none origin main;git -C "$t/relay" show "FETCH_HEAD:relay/control-outbox/$release.json" >"$t/release.json"
python3 - "$t" "$sha" "$release" <<'PY'
from pathlib import Path
import json,re,sys
p=Path(sys.argv[1]);r=json.loads((p/'release.json').read_text());s=r['result']['operation_script'];assert r['id']==sys.argv[3] and r['status']=='completed' and s['exit_code']==0 and r['operation_status']['process_exit_code']==0
assert s['path']=='scripts/ops/release-rat182-shared-discovery.sh';o=s['output'];assert 'RAT182_ROLLOUT_AND_PROTECTED_AUDIT_PASS' in o and 'sha='+sys.argv[2] in o
for c in ('api','rate-anything'):
 a=re.findall(r'(?m)^PUBLISHED '+c+r' [0-9]+ '+sys.argv[2]+r' (ghcr\.io/daisycloversoftware/rum-'+c+r'@sha256:[a-f0-9]{64})$',o);assert len(a)==1;(p/c).write_text(a[0]);print('ACCEPTING_ROLLOUT_IMAGE',c,a[0])
PY
k(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ $(k -n rum-dev-isolated get deploy rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}') == "$(cat "$t/api")" ]]
[[ $(k -n rum-rate-anything-preview get deploy rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}') == "$(cat "$t/rate-anything")" ]]
[[ $(curl -fsS --retry 2 --max-time 25 https://dev-rum-ra.daisycloversoftware.uk/VERSION) == "rat-candidate+$sha" ]]
curl -fsS --retry 2 --max-time 25 https://dev-rum.daisycloversoftware.uk/api/v1/health | python3 -c 'import json,sys;d=json.load(sys.stdin);assert d["status"]=="ok" and sys.argv[1] in d["version"];print("ACCEPTING_API",d["version"])' "$sha"
bash "$ops/check-rat181-protected-baseline.sh" >"$t/before"
mkdir "$t/source";git -C "$t/source" init -q;git -C "$t/source" remote add origin https://github.com/DaisyCloverSoftware/rum.git;git -C "$t/source" fetch -q --depth=24 origin "$sha";git -C "$t/source" checkout -q --detach FETCH_HEAD
base=b135949c8d97072cca3b24c2611984ae6d6fc30c
git -C "$t/source" diff --exit-code "$base" "$sha" -- apps/rate-anything/src/RatBrand.tsx apps/rate-anything/src/brandAssets.ts apps/rate-anything/src/RumThumbSelector.tsx apps/rate-anything/src/rat-explorer.css apps/rate-anything/src/rat-results-mascot.css apps/rate-anything/public/brand infra/
echo APPROVED_BRANDING_LAYOUT_THUMBS_AND_INFRA_UNCHANGED
cd "$t/source/apps/rate-anything";npm ci --ignore-scripts --no-audit --no-fund >"$t/npm.log" 2>&1;npx playwright install chromium >"$t/browser-install.log" 2>&1
set +e
node --input-type=module - "$sha" <<'JS'
import {chromium,expect} from '@playwright/test';import assert from 'node:assert/strict';
const sha=process.argv[2],rat='https://dev-rum-ra.daisycloversoftware.uk',dev='https://dev-rum.daisycloversoftware.uk';
const person='01m0mw9hkbynssr7kbc0gk534j',main='01m7x6ez57zpsgbv486y9t0y9h',second='01m81mbmqvwr3d1nvfp9gkp3xf',knownSecondEdge='01m81n8fsgxf1p52bw62e9rzyp';
const browser=await chromium.launch({headless:true,args:['--no-sandbox','--disable-dev-shm-usage']});const page=await browser.newPage({viewport:{width:1440,height:1000},serviceWorkers:'block'});page.setDefaultTimeout(30000);page.setDefaultNavigationTimeout(40000);const writes=[];const evidence={sha,chromium:browser.version(),person,main,second};
await page.route('**/*',r=>{const q=r.request();if(!['GET','HEAD','OPTIONS'].includes(q.method())){writes.push({method:q.method(),path:new URL(q.url()).pathname});return r.abort();}return [rat,dev].includes(new URL(q.url()).origin)?r.continue():r.abort();});
async function node(id){const r=await page.request.get(dev+'/api/v1/public/entities/rat-catalogue/nodes/'+id);assert.equal(r.status(),200);const j=await r.json();assert.equal(j.data.active.id,id);assert.equal(j.meta.privacy,'private-rum-people-excluded');return j.data;}
function path(data,id,other){assert.equal(data.categories.reduce((n,c)=>n+c.total,0),1);assert.equal(data.categories[0].key,'public-personas');assert.equal(data.categories[0].items[0].id,person);assert.equal(data.relatedThrough.length,1);const g=data.relatedThrough[0];assert.equal(g.via.id,person);assert.equal(g.label,'Other channels connected through Colin Furze');assert.equal(g.total,1);assert.equal(g.items.length,1);const item=g.items[0];assert.equal(item.id,other);assert.notEqual(item.id,id);assert.ok(!('relationship' in item));assert.equal(item.path.length,2);assert.equal(new Set(item.path.map(e=>e.id)).size,2);assert.ok(item.path.some(e=>e.id===knownSecondEdge));for(const edge of item.path){assert.equal(edge.type,'related_to');assert.ok(['verified','unverified'].includes(edge.verificationState));assert.ok([id,person,other].includes(edge.sourceId)&&[id,person,other].includes(edge.targetId));}return item.path;}
async function workspace(id,other){await expect(page.locator('.rat-workspace-id')).toHaveText(id);await expect(page.locator('#connections [aria-label="Direct public connections"]')).toHaveText('1 direct public');const direct=page.locator('#connections [data-connection-kind="direct"]');await expect(direct).toHaveCount(1);await expect(direct.locator('.rat-workspace-connection-group__head strong')).toHaveText('Public personas');const derived=page.getByRole('region',{name:'Indirect connected Things'});await expect(derived).toBeVisible();await expect(derived.getByText('Other channels connected through Colin Furze',{exact:true})).toBeVisible();await expect(derived.getByText(/not direct links or verified shared ownership/)).toBeVisible();await expect(derived.locator('[data-entity-id]')).toHaveCount(1);await expect(derived.locator(`[data-entity-id="${other}"]`)).toBeVisible();await expect(page.locator('.rat-wordmark-compact')).toBeVisible();await expect(page.locator('.rum-rating img').first()).toHaveAttribute('alt','RUM Thumb');assert.ok(await page.locator('.rum-rating img').first().evaluate(img=>img.complete&&img.naturalWidth>0));return derived;}
try{
const a=await node(main),b=await node(second);evidence.mainPath=path(a,main,second);evidence.secondPath=path(b,second,main);assert.deepEqual(new Set(evidence.mainPath.map(e=>e.id)),new Set(evidence.secondPath.map(e=>e.id)));const p=await node(person);assert.equal(p.categories.reduce((n,c)=>n+c.total,0),2);assert.equal(p.relatedThrough.length,0);assert.equal(p.active.ratingOwner,'rum');assert.equal(a.active.ratingOwner,'rat');assert.equal(b.active.ratingOwner,'rat');
await page.goto(rat+'/e/thing~'+main);let derived=await workspace(main,second);await derived.locator(`[data-entity-id="${second}"]`).click();derived=await workspace(second,main);assert.ok(page.url().endsWith('~'+second));await page.reload();derived=await workspace(second,main);await derived.locator(`[data-entity-id="${main}"]`).click();await workspace(main,second);await page.reload();derived=await workspace(main,second);await derived.getByRole('button',{name:'View connecting Thing: Colin Furze',exact:true}).click();await expect(page.locator('.rat-workspace-id')).toHaveText(person);await expect(page.locator('#connections [aria-label="Direct public connections"]')).toHaveText('2 direct public');await expect(page.getByRole('region',{name:'Indirect connected Things'})).toHaveCount(0);await page.goBack();await workspace(main,second);await page.goForward();await expect(page.locator('.rat-workspace-id')).toHaveText(person);evidence.workspaceReciprocalRefreshHistory=true;
for(const [id,other,otherName] of [[main,second,'2 Much ColinFurze'],[second,main,'colinfurze']]){
 await page.goto(rat+'/search?q=Colin%20Furze&active='+id);await expect(page.locator('.rat-active-card__type')).toHaveText('YouTube Channel');let section=page.locator('section[data-connection-kind="indirect"]');await expect(section).toHaveCount(1);await expect(section.getByRole('heading',{name:'Other channels connected through Colin Furze',exact:true})).toBeVisible();await expect(section.locator('.rat-linked-card')).toHaveCount(1);await expect(section.locator('.rat-linked-card strong')).toHaveText(otherName);await expect(page.locator('section[data-connection-kind="direct"] h2')).toHaveText('Public personas');await expect(page.locator('.rat-results-mascot')).toBeVisible();assert.equal(await page.locator('.rat-active-card .rat-results-mascot').count(),0);await section.locator('.rat-linked-card').click();await expect.poll(()=>new URL(page.url()).searchParams.get('active')).toBe(other);await expect(page.locator('.rat-active-card h1')).toHaveText(otherName);await page.reload();await expect(page.locator('.rat-active-card h1')).toHaveText(otherName);await expect(page.locator('section[data-connection-kind="indirect"] .rat-linked-card')).toHaveCount(1);
}
evidence.searchReciprocalRefresh=true;evidence.directCounts={main:1,second:1,person:2};evidence.brandAndThumbsLoaded=true;assert.deepEqual(writes,[]);evidence.attemptedWrites=0;console.log('RAT182_PUBLIC_CHROMIUM_ACCEPTANCE_PASS '+JSON.stringify(evidence));
}catch(e){console.log('PUBLIC_BROWSER_FAILURE',new URL(page.url()).pathname,(await page.locator('body').innerText()).slice(-1800));throw e;}finally{await browser.close();}
JS
rc=$?;set -e
bash "$ops/check-rat181-protected-baseline.sh" >"$t/after"
for f in before after;do grep -E '^((entities|entity_identifiers|rating_events|relationships) rows=|AUDIENCE |ROUTING )' "$t/$f" >"$t/$f.compare";done
diff -u "$t/before.compare" "$t/after.compare";cat "$t/after.compare"
echo RAT182_COLIN_RECORDS_RATINGS_IDENTIFIERS_AND_EDGES_UNCHANGED
[[ $rc == 0 ]] || exit "$rc"
echo "RAT182_DEPLOYED_BROWSER_ACCEPTANCE_PASS sha=$sha rollout=$release canonical_writes=0"
