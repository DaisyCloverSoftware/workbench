#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]] || exit 64
# Acceptance only: no application writes, deployments, cleanup or rating operations.
sha=bb7dc6f8659b21fe612c1a003b8054b29a8317e9
api=ghcr.io/daisycloversoftware/rum-api@sha256:5e8486292d3f1b0ab0b834cd98028892ccf73a600548234d737ea7402e60d336
rat=ghcr.io/daisycloversoftware/rum-rate-anything@sha256:fe9a94d119be941479a655858e548b1cdb5b1f1020b9cae54a2de7985db58081
ops=$(cd "$(dirname "$0")" && pwd); t=$(mktemp -d); umask 077; trap 'rm -rf "$t"' EXIT
k(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ $(k -n rum-dev-isolated get namespace rum-dev-isolated -o 'jsonpath={.metadata.uid}') == 6804e79b-3000-42c1-b37e-f07a57252425 ]] || exit 78
[[ $(k -n rum-dev-isolated get deploy rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}') == "$api" ]] || exit 78
[[ $(k -n rum-rate-anything-preview get deploy rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}') == "$rat" ]] || exit 78
[[ $(curl -fsS --max-time 25 https://dev-rum-ra.daisycloversoftware.uk/VERSION) == "rat-candidate+$sha" ]] || exit 78
bash "$ops/check-rat181-protected-baseline.sh" >"$t/before"
# Read public CORS configuration and route middleware; never print credentials.
k -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php <<'PHP'
<?php
require 'vendor/autoload.php';$app=require 'bootstrap/app.php';$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
echo 'PUBLIC_CORS_RUNTIME '.json_encode(['paths'=>config('cors.paths'),'origins'=>config('cors.allowed_origins'),'credentials'=>config('cors.supports_credentials')]).PHP_EOL;
foreach(app('router')->getRoutes() as $route){if(str_contains($route->uri(),'rat-catalogue/nodes'))echo 'CANONICAL_READ_MIDDLEWARE '.json_encode(['uri'=>$route->uri(),'middleware'=>$route->gatherMiddleware()]).PHP_EOL;}
PHP
mkdir "$t/source";git -C "$t/source" init -q;git -C "$t/source" remote add origin https://github.com/DaisyCloverSoftware/rum.git;git -C "$t/source" fetch -q --depth=1 origin "$sha";git -C "$t/source" checkout -q --detach FETCH_HEAD
cd "$t/source/apps/rate-anything";npm ci --ignore-scripts --no-audit --no-fund >"$t/npm" 2>&1;npx playwright install chromium >"$t/browser-install" 2>&1
set +e
node --input-type=module - "$sha" <<'JS'
import {chromium,expect as baseExpect} from '@playwright/test';import assert from 'node:assert/strict';
const expect=baseExpect.configure({timeout:20000});
const sha=process.argv[2],rat='https://dev-rum-ra.daisycloversoftware.uk',dev='https://dev-rum.daisycloversoftware.uk';
const person='01m0mw9hkbynssr7kbc0gk534j',main='01m7x6ez57zpsgbv486y9t0y9h',second='01m81mbmqvwr3d1nvfp9gkp3xf';
const browser=await chromium.launch({headless:true,args:['--no-sandbox','--disable-dev-shm-usage']});
const writes=[],unexpectedFailures=[],httpErrors=[],apiEvidence=[],blocked=[];
let stage='baseline',failurePage,blockEnabled=false,photo,photoURL;
function publicHeaders(headers){return Object.fromEntries(Object.entries(headers).filter(([k])=>['access-control-allow-origin','access-control-allow-credentials','vary','retry-after','x-ratelimit-limit','x-ratelimit-remaining','server','cf-ray','content-type'].includes(k.toLowerCase())));}
function exactImage(q){const u=new URL(q.url());return photoURL&&q.resourceType()==='image'&&['https:','http:'].includes(u.protocol)&&u.hostname===photoURL.hostname&&u.pathname===photoURL.pathname&&u.search===photoURL.search;}
async function context(label){
 const c=await browser.newContext({viewport:{width:1440,height:1000},serviceWorkers:'block'});
 c.setDefaultTimeout(20000);c.setDefaultNavigationTimeout(35000);
 // Do not proxy, fetch or fulfil canonical API responses. Non-image GETs pass unchanged.
 await c.route('**/*',r=>{const q=r.request();if(!['GET','HEAD','OPTIONS'].includes(q.method())){writes.push({label,method:q.method(),path:new URL(q.url()).pathname});return r.abort();}return r.continue();});
 c.on('response',r=>{const u=new URL(r.url());if(u.origin===dev&&u.pathname.includes('/rat-catalogue/')){const e={label,path:u.pathname,status:r.status(),headers:publicHeaders(r.headers())};apiEvidence.push(e);if(r.status()>=400){httpErrors.push(e);console.log('CANONICAL_HTTP_ERROR',JSON.stringify(e));}}});
 c.on('requestfailed',q=>{const u=new URL(q.url()),e={label,path:u.pathname,type:q.resourceType(),error:q.failure()?.errorText};if(!(blockEnabled&&exactImage(q))&&q.failure()?.errorText!=='net::ERR_ABORTED'){unexpectedFailures.push(e);console.log('UNEXPECTED_NETWORK_FAILURE',JSON.stringify(e));}});
 return c;
}
async function page(c,label){const p=await c.newPage();const cd=await c.newCDPSession(p);await cd.send('Network.enable');const requests=new Map();cd.on('Network.requestWillBeSent',e=>requests.set(e.requestId,e.request.url));cd.on('Network.responseReceivedExtraInfo',e=>{const u=requests.get(e.requestId);if(u&&new URL(u).origin===dev&&new URL(u).pathname.includes('/rat-catalogue/')&&e.statusCode>=400)console.log('RAW_CANONICAL_HTTP',JSON.stringify({label,path:new URL(u).pathname,status:e.statusCode,headers:publicHeaders(e.headers)}));});cd.on('Network.loadingFailed',e=>{if(e.corsErrorStatus)console.log('CDP_CORS_FAILURE',JSON.stringify({label,url:requests.get(e.requestId),error:e.errorText,cors:e.corsErrorStatus}));});p.on('console',m=>{if(m.type()==='error'&&/CORS|Failed to fetch/.test(m.text()))console.log('BROWSER_CORS',label,m.text().slice(0,800));});return p;}
function row(p,kind){return kind==='direct'?p.locator('#connections [data-connection-kind="direct"] .rat-workspace-connection').filter({has:p.locator('strong').filter({hasText:/^colinfurze$/})}):p.getByRole('region',{name:'Indirect connected Things'}).locator(`[data-entity-id="${main}"]`);}
async function ready(p,id){await expect(p.locator('.rat-workspace-id')).toHaveText(id);}
async function open(p,id){await p.goto(rat+'/e/thing~'+id+'#connections',{waitUntil:'domcontentloaded'});await ready(p,id);}
async function loaded(r){await expect(r).toHaveCount(1);const mark=r.locator('.rat-workspace-connection__mark'),img=mark.locator('img');await r.scrollIntoViewIfNeeded();await expect(img).toHaveAttribute('src',photo);await expect(img).toBeVisible();await expect.poll(()=>img.evaluate(i=>i.complete&&i.naturalWidth>0)).toBe(true);const b=await mark.boundingBox();assert.equal(b.width,42);assert.equal(b.height,42);}
async function initials(r){await expect(r).toHaveCount(1);await r.scrollIntoViewIfNeeded();const mark=r.locator('.rat-workspace-connection__mark');await expect(mark).toHaveText('C');await expect(mark.locator('img')).toHaveCount(0);const b=await mark.boundingBox();assert.equal(b.width,42);assert.equal(b.height,42);}
const evidence={sha,chromium:browser.version(),main,person,second};
try{
 const normal=await context('normal');const control=await page(normal,'normal');
 const meta=await normal.request.get(dev+'/api/v1/public/entities/rat-catalogue/nodes/'+main,{headers:{Origin:rat,Accept:'application/json'},timeout:25000});
 console.log('UNMODIFIED_ORIGIN_GET',JSON.stringify({status:meta.status(),headers:publicHeaders(meta.headers())}));assert.equal(meta.status(),200);assert.equal(meta.headers()['access-control-allow-origin'],rat);
 const j=(await meta.json()).data;assert.equal(j.active.id,main);assert.equal(j.active.images?.profile?.status,'published');photo=j.active.images.profile.url;photoURL=new URL(photo,rat);evidence.profilePath=photoURL.pathname;
 await open(control,person);await loaded(row(control,'direct'));await expect(control.locator('#connections [aria-label="Direct public connections"]')).toHaveText('2 direct public');
 await row(control,'direct').click();await ready(control,main);await control.reload({waitUntil:'domcontentloaded'});await ready(control,main);
 await open(control,second);await loaded(row(control,'indirect'));await expect(control.locator('#connections [aria-label="Direct public connections"]')).toHaveText('1 direct public');
 assert.equal(httpErrors.length,0);assert.equal(unexpectedFailures.length,0);evidence.normalNavigationRefresh=true;
 console.log('NORMAL_PUBLIC_NAVIGATION_API_AND_PHOTOGRAPHS_PASS',JSON.stringify({apiRequests:apiEvidence.length,apiStatuses:[...new Set(apiEvidence.map(x=>x.status))],apiOrigins:[...new Set(apiEvidence.map(x=>x.headers['access-control-allow-origin']))]}));
 stage='isolated-image-failure';const isolated=await context('image-failure');failurePage=await page(isolated,'image-failure');
 // A separate clean context. Only this exact image resource is failed; not a URL prefix or API endpoint.
 const match=u=>['http:','https:'].includes(u.protocol)&&u.hostname===photoURL.hostname&&u.pathname===photoURL.pathname&&u.search===photoURL.search;
 await failurePage.route(match,r=>{const q=r.request();if(blockEnabled&&exactImage(q)){const e={path:new URL(q.url()).pathname,type:q.resourceType(),method:q.method()};blocked.push(e);console.log('BLOCKED_ONLY_INTENDED_IMAGE',JSON.stringify(e));return r.abort('failed');}return r.continue();});
 blockEnabled=true;await open(failurePage,second);await initials(row(failurePage,'indirect'));assert.ok(blocked.length>0);
 // Confirm an independently loaded real published photograph remains visible, without modifying metadata.
 await loaded(row(control,'indirect'));
 console.log('INDIRECT_FAILED_IMAGE_INITIALS_PASS',JSON.stringify({fallback:'C',blocked:blocked.length,unaffectedPublishedPhotoVisible:true}));
 await open(failurePage,person);await initials(row(failurePage,'direct'));await loaded(row(control,'indirect'));
 console.log('DIRECT_FAILED_IMAGE_INITIALS_PASS',JSON.stringify({fallback:'C',blocked:blocked.length,unaffectedPublishedPhotoVisible:true}));
 // Click-through still works while the image alone is unavailable.
 await row(failurePage,'direct').click();await ready(failurePage,main);
 evidence.failedImageDirectAndIndirect=true;evidence.unaffectedRealPublishedImage=true;evidence.navigationDuringImageFailure=true;
 blockEnabled=false;await failurePage.unroute(match);
 stage='recovery';await open(failurePage,second);await loaded(row(failurePage,'indirect'));await failurePage.reload({waitUntil:'domcontentloaded'});await ready(failurePage,second);await loaded(row(failurePage,'indirect'));await row(failurePage,'indirect').click();await ready(failurePage,main);await open(failurePage,person);await loaded(row(failurePage,'direct'));
 await expect(failurePage.locator('.rat-wordmark-compact')).toBeVisible();await expect(failurePage.locator('.rum-rating img').first()).toHaveAttribute('alt','RUM Thumb');
 assert.deepEqual(writes,[]);assert.deepEqual(httpErrors,[]);assert.deepEqual(unexpectedFailures,[]);
 assert.ok(apiEvidence.length>0);assert.ok(apiEvidence.every(x=>x.status===200));assert.ok(apiEvidence.filter(x=>x.label==='image-failure').length>0);
 evidence.recoveryRefreshNavigation=true;evidence.blockedImageRequests=blocked.length;evidence.apiRequests=apiEvidence.length;evidence.apiStatuses=[...new Set(apiEvidence.map(x=>x.status))];evidence.httpWrites=0;evidence.apiFailures=0;evidence.applicationChanges=0;
 console.log('PR183_ISOLATED_FAILED_IMAGE_FALLBACK_ACCEPTANCE_PASS',JSON.stringify(evidence));
}catch(error){console.log('FINAL_ACCEPTANCE_STAGE',stage);console.log('NETWORK_DIAGNOSTICS',JSON.stringify({blocked,unexpectedFailures,httpErrors,apiEvidence}));if(failurePage)console.log('VISIBLE_STATE',(await failurePage.locator('body').innerText().catch(()=>'' )).slice(-1000));throw error;}finally{await browser.close();}
JS
rc=$?;set -e
bash "$ops/check-rat181-protected-baseline.sh" >"$t/after"
for f in before after;do grep -E '^((entities|entity_identifiers|rating_events|relationships) rows=|AUDIENCE |ROUTING )' "$t/$f" >"$t/$f.compare";done
diff -u "$t/before.compare" "$t/after.compare";cat "$t/after.compare"
echo PR183_FINAL_ACCEPTANCE_PROTECTED_RECORDS_AND_ROUTING_UNCHANGED
[[ $rc == 0 ]] || exit "$rc"
echo PR183_FINAL_THUMBNAIL_FALLBACK_COMPLETE_NO_APPLICATION_OR_DATA_MUTATIONS
