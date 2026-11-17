#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only RAT174 search-profile acceptance using canonical published image data.
# No user, rating, revision, image, project or LIVE mutations.
[[ $# -eq 4 ]] || { echo 'usage: source-sha canonical-api@sha256 rat-image@sha256 unique-run-id' >&2; exit 64; }
src="$1"; api="$2"; rat="$3"; rid="$4"
[[ "$rid" =~ ^[a-z0-9-]{12,44}$ ]] || exit 64
out="/tmp/rat174-search-profile-$rid"
[[ ! -e "$out" ]] || { echo "Blocked: duplicate screenshot run id; inspect existing result" >&2; exit 65; }
mkdir -m 700 "$out"
[[ "$src" == "cd1910ba808919d74f748456fd04202da052ddf6" ]] || exit 78
[[ "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$api" == "ghcr.io/daisycloversoftware/rum-api@sha256:5fc614b352a596e41d35a33f55b9f4b01d4c6c2bfbe2d7dda700d92267761a13" ]] || exit 78
[[ "$rat" == "ghcr.io/daisycloversoftware/rum-rate-anything@sha256:31651a79f1bd59c8ff6c04a627e392a10b7756a7579633be81230cf96673ad7f" ]] || exit 78
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --retry 4 --retry-delay 2 --max-time 20 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$src" ]] || exit 78
health="$(curl -fsS --retry 4 --retry-delay 2 --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$health" "$src" <<'PY'
import json,sys
obj=json.loads(sys.argv[1])
assert obj.get('status')=='ok' and obj.get('service')=='rum-api'
assert sys.argv[2] in str(obj.get('version',''))
PY
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$src"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$src" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for p in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser;do
  if [[ -x "$p" ]];then chrome="$p";break;fi
done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
CHROME="$chrome" OUT="$out" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { chromium, expect } from '@playwright/test'
import { stat } from 'node:fs/promises'
const RAT='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk'
const ID='01m7bswfh0a3evf271ye4taaf0'
const capture=async(page,stem,fullPage=false)=>{const path=process.env.OUT+'/'+stem+'.jpg';await page.screenshot({path,type:'jpeg',quality:67,fullPage,animations:'disabled'});const bytes=(await stat(path)).size;assert.ok(bytes>1500&&bytes<750000,'Unexpected screenshot size');console.log('RAT174_ACTUAL_BROWSER_SCREENSHOT='+stem+'.jpg bytes='+bytes)}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
try{
 for(const width of [1440,390]){
  const ctx=await browser.newContext({viewport:{width,height:900}})
  const response=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+ID)
  assert.equal(response.status(),200,'Canonical public Thing unavailable')
  const payload=await response.json()
  const active=payload.data?.active
  assert.equal(active?.id,ID)
  assert.equal(payload.meta?.privacy,'private-rum-people-excluded')
  assert.equal(payload.meta?.audience,'rat-public')
  assert.equal(active?.images?.profile?.status,'published','Expected existing approved public profile')
  const expected=active.images.profile.url
  assert.ok(expected.includes('/api/v1/public/entities/rat-image/'),'Expected canonical public processed image')
  const page=await ctx.newPage()
  const errors=[]
  page.on('pageerror',e=>errors.push(e.message))
  await page.goto(RAT+'/search?q=professional&active='+ID,{waitUntil:'domcontentloaded',timeout:40000})
  await expect(page.locator('.rat-active-card h1')).toContainText('The Professional Idiots',{timeout:35000})
  const avatar=page.locator('.rat-active-card__avatar > img')
  await expect(avatar).toHaveAttribute('src',expected)
  await expect.poll(async()=>avatar.evaluate(el=>el instanceof HTMLImageElement && el.complete && el.naturalWidth>0 && el.naturalHeight>0),{timeout:20000}).toBe(true)
  const visual=await avatar.evaluate(el=>{const img=getComputedStyle(el),parent=getComputedStyle(el.parentElement);return {fit:img.objectFit,borderRadius:parent.borderRadius,overflow:parent.overflow}})
  assert.equal(visual.fit,'cover')
  assert.equal(visual.overflow,'hidden')
  const bounds=await avatar.evaluate(el=>{const r=el.parentElement.getBoundingClientRect();return {w:r.width,h:r.height}})
  assert.ok(Math.abs(bounds.w-(width>700?145:76))<=1,JSON.stringify(bounds))
  assert.ok(Math.abs(bounds.w-bounds.h)<=2,'Original circular avatar proportions changed')
  assert.ok((await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth))<=2,'Search document has horizontal overflow')
  const actualMedia=await ctx.request.get(expected)
  assert.equal(actualMedia.status(),200,'Published canonical profile media is not public')
  assert.ok((actualMedia.headers()['content-type']||'').startsWith('image/'),'Canonical media is not an image')
  const thing=await ctx.newPage()
  await thing.goto(RAT+'/e/the-professional-idiots~'+ID,{waitUntil:'domcontentloaded',timeout:40000})
  await expect(thing.locator('.rat-workspace-header h1')).toContainText('The Professional Idiots',{timeout:25000})
  const profile=thing.locator('.rat-workspace-avatar > img')
  await expect(profile).toHaveAttribute('src',expected)
  await expect.poll(async()=>profile.evaluate(el=>el.complete&&el.naturalWidth>0),{timeout:20000}).toBe(true)
  await thing.close()
  // The explorer must provide the same approved public image to compact linked cards.
  const linked=payload.data.topLinked||[]
  const cards=page.locator('.rat-top-linked .rat-linked-card')
  await expect(cards).toHaveCount(linked.length)
  for(let i=0;i<linked.length;i++){
    const item=linked[i],card=cards.nth(i)
    const url=item.images?.profile?.status==='published'?item.images.profile.url:null
    if(url){
      await expect(card.locator('.rat-linked-card__media > img')).toHaveAttribute('src',url)
      await expect.poll(async()=>card.locator('.rat-linked-card__media > img').evaluate(el=>el.complete&&el.naturalWidth>0),{timeout:20000}).toBe(true)
    }else{
      await expect(card.locator('.rat-linked-card__media > img')).toHaveCount(0)
      await expect(card.locator('.rat-linked-card__media')).toContainText(item.name.slice(0,2).toUpperCase())
    }
  }
  await capture(page,'rat174-search-'+width+'-full',true)
  await capture(page.locator('.rat-active-card'),'rat174-search-'+width+'-main-card')
  if(linked.length) await capture(page.locator('.rat-top-linked'),'rat174-search-'+width+'-linked-cards')
  await page.reload({waitUntil:'domcontentloaded'})
  await expect(page.locator('.rat-active-card h1')).toContainText('The Professional Idiots',{timeout:25000})
  await expect(page.locator('.rat-active-card__avatar > img')).toHaveAttribute('src',expected)
  await expect.poll(async()=>page.locator('.rat-active-card__avatar > img').evaluate(el=>el.complete&&el.naturalWidth>0),{timeout:20000}).toBe(true)
  if(linked.length){
    const item=linked[0]
    await page.locator('.rat-top-linked .rat-linked-card').first().click()
    await expect(page.locator('.rat-active-card h1')).toContainText(item.name,{timeout:25000})
    const linkedPic=item.images?.profile?.status==='published'?item.images.profile.url:null
    if(linkedPic)await expect(page.locator('.rat-active-card__avatar > img')).toHaveAttribute('src',linkedPic)
    else await expect(page.locator('.rat-active-card__avatar > img')).toHaveCount(0)
    await page.goBack({waitUntil:'domcontentloaded'})
    await expect(page.locator('.rat-active-card h1')).toContainText('The Professional Idiots',{timeout:25000})
    await expect(page.locator('.rat-active-card__avatar > img')).toHaveAttribute('src',expected)
  }
  const after=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+ID)
  assert.equal(after.status(),200)
  const afterActive=(await after.json()).data.active
  assert.equal(afterActive.images.profile.url,expected,'Owner published profile changed during read-only test')
  assert.deepEqual(afterActive.reputation,active.reputation,'Owner rating aggregate changed during read-only browser test')
  for(const [q,name] of [['DJI','DJI'],['CJ','CJ Investigates']]){
    await page.goto(RAT+'/search?q='+q,{waitUntil:'domcontentloaded',timeout:35000})
    await expect(page.locator('.rat-active-card h1')).toContainText(name,{timeout:25000})
  }
  assert.deepEqual(errors,[],'Unexpected RAT browser errors')
  console.log('RAT174_SEARCH_PROFILE_BROWSER_PASS='+JSON.stringify({width,active:active.name,url:expected,linked:linked.length,visual,bounds,privacy:payload.meta.privacy,errors:errors.length}))
  await ctx.close()
 }
}finally{await browser.close()}
NODE
echo RAT174_SEARCH_PROFILE_BROWSER_ACCEPTANCE_PASS=true
printf 'source=%s\napi=%s\nrat=%s\nstatus=PASS\n' "$src" "$api" "$rat" > "$out/accepted.txt"
echo "RAT174_ACCEPTED_BROWSER_SCREENSHOTS=$out"
