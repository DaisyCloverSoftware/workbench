#!/usr/bin/env bash
set -euo pipefail
# Canonical DEV only; verify harmless acceptance names are genuine no-matches
# before any test account or Thing creation. Absolutely no mutation.
[[ $# -eq 2 ]] || exit 64
rootId="$1"; missingId="$2"
for rid in "$rootId" "$missingId"; do [[ "$rid" =~ ^[a-z0-9-]{8,42}$ ]] || exit 64; done
rootName="Carmine Nebula Archive $rootId"
missingName="Iridescent Monocle Transit $missingId"
for name in "$rootName" "$missingName"; do
 tmp="$(mktemp)"; trap 'rm -f "$tmp"' EXIT
 status="$(curl -fsS --max-time 25 --retry 2 -G \
   --data-urlencode "q=$name" -o "$tmp" -w '%{http_code}' \
   'https://dev-rum.daisycloversoftware.uk/api/v1/public/entities/rat-catalogue/search')"
 [[ "$status" == '200' ]] || exit 78
 python3 - "$name" "$tmp" <<'PY'
import json,sys
name,path=sys.argv[1:]
with open(path,encoding='utf-8') as source: body=json.load(source)
meta=body.get('meta',{})
found=meta.get('found')
print('RAT170_DEV_TEST_NAME='+json.dumps({'name':name,'found':found,'source':meta.get('source'),'privacy':meta.get('privacy')}))
if found is not False: raise SystemExit('Not an independently verified no-match')
PY
 rm -f "$tmp"; trap - EXIT
done
echo RAT170_TEST_NAMES_READ_ONLY_NO_MATCH=true
