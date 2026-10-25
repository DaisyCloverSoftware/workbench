#!/usr/bin/env bash
set -euo pipefail
umask 077

MODE="${1:-inspect}"
case "$MODE" in
  inspect|repair) ;;
  *) echo "usage: $0 [inspect|repair]" >&2; exit 64 ;;
esac

DEV_NAMESPACE="rum-dev-isolated"
LIVE_NAMESPACE="rum-dev"
RELEASE="rum"
DEV_HOST="dev-rum.daisycloversoftware.uk"
LIVE_HOST="rateurmate.online"
MEDIA_IDS=(
  "01kzgsgvdagc48ptewqqqgy9xs"
  "01m0b6y8n9ca0qhwer9hevq1ng"
)

for command in kubectl mktemp sha256sum; do
  command -v "$command" >/dev/null 2>&1 || {
    echo "ERROR: required command unavailable: $command" >&2
    exit 2
  }
done

kctl() {
  if command -v k3s >/dev/null 2>&1; then
    sudo k3s kubectl "$@"
  else
    kubectl "$@"
  fi
}

# Fail closed unless the DEV/LIVE host boundary is exactly as expected.
dev_hosts="$(kctl -n "$DEV_NAMESPACE" get ingress "$RELEASE" -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}')"
live_hosts="$(kctl -n "$LIVE_NAMESPACE" get ingress "$RELEASE" -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}')"
[[ "$dev_hosts" == "$DEV_HOST" ]] || { echo "BLOCKED: isolated DEV ingress boundary mismatch" >&2; exit 78; }
[[ "$live_hosts" == "$LIVE_HOST" ]] || { echo "BLOCKED: LIVE ingress boundary mismatch" >&2; exit 78; }

dev_api_pod="$(kctl -n "$DEV_NAMESPACE" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
live_api_pod="$(kctl -n "$LIVE_NAMESPACE" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
[[ -n "$dev_api_pod" && -n "$live_api_pod" ]] || { echo "BLOCKED: API pod missing" >&2; exit 78; }

PHP_BOOTSTRAP='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();'
PHP_META="$PHP_BOOTSTRAP \$a=App\\Models\\MediaAsset::find(\$argv[1]); if(!\$a){fwrite(STDERR,\"media record missing\\n\"); exit(3);} \$k=\$a->thumbnail_key; if(!\$k){fwrite(STDERR,\"thumbnail key missing\\n\"); exit(4);} try { \$d=Storage::disk(\$a->storage_disk); \$e=\$d->exists(\$k); if(!\$e){echo \"missing\"; exit(0);} \$b=\$d->get(\$k); echo \"present \".strlen(\$b).\" \".hash('sha256',\$b); } catch(Throwable \$e){ echo \"error \".get_class(\$e); exit(5); }"
PHP_READ="$PHP_BOOTSTRAP \$a=App\\Models\\MediaAsset::find(\$argv[1]); if(!\$a || !\$a->thumbnail_key){exit(3);} \$s=Storage::disk(\$a->storage_disk)->readStream(\$a->thumbnail_key); if(!is_resource(\$s)){exit(4);} while(!feof(\$s)){\$c=fread(\$s,1048576); if(\$c===false){exit(5);} echo \$c;}"
PHP_WRITE="$PHP_BOOTSTRAP \$a=App\\Models\\MediaAsset::find(\$argv[1]); if(!\$a || !\$a->thumbnail_key){exit(3);} \$b=stream_get_contents(STDIN); if(\$b===false || strlen(\$b)===0){fwrite(STDERR,\"empty input\\n\"); exit(4);} if(!Storage::disk(\$a->storage_disk)->put(\$a->thumbnail_key,\$b)){fwrite(STDERR,\"storage write failed\\n\"); exit(5);} echo strlen(\$b).\" \".hash('sha256',\$b);"

meta() {
  local ns="$1" pod="$2" id="$3"
  kctl -n "$ns" exec "$pod" -c php-fpm -- php -r "$PHP_META" -- "$id"
}

printf 'RUM_MEDIA_RECONCILE_MODE=%s\n' "$MODE"
printf 'RUM_MEDIA_BOUNDARY=dev:%s live:%s\n' "$DEV_HOST" "$LIVE_HOST"

if [[ "$MODE" == "inspect" ]]; then
  for id in "${MEDIA_IDS[@]}"; do
    live_meta="$(meta "$LIVE_NAMESPACE" "$live_api_pod" "$id")"
    dev_meta="$(meta "$DEV_NAMESPACE" "$dev_api_pod" "$id")"
    printf 'MEDIA id=%s live=%s dev=%s\n' "$id" "$live_meta" "$dev_meta"
  done
  printf 'RUM_MEDIA_INSPECT_COMPLETE=1\n'
  exit 0
fi

work="$(mktemp -d)"
cleanup() { rm -rf "$work"; }
trap cleanup EXIT HUP INT TERM

for id in "${MEDIA_IDS[@]}"; do
  live_meta="$(meta "$LIVE_NAMESPACE" "$live_api_pod" "$id")"
  dev_meta="$(meta "$DEV_NAMESPACE" "$dev_api_pod" "$id")"
  [[ "$live_meta" == present\ * ]] || { echo "BLOCKED: LIVE source thumbnail unavailable for $id ($live_meta)" >&2; exit 78; }

  live_size="$(awk '{print $2}' <<<"$live_meta")"
  live_hash="$(awk '{print $3}' <<<"$live_meta")"

  if [[ "$dev_meta" == present\ * ]]; then
    dev_size="$(awk '{print $2}' <<<"$dev_meta")"
    dev_hash="$(awk '{print $3}' <<<"$dev_meta")"
    [[ "$dev_size" == "$live_size" && "$dev_hash" == "$live_hash" ]] || {
      echo "BLOCKED: isolated DEV already has a different thumbnail for $id" >&2
      exit 78
    }
    printf 'MEDIA_ALREADY_MATCHED id=%s size=%s sha256=%s\n' "$id" "$dev_size" "$dev_hash"
    continue
  fi
  [[ "$dev_meta" == "missing" ]] || { echo "BLOCKED: isolated DEV thumbnail state is not safely repairable for $id ($dev_meta)" >&2; exit 78; }

  object_file="$work/$id.thumbnail"
  kctl -n "$LIVE_NAMESPACE" exec "$live_api_pod" -c php-fpm -- php -r "$PHP_READ" -- "$id" >"$object_file"
  local_size="$(wc -c <"$object_file" | tr -d ' ')"
  local_hash="$(sha256sum "$object_file" | awk '{print $1}')"
  [[ "$local_size" == "$live_size" && "$local_hash" == "$live_hash" ]] || {
    echo "BLOCKED: exported LIVE thumbnail fingerprint mismatch for $id" >&2
    exit 78
  }

  write_meta="$(kctl -n "$DEV_NAMESPACE" exec -i "$dev_api_pod" -c php-fpm -- php -r "$PHP_WRITE" -- "$id" <"$object_file")"
  [[ "$write_meta" == "$local_size $local_hash" ]] || {
    echo "ERROR: isolated DEV thumbnail write verification failed for $id" >&2
    exit 1
  }
  final_meta="$(meta "$DEV_NAMESPACE" "$dev_api_pod" "$id")"
  [[ "$final_meta" == "present $live_size $live_hash" ]] || {
    echo "ERROR: isolated DEV thumbnail post-write fingerprint mismatch for $id" >&2
    exit 1
  }
  printf 'MEDIA_REPAIRED id=%s size=%s sha256=%s\n' "$id" "$live_size" "$live_hash"
done

printf 'RUM_MEDIA_REPAIR_COMPLETE=1\n'
