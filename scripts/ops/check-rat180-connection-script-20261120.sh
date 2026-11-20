#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 0 ]] || exit 64
bash -n scripts/ops/connect-rat180-colin-channel-unverified-20261120.sh
grep -Fq "RAT180_UNVERIFIED_CANONICAL_EDGE_CREATED" scripts/ops/connect-rat180-colin-channel-unverified-20261120.sh
printf 'RAT180_CANONICAL_CONNECTION_OPERATION_SYNTAX_PASS=true\n'
