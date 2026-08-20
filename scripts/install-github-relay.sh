#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
bin_dir="$HOME/.local/bin"
config_dir="$HOME/.config/workbench"
state_dir="$HOME/.local/state/workbench"
auth_file="$config_dir/mcp-loopback-auth-value"
mcp_url="${WORKBENCH_MCP_URL:-http://127.0.0.1:8765/mcp}"
relay_repo="${WORKBENCH_RELAY_REPO_DIR:-$repo_root}"
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

relay_repo="$(cd "$relay_repo" 2>/dev/null && pwd)" || {
  echo "Relay transport repository does not exist: $relay_repo" >&2
  exit 1
}
[ -d "$relay_repo/.git" ] || {
  echo "Relay transport must be a git clone: $relay_repo" >&2
  exit 1
}

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
  echo "Refusing report mode on a public relay transport." >&2
  exit 1
fi
case "$result_mode" in status|report) ;; *) echo "WORKBENCH_RELAY_RESULT_MODE must be status or report." >&2; exit 1 ;; esac

# The daemon needs read + write Git transport: Chat writes inbox/answers and the
# runner writes outbox status/results. It never creates or stores a GitHub token.
echo "Checking relay git transport..."
git -C "$relay_repo" fetch --quiet "$relay_remote" "$relay_branch"
if ! git -C "$relay_repo" push --dry-run "$relay_remote" "refs/remotes/$relay_remote/$relay_branch:refs/heads/$relay_branch" >/dev/null 2>&1; then
  relay_url="$(git -C "$relay_repo" remote get-url "$relay_remote")"
  case "$relay_url" in
    https://github.com/*)
      slug="${relay_url#https://github.com/}"
      slug="${slug%.git}"
      ssh_url="git@github.com:${slug}.git"
      if git ls-remote "$ssh_url" HEAD >/dev/null 2>&1; then
        relay_remote="workbench-relay-write"
        if git -C "$relay_repo" remote get-url "$relay_remote" >/dev/null 2>&1; then
          git -C "$relay_repo" remote set-url "$relay_remote" "$ssh_url"
        else
          git -C "$relay_repo" remote add "$relay_remote" "$ssh_url"
        fi
        git -C "$relay_repo" fetch --quiet "$relay_remote" "$relay_branch"
      fi
      ;;
  esac
fi

if ! git -C "$relay_repo" push --dry-run "$relay_remote" "refs/remotes/$relay_remote/$relay_branch:refs/heads/$relay_branch" >/dev/null 2>&1; then
  echo "Relay repository is readable but no non-interactive Git push credential was found." >&2
  echo "Configure an authenticated Git remote (SSH is recommended) or set WORKBENCH_RELAY_REMOTE to one." >&2
  exit 1
fi

# Report mode can contain source/task detail. For GitHub transports, fail closed
# unless the repository can be verified as private. Fetch already proved the
# repository exists, so an unauthenticated GitHub API 404 means it is private.
if [ "$result_mode" = report ]; then
  relay_url="$(git -C "$relay_repo" remote get-url "$relay_remote")"
  github_slug=""
  case "$relay_url" in
    https://github.com/*)
      github_slug="${relay_url#https://github.com/}"
      ;;
    git@github.com:*)
      github_slug="${relay_url#git@github.com:}"
      ;;
    ssh://git@github.com/*)
      github_slug="${relay_url#ssh://git@github.com/}"
      ;;
  esac
  github_slug="${github_slug%.git}"
  if [ -n "$github_slug" ]; then
    visibility=""
    if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
      visibility="$(gh repo view "$github_slug" --json visibility --jq .visibility 2>/dev/null || true)"
    fi
    if [ -z "$visibility" ] && command -v curl >/dev/null 2>&1; then
      code="$(curl -sS -o /dev/null -w '%{http_code}' "https://api.github.com/repos/$github_slug" || true)"
      case "$code" in
        200) visibility="PUBLIC" ;;
        404) visibility="PRIVATE" ;;
      esac
    fi
    if [ "$visibility" != PRIVATE ]; then
      echo "Refusing relay report mode: GitHub transport could not be verified as PRIVATE." >&2
      exit 1
    fi
  elif [ "${WORKBENCH_RELAY_ASSUME_PRIVATE:-0}" != 1 ]; then
    echo "Refusing report mode for an unverified non-GitHub transport." >&2
    echo "Set WORKBENCH_RELAY_ASSUME_PRIVATE=1 only after independently verifying repository privacy." >&2
    exit 1
  fi
fi

echo "Building Workbench Git relay with $($go_bin version)..."
cd "$repo_root"
"$go_bin" test ./...
"$go_bin" build -trimpath -o "$bin_dir/workbench-relay" ./cmd/workbench-relay
chmod 0755 "$bin_dir/workbench-relay"

relay_args=(
  --repo-dir "$relay_repo"
  --remote "$relay_remote"
  --branch "$relay_branch"
  --mcp-url "$mcp_url"
  --auth-file "$auth_file"
  --result-mode "$result_mode"
  --public-transport="$public_transport"
)

# Validate the newly built binary and relay configuration without consuming any
# live inbox/control work. Polling here could block the updater on an unrelated
# long operation and leave the old supervised process running the previous
# binary while the new binary had already published newer metadata.
"$bin_dir/workbench-relay" "${relay_args[@]}" --smoke-only

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
ExecStart=$bin_dir/workbench-relay --repo-dir=$relay_repo --remote=$relay_remote --branch=$relay_branch --interval=$relay_interval --mcp-url=$mcp_url --auth-file=$auth_file --result-mode=$result_mode --public-transport=$public_transport
Restart=always
RestartSec=3

[Install]
WantedBy=default.target
EOF
  systemctl --user daemon-reload
  # `enable --now` does not restart an already-active unit after its ExecStart
  # changes. Explicitly restart so rerunning the installer actually switches an
  # existing relay process to the newly installed binary.
  systemctl --user enable workbench-github-relay.service >/dev/null
  systemctl --user restart workbench-github-relay.service
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    if systemctl --user is-active --quiet workbench-github-relay.service; then
      break
    fi
    sleep 1
  done
  systemctl --user is-active --quiet workbench-github-relay.service || {
    echo "Workbench relay service did not become active after restart." >&2
    exit 1
  }
  service_mode="systemd --user"
else
  start_fallback
fi

echo
echo "WORKBENCH GIT RELAY READY"
echo "  transport repo: $relay_repo"
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
  echo "Private relay mode publishes reports and attention questions back to Chat."
fi
