#!/usr/bin/env bash
set -euo pipefail

before="$(git status --porcelain=v1 --untracked-files=all)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

printf '%s\n' '[1/4] portable core + private relay tests'
go test ./internal/core ./cmd/workbench-relay

printf '%s\n' '[2/4] Windows core test-binary compile'
GOOS=windows GOARCH=amd64 go test -c -o "$tmp/core.test.exe" ./internal/core

printf '%s\n' '[3/4] Windows private-relay test-binary compile'
GOOS=windows GOARCH=amd64 go test -c -o "$tmp/relay.test.exe" ./cmd/workbench-relay

printf '%s\n' '[4/4] production Windows app cross-build'
GOOS=windows GOARCH=amd64 go build -ldflags='-s -w -H=windowsgui' -o "$tmp/Workbench.exe" ./cmd/workbench

after="$(git status --porcelain=v1 --untracked-files=all)"
if [[ "$before" != "$after" ]]; then
  printf '%s\n' 'preflight changed the disposable source worktree' >&2
  diff -u <(printf '%s\n' "$before") <(printf '%s\n' "$after") || true
  exit 1
fi

printf 'wardrobe_inventory_preflight=pass\n'
printf 'windows_core_test_compile=pass\n'
printf 'windows_relay_test_compile=pass\n'
printf 'windows_app_cross_build=pass\n'
printf 'source_worktree_unchanged=true\n'
