#!/usr/bin/env bash
set -euo pipefail
umask 077

DEV_NAMESPACE="rum-dev-isolated"
DEV_HOST="dev-rum.daisycloversoftware.uk"
LIVE_HOST="rateurmate.online"

kctl(){ if command -v k3s >/dev/null 2>&1; then sudo k3s kubectl "$@"; else kubectl "$@"; fi; }

dev_hosts="$(kctl -n "$DEV_NAMESPACE" get ingress rum -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}' 2>/dev/null || true)"
[[ "$dev_hosts" == "$DEV_HOST" ]] || { echo "INSPECT BLOCKED: isolated DEV ingress mismatch" >&2; exit 78; }
if grep -Fqx "$LIVE_HOST" <<<"$dev_hosts"; then echo "INSPECT BLOCKED: DEV ingress contains LIVE host" >&2; exit 78; fi
api_pod="$(kctl -n "$DEV_NAMESPACE" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
[[ -n "$api_pod" ]] || { echo "INSPECT BLOCKED: DEV API pod unavailable" >&2; exit 78; }

read -r -d '' php_code <<'PHP' || true
require '/var/www/html/vendor/autoload.php';
$app = require '/var/www/html/bootstrap/app.php';
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$entities = App\Models\Entity::query()
    ->where('canonical_name', 'CJ Investigates')
    ->get(['id','canonical_name','visibility','rateability','status','publication_state','entity_type_id']);
$entityIds = $entities->pluck('id')->all();
$events = $entityIds === [] ? collect() : App\Models\RatingEvent::query()
    ->whereIn('target_entity_id', $entityIds)
    ->orderBy('submitted_at')
    ->get(['id','target_entity_id','status','source','audience','submitted_at']);
$eventIds = $events->pluck('id')->all();
$hasAdmission = Illuminate\Support\Facades\Schema::hasTable('rating_event_admission_states');
$states = (!$hasAdmission || $eventIds === []) ? collect() : Illuminate\Support\Facades\DB::table('rating_event_admission_states')
    ->whereIn('rating_event_id', $eventIds)
    ->orderBy('rating_event_id')
    ->get(['rating_event_id','state','keep_votes','sin_bin_votes','score']);
$result = [
    'entities' => $entities->map(fn ($e) => [
        'id' => (string) $e->id,
        'name' => (string) $e->canonical_name,
        'visibility' => (string) $e->visibility,
        'rateability' => (string) $e->rateability,
        'status' => (string) $e->status,
        'publicationState' => (string) $e->publication_state,
        'entityTypeId' => (string) $e->entity_type_id,
    ])->values()->all(),
    'events' => $events->map(fn ($e) => [
        'id' => (string) $e->id,
        'targetEntityId' => (string) $e->target_entity_id,
        'status' => (string) $e->status,
        'source' => (string) $e->source,
        'audience' => (string) $e->audience,
        'submittedAt' => $e->submitted_at?->toAtomString(),
    ])->values()->all(),
    'admissionTablePresent' => $hasAdmission,
    'admissionStates' => $states->map(fn ($s) => [
        'ratingEventId' => (string) $s->rating_event_id,
        'state' => (string) $s->state,
        'keepVotes' => (int) $s->keep_votes,
        'sinBinVotes' => (int) $s->sin_bin_votes,
        'score' => (int) $s->score,
    ])->values()->all(),
];
echo json_encode($result, JSON_UNESCAPED_SLASHES|JSON_THROW_ON_ERROR), PHP_EOL;
PHP

# This probe is deliberately read-only. Reject accidental write-capable PHP before execution.
if grep -Eqi -- '->(create|update|delete|save|forceFill|insert|upsert|increment|decrement)\s*\(|::(create|update|delete|insert|upsert)\s*\(' <<<"$php_code"; then
  echo "INSPECT BLOCKED: write-capable PHP detected" >&2
  exit 78
fi
encoded="$(printf '%s' "$php_code" | base64 -w0)"
kctl -n "$DEV_NAMESPACE" exec "$api_pod" -- php -r "eval(base64_decode('${encoded}'));"
printf 'DEV_READ_ONLY=YES\n'
printf 'LIVE_MUTATION=NO\n'
printf 'RATE_ANYTHING_AFFECTED=NO\n'
