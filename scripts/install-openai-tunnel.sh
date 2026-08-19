#!/usr/bin/env bash
set -euo pipefail

# Connects Workbench's loopback MCP server to an OpenAI Secure MCP Tunnel.
# The runtime key and MCP bearer remain local 0600 files and are referenced by
# tunnel-client rather than placed on argv or committed into a profile.

bin_dir="$HOME/.local/bin"
config_dir="$HOME/.config/workbench"
state_dir="$HOME/.local/state/workbench"
key_file="$config_dir/openai-tunnel-runtime-key"
mcp_auth_file="$config_dir/mcp-loopback-auth-value"
health_file="$state_dir/openai-tunnel-health.url"
mcp_url="${WORKBENCH_MCP_URL:-http://127.0.0.1:8765/mcp}"
profile="${WORKBENCH_TUNNEL_PROFILE:-workbench}"
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

fetch_latest_tunnel_client_tag() {
  local json tag
  if [ -n "${TUNNEL_CLIENT_VERSION:-}" ]; then
    tag="$TUNNEL_CLIENT_VERSION"
  else
    if command -v curl >/dev/null 2>&1; then
      json="$(curl -fsSL --retry 3 --connect-timeout 15 https://api.github.com/repos/openai/tunnel-client/releases/latest)"
    elif command -v wget >/dev/null 2>&1; then
      json="$(wget -qO- https://api.github.com/repos/openai/tunnel-client/releases/latest)"
    else
      echo "curl or wget is required." >&2
      exit 1
    fi
    tag="$(printf '%s\n' "$json" | sed -n 's/^[[:space:]]*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  fi
  if ! printf '%s' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "Could not resolve a stable tunnel-client release tag; got: ${tag:-empty}." >&2
    exit 1
  fi
  printf '%s\n' "$tag"
}

current_client_usable() {
  local init_help doctor_help
  [ -x "$bin_dir/tunnel-client" ] || return 1
  init_help="$("$bin_dir/tunnel-client" init --help 2>&1)" || return 1
  doctor_help="$("$bin_dir/tunnel-client" doctor --help 2>&1)" || return 1
  printf '%s\n' "$init_help" | grep -q -- '--sample' || return 1
  printf '%s\n' "$init_help" | grep -q -- '--profile' || return 1
  printf '%s\n' "$doctor_help" | grep -q -- '--profile' || return 1
  return 0
}

install_tunnel_client() {
  local machine platform tag asset tmp archive sums expected actual candidate base
  machine="$(uname -m)"
  case "$machine" in
    x86_64|amd64) platform="linux-amd64" ;;
    aarch64|arm64) platform="linux-arm64" ;;
    *) echo "Unsupported tunnel-client platform: linux/$machine" >&2; exit 1 ;;
  esac

  if [ "${WORKBENCH_TUNNEL_CLIENT_REFRESH:-0}" != "1" ] && current_client_usable; then
    echo "Using existing $($bin_dir/tunnel-client --version 2>/dev/null || echo tunnel-client)"
    return
  fi
  if [ -x "$bin_dir/tunnel-client" ]; then
    echo "Refreshing tunnel-client because the installed binary is missing the current profile/doctor command surface."
  fi

  tag="$(fetch_latest_tunnel_client_tag)"
  asset="tunnel-client-$tag-$platform.zip"
  base="https://persistent.oaistatic.com/tunnel-client/$tag"
  echo "Installing OpenAI tunnel-client $tag ($platform)..."
  tmp="$(mktemp -d)"
  trap 'rm -rf "${tmp:-}"' EXIT
  archive="$tmp/$asset"
  sums="$tmp/SHA256SUMS.txt"
  download "$base/$asset" "$archive"
  download "$base/SHA256SUMS.txt" "$sums"

  expected="$(awk -v f="$asset" '$2 == f || $2 == "*" f {print $1; exit}' "$sums")"
  [ -n "$expected" ] || { echo "Could not find $asset in OpenAI's SHA256SUMS.txt." >&2; exit 1; }
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$archive" | awk '{print $1}')"
  elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$archive" | awk '{print $1}')"
  else
    echo "sha256sum or shasum is required to verify tunnel-client." >&2
    exit 1
  fi
  [ "$actual" = "$expected" ] || { echo "tunnel-client checksum verification failed." >&2; exit 1; }

  command -v unzip >/dev/null 2>&1 || { echo "unzip is required to install tunnel-client." >&2; exit 1; }
  mkdir -p "$tmp/unpacked"
  unzip -q "$archive" -d "$tmp/unpacked"
  candidate="$(find "$tmp/unpacked" -type f -name 'tunnel-client' | head -n 1)"
  [ -n "$candidate" ] || { echo "Could not find tunnel-client in release archive." >&2; exit 1; }
  install -m 0755 "$candidate" "$bin_dir/tunnel-client"
  echo "Installed $($bin_dir/tunnel-client --version 2>/dev/null || echo tunnel-client)"
  rm -rf "$tmp"
  trap - EXIT
}

tunnel_id="${CONTROL_PLANE_TUNNEL_ID:-${WORKBENCH_TUNNEL_ID:-}}"
if [ -z "$tunnel_id" ]; then
  echo
  echo "Workbench needs the tunnel ID created in OpenAI Platform tunnel settings."
  read -r -p "Tunnel ID (tunnel_...): " tunnel_id
fi
if ! printf '%s' "$tunnel_id" | grep -Eq '^tunnel_[0-9a-f]{32}$'; then
  echo "Invalid tunnel ID. Expected tunnel_ followed by 32 lowercase hexadecimal characters." >&2
  exit 1
fi

install_tunnel_client

if [ ! -s "$mcp_auth_file" ]; then
  echo "Workbench MCP auth file is missing: $mcp_auth_file" >&2
  echo "Run scripts/install-cluster-mcp.sh first." >&2
  exit 1
fi
chmod 0600 "$mcp_auth_file"

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

if command -v curl >/dev/null 2>&1; then
  curl -fsS "${mcp_url%/mcp}/health" >/dev/null || {
    echo "Workbench MCP is not healthy at ${mcp_url%/mcp}/health." >&2
    echo "Run scripts/install-cluster-mcp.sh first." >&2
    exit 1
  }
fi

# Current tunnel-client uses named profiles. The profile stores only a file
# reference to the runtime key; the Workbench MCP bearer is supplied at runtime
# through MCP_EXTRA_HEADERS and never written into the profile.
echo
echo "Preparing tunnel-client profile '$profile'..."
"$bin_dir/tunnel-client" init \
  --sample sample_mcp_remote_no_auth \
  --profile "$profile" \
  --tunnel-id "$tunnel_id" \
  --mcp-server-url "$mcp_url" \
  --control-plane-api-key-ref "file:$key_file" \
  --health-listen-addr 127.0.0.1:0 \
  --force

echo "Running tunnel-client preflight..."
MCP_EXTRA_HEADERS="Authorization: file:$mcp_auth_file" \
  "$bin_dir/tunnel-client" doctor --profile "$profile" --explain

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
  MCP_EXTRA_HEADERS="Authorization: file:$mcp_auth_file" nohup "$bin_dir/tunnel-client" run \
    --profile "$profile" \
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
  env_file="$config_dir/openai-tunnel.env"
  mkdir -p "$unit_dir"
  umask 077
  printf 'MCP_EXTRA_HEADERS="Authorization: file:%s"\n' "$mcp_auth_file" > "$env_file"
  chmod 0600 "$env_file"
  cat > "$unit_dir/workbench-openai-tunnel.service" <<EOF
[Unit]
Description=Workbench OpenAI Secure MCP Tunnel
After=network-online.target workbench-mcp.service
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=$env_file
ExecStart=$bin_dir/tunnel-client run --profile=$profile --health.listen-addr=127.0.0.1:0 --health.url-file=$health_file --log.level=info
Restart=always
RestartSec=3

[Install]
WantedBy=default.target
EOF
  systemctl --user daemon-reload
  systemctl --user enable workbench-openai-tunnel.service >/dev/null
  systemctl --user restart workbench-openai-tunnel.service
  service_mode="systemd --user"
else
  start_fallback
fi

for _ in $(seq 1 80); do
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
echo "  profile: $profile"
echo "  local MCP: $mcp_url"
echo "  supervisor: $service_mode"
echo "  inbound ports opened: none"
echo "  Workbench MCP bearer: injected from local 0600 file (not displayed)"
echo "  OpenAI runtime key: stored locally and referenced by profile (not displayed)"
echo
echo "Next: in ChatGPT Plugins developer mode create one Workbench app using Tunnel and this tunnel ID."
