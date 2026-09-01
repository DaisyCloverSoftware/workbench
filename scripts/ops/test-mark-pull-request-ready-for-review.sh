#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
target="$repo_root/scripts/ops/mark-pull-request-ready-for-review.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/bin"

cat >"$tmp/bin/gh" <<'FAKEGH'
#!/usr/bin/env bash
set -euo pipefail
scenario="${GH_READY_TEST_SCENARIO:?}"
state_dir="${GH_READY_TEST_STATE:?}"
mkdir -p "$state_dir"

if [ "${1:-}" = auth ] && [ "${2:-}" = status ]; then
  [ "$scenario" != auth_unavailable ] || exit 1
  exit 0
fi

if [ "${1:-}" != api ] || [ "${2:-}" != graphql ]; then
  exit 2
fi

joined="$*"
if [[ "$joined" == *"mutation MarkPullRequestReadyForReview"* ]]; then
  : >"$state_dir/mutation-attempted"
  if [ "$scenario" = mutation_failure ]; then
    exit 1
  fi
  : >"$state_dir/mutated"
  printf '%s\n' '{"data":{"markPullRequestReadyForReview":{"pullRequest":{"id":"PR_node"}}}}'
  exit 0
fi

sha='aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
case "$scenario" in
  success)
    if [ -f "$state_dir/mutated" ]; then draft=false; else draft=true; fi
    state=OPEN; merged=false ;;
  wrong_sha)
    state=OPEN; draft=true; merged=false ;;
  already_ready)
    state=OPEN; draft=false; merged=false ;;
  closed)
    state=CLOSED; draft=true; merged=false ;;
  mutation_failure)
    state=OPEN; draft=true; merged=false ;;
  auth_unavailable)
    exit 3 ;;
  *) exit 4 ;;
esac

# The production script asks gh to apply --jq. Emit the final TSV directly;
# this fake intentionally models only the exact bounded calls under test.
printf 'PR_node\t%s\t%s\t%s\t%s\n' "$state" "$draft" "$merged" "$sha"
FAKEGH
chmod 0755 "$tmp/bin/gh"

run_case() {
  local name="$1" expected_rc="$2" expected_mutation="$3" expected_sha="$4"
  local state_dir="$tmp/$name"
  mkdir -p "$state_dir"
  set +e
  output="$(PATH="$tmp/bin:$PATH" GH_READY_TEST_SCENARIO="$name" GH_READY_TEST_STATE="$state_dir" \
    bash "$target" DaisyCloverSoftware/rum 155 "$expected_sha" 2>&1)"
  rc=$?
  set -e
  if [ "$rc" -ne "$expected_rc" ]; then
    echo "$name: expected rc $expected_rc, got $rc: $output" >&2
    exit 1
  fi
  if [ "$expected_mutation" = yes ] && [ ! -f "$state_dir/mutation-attempted" ]; then
    echo "$name: expected mutation attempt" >&2
    exit 1
  fi
  if [ "$expected_mutation" = no ] && [ -f "$state_dir/mutation-attempted" ]; then
    echo "$name: mutation must not be attempted" >&2
    exit 1
  fi
  printf 'PASS %s\n' "$name"
}

sha='aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
wrong='bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb'
run_case success 0 yes "$sha"
run_case wrong_sha 65 no "$wrong"
run_case already_ready 65 no "$sha"
run_case closed 65 no "$sha"
run_case auth_unavailable 69 no "$sha"
run_case mutation_failure 70 yes "$sha"

echo "ALL MARK_PULL_REQUEST_READY_FOR_REVIEW REGRESSIONS PASSED"
