#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="rum-dev"
LIVE_HOST="rateurmate.online"
KUBECTL=(sudo -n k3s kubectl)

command -v sudo >/dev/null 2>&1 || { echo "required command unavailable: sudo" >&2; exit 2; }
command -v k3s >/dev/null 2>&1 || { echo "required command unavailable: k3s" >&2; exit 2; }

actual_host="$(${KUBECTL[@]} get ingress rum -n "$NAMESPACE" -o jsonpath='{.spec.rules[0].host}')"
[[ "$actual_host" == "$LIVE_HOST" ]] || { echo "refusing: $NAMESPACE ingress host is $actual_host" >&2; exit 78; }

pod="$(${KUBECTL[@]} get pods -n "$NAMESPACE" -l app.kubernetes.io/component=api --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
[[ -n "$pod" ]] || { echo "no running LIVE API pod" >&2; exit 2; }

php_code='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\\Contracts\\Console\\Kernel::class)->bootstrap(); $rows=App\\Models\\RatingCategory::query()->where("status","active")->orderBy("sort_order")->orderBy("name")->get(["id","slug","name"]); echo "ACTIVE_CATEGORY_COUNT=".$rows->count().PHP_EOL; foreach($rows as $row){echo "CATEGORY=".$row->slug."|".$row->name.PHP_EOL;}'

${KUBECTL[@]} exec -n "$NAMESPACE" "$pod" -c php-fpm -- php -r "$php_code"
printf 'LIVE_READ_ONLY=YES\n'
printf 'LIVE_MUTATION=NO\n'
printf 'LIVE_HOST=%s\n' "$actual_host"
