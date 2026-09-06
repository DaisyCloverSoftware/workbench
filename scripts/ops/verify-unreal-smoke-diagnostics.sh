#!/usr/bin/env bash
# Verification only in the exact detached operations checkout; no deployment.
set -euo pipefail
[ "$#" -eq 0 ] || { echo 'This operation accepts no arguments' >&2; exit 2; }
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
if command -v go >/dev/null 2>&1; then
  go_bin="$(command -v go)"
else
  go_bin="$(find "$HOME/.local/share/workbench/toolchains" -type f -path '*/go/bin/go' -perm -u+x 2>/dev/null | sort -V | tail -n 1 || true)"
fi
[ -n "${go_bin:-}" ] && [ -x "$go_bin" ] || { echo 'Workbench Go toolchain is unavailable' >&2; exit 1; }
export GOTOOLCHAIN=local
printf 'source_commit=%s\n' "$(git rev-parse HEAD)"
"$go_bin" version
fmt_bin="$("$go_bin" env GOROOT)/bin/gofmt"
[ -z "$("$fmt_bin" -l internal/core/host_bridge_unreal*.go)" ] || { echo 'Unreal source is not gofmt clean' >&2; exit 1; }
"$go_bin" test -count=1 ./internal/core -run 'Test(Unreal|.*HostBridge)'
"$go_bin" test -race -count=20 ./internal/core -run 'TestUnrealSmoke(Evidence|Zen|Process)'
"$go_bin" test -count=1 ./internal/mcp ./cmd/workbench-relay ./internal/desktop
out="$(mktemp -d)"
trap 'rm -rf -- "$out"' EXIT
env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 "$go_bin" build -trimpath -o "$out/Workbench.exe" ./cmd/workbench
[ -z "$(git status --porcelain)" ] || { echo 'Verification changed the checkout' >&2; exit 1; }
printf 'focused_checks=passed\nwindows_cross_build=passed\nwindows_execution=not_performed\n'
# Full go test ./... remains mandatory in ordinary CI. Keep all operations
# origin/authority guards intact; do not disable them for governance fixtures.
