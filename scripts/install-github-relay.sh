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
relay_private="${WORKBENCH_RELAY_PRIVATE:-0}"
result_mode="${WORKBENCH_RELAY_RESULT_MODE:-}"
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

case "$relay_private" in
  1|true|yes)
    public_transport=false
    [ -n "$result_mode" ] || result_mode=report
    ;;
  0|false|no|'')
    public_transport=true
    [ -n "$result_mode" ] || result_mode=status
    ;;
  *) echo "WORKBENCH_RELAY_PRIVATE must be 0/1 (or false/true)." >&2; exit 1 ;;
esac

if [ "$public_transport" = true ] && [ "$result_mode" = report ]; then
  echo "Refusing report mode on a public relay transport. Set WORKBENCH_RELAY_PRIVATE=1 only for a genuinely private relay repository." >&2
  exit 1
fi
case "$result_mode" in status|report) ;; *) echo "WORKBENCH_RELAY_RESULT_MODE must be status or report." >&2; exit 1 ;; esac

# The daemon needs read + write Git transport: Chat writes inbox/answers and the
# cluster writes outbox status/results. Prefer the configured remote. If a
# github.com HTTPS remote is readable but has no non-interactive push credential,
# automatically use an SSH sibling remote when the operator already has SSH Git
# access. No token is created or stored by Workbench.
echo "Checking relay git transport..."
git -C "$repo_root" fetch --quiet "$relay_remote" "$relay_branch"
if ! git -C "$repo_root" push --dry-run "$relay_remote" "refs/remotes/$relay_remote/$relay_branch:refs/heads/$relay_branch" >/dev/null 2>&1; then
  relay_url="$(git -C "$repo_root" remote get-url "$relay_remote")"
  case "$relay_url" in
    https://github.com/*)
      slug="${relay_url#https://github.com/}"
      slug="${slug%.git}"
      ssh_url="git@github.com:${slug}.git"
      if git ls-remote "$ssh_url" HEAD >/dev/null 2>&1; then
        relay_remote="workbench-relay-write"
        if git -C "$repo_root" remote get-url "$relay_remote" >/dev/null 2>&1; then
          git -C "$repo_root" remote set-url "$relay_remote" "$ssh_url"
        else
          git -C "$repo_root" remote add "$relay_remote" "$ssh_url"
        fi
        git -C "$repo_root" fetch --quiet "$relay_remote" "$relay_branch"
      fi
      ;;
  esac
fi

if ! git -C "$repo_root" push --dry-run "$relay_remote" "refs/remotes/$relay_remote/$relay_branch:refs/heads/$relay_branch" >/dev/null 2>&1; then
  echo "Relay repository is readable but no non-interactive Git push credential was found." >&2
  echo "Configure an authenticated Git remote (SSH is recommended) or set WORKBENCH_RELAY_REMOTE to one." >&2
  exit 1
fi

echo "Building Workbench Git relay with $($go_bin version)..."
cd "$repo_root"
"$go_bin" test ./...
"$go_bin" build -trimpath -o "$bin_dir/workbench-relay" ./cmd/workbench-relay
chmod 0755 "$bin_dir/workbench-relay"

relay_args=(
  --repo-dir "$repo_root"
  --remote "$relay_remote"
  --branch "$relay_branch"
  --mcp-url "$mcp_url"
  --auth-file "$auth_file"
  --result-mode "$result_mode"
  --public-transport="$public_transport"
)

# Smoke the full daemon path once before installing supervision. An empty inbox
# is healthy; any existing relay state may also publish an idempotent outbox.
"$bin_dir/workbench-relay" "${relay_args[@]}" --once

start_fallback() {
  local pid_file="$state_dir/workbench-github-relay.pid"
  if [ -f "$pid_file" ]; then
    old_pid="$(cat "$pid_file" 2>/dev/null || true)"
    if [ -n "${old_pid:-}" ] && kill -0 "$old_pid" 2>/dev/null; then
      kill "$old_pid" 2>/dev/null || true
      sleep 1
    fi
  fi
  nohup "$bin_dir/workbench-relay" "${relay_args[@]}" --interval "$relay_interval" \
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
Description=Workbench bidirectional Git task relay
After=network-online.target workbench-mcp.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=$bin_dir/workbench-relay --repo-dir=$repo_root --remote=$relay_remote --branch=$relay_branch --interval=$relay_interval --mcp-url=$mcp_url --auth-file=$auth_file --result-mode=$result_mode --public-transport=$public_transport
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
echo "WORKBENCH GIT RELAY READY"
echo "  inbox: relay/inbox/*.json on $relay_remote/$relay_branch"
echo "  answers: relay/answers/*.json"
echo "  outbox: relay/outbox/*.json"
echo "  poll interval: $relay_interval"
echo "  local handoff: authenticated Workbench MCP"
echo "  result mode: $result_mode"
echo "  public-safe transport: $public_transport"
echo "  supervisor: $service_mode"
echo
if [ "$public_transport" = true ]; then
  echo "Public relay mode publishes task status only. Use it for harmless dogfood, never private task intent."
else
  echo "Private relay mode can publish task reports and attention questions back to Chat."
fi
