#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# == 0 ]] || exit 64
ops=$(cd "$(dirname "$0")" && pwd);sha=b135949c8d97072cca3b24c2611984ae6d6fc30c
# No deployment command is reached without successfully reading the exact run/job receipts.
# Both image names and immutable digests come from successful GitHub publication outputs, not caller input or mutable tags.
lock="${TMPDIR:-/tmp}/rat181-isolated-deploy-b135949c.lock";exec 9>"$lock";flock -n 9 || { echo CONCURRENT_RELEASE_NO_WRITE;exit 75; }
t=$(mktemp -d);trap 'rm -rf "$t"' EXIT
bash "$ops/rat181-published-image-receipts.sh" | tee "$t/published";grep -q '^NINE_JOBS_GREEN ' "$t/published" || exit 78
api=$(awk -v s="$sha" '$1=="PUBLISHED" && $2=="api" && $4==s {print $5}' "$t/published");rat=$(awk -v s="$sha" '$1=="PUBLISHED" && $2=="rate-anything" && $4==s {print $5}' "$t/published")
[[ $api =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ && $rat =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || exit 78
# Double-check protected canonical DB records and their audience disposition before any workload patch.
bash "$ops/check-rat181-protected-baseline.sh" >"$t/before";grep -Fxq 'entities rows=3 sha256=5112dc1f68d8b48f6c5c41968619ff9d362de16d66b637b2dd6c60271ff9e443' "$t/before";grep -Fxq 'entity_identifiers rows=1 sha256=db111e02ca58748931dd5252ff6fc4ab03332bcf37678297ff241c3ca821f791' "$t/before";grep -Fxq 'rating_events rows=2 sha256=51fd8c48c16b3ec3ad33a5d7a70031c9f0122a51bad8bbb8cd4b8d45c25336e4' "$t/before";grep -Fxq 'relationships rows=2 sha256=688a57ffb9558a55e5098d3a04006abdb733446d15bd6828077263538b088c7c' "$t/before"
bash "$ops/deploy-rat181-linked-things-isolated.sh" "$sha" "$api" "$rat"
bash "$ops/check-rat181-protected-baseline.sh" >"$t/after";grep -E '^((entities|entity_identifiers|rating_events|relationships) rows=|AUDIENCE |ROUTING )' "$t/before" >"$t/b";grep -E '^((entities|entity_identifiers|rating_events|relationships) rows=|AUDIENCE |ROUTING )' "$t/after" >"$t/a";diff -u "$t/b" "$t/a";cat "$t/a";echo 'PR181_ROLLOUT_AND_PROTECTED_AUDIT_PASS'
