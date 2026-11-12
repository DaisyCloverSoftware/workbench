#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <40-char-source-sha> <api-image@sha256:...>" >&2
  exit 64
fi

SOURCE_SHA="$1"
API_IMAGE="$2"
DEV_NS="rum-dev-isolated"
RAT_NS="rum-rate-anything-preview"
LIVE_NS="rum-dev"
DEV_HOST="dev-rum.daisycloversoftware.uk"
RAT_HOST="dev-rum-ra.daisycloversoftware.uk"
LIVE_HOST="rateurmate.online"
CONFIGMAP="rum-config"

[[ "$SOURCE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "invalid source SHA" >&2; exit 64; }
[[ "$API_IMAGE" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || { echo "invalid immutable API image" >&2; exit 64; }

kctl() {
  if command -v k3s >/dev/null 2>&1; then
    sudo -n k3s kubectl "$@"
  else
    kubectl "$@"
  fi
}

image() {
  local ns="$1" deploy="$2" container="$3"
  kctl -n "$ns" get deployment "$deploy" -o "jsonpath={.spec.template.spec.containers[?(@.name==\"$container\")].image}"
}

hosts() {
  local ns="$1"
  kctl -n "$ns" get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u
}

for ns in "$DEV_NS" "$RAT_NS" "$LIVE_NS"; do
  kctl get namespace "$ns" >/dev/null
done

[[ "$(hosts "$DEV_NS")" == "$DEV_HOST" ]] || { echo "blocked: isolated DEV host mismatch" >&2; exit 78; }
[[ "$(hosts "$RAT_NS")" == "$RAT_HOST" ]] || { echo "blocked: RAT preview host mismatch" >&2; exit 78; }
live_hosts_before="$(hosts "$LIVE_NS")"
grep -Fxq "$LIVE_HOST" <<<"$live_hosts_before" || { echo "blocked: LIVE host missing" >&2; exit 78; }

dev_api="$(image "$DEV_NS" rum-api php-fpm)"
[[ "$dev_api" == "$API_IMAGE" ]] || { echo "blocked: isolated DEV API is $dev_api, expected $API_IMAGE" >&2; exit 78; }
rat_front_before="$(image "$RAT_NS" rum-rate-anything rate-anything)"
rat_api_before="$(image "$RAT_NS" rum-api php-fpm)"
live_api_before="$(image "$LIVE_NS" rum-api php-fpm)"
live_web_before="$(image "$LIVE_NS" rum-web web)"
live_worker_before="$(image "$LIVE_NS" rum-worker worker)"

health="$(curl -fsS --retry 6 --retry-delay 2 --max-time 20 "https://$DEV_HOST/api/v1/health")"
python3 - "$health" "$SOURCE_SHA" <<'PY'
import json,sys
obj=json.loads(sys.argv[1])
assert obj.get("status") == "ok", obj
assert obj.get("service") == "rum-api", obj
assert sys.argv[2] in str(obj.get("version","")), obj
PY

app_url="$(kctl -n "$DEV_NS" get configmap "$CONFIGMAP" -o jsonpath='{.data.APP_URL}')"
[[ "$app_url" == "https://$DEV_HOST" ]] || { echo "blocked: isolated DEV APP_URL mismatch" >&2; exit 78; }

old_guard="$(kctl -n "$DEV_NS" get configmap "$CONFIGMAP" -o jsonpath='{.data.RAT_CANONICAL_DEV_FIXTURE_SEED_ALLOWED}' 2>/dev/null || true)"
restore_guard() {
  local rc=$?
  trap - ERR
  if [[ "$old_guard" == "true" ]]; then
    kctl -n "$DEV_NS" patch configmap "$CONFIGMAP" --type merge -p '{"data":{"RAT_CANONICAL_DEV_FIXTURE_SEED_ALLOWED":"true"}}' >/dev/null 2>&1 || true
  else
    kctl -n "$DEV_NS" patch configmap "$CONFIGMAP" --type json -p='[{"op":"remove","path":"/data/RAT_CANONICAL_DEV_FIXTURE_SEED_ALLOWED"}]' >/dev/null 2>&1 || true
  fi
  kctl -n "$DEV_NS" rollout restart deployment/rum-api >/dev/null 2>&1 || true
  kctl -n "$DEV_NS" rollout status deployment/rum-api --timeout=240s >/dev/null 2>&1 || true
  exit "$rc"
}
trap restore_guard ERR

kctl -n "$DEV_NS" patch configmap "$CONFIGMAP" --type merge -p '{"data":{"RAT_CANONICAL_DEV_FIXTURE_SEED_ALLOWED":"true"}}' >/dev/null
kctl -n "$DEV_NS" rollout restart deployment/rum-api >/dev/null
kctl -n "$DEV_NS" rollout status deployment/rum-api --timeout=300s

guard="$(kctl -n "$DEV_NS" exec deployment/rum-api -c php-fpm -- printenv RAT_CANONICAL_DEV_FIXTURE_SEED_ALLOWED | tr -d '\r\n')"
runtime_url="$(kctl -n "$DEV_NS" exec deployment/rum-api -c php-fpm -- printenv APP_URL | tr -d '\r\n')"
[[ "$guard" == "true" ]] || { echo "blocked: canonical DEV fixture guard not active" >&2; exit 78; }
[[ "$runtime_url" == "https://$DEV_HOST" ]] || { echo "blocked: runtime APP_URL mismatch" >&2; exit 78; }

# Run twice deliberately: the second run is the deployed idempotency proof.
for pass in 1 2; do
  kctl -n "$DEV_NS" exec deployment/rum-api -c php-fpm -- \
    php artisan db:seed --class='Database\Seeders\RateAnythingCanonicalDevSeeder' --force
  echo "canonical_seed_pass=$pass"
done

verify_search() {
  local query="$1" expected="$2"
  local body
  body="$(mktemp)"
  curl -fsS --retry 6 --retry-delay 2 --max-time 20 \
    -H "Origin: https://$RAT_HOST" -H 'Accept: application/json' \
    "https://$DEV_HOST/api/v1/public/entities/rat-catalogue/search?q=$query" -o "$body"
  python3 - "$body" "$expected" <<'PY'
import json,sys
obj=json.load(open(sys.argv[1], encoding="utf-8"))
meta=obj.get("meta",{})
assert meta.get("source") == "rum-dev-canonical", obj
assert meta.get("privacy") == "private-rum-people-excluded", obj
assert meta.get("found") is True, obj
active=(obj.get("data") or {}).get("active") or {}
assert active.get("name") == sys.argv[2], obj
print(json.dumps({"active":active,"categories":obj.get("data",{}).get("categories",[])}, separators=(",",":")))
PY
  rm -f "$body"
}

dji_payload="$(verify_search DJI DJI)"
grep -Fq '"DJI Mini Pro 5"' <<<"$dji_payload"
grep -Fq '"Osmo Pocket 3"' <<<"$dji_payload"
grep -Fq '"DJI Inspire 3"' <<<"$dji_payload"
grep -Fq '"DJI Air 3S"' <<<"$dji_payload"
grep -Fq '"DJI Mavic 3 Pro"' <<<"$dji_payload"
grep -Fq '"DJI Mic 2"' <<<"$dji_payload"

verify_search CJ "CJ Investigates" >/dev/null

no_match="$(curl -fsS --max-time 20 -H "Origin: https://$RAT_HOST" -H 'Accept: application/json' "https://$DEV_HOST/api/v1/public/entities/rat-catalogue/search?q=owner-no-match-dji-recovery-7cc91")"
python3 - "$no_match" <<'PY'
import json,sys
obj=json.loads(sys.argv[1])
assert obj.get("meta",{}).get("found") is False, obj
assert obj.get("data") is None, obj
assert obj.get("meta",{}).get("privacy") == "private-rum-people-excluded", obj
PY

# Output only the fixture identities and graph edges; no user/person rows.
probe='
$names=["DJI","DJI Mini Pro 5","Osmo Pocket 3","DJI Inspire 3","DJI Air 3S","DJI Mavic 3 Pro","DJI Mic 2","DJI Mini Pro 5 Intelligent Flight Battery","DJI Fly App","DJI LightCut","DJI Mini Pro 5 — First Look","RAT Demo Drone Creator"];
$rows=App\\Models\\Entity::query()->with("type")->whereIn("canonical_name",$names)->where("status","active")->get()->map(fn($e)=>["id"=>(string)$e->id,"name"=>$e->canonical_name,"slug"=>$e->canonical_slug,"type"=>$e->type?->key,"visibility"=>$e->visibility,"publication"=>$e->publication_state])->values();
$ids=$rows->pluck("id")->all();
$rels=Illuminate\\Support\\Facades\\DB::table("entity_relationships as r")->join("relationship_types as t","t.id","=","r.relationship_type_id")->join("entities as s","s.id","=","r.source_entity_id")->join("entities as d","d.id","=","r.target_entity_id")->whereNull("r.valid_to")->whereIn("r.source_entity_id",$ids)->whereIn("r.target_entity_id",$ids)->get(["s.canonical_name as source","t.key as relationship","d.canonical_name as target"])->values();
echo json_encode(["entities"=>$rows,"relationships"=>$rels],JSON_UNESCAPED_SLASHES);
'
fixture_state="$(kctl -n "$DEV_NS" exec deployment/rum-api -c php-fpm -- php artisan tinker --execute="$probe" 2>/dev/null | tail -n 1)"
python3 - "$fixture_state" <<'PY'
import json,sys
obj=json.loads(sys.argv[1])
names=[x["name"] for x in obj["entities"]]
expected=["DJI","DJI Mini Pro 5","Osmo Pocket 3","DJI Inspire 3","DJI Air 3S","DJI Mavic 3 Pro","DJI Mic 2","DJI Mini Pro 5 Intelligent Flight Battery","DJI Fly App","DJI LightCut","DJI Mini Pro 5 — First Look","RAT Demo Drone Creator"]
for name in expected:
    assert names.count(name) == 1, (name,names)
for e in obj["entities"]:
    assert e["visibility"] == "public", e
    assert e["publication"] == "published", e
rels={(x["source"],x["relationship"],x["target"]) for x in obj["relationships"]}
for product in ["DJI Mini Pro 5","Osmo Pocket 3","DJI Inspire 3","DJI Air 3S","DJI Mavic 3 Pro","DJI Mic 2"]:
    assert (product,"manufactured_by","DJI") in rels, (product,rels)
print(json.dumps(obj,separators=(",",":")))
PY

[[ "$(image "$RAT_NS" rum-rate-anything rate-anything)" == "$rat_front_before" ]]
[[ "$(image "$RAT_NS" rum-api php-fpm)" == "$rat_api_before" ]]
[[ "$(image "$LIVE_NS" rum-api php-fpm)" == "$live_api_before" ]]
[[ "$(image "$LIVE_NS" rum-web web)" == "$live_web_before" ]]
[[ "$(image "$LIVE_NS" rum-worker worker)" == "$live_worker_before" ]]
[[ "$(hosts "$LIVE_NS")" == "$live_hosts_before" ]]

trap - ERR
echo "RAT_DJI_CANONICAL_DEV_RECOVERED source_sha=$SOURCE_SHA"
echo "fixture_state=$fixture_state"
echo "live_unchanged=true"
