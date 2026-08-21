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

origin="$(git -C "$root" remote get-url origin 2>/dev/null || true)"
origin_ok=0
case "$origin" in
  https://github.com/DaisyCloverSoftware/workbench|https://github.com/DaisyCloverSoftware/workbench.git|git@github.com:DaisyCloverSoftware/workbench|git@github.com:DaisyCloverSoftware/workbench.git|ssh://git@github.com/DaisyCloverSoftware/workbench|ssh://git@github.com:DaisyCloverSoftware/workbench.git)
    origin_ok=1
    ;;
esac
if [ "$origin_ok" -ne 1 ]; then
  if [ "${WORKBENCH_GOVERNANCE_TEST_ALLOW_LOCAL_ORIGIN:-0}" = "1" ] && [ "${WORKBENCH_OPERATION_SCRIPT:-0}" != "1" ]; then
    origin_ok=1
  fi
fi
if [ "$origin_ok" -ne 1 ]; then
  printf 'error=unexpected-origin\n' >&2
  exit 68
fi

branch="$(git -C "$root" branch --show-current 2>/dev/null || true)"
if [ "$branch" != "main" ]; then
  printf 'error=unexpected-branch\n' >&2
  exit 69
fi
if [ -n "$(git -C "$root" status --porcelain=v1 --untracked-files=all)" ]; then
  printf 'error=checkout-dirty\n' >&2
  exit 70
fi

expected_head='2defa97101447c04e8350bfae88414cbacafe237'
if [ "${WORKBENCH_GOVERNANCE_TEST_ALLOW_LOCAL_ORIGIN:-0}" = "1" ] && [ "${WORKBENCH_OPERATION_SCRIPT:-0}" != "1" ]; then
  expected_head="${WORKBENCH_GOVERNANCE_TEST_EXPECTED_HEAD:-$expected_head}"
fi
current="$(git -C "$root" rev-parse HEAD)"
if [ "$current" != "$expected_head" ]; then
  printf 'error=unexpected-current-head\n' >&2
  exit 71
fi

subject="$(git -C "$root" log -1 --format=%s HEAD)"
if [ "$subject" != 'sec: pin GitHub Actions to full commit SHAs (SEC-008)' ]; then
  printf 'error=unexpected-local-commit-subject\n' >&2
  exit 72
fi
changed="$(git -C "$root" diff-tree --no-commit-id --name-only -r HEAD | LC_ALL=C sort)"
expected_changed="$(printf '%s\n' \
  '.github/workflows/build.yml' \
  '.github/workflows/release.yml' \
  '.github/workflows/runner.yml' | LC_ALL=C sort)"
if [ "$changed" != "$expected_changed" ]; then
  printf 'error=unexpected-local-commit-files\n' >&2
  exit 73
fi

desired="${WORKBENCH_OPERATION_COMMIT:-}"
case "$desired" in
  ''|*[!0-9a-f]* )
    printf 'error=operation-commit-required\n' >&2
    exit 74
    ;;
esac
if [ "${#desired}" -ne 40 ]; then
  printf 'error=operation-commit-required\n' >&2
  exit 74
fi

audit_branch='audit/pre-governance-reset-20260821'
if existing="$(git -C "$root" rev-parse --verify "refs/heads/$audit_branch" 2>/dev/null)"; then
  if [ "$existing" != "$current" ]; then
    printf 'error=audit-branch-conflict\n' >&2
    exit 75
  fi
else
  git -C "$root" branch "$audit_branch" "$current"
fi

# Fetch only the canonical main tip. The operation itself is already bound to an
# exact current public Workbench commit by RunOperationsScript.
git -C "$root" fetch --quiet origin main
fetched="$(git -C "$root" rev-parse FETCH_HEAD)"
if [ "$fetched" != "$desired" ]; then
  printf 'error=fetched-main-does-not-match-operation-commit\n' >&2
  exit 76
fi

# The one divergent local commit is preserved by the audit branch above and its
# security intent is already present in current public workflows. Realign main
# exactly to the reviewed operation commit; do not merge the obsolete v0.5 line.
git -C "$root" reset --hard "$desired" >/dev/null

if [ "$(git -C "$root" branch --show-current)" != "main" ]; then
  printf 'error=branch-changed\n' >&2
  exit 77
fi
if [ "$(git -C "$root" rev-parse HEAD)" != "$desired" ]; then
  printf 'error=head-not-realigned\n' >&2
  exit 78
fi
if [ "$(git -C "$root" rev-parse "refs/heads/$audit_branch")" != "$current" ]; then
  printf 'error=audit-branch-not-preserved\n' >&2
  exit 79
fi
if [ -n "$(git -C "$root" status --porcelain=v1 --untracked-files=all)" ]; then
  printf 'error=checkout-dirty-after-realignment\n' >&2
  exit 80
fi

printf 'realign=ok\n'
printf 'previous_head=%s\n' "$current"
printf 'current_head=%s\n' "$desired"
printf 'audit_branch=%s\n' "$audit_branch"
