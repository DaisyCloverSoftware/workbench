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
if [ -z "$root" ] || [ "$(cd "$root" && pwd -P)" != "$target" ]; then
  printf 'error=invalid-target-root\n' >&2
  exit 66
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
  exit 67
fi

# Refresh public branch refs only. This is an audit operation: no local branch,
# worktree or remote branch is changed.
git -C "$root" fetch --quiet --prune origin '+refs/heads/*:refs/remotes/origin/*'
main_ref='refs/remotes/origin/main'
if ! git -C "$root" rev-parse --verify "$main_ref" >/dev/null 2>&1; then
  printf 'error=origin-main-missing\n' >&2
  exit 68
fi

# GitHub exposes immutable-ish PR head refs even after the source branch is later
# deleted. Build a SHA -> PR-number map so branch review provenance can be
# separated from an unreviewed unique ref without needing GitHub credentials here.
declare -A prs_by_sha=()
while IFS=$'\t' read -r sha ref; do
  [ -n "$sha" ] || continue
  case "$ref" in
    refs/pull/*/head)
      pr="${ref#refs/pull/}"
      pr="${pr%/head}"
      if [ -n "${prs_by_sha[$sha]:-}" ]; then
        prs_by_sha[$sha]="${prs_by_sha[$sha]},$pr"
      else
        prs_by_sha[$sha]="$pr"
      fi
      ;;
  esac
done < <(git -C "$root" ls-remote origin 'refs/pull/*/head')

index=0
patch_equivalent=0
novel=0
reviewed_equivalent=0
unreviewed_equivalent=0
while IFS= read -r branch; do
  [ -n "$branch" ] || continue
  [ "$branch" = 'HEAD' ] && continue
  [ "$branch" = 'main' ] && continue
  index=$((index + 1))
  ref="refs/remotes/origin/$branch"
  tip="$(git -C "$root" rev-parse "$ref")"
  plus=0
  minus=0
  while IFS= read -r cherry_line || [ -n "$cherry_line" ]; do
    case "$cherry_line" in
      '+ '*) plus=$((plus + 1)) ;;
      '- '*) minus=$((minus + 1)) ;;
    esac
  done < <(git -C "$root" cherry "$main_ref" "$ref")
  prs="${prs_by_sha[$tip]:-}"
  if [ "$plus" -eq 0 ]; then
    status='patch-equivalent'
    patch_equivalent=$((patch_equivalent + 1))
    if [ -n "$prs" ]; then
      provenance='pr-head'
      reviewed_equivalent=$((reviewed_equivalent + 1))
    else
      provenance='no-pr-head-match'
      unreviewed_equivalent=$((unreviewed_equivalent + 1))
    fi
  else
    status='novel-patches'
    provenance='protected'
    novel=$((novel + 1))
  fi
  printf 'branch_%d_name=%s\n' "$index" "$branch"
  printf 'branch_%d_tip=%s\n' "$index" "$tip"
  printf 'branch_%d_patch_status=%s\n' "$index" "$status"
  printf 'branch_%d_cherry_plus=%d\n' "$index" "$plus"
  printf 'branch_%d_cherry_minus=%d\n' "$index" "$minus"
  printf 'branch_%d_provenance=%s\n' "$index" "$provenance"
  if [ -n "$prs" ]; then
    printf 'branch_%d_pr_numbers=%s\n' "$index" "$prs"
  fi
done < <(git -C "$root" for-each-ref --format='%(refname:strip=3)' refs/remotes/origin/ | LC_ALL=C sort)

printf 'branch_total=%d\n' "$index"
printf 'patch_equivalent=%d\n' "$patch_equivalent"
printf 'novel_patches=%d\n' "$novel"
printf 'reviewed_patch_equivalent=%d\n' "$reviewed_equivalent"
printf 'unreviewed_patch_equivalent=%d\n' "$unreviewed_equivalent"
