#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only real Chromium acceptance at 1440/768/390, no signups/votes/edits.
[[ $# -eq 3 ]] || exit 64
src="$1"; api="$2"; rid="$3"
[[ "$src" == 'acebcc208f19348ab9eb01d05434b0e20a5e256c' ]] || exit 78
[[ "$api" == 'ghcr.io/daisycloversoftware/rum-api@sha256:a23e36323b55f3bce46d5b229146e2e7a9d70bdff9e16ea366380940e0629430' ]] || exit 78
[[ "$rid" =~ ^rat178-[a-z0-9-]{12,34}$ ]] || exit 64
out="/tmp/rat178-linked-category-browser-$rid"
[[ ! -e "$out" ]] || exit 65
mkdir -m 700 "$out"
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == 'ghcr.io/daisycloversoftware/rum-rate-anything@sha256:ec11bfec432ff10d508850a0a071b3af3787bff656d2decca90eed5951cd0003' ]] || exit 78
[[ "$(curl -fsS --retry 4 --max-time 25 https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == 'rat-candidate+887b5e02f4be6ec7e52371666effa9eb47892d6e' ]] || exit 78
health="$(curl -fsS --retry 4 --max-time 25 https://dev-rum.daisycloversoftware.uk/api/v1/health)"
python3 - "$health" "$src" <<'PY'
import json,sys
h=json.loads(sys.argv[1])
assert h.get('status')=='ok' and h.get('service')=='rum-api' and sys.argv[2] in str(h.get('version',''))
PY
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
export TMPDIR="$tmp"
git -C "$tmp" init -q
git -C "$tmp" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$tmp" fetch -q --depth=1 origin "$src"
git -C "$tmp" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$tmp" rev-parse HEAD)" == "$src" ]] || exit 78
cd "$tmp/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >/dev/null
chrome=''
for p in /usr/bin/google-chrome /usr/bin/google-chrome-stable /usr/bin/chromium /usr/bin/chromium-browser;do if [[ -x "$p" ]];then chrome="$p";break;fi;done
[[ -n "$chrome" ]] || npx playwright install chromium >/dev/null
CHROME="$chrome" REPORT_DIR="$out" REPORT_RUN="$rid" node --input-type=module <<'NODE'
import assert from 'node:assert/strict'
import {createHash} from 'node:crypto'
import {stat,writeFile} from 'node:fs/promises'
import {chromium,expect} from '@playwright/test'
const DEV='https://dev-rum.daisycloversoftware.uk', RAT='https://dev-rum-ra.daisycloversoftware.uk'
const SOURCE='01m7bswfh0a3evf271ye4taaf0', TARGET='01m7jss48cpff8c59j6mdbbb7f'
const REL='01m7jswzw09b94n00am5003dej',KEY='community_788e6ae3bfcd391d111e3ca31a1332ebf92906d0d0868e7cf7bb'
const dir=process.env.REPORT_DIR
const report={status:'NOT_ASSESSED',run:process.env.REPORT_RUN,source:'acebcc208f19348ab9eb01d05434b0e20a5e256c',checks:[],screenshots:[],errors:[]}
const pass=x=>{report.checks.push(x);console.log('RAT178_BROWSER_PASS='+x)}
const browser=await chromium.launch({headless:true,executablePath:process.env.CHROME||undefined,args:['--no-sandbox','--disable-dev-shm-usage']})
const ctx=await browser.newContext({viewport:{width:1440,height:940}})
const page=await ctx.newPage()
await page.route('**/*',r=>['GET','HEAD','OPTIONS'].includes(r.request().method())?r.continue():r.abort('blockedbyclient'))
let stage='api-baseline'
const capture=async (width,kind)=>{
 const filename='professional-idiots-'+width+'-'+kind+'.jpg'
 const path=dir+'/'+filename
 await page.screenshot({path,type:'jpeg',quality:68,fullPage:true,animations:'disabled'})
 const bytes=(await stat(path)).size
 assert.ok(bytes>12000&&bytes<800000,'Screenshot bounded size '+filename+': '+bytes)
 const checksum=createHash('sha256').update(await (await import('node:fs/promises')).readFile(path)).digest('hex')
 report.screenshots.push({name:filename,bytes,sha256:checksum})
 console.log('RAT178_REAL_SCREENSHOT='+filename+' bytes='+bytes+' sha256='+checksum)
}
try{
 const r=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+SOURCE)
 assert.equal(r.status(),200)
 const payload=await r.json()
 assert.equal(payload.meta.privacy,'private-rum-people-excluded')
 assert.equal(payload.data.active.id,SOURCE)
 assert.equal(payload.data.active.entityType,'digital_resource')
 assert.equal(payload.data.active.classifications.find(x=>x.primary)?.key,KEY)
 const classes=payload.data.categories
 assert.equal(new Set(classes.map(x=>x.key)).size,classes.length,'Duplicated category')
 const channels=classes.filter(x=>x.key==='channels')
 assert.equal(channels.length,1)
 assert.equal(channels[0].label,'Channels')
 const item=channels[0].items.filter(x=>x.id===TARGET)
 assert.equal(item.length,1)
 assert.equal(item[0].name,'The Unprofessional Idiots')
 assert.equal(item[0].entityType,'digital_resource')
 assert.equal(item[0].classifications.find(x=>x.primary)?.key,KEY)
 assert.equal(item[0].relationship.id,REL)
 assert.equal(item[0].relationship.type,'related_to')
 assert.equal(item[0].relationship.direction,'outgoing')
 assert.ok(!classes.some(x=>x.key==='projects-tools'&&x.items.some(y=>y.id===TARGET)))
 assert.equal(payload.data.topLinked.filter(x=>x.id===TARGET).length,1)
 assert.equal(payload.data.topLinked.find(x=>x.id===TARGET)?.relationship.id,REL)
 const all=classes.flatMap(x=>x.items.map(y=>y.relationship.id))
 assert.equal(all.length,new Set(all).size,'Linked relationships duplicated across categories')
 pass('canonical-api-primary-youtube-channel-categorised-once-with-unchanged-relationship-and-top-linked')
 const selected=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+TARGET)
 assert.equal(selected.status(),200)
 const targetNode=(await selected.json()).data.active
 assert.equal(targetNode.id,TARGET)
 assert.equal(targetNode.entityType,'digital_resource')
 assert.equal(targetNode.classifications.find(x=>x.primary)?.key,KEY)
 pass('both-canonical-channel-types-and-source-target-identity-preserved')
 for(const width of [1440,768,390]){
  await page.setViewportSize({width,height:width===390?850:940})
  stage='search-'+width
  await page.goto(RAT+'/search?q=The%20Professional%20Idiots&active='+SOURCE,{waitUntil:'domcontentloaded',timeout:45000})
  await expect(page.locator('.rat-active-card h1')).toHaveText('The Professional Idiots',{timeout:30000})
  await expect(page.locator('.rat-active-card__type')).toHaveText('YouTube Channel')
  const channel=page.locator('section.rat-category').filter({has:page.getByRole('heading',{name:'Channels',exact:true})})
  await expect(channel).toHaveCount(1)
  await expect(channel.getByRole('heading',{name:'Channels',exact:true})).toBeVisible()
  const card=channel.locator('button.rat-linked-card').filter({hasText:'The Unprofessional Idiots'})
  await expect(card).toHaveCount(1)
  await expect(card.locator('strong')).toHaveText('The Unprofessional Idiots')
  await expect(card.locator('small')).toContainText('YouTube Channel')
  await expect(card.locator('img[alt="RUM Thumb"]')).toBeVisible()
  const thumbnail=card.locator('.rat-linked-card__media img')
  const published=item[0].images?.profile?.url??null
  if(published){
   await expect(thumbnail).toHaveAttribute('src',published)
   await expect.poll(()=>thumbnail.evaluate(el=>el.complete&&el.naturalWidth>0),{timeout:25000}).toBe(true)
  }else{
   await expect(card.locator('.rat-linked-card__media')).toBeVisible()
  }
  const projectSection=page.locator('section.rat-category').filter({has:page.getByRole('heading',{name:'Projects & tools',exact:true})})
  if(await projectSection.count()>0)await expect(projectSection.locator('button.rat-linked-card').filter({hasText:'The Unprofessional Idiots'})).toHaveCount(0)
  await expect(page.locator('.rat-top-linked')).toContainText('The Unprofessional Idiots')
  await expect(page.locator('.rat-active-card__actions').getByRole('button',{name:'Rate'})).toBeVisible()
  await expect(page.locator('.rat-active-card__actions').getByRole('button',{name:'See ratings'})).toBeVisible()
  assert.ok((await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth))<=2,'Search horizontal overflow '+width)
  await capture(width,'search')
  pass('search-'+width+'-Channels-not-Projects-Thumbnails-Thumb-TopLinked-and-no-overflow')
  stage='rebase-'+width
  await card.click()
  await expect(page.locator('.rat-active-card h1')).toHaveText('The Unprofessional Idiots',{timeout:30000})
  await expect(page.locator('.rat-active-card__type')).toHaveText('YouTube Channel')
  assert.equal(new URL(page.url()).searchParams.get('active'),TARGET)
  const reverse=page.locator('section.rat-category').filter({has:page.getByRole('heading',{name:'Channels',exact:true})})
  await expect(reverse).toContainText('The Professional Idiots')
  await page.reload({waitUntil:'domcontentloaded'})
  await expect(page.locator('.rat-active-card h1')).toHaveText('The Unprofessional Idiots',{timeout:30000})
  assert.ok((await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth))<=2,'Rebase overflow '+width)
  await capture(width,'linked-active')
  pass('rebase-'+width+'-click-promotes-existing-target-and-persists-after-refresh')
 }
 stage='wider-canonical-fallback'
 const djiId='01m784s4xmkdbfxxd3mzwhp991'
 const djiRes=await ctx.request.get(DEV+'/api/v1/public/entities/rat-catalogue/nodes/'+djiId)
 assert.equal(djiRes.status(),200)
 const dji=(await djiRes.json()).data
 assert.equal(dji.active.name,'DJI')
 const otherDigital=dji.categories.filter(c=>c.key==='other-digital-resources')
 assert.equal(otherDigital.length,1)
 assert.equal(otherDigital[0].label,'Other digital resources')
 const otherIds=otherDigital[0].items.map(i=>i.id)
 assert.ok(otherIds.includes('01m784s53872kny018gv7pjav2'),'DJI Fly App missing from truthful fallback')
 assert.ok(otherIds.includes('01m784s53jq0yypvdzr8jxn4m1'),'DJI LightCut missing from truthful fallback')
 assert.ok(!dji.categories.some(c=>c.key==='projects-tools'&&c.items.some(i=>otherIds.includes(i.id))))
 await page.setViewportSize({width:1440,height:940})
 await page.goto(RAT+'/search?q=DJI&active='+djiId,{waitUntil:'domcontentloaded',timeout:45000})
 await expect(page.locator('.rat-active-card h1')).toHaveText('DJI',{timeout:30000})
 const otherSection=page.locator('section.rat-category').filter({has:page.getByRole('heading',{name:'Other digital resources',exact:true})})
 await expect(otherSection).toContainText('DJI Fly App')
 await expect(otherSection).toContainText('DJI LightCut')
 assert.ok((await page.evaluate(()=>document.documentElement.scrollWidth-innerWidth))<=2)
 pass('existing-unclassified-DJI-digital-resources-appear-in-truthful-other-category-not-tools')
 stage='controls'
 await page.setViewportSize({width:1440,height:940})
 await page.goto(RAT+'/search?q=The%20Professional%20Idiots&active='+SOURCE,{waitUntil:'domcontentloaded'})
 await expect(page.locator('.rat-active-card h1')).toHaveText('The Professional Idiots',{timeout:30000})
 await page.locator('.rat-active-card__actions').getByRole('button',{name:'Rate',exact:true}).click()
 await expect(page.locator('.rat-workspace-header h1')).toHaveText('The Professional Idiots',{timeout:30000})
 await expect(page.getByRole('dialog',{name:'Rate The Professional Idiots'})).toBeVisible({timeout:20000})
 await page.getByRole('button',{name:'Close rating'}).click()
 pass('existing-rating-navigation-opens-without-rating-submission')
 report.status='PASS'
 console.log('RAT178_REAL_CHROMIUM_ACCEPTANCE_PASS=true')
}catch(e){report.status='FAIL';report.errors.push({stage,message:String(e).slice(0,1800)});process.exitCode=1;console.log('RAT178_BROWSER_ERROR='+JSON.stringify(report.errors.at(-1)))}
finally{await writeFile(dir+'/report.json',JSON.stringify(report,null,2));console.log('RAT178_BROWSER_REPORT='+JSON.stringify(report));await browser.close()}
NODE
