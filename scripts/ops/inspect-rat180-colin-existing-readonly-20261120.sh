#!/usr/bin/env bash
set -Eeuo pipefail
# Read only two public RUM DEV endpoints. No authentication, writes, LIVE queries or DB access.
[[ $# -eq 0 ]] || exit 64
base='https://dev-rum.daisycloversoftware.uk'
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
rat_status="$(curl -sS --retry 3 --max-time 20 -w '%{http_code}' -o "$tmp/rat.json" -H 'Accept: application/json' "$base/api/v1/public/entities/rat-catalogue/search?q=Colin%20Furze")"
rum_status="$(curl -sS --retry 3 --max-time 20 -w '%{http_code}' -o "$tmp/rum.json" -H 'Accept: application/json' "$base/api/v1/public/entities/search?q=Colin%20Furze")"
printf 'RAT_CATALOGUE_HTTP=%s\\nRUM_PUBLIC_SEARCH_HTTP=%s\\n' "$rat_status" "$rum_status"
[[ "$rat_status" == 200 ]] || exit 22
if [[ "$rum_status" != 200 ]];then printf '{"data":[]}' > "$tmp/rum.json";fi
channel_id='UCp68_FLety0O-n9QU6phsgw'
channel_status="$(curl -sS --retry 3 --max-time 20 -w '%{http_code}' -o "$tmp/channel.json" -H 'Accept: application/json' "$base/api/v1/public/entities/rat-catalogue/search?q=$channel_id")"
printf 'RAT_CHANNEL_IDENTIFIER_HTTP=%s\n' "$channel_status"
if [[ "$channel_status" == 200 ]];then
python3 - "$tmp/channel.json" <<'PYCHANNEL'
import json,sys
body=json.load(open(sys.argv[1]))
print('RAT_CHANNEL_IDENTIFIER_FOUND='+str(body.get('meta',{}).get('found')))
active=(body.get('data') or {}).get('active') or {}
print('RAT_CHANNEL_IDENTIFIER_ACTIVE='+str({'id':active.get('id'),'name':active.get('name'),'entityType':active.get('entityType')}))
PYCHANNEL
fi
python3 - "$tmp/rat.json" "$tmp/rum.json" <<'PY'
import json,sys
rat=json.load(open(sys.argv[1]))
rum=json.load(open(sys.argv[2]))
print('RAT180_COLIN_READONLY_PUBLIC_CHECK=true')
print('RAT_FOUND='+str(rat.get('meta',{}).get('found')))
active=(rat.get('data') or {}).get('active') or {}
print('RAT_ACTIVE_ID='+str(active.get('id','')))
print('RAT_ACTIVE_NAME='+str(active.get('name','')))
print('RAT_ACTIVE_TYPE='+str(active.get('entityType','')))
print('RAT_ACTIVE_PRIMARY='+str(next((c.get('name') for c in active.get('classifications',[]) if c.get('primary')),'')))
print('RAT_ACTIVE_RATING_OWNER='+str(active.get('ratingOwner','')))
for cat in (rat.get('data') or {}).get('categories',[]):
 for item in cat.get('items',[]):
  if str(item.get('name','')).strip().casefold()=='colin furze':
   print('RAT_LINKED_SAME_NAME='+str({'id':item.get('id'),'entityType':item.get('entityType'),'classification':next((c.get('name') for c in item.get('classifications',[]) if c.get('primary')),''),'relationshipId':item.get('relationship',{}).get('id'),'verificationState':item.get('relationship',{}).get('verificationState')}))
results=rum.get('data') or []
matching=[]
for entry in results:
 entity=entry.get('entity') or {}
 if str(entity.get('name','')).strip().casefold()=='colin furze':
  matching.append({'id':entity.get('id'),'entityType':entity.get('entityType'),'classification':next((c.get('name') for c in entity.get('classifications',[]) if c.get('primary')),''),'rateability':entity.get('rateability')})
print('RUM_PUBLIC_SAME_NAME='+json.dumps(matching,sort_keys=True,separators=(',',':')))
PY
