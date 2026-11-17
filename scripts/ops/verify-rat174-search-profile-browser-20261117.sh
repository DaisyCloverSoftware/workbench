#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only RAT174 search-profile acceptance using canonical published image data.
# No user, rating, revision, image, project or LIVE mutations.
[[ $# -eq 3 ]] || { echo 'usage: source-sha canonical-api@sha256 rat-image@sha256' >&2; exit 64; }
src="$1"; api="$2"; rat="$3"
[[ "$src" == "cd1910ba808919d74f748456fd04202da052ddf6" ]] || exit 78
[[ "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || exit 64
[[ "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 64
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
CHROME="$chrome" OUT="$tmp" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import { chromium, expect } from '@playwright/test'
const RAT='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk'
const ID='01m7bswfh0a3evf271ye4taaf0'
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
try{
 for(const width of [1440,390]){
  const ctx=await browser.newContext({viewport:{width,height:900}})
  const response=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+ID)
  assert.equal(response.status(),200,'Canonical public Thing unavailable')
  const payload=await response.json()
  const active=payload.data?.active
  assert.equal(active?.id,ID)
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
  assert.deepEqual(errors,[],'Unexpected RAT browser errors')
  await page.screenshot({path:process.env.OUT+'/rat174-search-'+width+'.png',fullPage:true,animations:'disabled'})
  console.log('RAT174_SEARCH_PROFILE_BROWSER_PASS='+JSON.stringify({width,active:active.name,url:expected,linked:linked.length,visual,errors:errors.length}))
  await ctx.close()
 }
}finally{await browser.close()}
NODE
echo RAT174_SEARCH_PROFILE_BROWSER_ACCEPTANCE_PASS=true
