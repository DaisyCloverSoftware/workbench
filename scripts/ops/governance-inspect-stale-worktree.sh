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

expected_branch='fix/preserve-changeset-file-modes'
expected_head='0d9311802485d063c80bc23a5a6c5eb599d0b581'
if [ "${WORKBENCH_GOVERNANCE_TEST_ALLOW_LOCAL_ORIGIN:-0}" = "1" ] && [ "${WORKBENCH_OPERATION_SCRIPT:-0}" != "1" ]; then
  expected_branch="${WORKBENCH_GOVERNANCE_TEST_EXPECTED_BRANCH:-$expected_branch}"
  expected_head="${WORKBENCH_GOVERNANCE_TEST_EXPECTED_HEAD:-$expected_head}"
fi

found_path=''
current_path=''
current_head=''
current_branch=''
check_record() {
  if [ "$current_branch" = "$expected_branch" ]; then
    if [ -n "$found_path" ]; then
      printf 'error=duplicate-target-worktree\n' >&2
      exit 68
    fi
    if [ "$current_head" != "$expected_head" ]; then
      printf 'error=unexpected-target-head\n' >&2
      exit 69
    fi
    found_path="$current_path"
  fi
}

while IFS= read -r line || [ -n "$line" ]; do
  case "$line" in
    worktree\ *)
      if [ -n "$current_path" ]; then
        check_record
      fi
      current_path="${line#worktree }"
      current_head=''
      current_branch=''
      ;;
    HEAD\ *) current_head="${line#HEAD }" ;;
    branch\ refs/heads/*) current_branch="${line#branch refs/heads/}" ;;
    detached) current_branch='detached' ;;
  esac
done < <(git -C "$root" worktree list --porcelain)
if [ -n "$current_path" ]; then
  check_record
fi
if [ -z "$found_path" ]; then
  printf 'error=target-worktree-not-found\n' >&2
  exit 70
fi

status_text="$(git -C "$found_path" status --porcelain=v1 --untracked-files=all)"
expected_status=$' M internal/core/changeset_prepare.go\n M internal/core/changeset_prepare_test.go'
if [ "$status_text" != "$expected_status" ]; then
  printf 'error=unexpected-target-dirty-set\n' >&2
  exit 71
fi

printf 'inspection=ok\n'
printf 'branch=%s\n' "$expected_branch"
printf 'head=%s\n' "$expected_head"
printf 'changeset_prepare_blob=%s\n' "$(git -C "$found_path" hash-object internal/core/changeset_prepare.go)"
printf 'changeset_prepare_test_blob=%s\n' "$(git -C "$found_path" hash-object internal/core/changeset_prepare_test.go)"
printf '%s\n' '--- diff begin ---'
git -C "$found_path" diff --no-ext-diff --unified=3 -- \
  internal/core/changeset_prepare.go \
  internal/core/changeset_prepare_test.go
printf '%s\n' '--- diff end ---'
