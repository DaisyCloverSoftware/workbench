#!/usr/bin/env bash
set -euo pipefail

state_file="${WORKBENCH_RELAY_PROGRESS_FILE:-$HOME/.local/state/workbench/workbench-github-relay-progress.json}"
unit="${WORKBENCH_RELAY_WATCHDOG_UNIT:-workbench-github-relay.service}"

if [ "${1:-}" = "--self-test" ]; then
  [ "$#" -eq 1 ]
  [ -n "$state_file" ]
  [ "$unit" = "workbench-github-relay.service" ]
  printf 'WORKBENCH_RELAY_WATCHDOG_SELF_TEST_OK\n'
  exit 0
fi

[ "$#" -eq 0 ] || {
  echo "Usage: workbench-relay-watchdog.sh [--self-test]" >&2
  exit 2
}

# Restart=always already handles a dead process. This watchdog is only for the
# live-but-wedged case, so an intentionally stopped/inactive unit is left alone.
if ! systemctl --user is-active --quiet "$unit"; then
  exit 0
fi
[ -r "$state_file" ] || exit 0

line="$(head -n 1 "$state_file" 2>/dev/null || true)"
pid="$(printf '%s\n' "$line" | sed -n 's/.*"pid":\([0-9][0-9]*\).*/\1/p')"
deadline="$(printf '%s\n' "$line" | sed -n 's/.*"deadline_unix":\([0-9][0-9]*\).*/\1/p')"
case "$pid:$deadline" in
  *[!0-9:]*|:*|*:)
    exit 0
    ;;
esac

main_pid="$(systemctl --user show "$unit" -p MainPID --value 2>/dev/null || true)"
[ "$main_pid" = "$pid" ] || exit 0

now="$(date +%s)"
case "$now" in *[!0-9]*|'') exit 0 ;; esac
if [ "$now" -le "$deadline" ]; then
  exit 0
fi

echo "Workbench relay watchdog: forward-progress lease expired for PID $pid; restarting $unit" >&2
exec systemctl --user restart "$unit"
