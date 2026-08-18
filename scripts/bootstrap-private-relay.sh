#!/usr/bin/env bash
set -euo pipefail

relay_url="${1:-}"
if [ -z "$relay_url" ]; then
  echo "Usage: bash scripts/bootstrap-private-relay.sh <private-git-repository-url>" >&2
  exit 2
fi

public_url="${WORKBENCH_PUBLIC_REPO_URL:-https://github.com/DaisyCloverSoftware/workbench.git}"
project_source_dir="${WORKBENCH_SOURCE_DIR:-$HOME/src/workbench}"
update_source_dir="$HOME/.local/share/workbench/update-source"
relay_dir="${WORKBENCH_RELAY_REPO_DIR:-$HOME/.local/share/workbench/relay-private}"
mkdir -p "$(dirname "$project_source_dir")" "$(dirname "$update_source_dir")" "$(dirname "$relay_dir")"

ensure_project_clone() {
  local url="$1"
  local dir="$2"

  if [ -d "$dir/.git" ]; then
    # This is the developer/project checkout. Never reset, clean, switch or
    # fast-forward it as part of maintenance; local work must survive updates.
    return 0
  fi
  if [ -e "$dir" ]; then
    echo "STOP: Workbench project source path exists but is not a git clone: $dir" >&2
    exit 1
  fi
  git clone --quiet "$url" "$dir"
}

refresh_update_clone() {
  local url="$1"
  local dir="$2"

  if [ -d "$dir/.git" ]; then
    git -C "$dir" remote get-url origin >/dev/null 2>&1 || git -C "$dir" remote add origin "$url"
    git -C "$dir" remote set-url origin "$url"
    git -C "$dir" fetch --quiet origin main
    git -C "$dir" reset --hard --quiet origin/main
    git -C "$dir" clean -fdx -q
    return 0
  fi
  if [ -e "$dir" ]; then
    backup="${dir}-invalid-$(date +%Y%m%d-%H%M%S)"
    mv "$dir" "$backup"
  fi
  git clone --quiet "$url" "$dir"
}

refresh_relay_clone() {
  local url="$1"
  local dir="$2"

  if [ -d "$dir/.git" ]; then
    if [ -n "$(git -C "$dir" status --porcelain)" ]; then
      echo "STOP: Private relay clone has unexpected local changes: $dir" >&2
      exit 1
    fi
    git -C "$dir" remote get-url origin >/dev/null 2>&1 || git -C "$dir" remote add origin "$url"
    git -C "$dir" remote set-url origin "$url"
    git -C "$dir" fetch --quiet origin main
    git -C "$dir" switch --quiet main
    git -C "$dir" merge --ff-only --quiet origin/main
    return 0
  fi
  if [ -e "$dir" ]; then
    echo "STOP: Private relay path exists but is not a git clone: $dir" >&2
    exit 1
  fi
  git clone --quiet "$url" "$dir"
}

echo "Checking Workbench project checkout without modifying it..."
ensure_project_clone "$public_url" "$project_source_dir"

echo "Refreshing disposable Workbench maintenance source..."
refresh_update_clone "$public_url" "$update_source_dir"

echo "Refreshing private relay transport..."
refresh_relay_clone "$relay_url" "$relay_dir"

# The desktop talks to workbench-runner directly over unattended SSH, while the
# private relay talks to the local MCP service. They must be upgraded together;
# otherwise the picker/dashboard can silently observe an older project protocol
# than ChatGPT's relay does.
echo "Installing current Workbench cluster runner..."
bash "$update_source_dir/scripts/install-runner.sh"

echo "Installing current Workbench MCP service..."
bash "$update_source_dir/scripts/install-cluster-mcp.sh" "$project_source_dir"

echo "Installing private bidirectional relay..."
WORKBENCH_RELAY_REPO_DIR="$relay_dir" \
WORKBENCH_RELAY_PRIVATE=1 \
bash "$update_source_dir/scripts/install-github-relay.sh"

echo
echo "WORKBENCH PRIVATE LOOP READY"
echo "  project source: $project_source_dir (left untouched by maintenance)"
echo "  update source: app-owned disposable checkout"
echo "  cluster runner: refreshed from the same maintenance source"
echo "  relay mode: private bidirectional safe hands + autonomous handoff"
echo "  secrets: remain local; relay messages must not contain raw credentials"
