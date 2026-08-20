#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = "--self-test" ]; then
  echo "restart-workbench-relay-delayed:self-test=ok"
  exit 0
fi
if [ "$#" -ne 0 ]; then
  echo "usage: restart-workbench-relay-delayed.sh" >&2
  exit 2
fi

systemd_run="$(command -v systemd-run || true)"
systemctl_bin="$(command -v systemctl || true)"
if [ -z "$systemd_run" ] || [ -z "$systemctl_bin" ]; then
  echo "systemd user tools are unavailable" >&2
  exit 1
fi
if ! "$systemctl_bin" --user show-environment >/dev/null 2>&1; then
  echo "systemd user manager is unavailable" >&2
  exit 1
fi

unit="workbench-relay-restart-$(date +%s)-$$"
"$systemd_run" --user --quiet --collect \
  --unit="$unit" \
  --on-active=60s \
  "$systemctl_bin" --user restart workbench-github-relay.service

printf 'scheduled_unit=%s\n' "$unit"
printf 'delay_seconds=60\n'
printf 'target=workbench-github-relay.service\n'
