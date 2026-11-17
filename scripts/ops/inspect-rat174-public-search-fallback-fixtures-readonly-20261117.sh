#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only public RAT174 canonical search preview; no credentials or mutations.
[[ $# -eq 0 ]] || exit 64
for query in DJI CJ professional; do
  body="$(curl -fsS --retry 3 --retry-delay 2 --max-time 20 -H 'Accept: application/json' "https://dev-rum.daisycloversoftware.uk/api/v1/public/entities/rat-catalogue/search?q=$query")"
  python3 - "$query" "$body" <<'PY'
import json,sys
q=sys.argv[1]; data=json.loads(sys.argv[2])
meta=data.get("meta") or {}
assert meta.get("source")=="rum-dev-canonical" and meta.get("privacy")=="private-rum-people-excluded" and meta.get("audience")=="rat-public"
node=data.get("data") or {}
active=node.get("active") or {}
def info(x):
    profile=(x.get("images") or {}).get("profile")
    assert profile is None or profile.get("status")=="published","Private image unexpectedly exposed"
    return {"id":x.get("id"),"name":x.get("name"),"published_profile":bool(profile),"category":x.get("entityType")}
top=[info(x) for x in node.get("topLinked") or []]
cats=[{"category":c.get("key"),"items":[info(x) for x in (c.get("items") or [])[:5]]} for c in node.get("categories") or []]
print("RAT174_PUBLIC_FALLBACK_INSPECTION="+json.dumps({"query":q,"found":meta.get("found"),"active":info(active),"topLinked":top,"categories":cats},separators=(",",":")))
PY
done
echo RAT174_PUBLIC_FALLBACK_INSPECTION_PASS=true