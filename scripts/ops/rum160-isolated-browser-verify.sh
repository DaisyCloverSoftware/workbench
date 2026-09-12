#!/usr/bin/env bash
set -euo pipefail
umask 077

SOURCE_SHA="${1:-}"
if ! [[ "$SOURCE_SHA" =~ ^[0-9a-f]{40}$ ]]; then
  echo "usage: rum160-isolated-browser-verify.sh <40-hex-source-sha>" >&2
  exit 64
fi

RAT_ORIGIN='https://dev-rum-ra.daisycloversoftware.uk'
RUM_ORIGIN='https://dev-rum.daisycloversoftware.uk'
BROWSER='/usr/bin/google-chrome'

[[ -x "$BROWSER" ]] || { echo "ERROR: Google Chrome is unavailable at $BROWSER" >&2; exit 3; }
command -v node >/dev/null 2>&1 || { echo 'ERROR: node is unavailable' >&2; exit 3; }

work="$(mktemp -d)"
chrome_pid=''
cleanup() {
  if [[ -n "$chrome_pid" ]]; then
    kill "$chrome_pid" >/dev/null 2>&1 || true
    wait "$chrome_pid" >/dev/null 2>&1 || true
  fi
  rm -rf "$work"
}
trap cleanup EXIT HUP INT TERM

mkdir -p "$work/profile"
"$BROWSER" \
  --headless=new \
  --no-sandbox \
  --disable-dev-shm-usage \
  --disable-gpu \
  --disable-background-networking \
  --disable-component-update \
  --disable-default-apps \
  --disable-features=OptimizationHints,MediaRouter \
  --no-first-run \
  --no-default-browser-check \
  --user-data-dir="$work/profile" \
  --remote-debugging-port=0 \
  about:blank >"$work/chrome.log" 2>&1 &
chrome_pid=$!

active_port="$work/profile/DevToolsActivePort"
for _ in $(seq 1 100); do
  [[ -s "$active_port" ]] && break
  kill -0 "$chrome_pid" >/dev/null 2>&1 || {
    echo 'ERROR: Chrome exited before DevTools became ready' >&2
    cat "$work/chrome.log" >&2 || true
    exit 4
  }
  sleep 0.1
done
[[ -s "$active_port" ]] || { echo 'ERROR: DevToolsActivePort was not created' >&2; cat "$work/chrome.log" >&2 || true; exit 4; }

DEVTOOLS_PORT="$(sed -n '1p' "$active_port")"
export SOURCE_SHA RAT_ORIGIN RUM_ORIGIN DEVTOOLS_PORT

node <<'NODE'
const sourceSha = process.env.SOURCE_SHA
const ratOrigin = process.env.RAT_ORIGIN
const rumOrigin = process.env.RUM_ORIGIN
const devtoolsPort = process.env.DEVTOOLS_PORT
const allowedOrigins = new Set([ratOrigin, rumOrigin])

const assert = (condition, message) => {
  if (!condition) throw new Error(message)
}

const getJson = async (url) => {
  const response = await fetch(url, {
    method: 'GET',
    headers: { Accept: 'application/json', 'Cache-Control': 'no-cache' },
    redirect: 'manual',
  })
  const text = await response.text()
  assert(response.status === 200, `${url} returned HTTP ${response.status}: ${text.slice(0, 400)}`)
  return JSON.parse(text)
}

const health = await getJson(`${rumOrigin}/api/v1/health`)
assert(health.status === 'ok' && health.service === 'rum-api', `unexpected RUM API health payload: ${JSON.stringify(health)}`)
assert(String(health.version || '').includes(sourceSha), `RUM API is not exact head ${sourceSha}: ${JSON.stringify(health)}`)

const ratVersionResponse = await fetch(`${ratOrigin}/VERSION`, { headers: { 'Cache-Control': 'no-cache' }, redirect: 'manual' })
const ratVersion = (await ratVersionResponse.text()).trim()
assert(ratVersionResponse.status === 200, `RAT VERSION returned HTTP ${ratVersionResponse.status}`)
assert(ratVersion.includes(sourceSha), `RAT frontend is not exact head ${sourceSha}: ${ratVersion}`)

const canonical = await getJson(`${rumOrigin}/api/v1/public/entities/rat-catalogue/search?q=CJ`)
assert(canonical?.meta?.source === 'rum-dev-canonical', `unexpected canonical source: ${JSON.stringify(canonical?.meta)}`)
assert(canonical?.meta?.audience === 'rat-public', `unexpected canonical audience: ${JSON.stringify(canonical?.meta)}`)
assert(canonical?.meta?.privacy === 'private-rum-people-excluded', `unexpected canonical privacy marker: ${JSON.stringify(canonical?.meta)}`)
assert(canonical?.data?.active?.name === 'CJ Investigates', `canonical CJ active was ${JSON.stringify(canonical?.data?.active?.name)}`)
assert(canonical?.data?.active?.ratingOwner === 'rum', `CJ is not owned by canonical RUM rating: ${JSON.stringify(canonical?.data?.active)}`)
const canonicalId = canonical.data.active.id
assert(typeof canonicalId === 'string' && canonicalId.length > 0, 'canonical CJ id is missing')

// The isolated RAT namespace may intentionally have no local CJ fixture. The
// browser contract is canonical-first: a positive RUM DEV catalogue match must
// be used directly and must not fall through to the local fixture resolver.
const localCj = await fetch(`${ratOrigin}/api/v1/public/entities/rat/search?q=CJ`, {
  method: 'GET',
  headers: { Accept: 'application/json', 'Cache-Control': 'no-cache' },
  redirect: 'manual',
})
assert([200, 422].includes(localCj.status), `unexpected local RAT CJ status: ${localCj.status}`)

const targetsResponse = await fetch(`http://127.0.0.1:${devtoolsPort}/json/list`)
const targets = await targetsResponse.json()
const target = targets.find((item) => item.type === 'page')
assert(target?.webSocketDebuggerUrl, 'Chrome page DevTools target is unavailable')

const ws = new WebSocket(target.webSocketDebuggerUrl)
await new Promise((resolve, reject) => {
  const timer = setTimeout(() => reject(new Error('DevTools websocket open timed out')), 10_000)
  ws.addEventListener('open', () => { clearTimeout(timer); resolve() }, { once: true })
  ws.addEventListener('error', () => { clearTimeout(timer); reject(new Error('DevTools websocket failed to open')) }, { once: true })
})

let nextId = 1
const pending = new Map()
const runtimeErrors = []
const blockedRequests = []
const seenRequests = []

const send = (method, params = {}) => new Promise((resolve, reject) => {
  const id = nextId++
  pending.set(id, { resolve, reject, method })
  ws.send(JSON.stringify({ id, method, params }))
})

ws.addEventListener('message', (event) => {
  let message
  try { message = JSON.parse(String(event.data)) } catch { return }
  if (message.id) {
    const waiter = pending.get(message.id)
    if (!waiter) return
    pending.delete(message.id)
    if (message.error) waiter.reject(new Error(`${waiter.method}: ${message.error.message}`))
    else waiter.resolve(message.result)
    return
  }

  if (message.method === 'Runtime.exceptionThrown') {
    const details = message.params?.exceptionDetails
    runtimeErrors.push(details?.exception?.description || details?.text || 'unknown runtime exception')
  }

  if (message.method === 'Network.requestWillBeSent') {
    const request = message.params?.request
    if (request?.url) seenRequests.push({ method: request.method, url: request.url })
  }

  if (message.method === 'Fetch.requestPaused') {
    const requestId = message.params.requestId
    const request = message.params.request
    let allowed = false
    try {
      const url = new URL(request.url)
      allowed = allowedOrigins.has(url.origin) && ['GET', 'HEAD', 'OPTIONS'].includes(request.method)
    } catch {
      allowed = request.url === 'about:blank' || request.url.startsWith('data:') || request.url.startsWith('blob:')
    }
    if (allowed) {
      void send('Fetch.continueRequest', { requestId }).catch((error) => runtimeErrors.push(String(error)))
    } else {
      blockedRequests.push({ method: request.method, url: request.url })
      void send('Fetch.failRequest', { requestId, errorReason: 'BlockedByClient' }).catch((error) => runtimeErrors.push(String(error)))
    }
  }
})

await send('Page.enable')
await send('Runtime.enable')
await send('Network.enable')
await send('Fetch.enable', { patterns: [{ urlPattern: '*' }] })

const evaluate = async (expression) => {
  const result = await send('Runtime.evaluate', {
    expression,
    returnByValue: true,
    awaitPromise: true,
  })
  if (result.exceptionDetails) {
    throw new Error(`Runtime.evaluate failed: ${result.exceptionDetails.text}`)
  }
  return result.result?.value
}

const waitFor = async (expression, description, timeoutMs = 30_000) => {
  const deadline = Date.now() + timeoutMs
  let last
  while (Date.now() < deadline) {
    try {
      last = await evaluate(expression)
      if (last) return last
    } catch (error) {
      last = String(error)
    }
    await new Promise((resolve) => setTimeout(resolve, 200))
  }
  throw new Error(`Timed out waiting for ${description}; last=${JSON.stringify(last)}`)
}

const navigate = async (url) => {
  await send('Page.navigate', { url })
  await waitFor(`document.readyState === 'complete'`, `document ready at ${url}`)
}

const pageState = `(() => ({
  href: location.href,
  heading: document.querySelector('.rat-active-card h1')?.textContent?.trim() || null,
  message: document.querySelector('.rat-message')?.textContent?.trim() || null,
  q: new URL(location.href).searchParams.get('q'),
  active: new URL(location.href).searchParams.get('active'),
  rate: Array.from(document.querySelectorAll('button')).some((button) => button.textContent?.trim() === 'Rate'),
  ratings: Array.from(document.querySelectorAll('button')).some((button) => button.textContent?.trim() === 'See ratings')
}))()`

const setSearch = async (value) => {
  const encoded = JSON.stringify(value)
  const changed = await evaluate(`(() => {
    const input = document.querySelector('#rat-search-results');
    if (!input) return false;
    const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set;
    setter.call(input, ${encoded});
    input.dispatchEvent(new Event('input', { bubbles: true }));
    return true;
  })()`)
  assert(changed, 'RAT results search input was unavailable')
  await waitFor(`(() => {
    const input = document.querySelector('#rat-search-results');
    const button = Array.from(document.querySelectorAll('button')).find((item) => item.textContent?.trim() === 'Search');
    return input?.value === ${encoded} && !!button && !button.disabled;
  })()`, `search input to accept ${value}`)
}

const clickButton = async (label) => {
  const encoded = JSON.stringify(label)
  const clicked = await evaluate(`(() => {
    const button = Array.from(document.querySelectorAll('button')).find((item) => item.textContent?.trim() === ${encoded});
    if (!button || button.disabled) return false;
    button.click();
    return true;
  })()`)
  assert(clicked, `button ${label} was unavailable`)
}

await navigate(`${ratOrigin}/search?q=DJI`)
const djiState = await waitFor(`(() => {
  const s = ${pageState};
  return s.heading === 'DJI' && !!s.active ? s : false;
})()`, 'DJI active result')
const djiActive = djiState.active

await setSearch('CJ')
await clickButton('Search')
const cjState = await waitFor(`(() => {
  const s = ${pageState};
  return s.heading === 'CJ Investigates' && s.q?.toLowerCase() === 'cj' && !!s.active ? s : false;
})()`, 'CJ Investigates active result')
assert(cjState.active === canonicalId, `browser CJ active ${cjState.active} does not match canonical ${canonicalId}`)
assert(cjState.active !== djiActive, `browser CJ retained DJI active id ${djiActive}`)
assert(cjState.rate && cjState.ratings, 'CJ active card is missing Rate or See ratings action')
const cjUrl = cjState.href

await clickButton('Rate')
const rateHandoff = await waitFor(`(() => {
  const u = new URL(location.href);
  return u.origin === ${JSON.stringify(rumOrigin)} && u.pathname === '/rate' ? {
    href: u.href,
    id: u.searchParams.get('ratPublicIdentity'),
    name: u.searchParams.get('ratPublicIdentityName'),
    intent: u.searchParams.get('ratIntent')
  } : false;
})()`, 'CJ Rate canonical RUM handoff')
assert(rateHandoff.id === canonicalId, `Rate handoff identity mismatch: ${JSON.stringify(rateHandoff)}`)
assert(rateHandoff.name === 'CJ Investigates', `Rate handoff name mismatch: ${JSON.stringify(rateHandoff)}`)
assert(rateHandoff.intent === 'rate', `Rate handoff intent mismatch: ${JSON.stringify(rateHandoff)}`)

await navigate(cjUrl)
await waitFor(`document.querySelector('.rat-active-card h1')?.textContent?.trim() === 'CJ Investigates'`, 'CJ result after returning from Rate handoff')
await clickButton('See ratings')
const ratingsHandoff = await waitFor(`(() => {
  const u = new URL(location.href);
  return u.origin === ${JSON.stringify(rumOrigin)} && u.pathname === '/rate' ? {
    href: u.href,
    id: u.searchParams.get('ratPublicIdentity'),
    name: u.searchParams.get('ratPublicIdentityName'),
    intent: u.searchParams.get('ratIntent')
  } : false;
})()`, 'CJ See ratings canonical RUM handoff')
assert(ratingsHandoff.id === canonicalId, `Ratings handoff identity mismatch: ${JSON.stringify(ratingsHandoff)}`)
assert(ratingsHandoff.name === 'CJ Investigates', `Ratings handoff name mismatch: ${JSON.stringify(ratingsHandoff)}`)
assert(ratingsHandoff.intent === 'ratings', `Ratings handoff intent mismatch: ${JSON.stringify(ratingsHandoff)}`)

await navigate(`${ratOrigin}/search?q=DJI`)
const djiForLinked = await waitFor(`(() => {
  const s = ${pageState};
  const linked = document.querySelector('.rat-linked-card strong')?.textContent?.trim();
  return s.heading === 'DJI' && !!s.active && !!linked ? { ...s, linked } : false;
})()`, 'DJI linked Thing')
const linkedName = djiForLinked.linked
const linkedClicked = await evaluate(`(() => {
  const button = document.querySelector('.rat-linked-card');
  if (!button) return false;
  button.click();
  return true;
})()`)
assert(linkedClicked, 'first linked Thing could not be selected')
const linkedState = await waitFor(`(() => {
  const s = ${pageState};
  return s.heading === ${JSON.stringify(linkedName)} && !!s.active ? s : false;
})()`, `linked Thing ${linkedName} to become active`)
assert(linkedState.active !== djiForLinked.active, 'linked Thing selection retained DJI active id')
assert(linkedState.q === 'DJI', `linked Thing selection lost DJI exploration query: ${linkedState.href}`)

await setSearch('zzzz-no-such-rat-entity-zzzz')
await clickButton('Search')
const noMatch = await waitFor(`(() => {
  const s = ${pageState};
  return !!s.message && s.heading === null && s.active === null ? s : false;
})()`, 'genuine no-match state')
assert(noMatch.q === 'zzzz-no-such-rat-entity-zzzz', `no-match query state mismatch: ${noMatch.href}`)

await new Promise((resolve) => setTimeout(resolve, 500))
assert(blockedRequests.length === 0, `browser attempted blocked request(s): ${JSON.stringify(blockedRequests)}`)
assert(runtimeErrors.length === 0, `browser runtime error(s): ${JSON.stringify(runtimeErrors)}`)

for (const request of seenRequests) {
  if (request.url === 'about:blank' || request.url.startsWith('data:') || request.url.startsWith('blob:')) continue
  const url = new URL(request.url)
  assert(allowedOrigins.has(url.origin), `browser contacted non-isolated origin: ${request.method} ${request.url}`)
  assert(['GET', 'HEAD', 'OPTIONS'].includes(request.method), `browser attempted non-read-only method: ${request.method} ${request.url}`)
}

const browserCjCanonical = seenRequests.filter((request) => {
  try {
    const url = new URL(request.url)
    return request.method === 'GET' && url.origin === rumOrigin && url.pathname === '/api/v1/public/entities/rat-catalogue/search' && url.searchParams.get('q')?.toLowerCase() === 'cj'
  } catch { return false }
})
const browserCjLocal = seenRequests.filter((request) => {
  try {
    const url = new URL(request.url)
    return request.method === 'GET' && url.origin === ratOrigin && url.pathname === '/api/v1/public/entities/rat/search' && url.searchParams.get('q')?.toLowerCase() === 'cj'
  } catch { return false }
})
assert(browserCjCanonical.length > 0, 'browser did not request the canonical RUM DEV CJ catalogue')
assert(browserCjLocal.length === 0, `browser incorrectly fell through to local CJ fixture resolver: ${JSON.stringify(browserCjLocal)}`)

console.log(`RUM160_SOURCE_SHA=${sourceSha}`)
console.log(`RUM160_CJ_CANONICAL_ID=${canonicalId}`)
console.log(`RUM160_CJ_BROWSER_URL=${cjUrl}`)
console.log(`RUM160_CJ_RATE_HANDOFF=${rateHandoff.href}`)
console.log(`RUM160_CJ_RATINGS_HANDOFF=${ratingsHandoff.href}`)
console.log(`RUM160_LINKED_SELECTED=${linkedName}`)
console.log(`RUM160_LOCAL_CJ_STATUS=${localCj.status}`)
console.log('RUM160_PRIVACY_META_PASS=true')
console.log('RUM160_CANONICAL_FIRST_PASS=true')
console.log('RUM160_CJ_SEARCH_SELECT_PASS=true')
console.log('RUM160_CJ_RATE_HANDOFF_PASS=true')
console.log('RUM160_CJ_RATINGS_HANDOFF_PASS=true')
console.log('RUM160_LINKED_SELECTION_PASS=true')
console.log('RUM160_NO_MATCH_PASS=true')
console.log('RUM160_READ_ONLY_BROWSER_PASS=true')
console.log('RUM160_ISOLATED_BROWSER_VERIFIED=true')

ws.close()
NODE
