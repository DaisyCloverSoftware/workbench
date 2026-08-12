#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
bin_dir="$HOME/.local/bin"
config_dir="$HOME/.config/workbench"
state_dir="$HOME/.local/state/workbench"
auth_file="$config_dir/mcp-loopback-auth-value"
mcp_url="${WORKBENCH_MCP_URL:-http://127.0.0.1:8765/mcp}"
relay_remote="${WORKBENCH_RELAY_REMOTE:-origin}"
relay_branch="${WORKBENCH_RELAY_BRANCH:-main}"
relay_interval="${WORKBENCH_RELAY_INTERVAL:-10s}"
mkdir -p "$bin_dir" "$config_dir" "$state_dir"
chmod 0700 "$config_dir" "$state_dir"

find_go() {
  if command -v go >/dev/null 2>&1; then
    command -v go
    return 0
  fi
  local found
  found="$(find "$HOME/.local/share/workbench/toolchains" -type f -path '*/go/bin/go' -perm -u+x 2>/dev/null | sort -V | tail -n 1 || true)"
  [ -n "$found" ] || return 1
  printf '%s\n' "$found"
}

if ! go_bin="$(find_go)"; then
  echo "Bootstrapping the Workbench toolchain first..."
  bash "$repo_root/scripts/install-runner.sh"
  go_bin="$(find_go)" || { echo "Could not locate Go after bootstrap." >&2; exit 1; }
fi

[ -s "$auth_file" ] || {
  echo "Workbench MCP auth file is missing. Run scripts/install-cluster-mcp.sh first." >&2
  exit 1
}
chmod 0600 "$auth_file"

# The relay uses ordinary git fetch as its transport. This works unauthenticated
# for a public relay repository and automatically uses the user's existing git
# credentials when the relay repository is private.
echo "Checking relay git transport..."
git -C "$repo_root" fetch --quiet "$relay_remote" "$relay_branch"

echo "Building Workbench GitHub relay with $($go_bin version)..."
cd "$repo_root"
"$go_bin" test ./...
"$go_bin" build -trimpath -o "$bin_dir/workbench-relay" ./cmd/workbench-relay
chmod 0755 "$bin_dir/workbench-relay"

# Smoke the daemon path once before installing supervision. An empty inbox is a
# healthy result; the command still proves git fetch and local MCP auth wiring.
"$bin_dir/workbench-relay" \
  --repo-dir "$repo_root" \
  --remote "$relay_remote" \
  --branch "$relay_branch" \
  --mcp-url "$mcp_url" \
  --auth-file "$auth_file" \
  --once

start_fallback() {
  local pid_file="$state_dir/workbench-github-relay.pid"
  if [ -f "$pid_file" ]; then
    old_pid="$(cat "$pid_file" 2>/dev/null || true)"
    if [ -n "${old_pid:-}" ] && kill -0 "$old_pid" 2>/dev/null; then
      kill "$old_pid" 2>/dev/null || true
      sleep 1
    fi
  fi
  nohup "$bin_dir/workbench-relay" \
    --repo-dir "$repo_root" \
    --remote "$relay_remote" \
    --branch "$relay_branch" \
    --interval "$relay_interval" \
    --mcp-url "$mcp_url" \
    --auth-file "$auth_file" \
    >"$state_dir/workbench-github-relay.log" 2>&1 < /dev/null &
  echo $! > "$pid_file"
  echo "Started relay with nohup (PID $(cat "$pid_file"))."
}

service_mode="fallback"
if command -v systemctl >/dev/null 2>&1 && systemctl --user show-environment >/dev/null 2>&1; then
  unit_dir="$HOME/.config/systemd/user"
  mkdir -p "$unit_dir"
  cat > "$unit_dir/workbench-github-relay.service" <<EOF
[Unit]
Description=Workbench GitHub task relay for personal ChatGPT plans
After=network-online.target workbench-mcp.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=$bin_dir/workbench-relay --repo-dir=$repo_root --remote=$relay_remote --branch=$relay_branch --interval=$relay_interval --mcp-url=$mcp_url --auth-file=$auth_file
Restart=always
RestartSec=3
Environment=WORKBENCH_RUNNER_ROOT=$HOME/src

[Install]
WantedBy=default.target
EOF
  systemctl --user daemon-reload
  systemctl --user enable --now workbench-github-relay.service
  service_mode="systemd --user"
else
  start_fallback
fi

echo
echo "WORKBENCH GITHUB RELAY READY"
echo "  transport repo: $repo_root"
echo "  inbox: relay/inbox/*.json on $relay_remote/$relay_branch"
echo "  poll interval: $relay_interval"
echo "  local handoff: authenticated Workbench MCP"
echo "  runner root: $HOME/src"
echo "  supervisor: $service_mode"
echo
echo "For private work, point the relay at a private clone with existing git credentials."
echo "The public Workbench relay is intended only for non-sensitive dogfood tasks."
