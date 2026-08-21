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

if [ "$(git -C "$root" branch --show-current 2>/dev/null || true)" != "main" ]; then
  printf 'error=unexpected-branch\n' >&2
  exit 69
fi
if [ -n "$(git -C "$root" status --porcelain=v1 --untracked-files=all)" ]; then
  printf 'error=main-checkout-dirty\n' >&2
  exit 70
fi

expected_main='72d19d14d0af628256b1042a86082dde9e331bcf'
expected_detached='2defa97101447c04e8350bfae88414cbacafe237'
expected_detached_count=6
expected_named_head='0d9311802485d063c80bc23a5a6c5eb599d0b581'
expected_named_branch='fix/preserve-changeset-file-modes'
expected_prepare_blob='40900322cf4e61c4a65ce9b769958a3445812994'
expected_prepare_test_blob='e90b7d17df6cdc7efa85f8c03739f756fd6bd260'
audit_branch='audit/pre-governance-reset-20260821'
if [ "${WORKBENCH_GOVERNANCE_TEST_ALLOW_LOCAL_ORIGIN:-0}" = "1" ] && [ "${WORKBENCH_OPERATION_SCRIPT:-0}" != "1" ]; then
  expected_main="${WORKBENCH_GOVERNANCE_TEST_EXPECTED_MAIN:-$expected_main}"
  expected_detached="${WORKBENCH_GOVERNANCE_TEST_EXPECTED_DETACHED:-$expected_detached}"
  expected_detached_count="${WORKBENCH_GOVERNANCE_TEST_EXPECTED_DETACHED_COUNT:-$expected_detached_count}"
  expected_named_head="${WORKBENCH_GOVERNANCE_TEST_EXPECTED_NAMED_HEAD:-$expected_named_head}"
  expected_named_branch="${WORKBENCH_GOVERNANCE_TEST_EXPECTED_NAMED_BRANCH:-$expected_named_branch}"
  expected_prepare_blob="${WORKBENCH_GOVERNANCE_TEST_EXPECTED_PREPARE_BLOB:-$expected_prepare_blob}"
  expected_prepare_test_blob="${WORKBENCH_GOVERNANCE_TEST_EXPECTED_PREPARE_TEST_BLOB:-$expected_prepare_test_blob}"
  audit_branch="${WORKBENCH_GOVERNANCE_TEST_AUDIT_BRANCH:-$audit_branch}"
fi

if [ "$(git -C "$root" rev-parse HEAD)" != "$expected_main" ]; then
  printf 'error=unexpected-main-head\n' >&2
  exit 71
fi
if [ "$(git -C "$root" rev-parse --verify "refs/heads/$audit_branch" 2>/dev/null || true)" != "$expected_detached" ]; then
  printf 'error=audit-branch-missing-or-wrong\n' >&2
  exit 72
fi

declare -a paths heads branches locks dirty
current_path=''
current_head=''
current_branch=''
current_locked=0

append_record() {
  if [ -z "$current_path" ]; then
    return
  fi
  local d=0
  if [ ! -d "$current_path" ] || [ -n "$(git -C "$current_path" status --porcelain=v1 --untracked-files=all 2>/dev/null || printf '__status_error__')" ]; then
    d=1
  fi
  paths+=("$current_path")
  heads+=("$current_head")
  branches+=("$current_branch")
  locks+=("$current_locked")
  dirty+=("$d")
}

while IFS= read -r line || [ -n "$line" ]; do
  case "$line" in
    worktree\ *)
      append_record
      current_path="${line#worktree }"
      current_head=''
      current_branch=''
      current_locked=0
      ;;
    HEAD\ *) current_head="${line#HEAD }" ;;
    branch\ refs/heads/*) current_branch="${line#branch refs/heads/}" ;;
    detached) current_branch='detached' ;;
    locked*) current_locked=1 ;;
  esac
done < <(git -C "$root" worktree list --porcelain)
append_record

if [ "${#paths[@]}" -ne $((expected_detached_count + 2)) ]; then
  printf 'error=unexpected-worktree-count\n' >&2
  exit 73
fi
if [ "$(cd "${paths[0]}" && pwd -P)" != "$target" ] || [ "${heads[0]}" != "$expected_main" ] || [ "${branches[0]}" != 'main' ] || [ "${locks[0]}" -ne 0 ] || [ "${dirty[0]}" -ne 0 ]; then
  printf 'error=unexpected-primary-worktree\n' >&2
  exit 74
fi

detached_count=0
named_count=0
named_path=''
named_duplicate_dirty=0
for ((i=1; i<${#paths[@]}; i++)); do
  if [ "${locks[$i]}" -ne 0 ]; then
    printf 'error=secondary-worktree-locked\n' >&2
    exit 75
  fi
  if [ "${branches[$i]}" = 'detached' ] && [ "${heads[$i]}" = "$expected_detached" ]; then
    if [ "${dirty[$i]}" -ne 0 ]; then
      printf 'error=detached-worktree-dirty\n' >&2
      exit 76
    fi
    detached_count=$((detached_count + 1))
    continue
  fi
  if [ "${branches[$i]}" = "$expected_named_branch" ] && [ "${heads[$i]}" = "$expected_named_head" ]; then
    named_count=$((named_count + 1))
    named_path="${paths[$i]}"
    if [ "${dirty[$i]}" -ne 0 ]; then
      status_text="$(git -C "$named_path" status --porcelain=v1 --untracked-files=all)"
      expected_status=$' M internal/core/changeset_prepare.go\n M internal/core/changeset_prepare_test.go'
      if [ "$status_text" != "$expected_status" ]; then
        printf 'error=unexpected-named-worktree-dirty-set\n' >&2
        exit 77
      fi
      prepare_blob="$(git -C "$named_path" hash-object internal/core/changeset_prepare.go)"
      prepare_test_blob="$(git -C "$named_path" hash-object internal/core/changeset_prepare_test.go)"
      if [ "$prepare_blob" != "$expected_prepare_blob" ] || [ "$prepare_test_blob" != "$expected_prepare_test_blob" ]; then
        printf 'error=named-worktree-content-not-published-duplicate\n' >&2
        exit 78
      fi
      named_duplicate_dirty=1
    fi
    continue
  fi
  printf 'error=unexpected-secondary-worktree\n' >&2
  exit 79
done
if [ "$detached_count" -ne "$expected_detached_count" ] || [ "$named_count" -ne 1 ]; then
  printf 'error=unexpected-secondary-topology\n' >&2
  exit 80
fi

if [ "$named_duplicate_dirty" -eq 1 ]; then
  # The exact dirty blobs are already public in 0b601ca. Restore only those two
  # duplicated files before removing the worktree; do not force-remove unknown work.
  git -C "$named_path" restore --staged --worktree -- \
    internal/core/changeset_prepare.go \
    internal/core/changeset_prepare_test.go
  if [ -n "$(git -C "$named_path" status --porcelain=v1 --untracked-files=all)" ]; then
    printf 'error=named-worktree-still-dirty-after-duplicate-restore\n' >&2
    exit 81
  fi
fi

for ((i=1; i<${#paths[@]}; i++)); do
  git -C "$root" worktree remove "${paths[$i]}"
done
git -C "$root" worktree prune

if [ "$(git -C "$root" worktree list --porcelain | grep -c '^worktree ')" -ne 1 ]; then
  printf 'error=secondary-worktrees-remain\n' >&2
  exit 82
fi
if git -C "$root" show-ref --verify --quiet "refs/heads/$expected_named_branch"; then
  if ! git -C "$root" merge-base --is-ancestor "refs/heads/$expected_named_branch" HEAD; then
    printf 'error=named-worktree-branch-not-merged\n' >&2
    exit 83
  fi
  git -C "$root" branch -d "$expected_named_branch" >/dev/null
fi

# Advance the now-clean registered main checkout to the exact reviewed commit
# carrying this cleanup operation. The audit branch keeps the old local-only
# SEC-008 line reachable.
desired="${WORKBENCH_OPERATION_COMMIT:-}"
case "$desired" in
  ''|*[!0-9a-f]*)
    printf 'error=operation-commit-required\n' >&2
    exit 84
    ;;
esac
if [ "${#desired}" -ne 40 ]; then
  printf 'error=operation-commit-required\n' >&2
  exit 84
fi
git -C "$root" fetch --quiet origin main
if [ "$(git -C "$root" rev-parse FETCH_HEAD)" != "$desired" ]; then
  printf 'error=fetched-main-does-not-match-operation-commit\n' >&2
  exit 85
fi
if ! git -C "$root" merge-base --is-ancestor HEAD "$desired"; then
  printf 'error=current-main-not-ancestor-of-operation-commit\n' >&2
  exit 86
fi
git -C "$root" merge --ff-only "$desired" >/dev/null

if [ "$(git -C "$root" rev-parse HEAD)" != "$desired" ] || [ -n "$(git -C "$root" status --porcelain=v1 --untracked-files=all)" ]; then
  printf 'error=final-main-state-invalid\n' >&2
  exit 87
fi
if [ "$(git -C "$root" rev-parse "refs/heads/$audit_branch")" != "$expected_detached" ]; then
  printf 'error=audit-branch-not-preserved\n' >&2
  exit 88
fi

printf 'cleanup=ok\n'
printf 'secondary_worktrees_removed=%d\n' $((expected_detached_count + 1))
printf 'detached_removed=%d\n' "$expected_detached_count"
printf 'published_duplicate_restored=%d\n' "$named_duplicate_dirty"
printf 'named_branch_removed=1\n'
printf 'current_head=%s\n' "$desired"
printf 'audit_branch=%s\n' "$audit_branch"
