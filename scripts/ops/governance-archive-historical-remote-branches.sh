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

archive_branch='archive/pre-governance-reset-20260821'
archive_ref="refs/remotes/origin/$archive_branch"

# Refresh all public branch refs, then prove canonical main is exactly the
# reviewed operation commit before creating or deleting any public ref.
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

main_tree="$(git -C "$root" rev-parse "$desired^{tree}")"
mapfile -t source_branches < <(
  git -C "$root" for-each-ref --format='%(refname:strip=3)' refs/remotes/origin/ \
    | LC_ALL=C sort \
    | grep -v -E '^(HEAD|main|archive/pre-governance-reset-20260821)$' || true
)

# If a previous partial execution already created the archive ref, reuse it only
# if it still has the canonical main tree and already preserves every current
# source branch tip. This makes retry safe after a later branch-delete failure.
archive_head=''
if git -C "$root" rev-parse --verify "$archive_ref" >/dev/null 2>&1; then
  archive_head="$(git -C "$root" rev-parse "$archive_ref")"
  if [ "$(git -C "$root" rev-parse "$archive_head^{tree}")" != "$main_tree" ]; then
    printf 'error=existing-archive-tree-mismatch\n' >&2
    exit 75
  fi
  for branch in "${source_branches[@]}"; do
    tip="$(git -C "$root" rev-parse "refs/remotes/origin/$branch")"
    if ! git -C "$root" merge-base --is-ancestor "$tip" "$archive_head"; then
      printf 'error=existing-archive-missing-source-tip\n' >&2
      exit 76
    fi
  done
else
  # Build a short chain of synthetic archive anchors. Every anchor has exactly
  # the canonical main tree; additional parents exist only to keep old public
  # branch commit graphs reachable under one explicit non-authoritative ref.
  archive_head="$desired"
  batch_size=32
  total="${#source_branches[@]}"
  tranche=0
  for ((start=0; start<total; start+=batch_size)); do
    end=$((start + batch_size))
    if [ "$end" -gt "$total" ]; then
      end="$total"
    fi
    parents=("$archive_head")
    for ((i=start; i<end; i++)); do
      parents+=("$(git -C "$root" rev-parse "refs/remotes/origin/${source_branches[$i]}")")
    done
    tranche=$((tranche + 1))
    args=(commit-tree "$main_tree")
    for parent in "${parents[@]}"; do
      args+=( -p "$parent" )
    done
    message="governance: archive historical Workbench branch tips tranche $tranche"
    archive_head="$(printf '%s\n\nTree intentionally identical to canonical main; parent links preserve public pre-reset branch history.\n' "$message" | git -C "$root" "${args[@]}")"
  done

  if [ "$(git -C "$root" rev-parse "$archive_head^{tree}")" != "$main_tree" ]; then
    printf 'error=archive-tree-mismatch\n' >&2
    exit 77
  fi
  git -C "$root" push --quiet origin "$archive_head:refs/heads/$archive_branch"
  git -C "$root" fetch --quiet origin "+refs/heads/$archive_branch:$archive_ref"
  if [ "$(git -C "$root" rev-parse "$archive_ref")" != "$archive_head" ]; then
    printf 'error=archive-push-verification-failed\n' >&2
    exit 78
  fi
fi

# Prove every branch tip is reachable from the archive before deleting any
# source ref. The archive tree itself must remain exactly canonical main.
if [ "$(git -C "$root" rev-parse "$archive_ref^{tree}")" != "$main_tree" ]; then
  printf 'error=archive-tree-verification-failed\n' >&2
  exit 79
fi
for branch in "${source_branches[@]}"; do
  tip="$(git -C "$root" rev-parse "refs/remotes/origin/$branch")"
  if ! git -C "$root" merge-base --is-ancestor "$tip" "$archive_ref"; then
    printf 'error=archive-missing-source-tip\n' >&2
    exit 80
  fi
done

# Source refs are now redundant pointers to history reachable through the
# archive. Remove them in bounded batches. If a server-side protection rejects
# a batch, the already-pushed archive keeps all histories safe for a retry.
batch_size=32
for ((start=0; start<${#source_branches[@]}; start+=batch_size)); do
  end=$((start + batch_size))
  if [ "$end" -gt "${#source_branches[@]}" ]; then
    end="${#source_branches[@]}"
  fi
  batch=("${source_branches[@]:start:end-start}")
  if [ "${#batch[@]}" -gt 0 ]; then
    git -C "$root" push --quiet origin --delete "${batch[@]}"
  fi
done

git -C "$root" fetch --quiet --prune origin '+refs/heads/*:refs/remotes/origin/*'
mapfile -t remaining < <(git -C "$root" for-each-ref --format='%(refname:strip=3)' refs/remotes/origin/ | LC_ALL=C sort | grep -v '^HEAD$' || true)
if [ "${#remaining[@]}" -ne 2 ] || [ "${remaining[0]}" != "$archive_branch" ] || [ "${remaining[1]}" != 'main' ]; then
  printf 'error=unexpected-remote-refs-after-archive-cleanup\n' >&2
  exit 81
fi
if [ "$(git -C "$root" rev-parse "$archive_ref^{tree}")" != "$(git -C "$root" rev-parse "$main_ref^{tree}")" ]; then
  printf 'error=archive-main-tree-diverged\n' >&2
  exit 82
fi
if [ -n "$(git -C "$root" status --porcelain=v1 --untracked-files=all)" ]; then
  printf 'error=checkout-dirty-after-archive-cleanup\n' >&2
  exit 83
fi

printf 'cleanup=ok\n'
printf 'archived_source_refs=%d\n' "${#source_branches[@]}"
printf 'archive_branch=%s\n' "$archive_branch"
printf 'archive_head=%s\n' "$(git -C "$root" rev-parse "$archive_ref")"
printf 'archive_tree=%s\n' "$(git -C "$root" rev-parse "$archive_ref^{tree}")"
printf 'main_head=%s\n' "$(git -C "$root" rev-parse "$main_ref")"
printf 'main_tree=%s\n' "$(git -C "$root" rev-parse "$main_ref^{tree}")"
printf 'remote_branch_count=2\n'
