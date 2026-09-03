#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = "--self-test" ]; then
  [ "$#" -eq 1 ]
  printf 'INSPECT_POSTGRES_USER_PROFILE_PRESENCE_SELF_TEST_OK\n'
  exit 0
fi

if [ "$#" -ne 2 ]; then
  echo "Usage: inspect-postgres-user-profile-presence.sh <namespace> <postgres-pod>" >&2
  exit 2
fi

NAMESPACE="$1"
POD="$2"
case "$NAMESPACE" in *[!A-Za-z0-9._-]*|'') echo "ERROR: invalid namespace" >&2; exit 2;; esac
case "$POD" in *[!A-Za-z0-9._-]*|'') echo "ERROR: invalid pod" >&2; exit 2;; esac

if sudo -n kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n kubectl)
elif sudo -n k3s kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n k3s kubectl)
else
  echo "ERROR: no sanctioned non-interactive Kubernetes client is available" >&2
  exit 1
fi

ready="$("${KUBECTL[@]}" get pod "$POD" -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses[0].ready}')"
if [ "$ready" != "true" ]; then
  echo "ERROR: PostgreSQL pod is not Ready" >&2
  exit 1
fi

SQL=$(cat <<'SQL'
SELECT 'db_connectivity=ok';
SELECT 'users_count=' || count(*)::text FROM users;
SELECT 'profiles_count=' || count(*)::text FROM profiles;
SELECT 'linked_user_profiles=' || count(*)::text FROM users u JOIN profiles p ON p.user_id = u.id;
SELECT 'founder_accounts=' || count(*)::text FROM users WHERE founder_number IS NOT NULL;
SELECT 'founder_1_profile_links=' || count(*)::text FROM users u JOIN profiles p ON p.user_id = u.id WHERE u.founder_number = 1;
SQL
)

printf '%s\n' "$SQL" | "${KUBECTL[@]}" exec -i -n "$NAMESPACE" "$POD" -- sh -ec 'exec psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -At -v ON_ERROR_STOP=1'
