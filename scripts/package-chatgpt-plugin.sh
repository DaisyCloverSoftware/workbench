#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
app_id="${1:-}"
dest="${2:-$HOME/.codex/plugins/workbench}"
marketplace="${WORKBENCH_PLUGIN_MARKETPLACE:-$HOME/.agents/plugins/marketplace.json}"

if ! printf '%s' "$app_id" | grep -Eq '^plugin_asdk_app_[A-Za-z0-9_-]+$'; then
  echo "Usage: $0 plugin_asdk_app_... [destination]" >&2
  echo "The app id must be the technical id of the registered Workbench MCP connection." >&2
  exit 2
fi
command -v python3 >/dev/null 2>&1 || { echo "python3 is required." >&2; exit 1; }

parent="$(dirname "$dest")"
mkdir -p "$parent"
tmp="$(mktemp -d "$parent/.workbench-plugin.XXXXXX")"
cleanup() { rm -rf "$tmp"; }
trap cleanup EXIT

mkdir -p "$tmp/.codex-plugin" "$tmp/skills"
cp -R "$repo_root/skills/workbench" "$tmp/skills/workbench"

python3 - "$repo_root/.codex-plugin/plugin.json" "$tmp/.codex-plugin/plugin.json" "$tmp/.app.json" "$app_id" <<'PY'
import json, pathlib, sys
source, manifest_out, app_out, app_id = sys.argv[1:]
with open(source, encoding="utf-8") as f:
    manifest = json.load(f)
manifest["apps"] = "./.app.json"
pathlib.Path(manifest_out).write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
pathlib.Path(app_out).write_text(json.dumps({"apps": {"workbench": {"id": app_id}}}, indent=2) + "\n", encoding="utf-8")
PY

cat > "$tmp/README.md" <<'EOF'
# Workbench personal ChatGPT plugin

This directory is generated from DaisyCloverSoftware/workbench. Its `.app.json` contains the workspace-specific technical id of the registered Workbench MCP connection. Regenerate it from the source repository rather than editing it by hand.
EOF

if [ -e "$dest" ]; then
  if [ ! -f "$dest/.codex-plugin/plugin.json" ] || ! grep -q '"name"[[:space:]]*:[[:space:]]*"workbench"' "$dest/.codex-plugin/plugin.json"; then
    echo "Refusing to replace non-Workbench destination: $dest" >&2
    exit 1
  fi
  old="$parent/.workbench-plugin.previous.$$"
  mv "$dest" "$old"
  if ! mv "$tmp" "$dest"; then
    mv "$old" "$dest"
    exit 1
  fi
  rm -rf "$old"
else
  mv "$tmp" "$dest"
fi
trap - EXIT

# Register the conventional personal install automatically when the destination
# is the standard ~/.codex/plugins/workbench path. Preserve unrelated entries.
if [ "$dest" = "$HOME/.codex/plugins/workbench" ]; then
  mkdir -p "$(dirname "$marketplace")"
  python3 - "$marketplace" <<'PY'
import json, pathlib, sys
path = pathlib.Path(sys.argv[1])
if path.exists():
    data = json.loads(path.read_text(encoding="utf-8"))
else:
    data = {"name": "personal", "interface": {"displayName": "Personal"}, "plugins": []}
plugins = data.setdefault("plugins", [])
entry = {
    "name": "workbench",
    "source": {"source": "local", "path": "./.codex/plugins/workbench"},
    "policy": {"installation": "AVAILABLE", "authentication": "ON_INSTALL"},
    "category": "Developer Tools",
}
for i, current in enumerate(plugins):
    if isinstance(current, dict) and current.get("name") == "workbench":
        plugins[i] = entry
        break
else:
    plugins.append(entry)
path.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
PY
fi

echo "WORKBENCH CHATGPT PLUGIN PACKAGE READY"
echo "  plugin: $dest"
echo "  app binding: workspace-specific id written locally"
if [ "$dest" = "$HOME/.codex/plugins/workbench" ]; then
  echo "  marketplace: $marketplace"
fi
