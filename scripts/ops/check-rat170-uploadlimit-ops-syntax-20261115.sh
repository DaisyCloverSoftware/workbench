#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 0 ]] || exit 64
file="$(dirname "$0")/deploy-rat170-uploadlimit-frontend-only-20261115.sh"
bash -n "$file"
bash -n "$(dirname "$0")/probe-rat170-uploadlimit-after-frontend-20261115.sh"
grep -q 'RAT170_UPLOADLIMIT_FRONTEND_ONLY_ISOLATED_OK' "$file"
grep -q 'client_max_body_size 16m' "$file"
grep -q 'api_source_sha' "$file"
grep -q 'protected_snapshot' "$file"
echo RAT170_UPLOADLIMIT_GUARDED_FRONTEND_STATIC_SYNTAX_PASS=true
