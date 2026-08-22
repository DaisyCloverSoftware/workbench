#!/usr/bin/env bash
set -euo pipefail

relay_dir="${WORKBENCH_RELAY_REPO_DIR:-$HOME/.local/share/workbench/relay-private}"
if [ ! -d "$relay_dir/.git" ]; then
  printf 'relay_checkout=missing\n'
  exit 65
fi

origin="$(git -C "$relay_dir" remote get-url origin 2>/dev/null || true)"
case "$origin" in
  git@github.com:DaisyCloverSoftware/workbench-relay-private.git|git@github.com:DaisyCloverSoftware/workbench-relay-private|https://github.com/DaisyCloverSoftware/workbench-relay-private.git|https://github.com/DaisyCloverSoftware/workbench-relay-private|ssh://git@github.com/DaisyCloverSoftware/workbench-relay-private.git|ssh://git@github.com/DaisyCloverSoftware/workbench-relay-private)
    ;;
  *)
    printf 'relay_checkout=unexpected-origin\n'
    exit 66
    ;;
esac

status_text="$(git -C "$relay_dir" status --porcelain=v1 --untracked-files=all)"
if [ -z "$status_text" ]; then
  printf 'relay_checkout=clean\n'
  printf 'dirty_count=0\n'
  exit 0
fi

printf 'relay_checkout=dirty\n'
count=0
tracked=0
untracked=0
while IFS= read -r line || [ -n "$line" ]; do
  count=$((count + 1))
  case "$line" in
    '?? '*) untracked=$((untracked + 1)) ;;
    *) tracked=$((tracked + 1)) ;;
  esac
  printf 'dirty_entry_%d=%s\n' "$count" "$line"
done <<< "$status_text"
printf 'dirty_count=%d\n' "$count"
printf 'tracked_dirty_count=%d\n' "$tracked"
printf 'untracked_count=%d\n' "$untracked"
