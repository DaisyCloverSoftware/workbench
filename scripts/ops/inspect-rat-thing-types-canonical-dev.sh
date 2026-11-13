#!/usr/bin/env bash
set -euo pipefail

# Read-only, canonical-DEV-specific inspection for PR #169.
# No SQL mutations, migrations, seeds, resets, workload changes or LIVE reads
# beyond checking ingress and image ownership.
[[ $# -eq 0 ]] || { echo 'usage: inspect-rat-thing-types-canonical-dev.sh' >&2; exit 64; }
kctl() {
  if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi
}
hosts() {
  kctl -n "$1" get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u
}
for ns in rum-dev-isolated rum-rate-anything-preview rum-dev; do
  kctl get namespace "$ns" >/dev/null
done
[[ "$(hosts rum-dev-isolated)" == 'dev-rum.daisycloversoftware.uk' ]] || exit 78
[[ "$(hosts rum-rate-anything-preview)" == 'dev-rum-ra.daisycloversoftware.uk' ]] || exit 78
[[ "$(hosts rum-dev)" == 'rateurmate.online' ]] || exit 78
runtime_url="$(kctl -n rum-dev-isolated get configmap rum-config -o jsonpath='{.data.APP_URL}')"
[[ "$runtime_url" == 'https://dev-rum.daisycloversoftware.uk' ]] || exit 78

api_image="$(kctl -n rum-dev-isolated get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}')"
frontend_image="$(kctl -n rum-rate-anything-preview get deployment rum-rate-anything -o 'jsonpath={.spec.template.spec.containers[?(@.name=="rate-anything")].image}')"
[[ -n "$api_image" && -n "$frontend_image" ]] || exit 78

# Fixed, read-only Eloquent/Schema queries only. Do not emit identities, tokens,
# connection strings, records or any classification/free-text names.
probe='
require "vendor/autoload.php";
$app = require "bootstrap/app.php";
$app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();
$migration = "2026_11_13_000001_map_thing_classifications_to_entity_types";
$column = Illuminate\Support\Facades\Schema::hasColumn("classifications", "entity_type_id");
$migrated = Illuminate\Support\Facades\DB::table("migrations")->where("migration", $migration)->exists();
$scheme = Illuminate\Support\Facades\DB::table("rating_schemes")->where("key", "rum_rate_anything_general_v1")->value("entity_type_constraints");
$constraints = is_string($scheme) ? json_decode($scheme, true) : $scheme;
$organisation = Illuminate\Support\Facades\DB::table("entity_types")->where("key", "organisation")->value("id");
$collective = Illuminate\Support\Facades\DB::table("classifications")->where("key", "creator_collective")->first();
$collectiveMapped = $column && $collective !== null && (string)$collective->entity_type_id === (string)$organisation;
$mapped = $column ? Illuminate\Support\Facades\DB::table("classifications")->whereNotNull("entity_type_id")->count() : null;
$counts = [
  "classifications" => Illuminate\Support\Facades\DB::table("classifications")->count(),
  "synonyms" => Illuminate\Support\Facades\DB::table("classification_synonyms")->count(),
  "assignments" => Illuminate\Support\Facades\DB::table("entity_classifications")->count(),
  "entities" => Illuminate\Support\Facades\DB::table("entities")->count(),
  "rating_events" => Illuminate\Support\Facades\DB::table("rating_events")->count(),
];
$result = [
  "migration" => $migration, "migration_recorded" => $migrated,
  "mapping_column_present" => $column, "mapped_classifications" => $mapped,
  "collective_mapped_to_organisation" => $collectiveMapped,
  "rating_scheme_present" => $scheme !== null,
  "digital_resource_rateable" => is_array($constraints) && in_array("digital_resource", $constraints, true),
  "counts" => $counts,
];
echo "RAT_CANONICAL_DEV_CATALOGUE_DB=".json_encode($result, JSON_THROW_ON_ERROR)."\n";
'
readback="$(kctl -n rum-dev-isolated exec deployment/rum-api -c php-fpm -- php -r "$probe")"
printf '%s\n' "$readback" | grep '^RAT_CANONICAL_DEV_CATALOGUE_DB='
echo "RAT_CANONICAL_DEV_API_IMAGE=$api_image"
echo "RAT_ISOLATED_FRONTEND_IMAGE=$frontend_image"
echo 'RAT_TYPE_CATALOGUE_INSPECTION_READ_ONLY=true'
