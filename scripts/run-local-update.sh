#!/usr/bin/env bash
set -euo pipefail

config_root="${XDG_CONFIG_HOME:-$HOME/.config}/workbench"
read_config() {
  local env_value="$1"
  local file="$2"
  if [ -n "$env_value" ]; then
    printf '%s' "$env_value"
    return 0
  fi
  if [ ! -f "$file" ]; then
    return 1
  fi
  local value
  IFS= read -r value < "$file" || true
  printf '%s' "$value"
}

source_dir="$(read_config "${WORKBENCH_SOURCE_DIR:-}" "$config_root/local-update-source" || true)"
relay_url="$(read_config "${WORKBENCH_RELAY_URL:-}" "$config_root/local-update-relay" || true)"
public_url="$(read_config "${WORKBENCH_PUBLIC_REPO_URL:-}" "$config_root/local-update-public" || true)"

if [ -z "$source_dir" ] || [ -z "$relay_url" ] || [ -z "$public_url" ]; then
  echo "workbench local updater: configured source, relay, and public repository are required" >&2
  exit 2
fi

if [ ! -d "$source_dir/.git" ]; then
  echo "workbench local updater: source is not a git clone" >&2
  exit 1
fi

mkdir -p "$config_root"
lock_dir="$config_root/local-update.lock"
if ! mkdir "$lock_dir" 2>/dev/null; then
  # Another local update is already active. This is ordinary contention, not
  # an error that should wake the human.
  exit 0
fi
trap 'rmdir "$lock_dir" 2>/dev/null || true' EXIT

if [ -n "$(git -C "$source_dir" status --porcelain)" ]; then
  echo "workbench local updater: source clone has local changes; leaving it untouched" >&2
  exit 0
fi

branch="$(git -C "$source_dir" branch --show-current)"
if [ "$branch" != "main" ]; then
  echo "workbench local updater: source clone is not on main; leaving it untouched" >&2
  exit 0
fi

configured_origin="$(git -C "$source_dir" remote get-url origin 2>/dev/null || true)"
if [ "$configured_origin" != "$public_url" ]; then
  echo "workbench local updater: source origin no longer matches the locally approved repository" >&2
  exit 1
fi

update_ref="refs/workbench/approved-main"
git -C "$source_dir" fetch --quiet --no-tags "$public_url" "+refs/heads/main:$update_ref"
current="$(git -C "$source_dir" rev-parse HEAD)"
target="$(git -C "$source_dir" rev-parse "$update_ref")"
if [ "$current" = "$target" ]; then
  exit 0
fi

if ! git -C "$source_dir" merge-base --is-ancestor "$current" "$target"; then
  echo "workbench local updater: approved main is not a fast-forward from the installed source" >&2
  exit 1
fi

# The existing bootstrap owns the transactional maintenance contract: clean
# clone, fast-forward only, full tests/builds, then service restarts. Passing the
# pinned repository values keeps the timer from trusting worker-controlled task
# text or a changed Git remote.
WORKBENCH_SOURCE_DIR="$source_dir" \
WORKBENCH_PUBLIC_REPO_URL="$public_url" \
bash "$source_dir/scripts/bootstrap-private-relay.sh" "$relay_url"
