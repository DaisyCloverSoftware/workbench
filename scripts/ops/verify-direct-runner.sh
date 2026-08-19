#!/usr/bin/env bash
set -euo pipefail

printf 'WORKBENCH_OPERATION_SCRIPT_OK\n'
printf 'commit=%s\n' "${WORKBENCH_OPERATION_COMMIT:-missing}"
printf 'host=%s\n' "$(hostname)"
