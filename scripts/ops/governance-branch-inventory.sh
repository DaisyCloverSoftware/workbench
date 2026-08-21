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
if [ "$origin_ok" -ne 1 ] && [ "${WORKBENCH_GOVERNANCE_TEST_ALLOW_LOCAL_ORIGIN:-0}" = "1" ] && [ "${WORKBENCH_OPERATION_SCRIPT:-0}" != "1" ]; then
  origin_ok=1
fi
if [ "$origin_ok" -ne 1 ]; then
  printf 'error=unexpected-origin\n' >&2
  exit 68
fi

# Refresh only remote-tracking branch refs. No local branch, worktree, tag or
# working-tree content is changed by this inventory operation.
git -C "$root" fetch --quiet --prune origin '+refs/heads/*:refs/remotes/origin/*'
main_ref='refs/remotes/origin/main'
if ! git -C "$root" rev-parse --verify "$main_ref" >/dev/null 2>&1; then
  printf 'error=origin-main-missing\n' >&2
  exit 69
fi

mapfile -t branches < <(git -C "$root" for-each-ref --format='%(refname:strip=3)' refs/remotes/origin/ | LC_ALL=C sort)
index=0
merged=0
diverged=0
for branch in "${branches[@]}"; do
  if [ -z "$branch" ] || [ "$branch" = 'HEAD' ]; then
    continue
  fi
  index=$((index + 1))
  ref="refs/remotes/origin/$branch"
  if [ "$branch" = 'main' ]; then
    status='current'
    unique=0
    behind=0
  elif git -C "$root" merge-base --is-ancestor "$ref" "$main_ref"; then
    status='merged'
    unique=0
    behind="$(git -C "$root" rev-list --count "$ref..$main_ref")"
    merged=$((merged + 1))
  else
    status='diverged'
    unique="$(git -C "$root" rev-list --count "$main_ref..$ref")"
    behind="$(git -C "$root" rev-list --count "$ref..$main_ref")"
    diverged=$((diverged + 1))
  fi
  printf 'branch_%d_name=%s\n' "$index" "$branch"
  printf 'branch_%d_status=%s\n' "$index" "$status"
  printf 'branch_%d_unique_commits=%s\n' "$index" "$unique"
  printf 'branch_%d_behind_commits=%s\n' "$index" "$behind"
done

printf 'branch_total=%d\n' "$index"
printf 'branch_merged=%d\n' "$merged"
printf 'branch_diverged=%d\n' "$diverged"
