#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]]||exit 64
sha=349d8178c4da2daa82434a3e3c37dad38c5aebeb;ops=$(cd "$(dirname "$0")" && pwd);t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
[[ $(curl -fsS --retry 2 --max-time 25 https://dev-rum-ra.daisycloversoftware.uk/VERSION) == "rat-candidate+$sha" ]]||exit 78
bash "$ops/check-rat181-protected-baseline.sh" >"$t/before"
mkdir "$t/source";git -C "$t/source" init -q;git -C "$t/source" remote add origin https://github.com/DaisyCloverSoftware/rum.git;git -C "$t/source" fetch -q --depth=1 origin "$sha";git -C "$t/source" checkout -q --detach FETCH_HEAD;cd "$t/source/apps/rate-anything";npm ci --ignore-scripts --no-audit --no-fund >"$t/npm" 2>&1;npx playwright install chromium >"$t/pw" 2>&1
node --input-type=module - "$sha" <<'JS'
import {chromium,expect} from '@playwright/test';import assert from 'node:assert/strict';
const sha=process.argv[2],rat='https://dev-rum-ra.daisycloversoftware.uk',dev='https://dev-rum.daisycloversoftware.uk';
const person='01m0mw9hkbynssr7kbc0gk534j',main='01m7x6ez57zpsgbv486y9t0y9h',second='01m81mbmqvwr3d1nvfp9gkp3xf';
const browser=await chromium.launch({headless:true,args:['--no-sandbox','--disable-dev-shm-usage']});const page=await browser.newPage({viewport:{width:1440,height:1000},serviceWorkers:'block'});page.setDefaultTimeout(30000);page.setDefaultNavigationTimeout(45000);const writes=[];
async function node(id){const r=await page.request.get(dev+'/api/v1/public/entities/rat-catalogue/nodes/'+id);assert.equal(r.status(),200);return (await r.json()).data;}
const data=await node(main);assert.equal(data.active.id,main);const profile=data.active.images?.profile;assert.equal(profile?.status,'published');assert.ok(profile.url);const photo=new URL(profile.url,rat).href;
await page.route('**/*',r=>{const q=r.request(),u=new URL(q.url());if(!['GET','HEAD','OPTIONS'].includes(q.method())){writes.push({method:q.method(),path:u.pathname});return r.abort();}return ([rat,dev].includes(u.origin)||(q.resourceType()==='image'&&u.href===photo))?r.continue():r.abort();});
const direct=()=>page.locator('#connections [data-connection-kind="direct"] .rat-workspace-connection').filter({has:page.locator('strong').filter({hasText:/^colinfurze$/})});
const indirect=()=>page.getByRole('region',{name:'Indirect connected Things'}).locator(`[data-entity-id="${main}"]`);
async function loaded(row,workspace=true){await expect(row).toHaveCount(1);const img=row.locator(workspace?'.rat-workspace-connection__mark img':'.rat-linked-card__media img');await expect(img).toHaveAttribute('src',profile.url);await expect(img).toBeVisible();await expect.poll(()=>img.evaluate(e=>e.complete&&e.naturalWidth>0)).toBe(true);if(workspace){const box=await row.locator('.rat-workspace-connection__mark').boundingBox();assert.equal(box.width,42);assert.equal(box.height,42);}return img;}
async function workspace(id){await page.goto(rat+'/e/thing~'+id+'#connections');await expect(page.locator('.rat-workspace-id')).toHaveText(id);}
const evidence={sha,chromium:browser.version(),profileRevision:profile.revisionId,profileStatus:profile.status,main,person,second};
try{
const p=await node(person),s=await node(second);const directItem=p.categories.flatMap(c=>c.items).find(i=>i.id===main);const indirectItem=s.relatedThrough.flatMap(g=>g.items).find(i=>i.id===main);assert.equal(directItem.images.profile.url,profile.url);assert.equal(indirectItem.images.profile.url,profile.url);
await workspace(person);await loaded(direct());await expect(page.locator('#connections [aria-label="Direct public connections"]')).toHaveText('2 direct public');await page.reload();await loaded(direct());await direct().click();await expect(page.locator('.rat-workspace-id')).toHaveText(main);await expect.poll(()=>page.locator('img').evaluateAll((imgs,url)=>imgs.some(i=>i.src===url&&i.complete&&i.naturalWidth>0),photo)).toBe(true);await page.goBack();await loaded(direct());evidence.directPublishedPhotoRefreshNavigation=true;
await workspace(second);await loaded(indirect());await expect(page.getByRole('region',{name:'Indirect connected Things'})).toContainText('Indirect through Colin Furze');await expect(page.locator('#connections [aria-label="Direct public connections"]')).toHaveText('1 direct public');await page.reload();await loaded(indirect());await indirect().click();await expect(page.locator('.rat-workspace-id')).toHaveText(main);await page.goBack();await loaded(indirect());evidence.indirectPublishedPhotoRefreshNavigation=true;
for(const [id,kind] of [[person,'direct'],[second,'indirect']]){await page.goto(rat+'/search?q=Colin%20Furze&active='+id);const row=page.locator(`section[data-connection-kind="${kind}"] .rat-linked-card`).filter({has:page.locator('strong').filter({hasText:/^colinfurze$/})});await loaded(row,false);await page.reload();await loaded(row,false);await row.click();await expect.poll(()=>new URL(page.url()).searchParams.get('active')).toBe(main);}
evidence.searchAndLinkedCategoriesPreserved=true;
// Browser-local network fault: no image or canonical metadata is modified.
const imageRequest=u=>u.href===photo;await page.route(imageRequest,r=>r.abort());await workspace(second);await expect(indirect().locator('.rat-workspace-connection__mark')).toHaveText('C');await expect(indirect().locator('.rat-workspace-connection__mark img')).toHaveCount(0);await indirect().click();await expect(page.locator('.rat-workspace-id')).toHaveText(main);await page.unroute(imageRequest);await workspace(second);await loaded(indirect());evidence.imageFailureFallsBackAndReloadRecovers=true;
// Exercise the client guard using the already-public photograph with simulated ineligible metadata.
for(const status of ['pending','private','rejected','unpublished']){const path=u=>u.href===dev+'/api/v1/public/entities/rat-catalogue/nodes/'+person;await page.route(path,async r=>{const response=await r.fetch();const json=await response.json();function walk(v){if(!v||typeof v!=='object')return;if(v.id===main&&v.images?.profile)v.images.profile.status=status;for(const x of Object.values(v))walk(x);}walk(json);await r.fulfill({response,json});});await workspace(person);await expect(direct().locator('.rat-workspace-connection__mark')).toHaveText('C');await expect(direct().locator('.rat-workspace-connection__mark img')).toHaveCount(0);await page.unroute(path);}
evidence.clientIneligibleStatusFallbacks=['pending','private','rejected','unpublished'];await workspace(person);await loaded(direct());
await expect(page.locator('.rat-wordmark-compact')).toBeVisible();await expect(page.locator('.rum-rating img').first()).toHaveAttribute('alt','RUM Thumb');await expect.poll(()=>page.locator('.rum-rating img').first().evaluate(i=>i.complete&&i.naturalWidth>0)).toBe(true);assert.deepEqual(writes,[]);evidence.brandAndThumbsPreserved=true;evidence.httpWrites=0;
const rr=await page.request.get(dev+'/api/v1/public/entities/rat-catalogue/nodes/'+main+'/ratings?offset=0&limit=20');const ratings=(await rr.json()).data;evidence.ratingCleanupNotPerformed=ratings.items.some(x=>x.id==='01m7x799g070af1g4je8hh468v');
console.log('PR183_PUBLIC_LINKED_THUMBNAIL_CHROMIUM_PASS',JSON.stringify(evidence));
}finally{await browser.close();}
JS
bash "$ops/check-rat181-protected-baseline.sh" >"$t/after";for f in before after;do grep -E '^((entities|entity_identifiers|rating_events|relationships) rows=|AUDIENCE |ROUTING )' "$t/$f" >"$t/$f.compare";done;diff -u "$t/before.compare" "$t/after.compare";cat "$t/after.compare"
echo PR183_THUMBNAILS_ACCEPTED_NO_CANONICAL_DATA_WRITES
