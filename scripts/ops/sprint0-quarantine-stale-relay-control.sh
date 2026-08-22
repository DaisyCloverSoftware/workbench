#!/usr/bin/env bash
set -euo pipefail

relay_dir="${WORKBENCH_RELAY_REPO_DIR:-$HOME/.local/share/workbench/relay-private}"
expected_rel='relay/control/override_build_list_projects_20260822_0310.json'
expected_outbox='relay/control-outbox/override_build_list_projects_20260822_0310.json'

if [ ! -d "$relay_dir/.git" ]; then
  printf 'result=relay-checkout-missing\n' >&2
  exit 65
fi
origin="$(git -C "$relay_dir" remote get-url origin 2>/dev/null || true)"
case "$origin" in
  git@github.com:DaisyCloverSoftware/workbench-relay-private.git|git@github.com:DaisyCloverSoftware/workbench-relay-private|https://github.com/DaisyCloverSoftware/workbench-relay-private.git|https://github.com/DaisyCloverSoftware/workbench-relay-private|ssh://git@github.com/DaisyCloverSoftware/workbench-relay-private.git|ssh://git@github.com/DaisyCloverSoftware/workbench-relay-private)
    ;;
  *)
    printf 'result=unexpected-origin\n' >&2
    exit 66
    ;;
esac

git -C "$relay_dir" fetch --quiet origin main
status_text="$(git -C "$relay_dir" status --porcelain=v1 --untracked-files=all)"
if [ "$status_text" != "?? $expected_rel" ]; then
  printf 'result=dirty-set-changed\n' >&2
  exit 67
fi

local_blob="$(git -C "$relay_dir" hash-object "$expected_rel")"
remote_blob="$(git -C "$relay_dir" rev-parse "origin/main:$expected_rel" 2>/dev/null || true)"
if [ -z "$remote_blob" ] || [ "$local_blob" != "$remote_blob" ]; then
  printf 'result=local-copy-not-canonical\n' >&2
  exit 68
fi
if ! git -C "$relay_dir" cat-file -e "origin/main:$expected_outbox" 2>/dev/null; then
  printf 'result=canonical-outbox-missing\n' >&2
  exit 69
fi

quarantine_dir="$HOME/.local/state/workbench/quarantine/relay-control"
mkdir -p "$quarantine_dir"
chmod 0700 "$quarantine_dir"
quarantine_file="$quarantine_dir/override_build_list_projects_20260822_0310.json"
cp "$relay_dir/$expected_rel" "$quarantine_file"
chmod 0600 "$quarantine_file"
rm -- "$relay_dir/$expected_rel"

if [ -n "$(git -C "$relay_dir" status --porcelain=v1 --untracked-files=all)" ]; then
  printf 'result=relay-checkout-still-dirty\n' >&2
  exit 70
fi
printf 'result=quarantined-redundant-canonical-copy\n'
printf 'relay_checkout=clean\n'
