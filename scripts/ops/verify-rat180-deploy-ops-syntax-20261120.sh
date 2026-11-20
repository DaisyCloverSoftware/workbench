#!/usr/bin/env bash
set -Eeuo pipefail
[[ $# -eq 0 ]] || exit 64
bash -n scripts/ops/deploy-rat180-identical-names-isolated-20261120.sh
bash -n scripts/ops/verify-rat180-identical-name-browser-readonly-20261120.sh
bash -n scripts/ops/accept-rat180-colin-channel-browser-20261120.sh
awk '/^NODE$/{exit} found{print} /<<.NODE.$/{found=1}' scripts/ops/accept-rat180-colin-channel-browser-20261120.sh | node --input-type=module --check
grep -q "sha256:7411cc02eb6568e6c6d228b551683294bfac49a6f89c320078af5550adce5d56" scripts/ops/deploy-rat180-identical-names-isolated-20261120.sh
grep -q "sha256:9a33ea315f52d5e14a5ece0e57895aae8d4eb4d7d7e9955110d49fc9676c64d9" scripts/ops/deploy-rat180-identical-names-isolated-20261120.sh
echo RAT180_GUARDED_DEPLOY_SCRIPT_PARSE_PASS=true
echo RAT180_GUARDED_RELEASE_DIGESTS_PINN=true
