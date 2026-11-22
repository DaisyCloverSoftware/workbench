#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 0 ]] || exit 64
src=24bcceb6bba37e16bf6416ccd1df946cfed5a422
img=ghcr.io/daisycloversoftware/rum-api@sha256:f4e700314dd2fe2b8b6dabee5f640f35b9463d914f8a2be79ff781862a80e978
command -v docker >/dev/null || { echo 'No disposable Docker runtime';exit 78; }; docker info --format '{{.ServerVersion}}' >/dev/null || exit 78
w=$(mktemp -d);run=rat181-fixture-$(basename "$w" | tr '[:upper:].' '[:lower:]-');db=$run-pg;net=$run-net;app=$run-php;ven=$run-deps;owner=$(id -u):$(id -g)
cleanup(){ docker rm -f "$app" "$db" >/dev/null 2>&1 || :;docker network rm "$net" >/dev/null 2>&1 || :;docker volume rm "$ven" >/dev/null 2>&1 || :;rm -rf "$w"; };trap cleanup EXIT
# No cluster context, host ports, production volumes, service credentials or worktree writes.
git -C "$w" init -q;git -C "$w" remote add origin https://github.com/DaisyCloverSoftware/rum.git;git -C "$w" fetch --depth=1 -q origin "$src";git -C "$w" checkout -q --detach FETCH_HEAD;[[ $(git -C "$w" rev-parse HEAD) == "$src" ]]
docker image inspect "$img" >/dev/null 2>&1 || docker pull -q "$img" >/dev/null
docker volume create "$ven" >/dev/null;docker run --rm --name "$app" --entrypoint sh -v "$w/apps/api:/fixture:ro" -v "$ven:/repo" -w /repo "$img" -ec 'cp -R /fixture/. /repo; composer install --no-interaction --no-progress --prefer-dist >/tmp/rat181-composer.log 2>&1 || { tail -n 14 /tmp/rat181-composer.log;exit 1; }; vendor/bin/pint --test app/Services/StandaloneConnectionIntent.php tests/Feature/Entity/StandaloneConnectionIntentTest.php' 
docker network create --internal "$net" >/dev/null;docker run -d --rm --name "$db" --network "$net" --network-alias fixturepg --tmpfs /var/lib/postgresql -e POSTGRES_USER=rum -e POSTGRES_DB=rum_test -e POSTGRES_PASSWORD=fixture-only postgres:18 >/dev/null
for i in {1..20};do docker exec "$db" pg_isready -U rum -d rum_test >/dev/null && break;sleep 1;done
# The test PHP process sees only the newly-created fixture network.
docker run --rm --name "$app" --network "$net" -v "$ven:/repo" -w /repo -e APP_ENV=testing -e DB_CONNECTION=pgsql -e DB_HOST=fixturepg -e DB_DATABASE=rum_test -e DB_USERNAME=rum -e DB_PASSWORD=fixture-only -e CACHE_STORE=array -e QUEUE_CONNECTION=sync -e SESSION_DRIVER=array -e MAIL_MAILER=array --entrypoint php "$img" vendor/bin/phpunit tests/Feature/Entity/StandaloneConnectionIntentTest.php
echo "DISPOSABLE_DB_LIFECYCLE_PASS=$src"