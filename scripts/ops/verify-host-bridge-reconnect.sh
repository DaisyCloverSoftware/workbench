#!/usr/bin/env bash
# Verification only. Never deploy, restart, or select an autonomous worker.
set -euo pipefail
if [ "$#" -ne 0 ]; then echo 'No arguments accepted' >&2; exit 2; fi
root="$(git rev-parse --show-toplevel)"
cd "$root"
out="$(mktemp -d)"
trap 'rm -rf -- "$out"' EXIT
export GOMAXPROCS=2 GOTOOLCHAIN=local
printf 'SOURCE_HEAD=%s\n' "$(git rev-parse HEAD)"
git diff --check
go test -p 2 -count=1 -run 'Test(HostBridge|OwnedWindowsStartupRunsOutboundHostBridge|.*SSH)' ./internal/core ./internal/desktop
go test -p 2 -count=1 ./cmd/workbench-relay ./internal/mcp
go test -p 2 -race -count=20 -run '^TestHostBridgeTargetLoop' ./internal/core
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o "$out/Workbench.exe" ./cmd/workbench
printf 'WINDOWS_CROSS_BUILD_SHA256='
sha256sum "$out/Workbench.exe" | cut -d ' ' -f 1
printf 'FOCUSED_VERIFICATION=passed\nFULL_SUITE=requires_standard_CI_environment\nWINDOWS_RUNTIME=not_executed\nCONNECTIVITY_RESTORED=not_assessed\n'
