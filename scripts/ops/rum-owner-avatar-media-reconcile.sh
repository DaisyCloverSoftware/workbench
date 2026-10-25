#!/usr/bin/env bash
set -euo pipefail
umask 077
MODE="${1:-inspect}"
case "$MODE" in inspect|repair) ;; *) echo "usage: $0 [inspect|repair]" >&2; exit 64;; esac
DEV_NAMESPACE="rum-dev-isolated"; LIVE_NAMESPACE="rum-dev"; DEV_HOST="dev-rum.daisycloversoftware.uk"; LIVE_HOST="rateurmate.online"
for command in kubectl mktemp sha256sum; do command -v "$command" >/dev/null 2>&1 || { echo "missing $command" >&2; exit 2; }; done
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n "$DEV_NAMESPACE" get ingress rum -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}')" == "$DEV_HOST" ]] || { echo "BLOCKED: DEV ingress mismatch" >&2; exit 78; }
[[ "$(kctl -n "$LIVE_NAMESPACE" get ingress rum -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}')" == "$LIVE_HOST" ]] || { echo "BLOCKED: LIVE ingress mismatch" >&2; exit 78; }
dev_pod="$(kctl -n "$DEV_NAMESPACE" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
live_pod="$(kctl -n "$LIVE_NAMESPACE" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
[[ -n "$dev_pod" && -n "$live_pod" ]] || { echo "BLOCKED: API pod missing" >&2; exit 78; }
BOOT='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();'
MEDIA_PROBE="$BOOT \$u=App\\Models\\User::query()->where('founder_number',2)->with('profile')->firstOrFail(); if(!\$u->profile?->avatar_media_id){fwrite(STDERR,'avatar missing');exit(3);} echo \$u->profile->avatar_media_id;"
dev_id="$(kctl -n "$DEV_NAMESPACE" exec "$dev_pod" -c php-fpm -- php -r "$MEDIA_PROBE" 2>/dev/null | tail -n 1)"
live_id="$(kctl -n "$LIVE_NAMESPACE" exec "$live_pod" -c php-fpm -- php -r "$MEDIA_PROBE" 2>/dev/null | tail -n 1)"
[[ "$dev_id" =~ ^[0-9a-z]{26}$ && "$dev_id" == "$live_id" ]] || { echo "BLOCKED: Founder #2 avatar media identity mismatch dev=$dev_id live=$live_id" >&2; exit 78; }
META="$BOOT \$a=App\\Models\\MediaAsset::findOrFail(\$argv[1]); \$k=\$a->thumbnail_key; if(!\$k){echo 'no-key';exit;} try{\$d=Illuminate\\Support\\Facades\\Storage::disk(\$a->storage_disk); if(!\$d->exists(\$k)){echo 'missing';exit;} \$b=\$d->get(\$k); echo 'present '.strlen(\$b).' '.hash('sha256',\$b);}catch(Throwable \$e){echo 'error '.get_class(\$e).' '.str_replace([chr(10),chr(13)],' ',\$e->getMessage()); exit(5);}"
READ="$BOOT \$a=App\\Models\\MediaAsset::findOrFail(\$argv[1]); \$s=Illuminate\\Support\\Facades\\Storage::disk(\$a->storage_disk)->readStream(\$a->thumbnail_key); if(!is_resource(\$s)){exit(4);} while(!feof(\$s)){\$c=fread(\$s,1048576); if(\$c===false){exit(5);} echo \$c;}"
WRITE="$BOOT \$a=App\\Models\\MediaAsset::findOrFail(\$argv[1]); \$b=stream_get_contents(STDIN); if(\$b===false||strlen(\$b)<100){exit(4);} \$ok=Illuminate\\Support\\Facades\\Storage::disk(\$a->storage_disk)->put(\$a->thumbnail_key,\$b); if(!\$ok){exit(5);} echo strlen(\$b).' '.hash('sha256',\$b);"
meta(){ kctl -n "$1" exec "$2" -c php-fpm -- php -r "$META" -- "$dev_id" 2>&1 | tail -n 1; }
live_meta="$(meta "$LIVE_NAMESPACE" "$live_pod")"; dev_meta="$(meta "$DEV_NAMESPACE" "$dev_pod")"
printf 'RUM_OWNER_AVATAR_MEDIA_ID=%s\nOWNER_AVATAR_LIVE=%s\nOWNER_AVATAR_DEV=%s\n' "$dev_id" "$live_meta" "$dev_meta"
[[ "$live_meta" == present\ * ]] || { echo "BLOCKED: LIVE stored avatar thumbnail cannot be read safely" >&2; exit 78; }
if [[ "$MODE" == inspect ]]; then printf 'OWNER_AVATAR_INSPECT_COMPLETE=YES\nLIVE_READ_ONLY=YES\nLIVE_MUTATION=NO\n'; exit 0; fi
live_size="$(awk '{print $2}' <<<"$live_meta")"; live_hash="$(awk '{print $3}' <<<"$live_meta")"
if [[ "$dev_meta" == present\ * ]]; then
  [[ "$(awk '{print $2}' <<<"$dev_meta")" == "$live_size" && "$(awk '{print $3}' <<<"$dev_meta")" == "$live_hash" ]] || { echo "BLOCKED: DEV avatar exists with different bytes" >&2; exit 78; }
  printf 'OWNER_AVATAR_ALREADY_MATCHED=YES size=%s sha256=%s\n' "$live_size" "$live_hash"
elif [[ "$dev_meta" == missing ]]; then
  work="$(mktemp -d)"; trap 'rm -rf "$work"' EXIT HUP INT TERM
  file="$work/avatar.thumbnail"
  kctl -n "$LIVE_NAMESPACE" exec "$live_pod" -c php-fpm -- php -r "$READ" -- "$dev_id" >"$file"
  [[ "$(wc -c <"$file" | tr -d ' ')" == "$live_size" && "$(sha256sum "$file" | awk '{print $1}')" == "$live_hash" ]] || { echo "BLOCKED: read-only LIVE export fingerprint mismatch" >&2; exit 78; }
  written="$(kctl -n "$DEV_NAMESPACE" exec -i "$dev_pod" -c php-fpm -- php -r "$WRITE" -- "$dev_id" <"$file")"
  [[ "$written" == "$live_size $live_hash" ]] || { echo "ERROR: isolated DEV avatar write verification failed" >&2; exit 1; }
  final="$(meta "$DEV_NAMESPACE" "$dev_pod")"; [[ "$final" == "present $live_size $live_hash" ]] || { echo "ERROR: isolated DEV avatar post-write mismatch: $final" >&2; exit 1; }
  printf 'OWNER_AVATAR_REPAIRED_IN_ISOLATED_DEV=YES size=%s sha256=%s\n' "$live_size" "$live_hash"
else
  echo "BLOCKED: DEV avatar state cannot be repaired safely: $dev_meta" >&2; exit 78
fi
printf 'OWNER_AVATAR_RECONCILE_COMPLETE=YES\nLIVE_READ_ONLY=YES\nLIVE_MUTATION=NO\nRATE_ANYTHING_AFFECTED=NO\n'
