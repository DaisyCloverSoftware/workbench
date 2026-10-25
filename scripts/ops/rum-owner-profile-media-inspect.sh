#!/usr/bin/env bash
set -euo pipefail
umask 077
DEV_NAMESPACE="rum-dev-isolated"
LIVE_NAMESPACE="rum-dev"
DEV_HOST="dev-rum.daisycloversoftware.uk"
LIVE_HOST="rateurmate.online"
for command in kubectl; do command -v "$command" >/dev/null 2>&1 || { echo "missing $command" >&2; exit 2; }; done
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo k3s kubectl "$@"; else kubectl "$@"; fi; }
[[ "$(kctl -n "$DEV_NAMESPACE" get ingress rum -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}')" == "$DEV_HOST" ]] || { echo "BLOCKED: DEV ingress mismatch" >&2; exit 78; }
[[ "$(kctl -n "$LIVE_NAMESPACE" get ingress rum -o jsonpath='{range .spec.rules[*]}{.host}{"\n"}{end}')" == "$LIVE_HOST" ]] || { echo "BLOCKED: LIVE ingress mismatch" >&2; exit 78; }
PHP_BOOT='require "/var/www/html/vendor/autoload.php"; $app=require "/var/www/html/bootstrap/app.php"; $app->make(Illuminate\Contracts\Console\Kernel::class)->bootstrap();'
inspect(){
 local ns="$1" label="$2" pod
 pod="$(kctl -n "$ns" get pods -l 'app.kubernetes.io/instance=rum,app.kubernetes.io/component=api' --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
 [[ -n "$pod" ]] || { echo "BLOCKED: $label api pod missing" >&2; exit 78; }
 kctl -n "$ns" exec "$pod" -c php-fpm -- php -r "$PHP_BOOT
 \$u=App\\Models\\User::query()->where('founder_number',2)->with('profile')->first();
 if(!\$u){echo '${label}_FOUNDER2=missing'; exit;}
 \$p=\$u->profile; \$m=\$p?->avatar_media_id ? App\\Models\\MediaAsset::query()->find(\$p->avatar_media_id) : null;
 \$exists='none'; \$thumb='none';
 if(\$m){try{\$d=Storage::disk(\$m->storage_disk); \$exists=\$d->exists(\$m->storage_key)?'present':'missing'; \$thumb=\$m->thumbnail_key ? (\$d->exists(\$m->thumbnail_key)?'present':'missing') : 'no-key';}catch(Throwable \$e){\$exists='error'; \$thumb='error';}}
 echo '${label}_FOUNDER2_USER_ID='.\$u->id.chr(10);
 echo '${label}_FOUNDER2_USERNAME='.\$u->username.chr(10);
 echo '${label}_FOUNDER2_AVATAR_MEDIA_ID='.(\$p?->avatar_media_id ?: 'none').chr(10);
 echo '${label}_FOUNDER2_MEDIA_ORIGINAL='.\$exists.chr(10);
 echo '${label}_FOUNDER2_MEDIA_THUMBNAIL='.\$thumb.chr(10);
 "
}
inspect "$DEV_NAMESPACE" DEV
inspect "$LIVE_NAMESPACE" LIVE
printf 'LIVE_READ_ONLY=YES\nLIVE_MUTATION=NO\nRATE_ANYTHING_AFFECTED=NO\n'
