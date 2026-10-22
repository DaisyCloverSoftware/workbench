#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <rating-event-ulid> <canonical-entity-ulid>" >&2
  exit 64
fi

RATING_EVENT_ID="${1,,}"
CANONICAL_ENTITY_ID="${2,,}"
RUM_NS="rum-dev-isolated"
RAT_NS="rum-rate-anything-preview"

[[ "$RATING_EVENT_ID" =~ ^[0-9a-hjkmnp-tv-z]{26}$ ]] || { echo "invalid rating event ULID" >&2; exit 64; }
[[ "$CANONICAL_ENTITY_ID" =~ ^[0-9a-hjkmnp-tv-z]{26}$ ]] || { echo "invalid canonical entity ULID" >&2; exit 64; }

kctl() {
  if command -v k3s >/dev/null 2>&1; then
    sudo -n k3s kubectl "$@"
  else
    kubectl "$@"
  fi
}

api_pod() {
  local ns="$1"
  local pod
  pod="$(kctl -n "$ns" get pods \
    -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' \
    --field-selector=status.phase=Running \
    -o jsonpath='{.items[0].metadata.name}')"
  [[ -n "$pod" ]] || { echo "no running API pod in $ns" >&2; exit 78; }
  printf '%s\n' "$pod"
}

RUM_API_POD="$(api_pod "$RUM_NS")"
RAT_API_POD="$(api_pod "$RAT_NS")"

canonical_json="$(kctl -n "$RUM_NS" exec "$RUM_API_POD" -c php-fpm -- php artisan tinker --execute="\
\$event=App\\Models\\RatingEvent::query()->find('${RATING_EVENT_ID}'); \
\$entity=App\\Models\\Entity::query()->find('${CANONICAL_ENTITY_ID}'); \
echo json_encode(['eventExists'=>(bool)\$event,'entityExists'=>(bool)\$entity,'eventId'=>\$event?->id,'targetEntityId'=>\$event?->target_entity_id,'status'=>\$event?->status,'source'=>\$event?->source,'entityId'=>\$entity?->id,'entityName'=>\$entity?->canonical_name,'entityCount'=>App\\Models\\Entity::query()->whereKey('${CANONICAL_ENTITY_ID}')->count(),'activeRatingCount'=>App\\Models\\RatingEvent::query()->where('id','${RATING_EVENT_ID}')->where('target_entity_id','${CANONICAL_ENTITY_ID}')->where('status','active')->count()]);")"

preview_json="$(kctl -n "$RAT_NS" exec "$RAT_API_POD" -c php-fpm -- php artisan tinker --execute="\
echo json_encode(['eventCount'=>App\\Models\\RatingEvent::query()->whereKey('${RATING_EVENT_ID}')->count(),'targetRatingCount'=>App\\Models\\RatingEvent::query()->where('target_entity_id','${CANONICAL_ENTITY_ID}')->where('status','active')->count()]);")"

python3 - "$canonical_json" "$preview_json" "$RATING_EVENT_ID" "$CANONICAL_ENTITY_ID" <<'PY'
import json,re,sys

def payload(raw):
    matches = re.findall(r'\{[^\n]*\}', raw)
    if not matches:
        raise SystemExit(f'no JSON payload in tinker output: {raw!r}')
    return json.loads(matches[-1])

canonical = payload(sys.argv[1])
preview = payload(sys.argv[2])
event_id = sys.argv[3]
entity_id = sys.argv[4]
assert canonical['eventExists'] is True, canonical
assert canonical['entityExists'] is True, canonical
assert str(canonical['eventId']).lower() == event_id, canonical
assert str(canonical['targetEntityId']).lower() == entity_id, canonical
assert str(canonical['entityId']).lower() == entity_id, canonical
assert canonical['status'] == 'active', canonical
assert canonical['source'] == 'rate_anything', canonical
assert canonical['entityCount'] == 1, canonical
assert canonical['activeRatingCount'] == 1, canonical
assert preview['eventCount'] == 0, preview
assert preview['targetRatingCount'] == 0, preview
print('canonical_rating_event=' + event_id)
print('canonical_entity=' + entity_id)
print('canonical_entity_name=' + str(canonical['entityName']))
print('canonical_source=' + str(canonical['source']))
print('canonical_entity_count=1')
print('canonical_exact_event_count=1')
print('rat_preview_exact_event_count=0')
print('rat_preview_target_rating_count=0')
print('RUM160_CANONICAL_RATING_DB_VERIFY_PASS=true')
PY
