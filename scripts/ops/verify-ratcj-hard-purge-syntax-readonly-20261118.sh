#!/usr/bin/env bash
set -Eeuo pipefail
# Read-only syntax/preflight checks for the pinned destructive maintenance script. Never performs cleanup.
[[ $# -eq 0 ]] || exit 64
dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
target="$dir/purge-ratcj-53-rating-events-guarded-20261118.sh"
[[ -f "$target" && ! -L "$target" ]] || exit 66
bash -n "$target"
tmp="$(mktemp -d)";trap 'rm -rf "$tmp"' EXIT
file="$tmp/ratcj-guarded-php.php"
printf '%s\n' '<?php' > "$file"
awk '/^require /{inside=1} /^RATCJ_HARD_DELETE_PHP$/{inside=0} inside{print}' "$target" >> "$file"
grep -F 'RATCJ_HARD_DELETE_TRANSACTION_PASS=' "$file" >/dev/null
grep -F 'SET TRANSACTION ISOLATION LEVEL SERIALIZABLE' "$file" >/dev/null
grep -F 'rating.dev_permanent_cleanup_completed.v1' "$file" >/dev/null
[[ "$(grep -c '^require "vendor/autoload.php";' "$file")" -eq 1 ]] || exit 78
kctl(){ if command -v k3s >/dev/null 2>&1;then sudo -n k3s kubectl "$@";else kubectl "$@";fi; }
kctl -n rum-dev-isolated exec -i deployment/rum-api -c php-fpm -- php -l < "$file"
echo RATCJ_HARD_PURGE_BASH_AND_PHP_SYNTAX_PASS=true
