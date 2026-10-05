#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 4 ]]; then
  echo "usage: $0 <40-char-source-sha> <api-image@sha256:...> <web-image@sha256:...> <rat-image@sha256:...>" >&2
  exit 64
fi

SOURCE_SHA="$1"
API_IMAGE="$2"
WEB_IMAGE="$3"
RAT_IMAGE="$4"

[[ "$SOURCE_SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "invalid source SHA" >&2; exit 64; }
[[ "$API_IMAGE" =~ ^ghcr\.io/daisycloversoftware/rum-api@sha256:[0-9a-f]{64}$ ]] || { echo "invalid immutable API image" >&2; exit 64; }
[[ "$WEB_IMAGE" =~ ^ghcr\.io/daisycloversoftware/rum-web@sha256:[0-9a-f]{64}$ ]] || { echo "invalid immutable web image" >&2; exit 64; }
[[ "$RAT_IMAGE" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || { echo "invalid immutable RAT image" >&2; exit 64; }

RUM_NS="rum-dev-isolated"
RAT_NS="rum-rate-anything-preview"
PUBLIC_NS="rum-dev"
RUM_HOST="dev-rum.daisycloversoftware.uk"
RAT_HOST="dev-rum-ra.daisycloversoftware.uk"
PUBLIC_HOST="rateurmate.online"
RUM_CONFIGMAP="rum-config"
RAT_CANONICAL_EGRESS_POLICY="rum160-canonical-rating-egress"
RUM_CANONICAL_INGRESS_POLICY="rum160-canonical-rating-ingress"
POLICY_OWNER_LABEL="rum160-canonical-rating"
CANONICAL_STATEFUL_DOMAINS="${RUM_HOST},${RAT_HOST}"
CANONICAL_API_POD_PORT=8080

kctl() {
  if command -v k3s >/dev/null 2>&1; then
    sudo -n k3s kubectl "$@"
  else
    kubectl "$@"
  fi
}

for ns in "$RUM_NS" "$RAT_NS" "$PUBLIC_NS"; do
  kctl get namespace "$ns" >/dev/null
done
kctl -n "$RUM_NS" get configmap "$RUM_CONFIGMAP" >/dev/null

host_set() {
  local ns="$1"
  kctl -n "$ns" get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u
}

[[ "$(host_set "$RUM_NS")" == "$RUM_HOST" ]] || { echo "blocked: RUM isolated ingress host mismatch" >&2; exit 78; }
[[ "$(host_set "$RAT_NS")" == "$RAT_HOST" ]] || { echo "blocked: RAT isolated ingress host mismatch" >&2; exit 78; }
public_hosts_before="$(host_set "$PUBLIC_NS")"
grep -Fxq "$PUBLIC_HOST" <<<"$public_hosts_before" || { echo "blocked: public host not found in public namespace" >&2; exit 78; }

deployment_image() {
  local ns="$1" deployment="$2" container="$3"
  kctl -n "$ns" get deployment "$deployment" -o "jsonpath={.spec.template.spec.containers[?(@.name==\"$container\")].image}"
}

config_key_snapshot() {
  local key="$1"
  kctl -n "$RUM_NS" get configmap "$RUM_CONFIGMAP" -o json | python3 -c '
import json,sys
key=sys.argv[1]
data=json.load(sys.stdin).get("data") or {}
print("1" if key in data else "0")
print(data.get(key, ""))
' "$key"
}

restore_config_patch() {
  python3 - "$rat_preview_accounts_present" "$rat_preview_accounts_before" "$stateful_domains_present" "$stateful_domains_before" <<'PY'
import json,sys
preview_present, preview_value, stateful_present, stateful_value = sys.argv[1:]
print(json.dumps({"data": {
    "FEATURE_RAT_PREVIEW_ACCOUNTS": preview_value if preview_present == "1" else None,
    "SANCTUM_STATEFUL_DOMAINS": stateful_value if stateful_present == "1" else None,
}}))
PY
}

policy_owned_or_absent() {
  local ns="$1" name="$2"
  if ! kctl -n "$ns" get networkpolicy "$name" >/dev/null 2>&1; then
    printf '0\n'
    return
  fi
  local owner
  owner="$(kctl -n "$ns" get networkpolicy "$name" -o jsonpath='{.metadata.labels.rum160-owner}' 2>/dev/null || true)"
  [[ "$owner" == "$POLICY_OWNER_LABEL" ]] || {
    echo "blocked: existing network policy $ns/$name is not owned by this bounded RUM160 operation" >&2
    exit 78
  }
  printf '1\n'
}

public_web_before="$(deployment_image "$PUBLIC_NS" rum-web web)"
public_api_before="$(deployment_image "$PUBLIC_NS" rum-api php-fpm)"
public_worker_before="$(deployment_image "$PUBLIC_NS" rum-worker worker)"

rum_api_before="$(deployment_image "$RUM_NS" rum-api php-fpm)"
rum_web_before="$(deployment_image "$RUM_NS" rum-web web)"
rat_api_before="$(deployment_image "$RAT_NS" rum-api php-fpm)"
rat_web_before="$(deployment_image "$RAT_NS" rum-rate-anything rate-anything)"

mapfile -t rat_preview_accounts_snapshot < <(config_key_snapshot FEATURE_RAT_PREVIEW_ACCOUNTS)
rat_preview_accounts_present="${rat_preview_accounts_snapshot[0]}"
rat_preview_accounts_before="${rat_preview_accounts_snapshot[1]:-}"
mapfile -t stateful_domains_snapshot < <(config_key_snapshot SANCTUM_STATEFUL_DOMAINS)
stateful_domains_present="${stateful_domains_snapshot[0]}"
stateful_domains_before="${stateful_domains_snapshot[1]:-}"
rat_policy_existed="$(policy_owned_or_absent "$RAT_NS" "$RAT_CANONICAL_EGRESS_POLICY")"
rum_policy_existed="$(policy_owned_or_absent "$RUM_NS" "$RUM_CANONICAL_INGRESS_POLICY")"

restore_isolated() {
  local rc=$?
  trap - ERR
  echo "isolated candidate deployment failed; restoring prior isolated state" >&2
  kctl -n "$RUM_NS" patch configmap "$RUM_CONFIGMAP" --type=merge -p "$(restore_config_patch)" >/dev/null 2>&1 || true
  kctl -n "$RUM_NS" set image deployment/rum-api "php-fpm=$rum_api_before" >/dev/null 2>&1 || true
  kctl -n "$RUM_NS" set image deployment/rum-web "web=$rum_web_before" >/dev/null 2>&1 || true
  kctl -n "$RAT_NS" set image deployment/rum-api "php-fpm=$rat_api_before" >/dev/null 2>&1 || true
  kctl -n "$RAT_NS" set image deployment/rum-rate-anything "rate-anything=$rat_web_before" >/dev/null 2>&1 || true
  kctl -n "$RUM_NS" rollout status deployment/rum-api --timeout=180s >/dev/null 2>&1 || true
  kctl -n "$RUM_NS" rollout status deployment/rum-web --timeout=180s >/dev/null 2>&1 || true
  kctl -n "$RAT_NS" rollout status deployment/rum-api --timeout=180s >/dev/null 2>&1 || true
  kctl -n "$RAT_NS" rollout status deployment/rum-rate-anything --timeout=180s >/dev/null 2>&1 || true
  if [[ "$rat_policy_existed" == "0" ]]; then
    kctl -n "$RAT_NS" delete networkpolicy "$RAT_CANONICAL_EGRESS_POLICY" --ignore-not-found >/dev/null 2>&1 || true
  fi
  if [[ "$rum_policy_existed" == "0" ]]; then
    kctl -n "$RUM_NS" delete networkpolicy "$RUM_CANONICAL_INGRESS_POLICY" --ignore-not-found >/dev/null 2>&1 || true
  fi
  exit "$rc"
}
trap restore_isolated ERR

# The RAT owner-review host keeps its own frontend shell, but authentication and
# rating persistence are served by the canonical isolated RUM API. The Service
# is exposed on 80 but targets the API nginx pod's named http port on 8080, so
# NetworkPolicy must permit that exact destination pod port. Both namespaces
# remain default-deny for every other cross-namespace application flow.
cat <<YAML | kctl apply -f - >/dev/null
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ${RAT_CANONICAL_EGRESS_POLICY}
  namespace: ${RAT_NS}
  labels:
    rum160-owner: ${POLICY_OWNER_LABEL}
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: rum
      app.kubernetes.io/instance: rum
      app.kubernetes.io/component: rate-anything
  policyTypes: [Egress]
  egress:
    - to:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: ${RUM_NS}
          podSelector:
            matchLabels:
              app.kubernetes.io/name: rum
              app.kubernetes.io/instance: rum
              app.kubernetes.io/component: api
      ports:
        - protocol: TCP
          port: ${CANONICAL_API_POD_PORT}
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ${RUM_CANONICAL_INGRESS_POLICY}
  namespace: ${RUM_NS}
  labels:
    rum160-owner: ${POLICY_OWNER_LABEL}
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: rum
      app.kubernetes.io/instance: rum
      app.kubernetes.io/component: api
  policyTypes: [Ingress]
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: ${RAT_NS}
          podSelector:
            matchLabels:
              app.kubernetes.io/name: rum
              app.kubernetes.io/instance: rum
              app.kubernetes.io/component: rate-anything
      ports:
        - protocol: TCP
          port: ${CANONICAL_API_POD_PORT}
YAML

# The candidate API sees the RAT hostname through Nginx. Enable only the
# existing DEV-host-only no-mail preview-account exception and include the RAT
# hostname in Sanctum's stateful domains. This does not enable universal entity
# creation and does not change any public/LIVE ConfigMap.
kctl -n "$RUM_NS" patch configmap "$RUM_CONFIGMAP" --type=merge -p "$(python3 - "$CANONICAL_STATEFUL_DOMAINS" <<'PY'
import json,sys
print(json.dumps({"data": {
    "FEATURE_RAT_PREVIEW_ACCOUNTS": "true",
    "SANCTUM_STATEFUL_DOMAINS": sys.argv[1],
}}))
PY
)" >/dev/null

kctl -n "$RUM_NS" set image deployment/rum-api "php-fpm=$API_IMAGE" >/dev/null
kctl -n "$RUM_NS" set image deployment/rum-web "web=$WEB_IMAGE" >/dev/null
kctl -n "$RAT_NS" set image deployment/rum-api "php-fpm=$API_IMAGE" >/dev/null
kctl -n "$RAT_NS" set image deployment/rum-rate-anything "rate-anything=$RAT_IMAGE" >/dev/null

kctl -n "$RUM_NS" rollout status deployment/rum-api --timeout=300s
kctl -n "$RUM_NS" rollout status deployment/rum-web --timeout=300s
kctl -n "$RAT_NS" rollout status deployment/rum-api --timeout=300s
kctl -n "$RAT_NS" rollout status deployment/rum-rate-anything --timeout=300s

[[ "$(deployment_image "$RUM_NS" rum-api php-fpm)" == "$API_IMAGE" ]]
[[ "$(deployment_image "$RUM_NS" rum-web web)" == "$WEB_IMAGE" ]]
[[ "$(deployment_image "$RAT_NS" rum-api php-fpm)" == "$API_IMAGE" ]]
[[ "$(deployment_image "$RAT_NS" rum-rate-anything rate-anything)" == "$RAT_IMAGE" ]]

[[ "$(host_set "$RUM_NS")" == "$RUM_HOST" ]]
[[ "$(host_set "$RAT_NS")" == "$RAT_HOST" ]]
[[ "$(host_set "$PUBLIC_NS")" == "$public_hosts_before" ]]
[[ "$(deployment_image "$PUBLIC_NS" rum-web web)" == "$public_web_before" ]]
[[ "$(deployment_image "$PUBLIC_NS" rum-api php-fpm)" == "$public_api_before" ]]
[[ "$(deployment_image "$PUBLIC_NS" rum-worker worker)" == "$public_worker_before" ]]

[[ "$(kctl -n "$RUM_NS" get configmap "$RUM_CONFIGMAP" -o jsonpath='{.data.FEATURE_RAT_PREVIEW_ACCOUNTS}')" == "true" ]]
[[ "$(kctl -n "$RUM_NS" get configmap "$RUM_CONFIGMAP" -o jsonpath='{.data.SANCTUM_STATEFUL_DOMAINS}')" == "$CANONICAL_STATEFUL_DOMAINS" ]]
[[ "$(kctl -n "$RAT_NS" get networkpolicy "$RAT_CANONICAL_EGRESS_POLICY" -o jsonpath='{.metadata.labels.rum160-owner}')" == "$POLICY_OWNER_LABEL" ]]
[[ "$(kctl -n "$RUM_NS" get networkpolicy "$RUM_CANONICAL_INGRESS_POLICY" -o jsonpath='{.metadata.labels.rum160-owner}')" == "$POLICY_OWNER_LABEL" ]]

# The RUM web image has no /VERSION asset: its nginx SPA fallback returns index.html.
# Exact frontend identity is therefore enforced by the immutable WEB_IMAGE equality
# above. The exact-head API runtime is independently proven through its versioned
# health response, and the RAT frontend exposes the candidate APP_VERSION at /VERSION.
rum_health="$(curl -fsS -H 'Cache-Control: no-cache' --retry 6 --retry-delay 2 --max-time 20 "https://$RUM_HOST/api/v1/health")"
python3 - "$rum_health" "$SOURCE_SHA" <<'PY'
import json,sys
obj=json.loads(sys.argv[1])
source_sha=sys.argv[2]
assert obj.get('status') == 'ok', obj
assert obj.get('service') == 'rum-api', obj
version=str(obj.get('version') or '')
assert source_sha in version, (source_sha, version)
print('rum_api_version=' + version)
PY

rat_version="$(curl -fsS -H 'Cache-Control: no-cache' --retry 6 --retry-delay 2 --max-time 20 "https://$RAT_HOST/VERSION" | tr -d '\r\n')"
grep -Fq "$SOURCE_SHA" <<<"$rat_version" || { echo "exact-head RAT VERSION mismatch at https://$RAT_HOST: $rat_version" >&2; false; }
printf 'rat_version=%s\n' "$rat_version"

catalogue="$(curl -fsS -H 'Cache-Control: no-cache' --retry 6 --retry-delay 2 --max-time 20 "https://$RUM_HOST/api/v1/public/entities/rat-catalogue/search?q=CJ")"
python3 - "$catalogue" <<'PY'
import json,sys
obj=json.loads(sys.argv[1])
assert obj.get('meta',{}).get('source') == 'rum-dev-canonical', obj
active=obj.get('data',{}).get('active') or {}
assert active.get('name') == 'CJ Investigates', obj
print('canonical_cj_id=' + str(active.get('id')))
PY

trap - ERR
printf 'RUM160_ISOLATED_DEPLOYED source_sha=%s\n' "$SOURCE_SHA"
printf 'rum_dev=%s\nrat_dev=%s\npublic_host_unchanged=%s\n' "$RUM_HOST" "$RAT_HOST" "$PUBLIC_HOST"
printf 'api_image=%s\nweb_image=%s\nrat_image=%s\n' "$API_IMAGE" "$WEB_IMAGE" "$RAT_IMAGE"
printf 'rat_canonical_policy=%s/%s\n' "$RAT_NS" "$RAT_CANONICAL_EGRESS_POLICY"
printf 'rum_canonical_policy=%s/%s\n' "$RUM_NS" "$RUM_CANONICAL_INGRESS_POLICY"
printf 'canonical_api_pod_port=%s\n' "$CANONICAL_API_POD_PORT"
printf 'rum_rat_preview_accounts=true\nstateful_domains=%s\n' "$CANONICAL_STATEFUL_DOMAINS"
