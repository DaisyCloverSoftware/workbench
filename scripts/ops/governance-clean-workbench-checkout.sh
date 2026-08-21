#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  printf 'error=target-required\n' >&2
  exit 64
fi

target="$1"
if [ ! -d "$target" ]; then
  printf 'error=target-missing\n' >&2
  exit 65
fi

target="$(cd "$target" && pwd -P)"
root="$(git -C "$target" rev-parse --show-toplevel 2>/dev/null || true)"
if [ -z "$root" ]; then
  printf 'error=not-git-checkout\n' >&2
  exit 66
fi
root="$(cd "$root" && pwd -P)"
if [ "$root" != "$target" ]; then
  printf 'error=target-not-repository-root\n' >&2
  exit 67
fi

branch="$(git -C "$root" branch --show-current 2>/dev/null || true)"
if [ "$branch" != "main" ]; then
  printf 'error=unexpected-branch\n' >&2
  exit 68
fi

origin="$(git -C "$root" remote get-url origin 2>/dev/null || true)"
case "$origin" in
  https://github.com/DaisyCloverSoftware/workbench|https://github.com/DaisyCloverSoftware/workbench.git|git@github.com:DaisyCloverSoftware/workbench|git@github.com:DaisyCloverSoftware/workbench.git|ssh://git@github.com/DaisyCloverSoftware/workbench|ssh://git@github.com/DaisyCloverSoftware/workbench.git)
    ;;
  *)
    printf 'error=unexpected-origin\n' >&2
    exit 69
    ;;
esac

tracked_dirty="$({ git -C "$root" diff --name-only; git -C "$root" diff --cached --name-only; } | LC_ALL=C sort -u)"
expected_tracked='internal/core/relay_state.go'
if [ "$tracked_dirty" != "$expected_tracked" ]; then
  printf 'error=unexpected-tracked-dirty-set\n' >&2
  exit 70
fi

untracked="$(git -C "$root" ls-files --others --exclude-standard | LC_ALL=C sort)"
expected_untracked="$(printf '%s\n' \
  'internal/core/relay_lock.go' \
  'internal/core/relay_lock_test.go' \
  'internal/core/relay_state_concurrency_test.go' | LC_ALL=C sort)"
if [ "$untracked" != "$expected_untracked" ]; then
  printf 'error=unexpected-untracked-set\n' >&2
  exit 71
fi

# The useful rationale/content of this abandoned experiment is preserved in
# docs/LOCAL_CHECKOUT_AUDIT_2026-08-21.md before this operation is authorised.
# Restore/remove only the exact audited paths; never clean the checkout broadly.
git -C "$root" restore --staged --worktree -- internal/core/relay_state.go
rm -- \
  "$root/internal/core/relay_lock.go" \
  "$root/internal/core/relay_lock_test.go" \
  "$root/internal/core/relay_state_concurrency_test.go"

remaining="$(git -C "$root" status --porcelain=v1 --untracked-files=all)"
if [ -n "$remaining" ]; then
  printf 'error=checkout-still-dirty\n' >&2
  exit 72
fi

printf 'cleanup=ok\n'
printf 'tracked_restored=1\n'
printf 'untracked_removed=3\n'
