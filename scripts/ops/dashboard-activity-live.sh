#!/usr/bin/env bash
set -euo pipefail

runner="$HOME/.local/bin/workbench-runner"
if [ ! -x "$runner" ]; then
  echo 'runner_available=false'
  exit 1
fi

printf 'runner_version='
"$runner" version
printf '%s\n' '{"action":"chat_activity","limit":10}' | "$runner" tool-json
