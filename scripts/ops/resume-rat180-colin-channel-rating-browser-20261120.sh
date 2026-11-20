#!/usr/bin/env bash
set -Eeuo pipefail
# Resume only the previously created pending canonical channel. Never create an
# Entity or account. Only one QA rating and one unverified public related_to edge.
[[ $# -eq 1 && "$1" == 'rat180-colin-resume-20261120-r3' ]] || exit 64
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
api='ghcr.io/daisycloversoftware/rum-api@sha256:7411cc02eb6568e6c6d228b551683294bfac49a6f89c320078af5550adce5d56'
rat='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:9a33ea315f52d5e14a5ece0e57895aae8d4eb4d7d7e9955110d49fc9676c64d9'
[[ "$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')" == "$api" ]] || exit 78
[[ "$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')" == "$rat" ]] || exit 78
[[ "$(curl -fsS --retry 4 --max-time 20 https://dev-rum-ra.daisycloversoftware.uk/VERSION | tr -d '\r\n')" == 'rat-candidate+061beefd23bd71cf5902ff2a7724ff78acb515f6' ]] || exit 78
original='/tmp/rat180-colin-auth-rat180-colin-create-20261120-r1'
[[ -f "$original/qa-account-private.json" && -f "$original/report.json" ]] || { echo 'RAT180_ORIGINAL_RECOVERY_FILES_MISSING'; exit 78; }
out='/tmp/rat180-colin-resume-20261120-r3'
[[ ! -e "$out" ]] || { echo RAT180_RESUME_ALREADY_SUBMITTED;exit 65; }
mkdir -m 700 "$out"
work="$(mktemp -d)";trap 'rm -rf "$work"' EXIT
git -C "$work" init -q
git -C "$work" remote add origin https://github.com/DaisyCloverSoftware/rum.git
git -C "$work" fetch -q --depth=1 origin '061beefd23bd71cf5902ff2a7724ff78acb515f6'
git -C "$work" checkout -q --detach FETCH_HEAD
[[ "$(git -C "$work" rev-parse HEAD)" == '061beefd23bd71cf5902ff2a7724ff78acb515f6' ]] || exit 78
cd "$work/apps/rate-anything"
npm ci --ignore-scripts --no-audit --no-fund >"$work/npm.log" 2>&1 || { tail -n 12 "$work/npm.log"; exit 1; }
npx playwright install chromium >"$work/chrome.log" 2>&1 || { tail -n 12 "$work/chrome.log"; exit 1; }
RAT180_ACCOUNT_FILE="$original/qa-account-private.json" RAT180_OUTPUT_DIR="$out" node --input-type=module <<'NODE'
import {chromium,expect} from '@playwright/test'
import assert from 'node:assert/strict'
import {readFile,writeFile,stat} from 'node:fs/promises'
import {createHash} from 'node:crypto'
const RAT='https://dev-rum-ra.daisycloversoftware.uk'
const DEV='https://dev-rum.daisycloversoftware.uk'
const PERSON='01m0mw9hkbynssr7kbc0gk534j'
const CHANNEL='01m7x6ez57zpsgbv486y9t0y9h'
const JOURNEY='01m7x6ez8pj1k53cj4y8nnw4q6'
const ACCOUNT_ID='01m7x6eyapaft7xxmk4cv0vj6x'
const dir=process.env.RAT180_OUTPUT_DIR
const auth=JSON.parse(await readFile(process.env.RAT180_ACCOUNT_FILE,'utf8'))
const report={status:'NOT_ASSESSED',personId:PERSON,channelId:CHANNEL,journeyId:JOURNEY,steps:[],ratingPosts:0,connectionPosts:0,ratingEventId:null,relationship:null,errors:[],screenshots:[]}
const pass=(s)=>{report.steps.push(s);console.log('RAT180_RESUME_PASS='+s)}
let browser=null,stage='start'
async function capture(page,label){
 const path=dir+'/'+label+'.jpg'
 await page.screenshot({path,type:'jpeg',quality:70,fullPage:true,animations:'disabled'})
 const bytes=(await stat(path)).size
 const sha=createHash('sha256').update(await readFile(path)).digest('hex')
 assert.ok(bytes>6000&&bytes<900000)
 report.screenshots.push({name:label+'.jpg',bytes,sha256:sha})
 console.log('RAT180_RESUME_SCREENSHOT='+label+' bytes='+bytes+' sha256='+sha)
}
try{
 browser=await chromium.launch({headless:true,args:['--no-sandbox','--disable-dev-shm-usage']})
 const context=await browser.newContext({viewport:{width:1440,height:940}})
 const page=await context.newPage()
 const visit=async(url)=>{
  for(let attempt=1;attempt<=3;attempt++){
   try{return await page.goto(url,{waitUntil:'domcontentloaded',timeout:45000})}
   catch(e){if(attempt===3||!String(e).includes('ERR_NETWORK_CHANGED'))throw e;await page.waitForTimeout(1200)}
  }
 }
 await page.route('**/*',async route=>{
  const req=route.request(),u=new URL(req.url()),method=req.method()
  if(['GET','HEAD','OPTIONS'].includes(method)){
   if(['dev-rum-ra.daisycloversoftware.uk','dev-rum.daisycloversoftware.uk'].includes(u.hostname))return route.continue()
   return route.abort('blockedbyclient')
  }
  const login=u.hostname==='dev-rum-ra.daisycloversoftware.uk'&&method==='POST'&&u.pathname==='/api/v1/rate-anything/auth/login'
  const rating=u.hostname==='dev-rum-ra.daisycloversoftware.uk'&&method==='POST'&&u.pathname==='/api/v1/rate-anything/rating'
  const connection=u.hostname==='dev-rum-ra.daisycloversoftware.uk'&&method==='POST'&&u.pathname==='/api/v1/rate-anything/things/'+CHANNEL+'/connections'
  if(!login&&!rating&&!connection)return route.abort('blockedbyclient')
  if(rating){
   const input=req.postDataJSON()
   assert.equal(input.entityId,CHANNEL)
   assert.equal(input.journeyId,JOURNEY)
   report.ratingPosts++
   assert.equal(report.ratingPosts,1,'Only one initial rating is authorised')
  }
  if(connection){
   const input=req.postDataJSON()
   assert.deepEqual(input,{targetEntityId:PERSON,relationshipType:'related_to',direction:'outgoing'})
   report.connectionPosts++
   assert.equal(report.connectionPosts,1,'Only one community relationship submission is authorised')
  }
  return route.continue()
 })
 stage='sign-in'
 await visit(RAT+'/search?q=Colin%20Furze&active='+PERSON)
 await expect(page.locator('.rat-active-card h1')).toHaveText('Colin Furze',{timeout:30000})
 await page.getByRole('button',{name:'Sign in',exact:true}).click()
 const login=page.getByRole('dialog',{name:'Sign in to RAT'})
 await expect(login).toBeVisible()
 await login.getByLabel('Username or email').fill(auth.username)
 await login.getByLabel('Password',{exact:true}).fill(auth.password)
 const authenticated=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/rate-anything/auth/login'&&r.request().method()==='POST',{timeout:30000})
 await login.getByRole('button',{name:'Log in and continue'}).click()
 const loginResponse=await authenticated
 assert.equal(loginResponse.status(),200)
 const member=(await loginResponse.json()).data
 assert.equal(member.id,ACCOUNT_ID,'Must resume the exact QA account owning the journey')
 pass('same-QA-account-authenticated-no-new-account')
 stage='pending-root'
 const rootRes=await context.request.get(RAT+'/api/v1/rate-anything/standalone-things/'+JOURNEY)
 assert.equal(rootRes.status(),200)
 const root=await rootRes.json()
 assert.equal(root.data.id,CHANNEL)
 assert.equal(root.data.name,'Colin Furze')
 assert.equal(root.data.publicationState,'pending')
 assert.equal(root.meta.journey.id,JOURNEY)
 assert.equal(root.meta.journey.state,'active')
 pass('exact-pending-channel-and-rating-journey-resumed')
 stage='rating-first'
 await visit(RAT+'/rate/'+CHANNEL+'?journey='+JOURNEY)
 const flow=page.locator('.rat-rating-flow')
 await expect(flow).toBeVisible({timeout:35000})
 // Do not select while the current-rating request can still overwrite the draft.
 await expect(flow.locator('.rating-card__status').filter({hasText:'Checking your rating account'})).toHaveCount(0,{timeout:45000})
 pass('initial-current-rating-load-settled-before-verdict')
 const thumb=flow.getByRole('radio',{name:/^3: Good/})
 await expect(thumb).toBeEnabled({timeout:30000})
 await thumb.click()
 await expect(thumb).toHaveAttribute('aria-checked','true')
 stage='rating-first-continue'
 await expect(flow.getByRole('button',{name:'Continue'})).toBeEnabled({timeout:16000})
 await flow.getByRole('button',{name:'Continue'}).click()
 await expect(flow).toHaveAttribute('data-rating-step','Category',{timeout:16000})
 pass('rating-thumb-selected-and-advanced')
 stage='rating-category'
 const choices=flow.locator('.rat-rating-choice-grid button')
 await expect(choices.first()).toBeVisible({timeout:30000})
 const choice=choices.first()
 await choice.click()
 await expect(choice).toHaveAttribute('aria-pressed','true',{timeout:16000})
 await expect(flow.getByRole('button',{name:'Continue'})).toBeEnabled({timeout:16000})
 await flow.getByRole('button',{name:'Continue'}).click()
 pass('existing-rating-category-selected-and-advanced')
 stage='rating-reason'
 await flow.getByLabel('Why this verdict?').fill('DEV acceptance: identified the official Colin Furze channel separately from its public creator persona.')
 await expect(flow.getByRole('button',{name:'Continue'})).toBeEnabled({timeout:16000})
 await flow.getByRole('button',{name:'Continue'}).click()
 pass('rating-reason-ready-no-creator-rating')
 stage='rating-submit'
 const responseWait=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/rate-anything/rating'&&r.request().method()==='POST',{timeout:45000})
 await flow.getByRole('button',{name:'Submit rating'}).click()
 const response=await responseWait
 assert.ok([200,201].includes(response.status()),'Channel rating was rejected: '+response.status())
 const rating=await response.json()
 report.ratingEventId=rating.data?.id??null
 await expect(flow).toContainText('Rating saved for Colin Furze',{timeout:30000})
 pass('only-canonical-channel-QA-rating-submitted')
 await capture(page,'colin-channel-initial-rating')
 stage='post-rating-public'
 const searchRes=await context.request.get(DEV+'/api/v1/public/entities/rat-catalogue/search?q=Colin%20Furze')
 assert.equal(searchRes.status(),200)
 const search=await searchRes.json()
 assert.equal(search.data.active.id,PERSON)
 const matches=search.meta.matches??[]
 assert.ok(matches.some(x=>x.id===PERSON&&x.ratingOwner==='rum'))
 assert.ok(matches.some(x=>x.id===CHANNEL&&x.ratingOwner==='rat'&&x.name==='Colin Furze'))
 pass('published-same-name-person-and-channel-found-in-canonical-search')
 stage='connection'
 const conn=await page.evaluate(async({person,channel})=>{
  const cookie=document.cookie.split('; ').find(x=>x.startsWith('XSRF-TOKEN='))
  const token=cookie?decodeURIComponent(cookie.split('=').slice(1).join('=')):null
  const response=await fetch('/api/v1/rate-anything/things/'+channel+'/connections',{
   method:'POST',credentials:'same-origin',
   headers:{Accept:'application/json','Content-Type':'application/json',...(token?{'X-XSRF-TOKEN':token}:{})},
   body:JSON.stringify({targetEntityId:person,relationshipType:'related_to',direction:'outgoing'}),
  })
  return {status:response.status,body:await response.json().catch(()=>({}))}
 },{person:PERSON,channel:CHANNEL})
 assert.ok([200,201].includes(conn.status),'Connection failed '+conn.status+' '+JSON.stringify(conn.body).slice(0,500))
 assert.equal(conn.body.data?.verificationState,'unverified')
 assert.equal(conn.body.data?.visibility,'public')
 assert.equal(conn.body.data?.sourceEntityId,CHANNEL)
 assert.equal(conn.body.data?.targetEntityId,PERSON)
 report.relationship=conn.body.data
 pass('explicit-public-unverified-provenanced-related-to-connection')
 report.status='PASS_CHANNEL_PUBLISHED_AND_LINKED'
}catch(e){
 report.status='FAIL'
 report.errors.push({stage,message:String(e).slice(0,2300)})
 console.log('RAT180_RESUME_ERROR='+JSON.stringify(report.errors.at(-1)))
 process.exitCode=1
}finally{
 await writeFile(dir+'/report.json',JSON.stringify(report,null,2))
 console.log('RAT180_RESUME_REPORT='+JSON.stringify(report))
 if(browser)await browser.close()
}
NODE
