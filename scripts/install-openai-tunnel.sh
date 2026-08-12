#!/usr/bin/env bash
set -euo pipefail

# Installs OpenAI's tunnel-client next to Workbench and connects the loopback
# Workbench MCP server to a user-provisioned Secure MCP Tunnel. Secrets are read
# only in this terminal and stored in 0600 local files; they are never printed.

bin_dir="$HOME/.local/bin"
config_dir="$HOME/.config/workbench"
state_dir="$HOME/.local/state/workbench"
key_file="$config_dir/openai-tunnel-runtime-key"
mcp_auth_file="$config_dir/mcp-loopback-auth-value"
health_file="$state_dir/openai-tunnel-health.url"
mcp_url="${WORKBENCH_MCP_URL:-http://127.0.0.1:8765/mcp}"
mkdir -p "$bin_dir" "$config_dir" "$state_dir"
chmod 0700 "$config_dir" "$state_dir"

download() {
  local url="$1" dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fL --retry 3 --connect-timeout 15 "$url" -o "$dest"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$dest" "$url"
  else
    echo "curl or wget is required." >&2
    exit 1
  fi
}

latest_stable_version() {
  if [ -n "${TUNNEL_CLIENT_VERSION:-}" ]; then
    printf '%s\n' "$TUNNEL_CLIENT_VERSION"
    return
  fi
  local json version
  if command -v curl >/dev/null 2>&1; then
    json="$(curl -fsSL --retry 2 --connect-timeout 10 https://api.github.com/repos/openai/tunnel-client/releases/latest 2>/dev/null || true)"
  elif command -v wget >/dev/null 2>&1; then
    json="$(wget -qO- https://api.github.com/repos/openai/tunnel-client/releases/latest 2>/dev/null || true)"
  fi
  version="$(printf '%s' "${json:-}" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  case "$version" in
    v[0-9]*.[0-9]*.[0-9]*) printf '%s\n' "$version" ;;
    *) printf '%s\n' "v0.0.11" ;;
  esac
}

install_tunnel_client() {
  local version machine platform tmp archive sums expected actual candidate base
  version="$(latest_stable_version)"
  machine="$(uname -m)"
  case "$machine" in
    x86_64|amd64) platform="linux-amd64" ;;
    aarch64|arm64) platform="linux-arm64" ;;
    *) echo "Unsupported tunnel-client platform: linux/$machine" >&2; exit 1 ;;
  esac

  if [ -x "$bin_dir/tunnel-client" ]; then
    echo "Using existing $($bin_dir/tunnel-client --version 2>/dev/null || echo tunnel-client)"
    return
  fi

  echo "Installing OpenAI tunnel-client $version ($platform)..."
  tmp="$(mktemp -d)"
  trap 'rm -rf "${tmp:-}"' EXIT
  archive="$tmp/$platform.zip"
  sums="$tmp/SHA256SUMS.txt"
  base="https://github.com/openai/tunnel-client/releases/download/$version"
  download "$base/$platform.zip" "$archive"
  download "$base/SHA256SUMS.txt" "$sums"

  expected="$(awk -v f="$platform.zip" '$2 == f || $2 == "*" f {print $1; exit}' "$sums")"
  if command -v sha256sum >/dev/null 2>&1 && [ -n "$expected" ]; then
    actual="$(sha256sum "$archive" | awk '{print $1}')"
    if [ "$actual" != "$expected" ]; then
      echo "tunnel-client checksum verification failed." >&2
      exit 1
    fi
  fi

  command -v unzip >/dev/null 2>&1 || { echo "unzip is required to install tunnel-client." >&2; exit 1; }
  mkdir -p "$tmp/unpacked"
  unzip -q "$archive" -d "$tmp/unpacked"
  candidate="$(find "$tmp/unpacked" -type f \( -name 'tunnel-client' -o -name 'client' \) | head -n 1)"
  [ -n "$candidate" ] || { echo "Could not find tunnel-client in release archive." >&2; exit 1; }
  install -m 0755 "$candidate" "$bin_dir/tunnel-client"
  echo "Installed $($bin_dir/tunnel-client --version 2>/dev/null || echo tunnel-client)"
}

install_tunnel_client

if [ ! -s "$mcp_auth_file" ]; then
  echo "Workbench MCP auth file is missing: $mcp_auth_file" >&2
  echo "Run scripts/install-cluster-mcp.sh first." >&2
  exit 1
fi
chmod 0600 "$mcp_auth_file"

# The tunnel ID is not secret, but it must be provisioned for the user's OpenAI
# organization/workspace. Prefer env for automated reinstalls; otherwise prompt.
tunnel_id="${CONTROL_PLANE_TUNNEL_ID:-${WORKBENCH_TUNNEL_ID:-}}"
if [ -z "$tunnel_id" ]; then
  echo
  echo "Workbench needs the tunnel ID you created in OpenAI Platform → Tunnels."
  read -r -p "Tunnel ID (tunnel_...): " tunnel_id
fi
if ! printf '%s' "$tunnel_id" | grep -Eq '^tunnel_[0-9a-f]{32}$'; then
  echo "Invalid tunnel ID. Expected tunnel_ followed by 32 lowercase hexadecimal characters." >&2
  exit 1
fi

if [ ! -s "$key_file" ]; then
  echo
  echo "Enter the OpenAI tunnel runtime API key locally."
  echo "It will be stored only in $key_file with mode 0600 and will not be printed."
  read -r -s -p "Runtime API key: " runtime_key
  echo
  [ -n "$runtime_key" ] || { echo "Runtime API key is empty." >&2; exit 1; }
  umask 077
  printf '%s' "$runtime_key" > "$key_file"
  unset runtime_key
fi
chmod 0600 "$key_file"

# Prove the local Workbench endpoint exists before involving the control plane.
if command -v curl >/dev/null 2>&1; then
  curl -fsS "${mcp_url%/mcp}/health" >/dev/null || {
    echo "Workbench MCP is not healthy at ${mcp_url%/mcp}/health." >&2
    echo "Run scripts/install-cluster-mcp.sh first." >&2
    exit 1
  }
fi

echo
echo "Running tunnel-client preflight..."
"$bin_dir/tunnel-client" doctor \
  --control-plane.tunnel-id="$tunnel_id" \
  --control-plane.api-key="file:$key_file" \
  --mcp.server-url="$mcp_url" \
  --mcp.extra-headers "Authorization: file:$mcp_auth_file" \
  --explain

rm -f "$health_file"

start_fallback() {
  local pid_file="$state_dir/openai-tunnel.pid"
  if [ -f "$pid_file" ]; then
    old_pid="$(cat "$pid_file" 2>/dev/null || true)"
    if [ -n "${old_pid:-}" ] && kill -0 "$old_pid" 2>/dev/null; then
      kill "$old_pid" 2>/dev/null || true
      sleep 1
    fi
  fi
  nohup "$bin_dir/tunnel-client" run \
    --control-plane.tunnel-id="$tunnel_id" \
    --control-plane.api-key="file:$key_file" \
    --mcp.server-url="$mcp_url" \
    --mcp.extra-headers "Authorization: file:$mcp_auth_file" \
    --health.listen-addr=127.0.0.1:0 \
    --health.url-file="$health_file" \
    --log.level=info \
    >"$state_dir/openai-tunnel.log" 2>&1 < /dev/null &
  echo $! > "$pid_file"
  echo "Started tunnel-client with fallback supervisor (PID $(cat "$pid_file"))."
}

service_mode="fallback"
if command -v systemctl >/dev/null 2>&1 && systemctl --user show-environment >/dev/null 2>&1; then
  unit_dir="$HOME/.config/systemd/user"
  mkdir -p "$unit_dir"
  cat > "$unit_dir/workbench-openai-tunnel.service" <<EOF
[Unit]
Description=Workbench OpenAI Secure MCP Tunnel
After=network-online.target workbench-mcp.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=$bin_dir/tunnel-client run --control-plane.tunnel-id=$tunnel_id --control-plane.api-key=file:$key_file --mcp.server-url=$mcp_url --mcp.extra-headers=Authorization:\ file:$mcp_auth_file --health.listen-addr=127.0.0.1:0 --health.url-file=$health_file --log.level=info
Restart=always
RestartSec=3

[Install]
WantedBy=default.target
EOF
  systemctl --user daemon-reload
  systemctl --user enable --now workbench-openai-tunnel.service
  service_mode="systemd --user"
else
  start_fallback
fi

for _ in $(seq 1 60); do
  [ -s "$health_file" ] && break
  sleep 0.25
done
if [ ! -s "$health_file" ]; then
  echo "tunnel-client did not publish its health URL." >&2
  [ "$service_mode" = "systemd --user" ] && systemctl --user status --no-pager workbench-openai-tunnel.service || true
  exit 1
fi

"$bin_dir/tunnel-client" health --url-file "$health_file"

echo
echo "WORKBENCH SECURE MCP TUNNEL READY"
echo "  tunnel: $tunnel_id"
echo "  local MCP: $mcp_url"
echo "  supervisor: $service_mode"
echo "  inbound ports opened: none"
echo "  Workbench MCP bearer: injected from local 0600 file (not displayed)"
echo "  OpenAI runtime key: stored locally (not displayed)"
echo
echo "In ChatGPT Plugins developer mode, create/select a Tunnel connection using the tunnel ID above."
