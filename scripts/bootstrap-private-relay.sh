#!/usr/bin/env bash
set -euo pipefail

relay_url="${1:-}"
if [ -z "$relay_url" ]; then
  echo "Usage: bash scripts/bootstrap-private-relay.sh <private-git-repository-url>" >&2
  exit 2
fi

public_url="${WORKBENCH_PUBLIC_REPO_URL:-https://github.com/DaisyCloverSoftware/workbench.git}"
source_dir="${WORKBENCH_SOURCE_DIR:-$HOME/src/workbench}"
relay_dir="${WORKBENCH_RELAY_REPO_DIR:-$HOME/.local/share/workbench/relay-private}"
mkdir -p "$(dirname "$source_dir")" "$(dirname "$relay_dir")"

refresh_clone() {
  local url="$1"
  local dir="$2"
  local label="$3"

  if [ -d "$dir/.git" ]; then
    if [ -n "$(git -C "$dir" status --porcelain)" ]; then
      echo "STOP: $label clone has local changes: $dir" >&2
      echo "Commit, stash, or move those changes before rerunning this bootstrap." >&2
      exit 1
    fi

    git -C "$dir" remote get-url origin >/dev/null 2>&1 || git -C "$dir" remote add origin "$url"
    git -C "$dir" remote set-url origin "$url"
    git -C "$dir" fetch --quiet origin main

    if git -C "$dir" merge-base HEAD origin/main >/dev/null 2>&1; then
      git -C "$dir" switch --quiet main
      git -C "$dir" pull --ff-only --quiet origin main
      return 0
    fi

    local backup="${dir}-pre-bootstrap-$(date +%Y%m%d-%H%M%S)"
    echo "$label clone has unrelated history; preserving it at $backup"
    mv "$dir" "$backup"
  elif [ -e "$dir" ]; then
    echo "STOP: $label path exists but is not a git clone: $dir" >&2
    exit 1
  fi

  git clone --quiet "$url" "$dir"
}

echo "Refreshing Workbench source..."
refresh_clone "$public_url" "$source_dir" "Workbench source"

echo "Refreshing private relay transport..."
refresh_clone "$relay_url" "$relay_dir" "Private relay"

echo "Installing current Workbench MCP service..."
bash "$source_dir/scripts/install-cluster-mcp.sh" "$source_dir"

echo "Installing private bidirectional relay..."
WORKBENCH_RELAY_REPO_DIR="$relay_dir" \
WORKBENCH_RELAY_PRIVATE=1 \
bash "$source_dir/scripts/install-github-relay.sh"

echo
echo "WORKBENCH PRIVATE LOOP READY"
echo "  source: $source_dir"
echo "  relay transport: $relay_dir"
echo "  relay mode: private bidirectional reports + attention answers"
echo "  secrets: remain local; relay messages must not contain raw credentials"
