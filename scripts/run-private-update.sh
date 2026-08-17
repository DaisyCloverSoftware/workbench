#!/usr/bin/env bash
set -uo pipefail

source_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
relay_url="${1:-}"
status_dir="$HOME/.local/state/workbench"
status_file="$status_dir/private-update-status.json"
mkdir -p "$status_dir"
chmod 0700 "$status_dir"

write_status() {
  local state="$1"
  local tmp="$status_file.tmp.$$"
  local now
  now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  printf '{\n  "state": "%s",\n  "updated_at": "%s"\n}\n' "$state" "$now" > "$tmp"
  chmod 0600 "$tmp"
  mv -f "$tmp" "$status_file"
}

if [ -z "$relay_url" ]; then
  write_status failed
  exit 2
fi

write_status running
if /bin/bash "$source_dir/scripts/bootstrap-private-relay.sh" "$relay_url"; then
  write_status succeeded
  exit 0
fi

rc=$?
write_status failed
exit "$rc"
