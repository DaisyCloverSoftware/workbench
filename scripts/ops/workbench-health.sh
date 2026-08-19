#!/usr/bin/env bash
set -euo pipefail

failed=0
state_dir="${WORKBENCH_STATE_DIR:-$HOME/.local/state/workbench}"
relay_dir="${WORKBENCH_RELAY_REPO_DIR:-$HOME/.local/share/workbench/relay-private}"
mcp_health="${WORKBENCH_MCP_HEALTH_URL:-http://127.0.0.1:8765/health}"

printf 'WORKBENCH_HEALTH\n'
printf 'host=%s\n' "$(hostname)"

check_binary() {
  local name="$1"
  local path="$HOME/.local/bin/$name"
  if [ -x "$path" ]; then
    printf 'binary_%s=ok\n' "$name"
  else
    printf 'binary_%s=missing\n' "$name"
    failed=1
  fi
}

check_service() {
  local key="$1"
  local unit="$2"
  local pid_file="$3"

  if command -v systemctl >/dev/null 2>&1 && systemctl --user show-environment >/dev/null 2>&1; then
    if systemctl --user is-active --quiet "$unit"; then
      printf '%s=active\n' "$key"
      return 0
    fi
  fi

  if [ -r "$pid_file" ]; then
    local pid
    pid="$(cat "$pid_file" 2>/dev/null || true)"
    case "$pid" in
      ''|*[!0-9]*) ;;
      *)
        if kill -0 "$pid" 2>/dev/null; then
          printf '%s=fallback-running\n' "$key"
          return 0
        fi
        ;;
    esac
  fi

  printf '%s=inactive\n' "$key"
  failed=1
}

check_binary workbench-runner
check_binary workbench-server
check_binary workbench-relay

check_service mcp_service workbench-mcp.service "$state_dir/workbench-mcp.pid"
check_service relay_service workbench-github-relay.service "$state_dir/workbench-github-relay.pid"

if command -v curl >/dev/null 2>&1; then
  if curl -fsS --max-time 3 "$mcp_health" >/dev/null 2>&1; then
    printf 'mcp_http=ok\n'
  else
    printf 'mcp_http=failed\n'
    failed=1
  fi
elif command -v wget >/dev/null 2>&1; then
  if wget -qO- --timeout=3 "$mcp_health" >/dev/null 2>&1; then
    printf 'mcp_http=ok\n'
  else
    printf 'mcp_http=failed\n'
    failed=1
  fi
else
  printf 'mcp_http=unchecked-no-client\n'
  failed=1
fi

if [ -d "$relay_dir/.git" ]; then
  if [ -z "$(git -C "$relay_dir" status --porcelain --untracked-files=no 2>/dev/null)" ]; then
    printf 'relay_checkout=clean\n'
  else
    printf 'relay_checkout=dirty\n'
    failed=1
  fi
else
  printf 'relay_checkout=missing\n'
  failed=1
fi

if [ "$failed" -eq 0 ]; then
  printf 'overall=ok\n'
  exit 0
fi

printf 'overall=degraded\n'
exit 1
