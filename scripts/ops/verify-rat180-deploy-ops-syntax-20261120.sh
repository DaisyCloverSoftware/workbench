#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 0 ]] || exit 64
bash -n scripts/ops/deploy-rat180-identical-names-isolated-20261120.sh
grep -q "__PR180_EXACT_HEAD_API_IMMUTABLE_DIGEST_PENDING__" scripts/ops/deploy-rat180-identical-names-isolated-20261120.sh
grep -q "__PR180_EXACT_HEAD_RAT_IMMUTABLE_DIGEST_PENDING__" scripts/ops/deploy-rat180-identical-names-isolated-20261120.sh
echo RAT180_GUARDED_DEPLOY_SCRIPT_PARSE_PASS=true
echo RAT180_DEPLOYMENT_LOCKED_PENDING_PUBLISHED_DIGEST_PIN=true
