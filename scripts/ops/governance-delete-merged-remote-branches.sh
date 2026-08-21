#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 2 ]; then
  printf 'error=target-and-confirmation-required\n' >&2
  exit 64
fi

target="$1"
confirmation="$2"
if [ "$confirmation" != "confirmed-no-open-prs" ]; then
  printf 'error=no-open-pr-confirmation-required\n' >&2
  exit 65
fi
if [ ! -d "$target" ]; then
  printf 'error=target-missing\n' >&2
  exit 66
fi

target="$(cd "$target" && pwd -P)"
root="$(git -C "$target" rev-parse --show-toplevel 2>/dev/null || true)"
if [ -z "$root" ] || [ "$(cd "$root" && pwd -P)" != "$target" ]; then
  printf 'error=invalid-target-root\n' >&2
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

if [ "$(git -C "$root" branch --show-current 2>/dev/null || true)" != "main" ]; then
  printf 'error=unexpected-branch\n' >&2
  exit 69
fi
if [ -n "$(git -C "$root" status --porcelain=v1 --untracked-files=all)" ]; then
  printf 'error=checkout-dirty\n' >&2
  exit 70
fi
if [ "$(git -C "$root" worktree list --porcelain | grep -c '^worktree ')" -ne 1 ]; then
  printf 'error=secondary-worktrees-present\n' >&2
  exit 71
fi

desired="${WORKBENCH_OPERATION_COMMIT:-}"
case "$desired" in
  ''|*[!0-9a-f]*)
    printf 'error=operation-commit-required\n' >&2
    exit 72
    ;;
esac
if [ "${#desired}" -ne 40 ]; then
  printf 'error=operation-commit-required\n' >&2
  exit 72
fi

# Refresh all remote-tracking heads and prove canonical main is the exact reviewed
# operation commit before deleting a single ref.
git -C "$root" fetch --quiet --prune origin '+refs/heads/*:refs/remotes/origin/*'
main_ref='refs/remotes/origin/main'
if [ "$(git -C "$root" rev-parse "$main_ref")" != "$desired" ]; then
  printf 'error=origin-main-does-not-match-operation-commit\n' >&2
  exit 73
fi
if ! git -C "$root" merge-base --is-ancestor HEAD "$desired"; then
  printf 'error=local-main-not-ancestor-of-operation-commit\n' >&2
  exit 74
fi
if [ "$(git -C "$root" rev-parse HEAD)" != "$desired" ]; then
  git -C "$root" merge --ff-only "$desired" >/dev/null
fi

mapfile -t remote_branches < <(git -C "$root" for-each-ref --format='%(refname:strip=3)' refs/remotes/origin/ | LC_ALL=C sort)
declare -a merged_branches diverged_branches
for branch in "${remote_branches[@]}"; do
  if [ -z "$branch" ] || [ "$branch" = 'HEAD' ] || [ "$branch" = 'main' ]; then
    continue
  fi
  ref="refs/remotes/origin/$branch"
  if git -C "$root" merge-base --is-ancestor "$ref" "$main_ref"; then
    merged_branches+=("$branch")
  else
    diverged_branches+=("$branch")
  fi
done

if [ "${#merged_branches[@]}" -eq 0 ]; then
  printf 'cleanup=ok\n'
  printf 'deleted=0\n'
  printf 'diverged_retained=%d\n' "${#diverged_branches[@]}"
  exit 0
fi

# Delete only refs whose complete tip history is already reachable from main.
# Use bounded batches so one enormous push cannot hide which operation failed.
batch_size=40
for ((start=0; start<${#merged_branches[@]}; start+=batch_size)); do
  end=$((start + batch_size))
  if [ "$end" -gt "${#merged_branches[@]}" ]; then
    end="${#merged_branches[@]}"
  fi
  batch=("${merged_branches[@]:start:end-start}")
  git -C "$root" push --quiet origin --delete "${batch[@]}"
done

git -C "$root" fetch --quiet --prune origin '+refs/heads/*:refs/remotes/origin/*'
remaining_merged=0
remaining_diverged=0
for ref in $(git -C "$root" for-each-ref --format='%(refname)' refs/remotes/origin/); do
  branch="${ref#refs/remotes/origin/}"
  if [ "$branch" = 'HEAD' ] || [ "$branch" = 'main' ]; then
    continue
  fi
  if git -C "$root" merge-base --is-ancestor "$ref" refs/remotes/origin/main; then
    remaining_merged=$((remaining_merged + 1))
  else
    remaining_diverged=$((remaining_diverged + 1))
  fi
done
if [ "$remaining_merged" -ne 0 ]; then
  printf 'error=merged-remote-refs-remain\n' >&2
  exit 75
fi
if [ -n "$(git -C "$root" status --porcelain=v1 --untracked-files=all)" ]; then
  printf 'error=checkout-dirty-after-remote-cleanup\n' >&2
  exit 76
fi

printf 'cleanup=ok\n'
printf 'deleted=%d\n' "${#merged_branches[@]}"
printf 'diverged_retained=%d\n' "$remaining_diverged"
printf 'current_head=%s\n' "$(git -C "$root" rev-parse HEAD)"
