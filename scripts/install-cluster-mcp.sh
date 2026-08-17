#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
project="${1:-$repo_root}"
port="${WORKBENCH_MCP_PORT:-8765}"
bin_dir="$HOME/.local/bin"
config_dir="$HOME/.config/workbench"
state_dir="$HOME/.local/state/workbench"
token_file="$config_dir/mcp-loopback-auth-value"
mkdir -p "$bin_dir" "$config_dir" "$state_dir"
chmod 0700 "$config_dir" "$state_dir"

find_go() {
  if command -v go >/dev/null 2>&1; then
    command -v go
    return 0
  fi
  local found
  found="$(find "$HOME/.local/share/workbench/toolchains" -type f -path '*/go/bin/go' -perm -u+x 2>/dev/null | sort -V | tail -n 1 || true)"
  if [ -n "$found" ]; then
    printf '%s\n' "$found"
    return 0
  fi
  return 1
}

if ! go_bin="$(find_go)"; then
  echo "Bootstrapping the Workbench toolchain first..."
  bash "$repo_root/scripts/install-runner.sh"
  go_bin="$(find_go)" || { echo "Could not locate Go after bootstrap." >&2; exit 1; }
fi

echo "Building headless Workbench MCP server with $($go_bin version)..."
cd "$repo_root"
"$go_bin" test ./...
"$go_bin" build -trimpath -o "$bin_dir/workbench-server" ./cmd/workbench-server
chmod 0755 "$bin_dir/workbench-server"

start_fallback() {
  local pid_file="$state_dir/workbench-mcp.pid"
  if [ -f "$pid_file" ]; then
    old_pid="$(cat "$pid_file" 2>/dev/null || true)"
    if [ -n "${old_pid:-}" ] && kill -0 "$old_pid" 2>/dev/null; then
      kill "$old_pid" 2>/dev/null || true
      sleep 1
    fi
  fi
  nohup "$bin_dir/workbench-server" \
    --port "$port" \
    --project "$project" \
    --token-file "$token_file" \
    >"$state_dir/workbench-mcp.log" 2>&1 < /dev/null &
  echo $! > "$pid_file"
  echo "Started Workbench MCP with nohup (PID $(cat "$pid_file"))."
}

service_ok=false
if command -v systemctl >/dev/null 2>&1 && systemctl --user show-environment >/dev/null 2>&1; then
  unit_dir="$HOME/.config/systemd/user"
  mkdir -p "$unit_dir"
  cat > "$unit_dir/workbench-mcp.service" <<EOF
[Unit]
Description=Workbench private MCP control plane
After=network-online.target

[Service]
Type=simple
ExecStart=$bin_dir/workbench-server --port=$port --project=$project --token-file=$token_file
Restart=always
RestartSec=3
Environment=PATH=$PATH

[Install]
WantedBy=default.target
EOF
  systemctl --user daemon-reload
  systemctl --user enable workbench-mcp.service >/dev/null
  systemctl --user restart workbench-mcp.service
  service_ok=true
  echo "Installed/restarted systemd user service: workbench-mcp.service"
else
  start_fallback
fi

health="http://127.0.0.1:$port/health"
mcp="http://127.0.0.1:$port/mcp"
health_ready=false
for _ in $(seq 1 120); do
  if [ -s "$token_file" ]; then
    if command -v curl >/dev/null 2>&1 && curl -fsS "$health" >/dev/null 2>&1; then
      health_ready=true
      break
    elif command -v wget >/dev/null 2>&1 && wget -qO- "$health" >/dev/null 2>&1; then
      health_ready=true
      break
    fi
  fi
  sleep 0.25
done

[ -s "$token_file" ] || { echo "Workbench MCP auth file was not created." >&2; exit 1; }
chmod 0600 "$token_file"
[ "$health_ready" = true ] || { echo "Workbench MCP did not become healthy within 30 seconds." >&2; exit 1; }

# Smoke-test initialize without putting the bearer value on the process command
# line. curl reads a temporary 0600 header file; Python is the no-curl fallback.
init_body='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
if command -v curl >/dev/null 2>&1; then
  header_file="$(mktemp)"
  trap 'rm -f "${header_file:-}"' EXIT
  chmod 0600 "$header_file"
  printf 'Authorization: %s\n' "$(cat "$token_file")" > "$header_file"
  init_resp="$(curl -fsS -H @"$header_file" -H 'Content-Type: application/json' -d "$init_body" "$mcp")"
elif command -v python3 >/dev/null 2>&1; then
  init_resp="$(WORKBENCH_MCP_URL="$mcp" WORKBENCH_MCP_TOKEN_FILE="$token_file" python3 - <<'PY'
import json, os, urllib.request
url = os.environ['WORKBENCH_MCP_URL']
with open(os.environ['WORKBENCH_MCP_TOKEN_FILE'], encoding='utf-8') as f:
    auth = f.read().strip()
body = json.dumps({'jsonrpc':'2.0','id':1,'method':'initialize','params':{}}).encode()
req = urllib.request.Request(url, data=body, headers={'Content-Type':'application/json','Authorization':auth})
with urllib.request.urlopen(req, timeout=5) as r:
    print(r.read().decode())
PY
)"
else
  echo "curl or python3 is required for the MCP smoke test." >&2
  exit 1
fi

printf '%s' "$init_resp" | grep -q 'Workbench' || { echo "MCP initialize smoke test failed." >&2; exit 1; }

echo
echo "WORKBENCH MCP READY"
echo "  endpoint: $mcp"
echo "  workspace: $project"
echo "  exposure: loopback only"
echo "  local auth: bearer value stored locally (not displayed)"
echo "  auth file: $token_file"
if [ "$service_ok" = true ]; then
  echo "  service: systemd --user workbench-mcp.service"
else
  echo "  service: nohup fallback (log: $state_dir/workbench-mcp.log)"
fi
echo
echo "Next: connect this authenticated private endpoint through OpenAI Secure MCP Tunnel."
