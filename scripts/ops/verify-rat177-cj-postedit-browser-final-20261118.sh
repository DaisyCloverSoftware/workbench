#!/usr/bin/env bash
set -Eeuo pipefail
# Post-transition Chromium acceptance of the existing CJ entity.
# Only authorized disposable RAT QA registration may POST; no CJ edits, ratings or votes.
[[ $# -eq 4 ]] || exit 64
src="$1";api="$2";rat="$3";rid="$4"
[[ "$src" == '887b5e02f4be6ec7e52371666effa9eb47892d6e' ]] || exit 78
[[ "$api" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[a-f0-9]{64}$ ]] || exit 64
[[ "$rat" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[a-f0-9]{64}$ ]] || exit 64
[[ "$api" == 'ghcr.io/daisycloversoftware/rum-api@sha256:c68f9e4902b7ab71abab2cf0ea50244bf80dd93809385add84b764ff1fc34d0c' ]] || exit 78
[[ "${rat#ghcr.io/daisycloversoftware/rum-rate-anything@sha256:}" =~ ^[a-f0-9]{64}$ ]] || exit 78
# RAT must be the new exact-head digest; the old deployed digest cannot satisfy this check.
[[ "$rat" != 'ghcr.io/daisycloversoftware/rum-rate-anything@sha256:6a506b08359eea423340c467ee6f330e22556c1a29002d0351323528123c8518' ]] || exit 78
[[ "$rid" =~ ^[a-z0-9-]{16,44}$ ]] || exit 64
out="/tmp/rat177-cj-postedit-$rid"
[[ ! -e "$out" ]] || exit 65
mkdir -m 700 "$out"
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --max-time 20 -H 'Cache-Control: no-cache' https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == "rat-candidate+$src" ]] || exit 78
health="$(curl -fsS --max-time 20 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$health" "$src" <<'PY'
import json,sys
obj=json.loads(sys.argv[1]); assert obj["status"]=="ok" and obj["service"]=="rum-api"
assert '628296e5c57665814799bfedf1ab3ef8f6f163df' in obj["version"]
PY
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
# Keep Chromium user-data profiles inside this writable isolated run directory.
export TMPDIR="$tmp"
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
CHROME="$chrome" READ_OUT="$out" READ_RUN="$rid" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import {writeFile,stat} from 'node:fs/promises'
import {randomBytes} from 'node:crypto'
import {chromium,expect} from '@playwright/test'
const SITE='https://dev-rum-ra.daisycloversoftware.uk',DEV='https://dev-rum.daisycloversoftware.uk',CJ='01m07d2f9fj05f4ypdbwp087qj',KEY='community_788e6ae3bfcd391d111e3ca31a1332ebf92906d0d0868e7cf7bb'
const report={run:process.env.READ_RUN,status:'NOT_ASSESSED',checks:[],screenshots:[],errors:[]}
const pass=x=>{report.checks.push(x);console.log('RAT177_PUBLIC_HEADER_PASS='+x)}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const ctx=await browser.newContext({viewport:{width:1440,height:940}})
const page=await ctx.newPage()
await page.route('**/*',r=>{
 const q=r.request(),url=new URL(q.url())
 const allow=['GET','HEAD','OPTIONS'].includes(q.method())||
  (q.method()==='POST'&&url.origin===SITE&&url.pathname==='/api/v1/rate-anything/auth/register')
 return allow?r.continue():r.abort('blockedbyclient')
})
let stage='starting'
try{
 const response=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+CJ)
 assert.equal(response.status(),200)
 const node=(await response.json()).data.active
 assert.equal(node.id,CJ)
 assert.equal(node.entityType,'digital_resource')
 assert.equal(node.classifications.find(c=>c.primary)?.name,'YouTube Channel')
 assert.equal(node.classifications.find(c=>c.primary)?.key,KEY)
 assert.ok(node.images.profile.url&&node.images.cover.url)
 const original={profile:node.images.profile.url,cover:node.images.cover.url,reputation:JSON.stringify(node.reputation)}
 pass('canonical-CJ-digital-resource-with-primary-YouTube-Channel')
 for(const width of [1440,768,390]){
  stage='header-'+width
  await page.setViewportSize({width,height:width===390?860:940})
  await page.goto(SITE+'/e/cj-investigates~'+CJ,{waitUntil:'domcontentloaded',timeout:40000})
  await expect(page.locator('.rat-workspace-header h1')).toHaveText('CJ Investigates',{timeout:30000})
  await expect(page.locator('.rat-workspace-kickers')).toContainText('YouTube Channel')
  const typeCell=page.locator('.rat-workspace-profile .rat-workspace-facts > div').filter({has:page.locator('dt',{hasText:'Type of Thing'})}).first()
  await expect(typeCell.locator('dd')).toHaveText('YouTube Channel')
  await expect(page.locator('.rat-workspace-avatar img')).toHaveAttribute('src',original.profile)
  await expect(page.locator('.rat-workspace-cover img')).toHaveAttribute('src',original.cover)
  const avatar=page.locator('.rat-workspace-avatar > img')
  const cover=page.locator('.rat-workspace-cover > img')
  await expect(avatar).toBeVisible({timeout:20000})
  await expect(cover).toBeVisible({timeout:20000})
  await expect.poll(()=>avatar.evaluate(el=>el.complete&&el.naturalWidth===160),{timeout:25000}).toBe(true)
  await expect.poll(()=>cover.evaluate(el=>el.complete&&el.naturalWidth>0),{timeout:25000}).toBe(true)
  const score=page.locator('.rat-workspace-score')
  await expect(score.locator('.rum-rating__empty')).toBeVisible({timeout:25000})
  const visibleText=await score.innerText()
  const pending=(visibleText.match(/At least \d+ ratings? awaiting community verdict/g)||[]).length
  // Anonymous visitors may not receive community Judge counts. Confirm
  // the appropriate branded empty state rather than fabricating votes.
  assert.ok(pending===0||pending===1,'Duplicate awaiting verdict in header')
  if(pending===0){
   await expect(score.locator('.rum-rating__empty')).toHaveText('No accepted rating yet')
   await expect(score).toContainText('No established public reputation yet')
  } else {
   await expect(score.locator('.rum-rating__empty')).toContainText('awaiting community verdict')
   assert.equal((visibleText.match(/awaiting community verdict/g)||[]).length,1)
  }
  await expect(score.locator('img[alt="RUM Thumb"]')).toBeVisible()
  await expect(score.getByRole('button',{name:'Rate',exact:true})).toBeVisible()
  await expect(score.locator('a[href="#ratings"]')).toBeVisible()
  const overflow=await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)
  assert.ok(overflow<=2,'Horizontal header/page overflow '+width+'px: '+overflow)
  const name='cj-'+width+'-public-header.png',path=process.env.READ_OUT+'/'+name
  await page.locator('.rat-workspace-header').screenshot({path,animations:'disabled'})
  const bytes=(await stat(path)).size
  assert.ok(bytes>1500&&bytes<650000)
  report.screenshots.push({name,bytes})
  pass('CJ-'+width+'-one-branded-pending-or-appropriate-public-empty-state-no-overflow')
  const full='cj-'+width+'-thing-full.jpg'
  await page.screenshot({path:process.env.READ_OUT+'/'+full,type:'jpeg',quality:60,fullPage:true,animations:'disabled'})
  report.screenshots.push({name:full,bytes:(await stat(process.env.READ_OUT+'/'+full)).size})
  await page.reload({waitUntil:'domcontentloaded'})
  await expect(page.locator('.rat-workspace-kickers')).toContainText('YouTube Channel')
  await page.goto(SITE+'/search?q=cj&active='+CJ,{waitUntil:'domcontentloaded',timeout:40000})
  await expect(page.locator('.rat-active-card h1')).toHaveText('CJ Investigates',{timeout:30000})
  await expect(page.locator('.rat-active-card__type')).toHaveText('YouTube Channel')
  await expect(page.locator('.rat-active-card__avatar img')).toHaveAttribute('src',original.profile)
  assert.ok((await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth))<=2,'Search overflow '+width)
  const searchFile='cj-'+width+'-search-full.jpg'
  await page.screenshot({path:process.env.READ_OUT+'/'+searchFile,type:'jpeg',quality:60,fullPage:true,animations:'disabled'})
  report.screenshots.push({name:searchFile,bytes:(await stat(process.env.READ_OUT+'/'+searchFile)).size})
  await page.reload({waitUntil:'domcontentloaded'})
  await expect(page.locator('.rat-active-card__type')).toHaveText('YouTube Channel')
  pass('CJ-'+width+'-header-about-search-images-refresh-no-overflow')
 }
 stage='linked-navigation-and-rate'
 await page.goto(SITE+'/e/cj-investigates~'+CJ,{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-connection').first()).toBeVisible()
 const first=page.locator('.rat-workspace-connection').first()
 const destinationName=(await first.locator('strong').first().innerText()).trim()
 await first.click()
 await expect(page.locator('.rat-workspace-header h1')).toHaveText(destinationName,{timeout:30000})
 assert.ok(!page.url().includes(CJ),'Connected navigation returned CJ rather than linked entity')
 pass('linked-Thing-navigation-preserves-entity-routing')
 await page.goto(SITE+'/e/cj-investigates~'+CJ,{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-workspace-header h1')).toHaveText('CJ Investigates')
 await page.locator('.rat-workspace-score').getByRole('button',{name:'Rate',exact:true}).click()
 await expect(page.getByRole('dialog',{name:'Rate CJ Investigates'})).toBeVisible()
 await page.getByRole('button',{name:'Close rating'}).click()
 await page.locator('.rat-workspace-score a[href="#ratings"]').click()
 await expect(page.locator('#ratings')).toBeVisible()
 pass('Rate-and-See-ratings-controls-open-without-submission')
 const thumbs=page.locator('.rat-judge-vote-row__hit img')
 if(await thumbs.count()>0){
  const count=await thumbs.count()
  for(let i=0;i<count;i++){
   const dims=await thumbs.nth(i).evaluate(el=>({width:el.getBoundingClientRect().width,height:el.getBoundingClientRect().height,
      hitW:el.closest('button')?.getBoundingClientRect().width,hitH:el.closest('button')?.getBoundingClientRect().height}))
   assert.equal(dims.width,22);assert.equal(dims.height,22);assert.equal(dims.hitW,44);assert.equal(dims.hitH,44)
  }
  pass('rendered-22px-voting-art-and-44px-hit-targets')
 }else pass('no-public-voting-cards-to-change-in-this-view')
 stage='authenticated-readonly-editor'
 await page.getByRole('button',{name:'Edit or suggest a correction'}).click()
 const panel=page.locator('.rat-workspace-profile .entity-about')
 await panel.getByRole('button',{name:'Create account',exact:true}).click()
 const name='ratpost'+process.env.READ_RUN.replace(/[^a-z0-9]/g,'').slice(-13)
 const password=randomBytes(24).toString('base64url')+'aA1!'
 await panel.getByLabel('Email',{exact:true}).fill(name+'@example.com')
 await panel.getByLabel('Username',{exact:true}).fill(name)
 await panel.getByLabel('Password',{exact:true}).fill(password)
 await panel.getByLabel('Confirm password').fill(password)
 await panel.getByRole('checkbox',{name:/Terms and Community Rules/}).check()
 await panel.getByRole('checkbox',{name:/Privacy Notice/}).check()
 const pending=page.waitForResponse(r=>r.request().method()==='POST'&&new URL(r.url()).pathname==='/api/v1/rate-anything/auth/register',{timeout:30000})
 await panel.getByRole('button',{name:'Create RAT account and continue'}).click()
 const registered=await pending
 assert.ok([200,201].includes(registered.status()),'QA account registration rejected: '+registered.status())
 await expect(panel.getByRole('button',{name:'Thing type',exact:true})).toBeVisible({timeout:25000})
 await panel.getByRole('button',{name:'Thing type',exact:true}).click()
 await expect(panel.locator('.rat-thing-type-editor p strong')).toHaveText('YouTube Channel')
 const revisions=page.locator('.rat-thing-revisions')
 await expect(revisions).toContainText('Thing type')
 await expect(revisions).toContainText('Applied')
 await expect(revisions).toContainText('Community contribution')
 const history=await ctx.request.get(SITE+'/api/v1/rate-anything/things/'+CJ+'/edits')
 assert.equal(history.status(),200)
 const rows=(await history.json()).data.filter(x=>x.field==='thing_type')
 assert.equal(rows.length,1)
 assert.equal(rows[0].id,'01m7qsxdjs1tg298py83xk2y49')
 assert.equal(rows[0].proposedTypeLabel,'YouTube Channel')
 const editorName='cj-1440-saved-type-editor.png'
 await page.setViewportSize({width:1440,height:940})
 await panel.screenshot({path:process.env.READ_OUT+'/'+editorName,animations:'disabled'})
 report.screenshots.push({name:editorName,bytes:(await stat(process.env.READ_OUT+'/'+editorName)).size})
 pass('new-community-member-editor-shows-saved-type-and-attributable-revision-without-mutating-CJ')
 stage='new-browser-context'
 const context=await browser.newContext({viewport:{width:390,height:850}})
 const fresh=await context.newPage()
 await fresh.goto(SITE+'/e/cj-investigates~'+CJ,{waitUntil:'domcontentloaded'})
 await expect(fresh.locator('.rat-workspace-header h1')).toHaveText('CJ Investigates')
 await expect(fresh.locator('.rat-workspace-kickers')).toContainText('YouTube Channel')
 await expect(fresh.locator('.rat-workspace-facts dd').first()).toHaveText('YouTube Channel')
 await fresh.goto(SITE+'/search?q=cj&active='+CJ,{waitUntil:'domcontentloaded'})
 await expect(fresh.locator('.rat-active-card__type')).toHaveText('YouTube Channel')
 await expect(fresh.locator('.rat-workspace-score .rum-rating__empty')).toBeVisible({timeout:25000})
 await expect.poll(()=>fresh.locator('.rat-workspace-avatar > img').evaluate(el=>el.complete&&el.naturalWidth===160),{timeout:25000}).toBe(true)
 await context.close()
 pass('independent-public-browser-context-preserves-saved-channel-in-Thing-and-search')
 const again=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+CJ)
 assert.equal(again.status(),200)
 const latest=(await again.json()).data.active
 assert.equal(latest.entityType,'digital_resource')
 assert.equal(latest.images.profile.url,original.profile)
 assert.equal(latest.images.cover.url,original.cover)
 assert.equal(JSON.stringify(latest.reputation),original.reputation)
 pass('final-canonical-identity-media-and-reputation-unchanged')
 report.status='PASS'
 console.log('RAT177_CJ_POSTEDIT_REAL_BROWSER_PASS=true')
}catch(e){
 report.status='FAIL'
 report.errors.push({stage,message:String(e).slice(0,1100)})
 console.log('RAT177_CJ_POSTEDIT_BROWSER_ERROR='+JSON.stringify(report.errors[0]))
 process.exitCode=1
}finally{
 await writeFile(process.env.READ_OUT+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT177_CJ_POSTEDIT_REPORT='+JSON.stringify(report))
 await browser.close()
}
NODE