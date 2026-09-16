#!/usr/bin/env bash
set -euo pipefail

[[ $# -eq 0 ]] || { echo "usage: $0" >&2; exit 64; }

RUM_NS="rum-dev-isolated"
RAT_NS="rum-rate-anything-preview"
PUBLIC_NS="rum-dev"
PUBLIC_HOST="rateurmate.online"
RUM_HOST="dev-rum.daisycloversoftware.uk"
RUM_CONFIGMAP="rum-config"
RAT_POLICY="rum160-canonical-rating-egress"
RUM_POLICY="rum160-canonical-rating-ingress"
POLICY_OWNER="rum160-canonical-rating"

if command -v k3s >/dev/null 2>&1; then
  KUBECTL=(sudo -n k3s kubectl)
else
  KUBECTL=(kubectl)
fi
kube() { "${KUBECTL[@]}" "$@"; }

for ns in "$RUM_NS" "$RAT_NS" "$PUBLIC_NS"; do kube get namespace "$ns" >/dev/null; done

host_set() {
  local ns="$1"
  kube -n "$ns" get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u
}
deployment_image() {
  local ns="$1" deployment="$2" container="$3"
  kube -n "$ns" get deployment "$deployment" -o "jsonpath={.spec.template.spec.containers[?(@.name==\"$container\")].image}"
}
ready_pod_image() {
  local ns="$1" component="$2" container="$3" snapshot
  snapshot="$(mktemp)"
  kube -n "$ns" get pods -l "app.kubernetes.io/instance=rum,app.kubernetes.io/name=rum,app.kubernetes.io/component=$component" -o json >"$snapshot"
  python3 - "$container" "$snapshot" <<'PY'
import json,sys
container,path=sys.argv[1:]
with open(path,encoding='utf-8') as fh:
    obj=json.load(fh)
found=[]
for pod in obj.get('items',[]):
    statuses={s.get('name'):s for s in (pod.get('status',{}).get('containerStatuses') or [])}
    status=statuses.get(container) or {}
    if status.get('ready') is not True:
        continue
    image=next((c.get('image') for c in pod.get('spec',{}).get('containers',[]) if c.get('name')==container),None)
    if image:
        found.append(image)
unique=sorted(set(found))
if len(unique)!=1:
    raise SystemExit(f'expected exactly one unique Ready image for {container}, found {unique}')
print(unique[0])
PY
  rm -f "$snapshot"
}

[[ "$(host_set "$RUM_NS")" == "$RUM_HOST" ]] || { echo "blocked: isolated RUM host mismatch" >&2; exit 78; }
public_hosts="$(host_set "$PUBLIC_NS")"
grep -Fxq "$PUBLIC_HOST" <<<"$public_hosts" || { echo "blocked: public host missing" >&2; exit 78; }

public_web_before="$(deployment_image "$PUBLIC_NS" rum-web web)"
public_api_before="$(deployment_image "$PUBLIC_NS" rum-api php-fpm)"
public_worker_before="$(deployment_image "$PUBLIC_NS" rum-worker worker)"

rum_api_ready="$(ready_pod_image "$RUM_NS" api php-fpm)"
rum_web_ready="$(ready_pod_image "$RUM_NS" web web)"
rat_api_ready="$(ready_pod_image "$RAT_NS" api php-fpm)"
rat_web_ready="$(ready_pod_image "$RAT_NS" rate-anything rate-anything)"

for pair in "$RUM_NS:$RUM_POLICY" "$RAT_NS:$RAT_POLICY"; do
  ns="${pair%%:*}"; name="${pair##*:}"
  if kube -n "$ns" get networkpolicy "$name" >/dev/null 2>&1; then
    owner="$(kube -n "$ns" get networkpolicy "$name" -o jsonpath='{.metadata.labels.rum160-owner}' 2>/dev/null || true)"
    [[ "$owner" == "$POLICY_OWNER" ]] || { echo "blocked: refusing to delete unowned policy $ns/$name" >&2; exit 78; }
  fi
done

printf 'Restoring isolated desired state to the images currently proven Ready.\n'
printf 'rum_api=%s\nrum_web=%s\nrat_api=%s\nrat_web=%s\n' "$rum_api_ready" "$rum_web_ready" "$rat_api_ready" "$rat_web_ready"

kube -n "$RUM_NS" set image deployment/rum-api "php-fpm=$rum_api_ready" >/dev/null
kube -n "$RUM_NS" set image deployment/rum-web "web=$rum_web_ready" >/dev/null
kube -n "$RAT_NS" set image deployment/rum-api "php-fpm=$rat_api_ready" >/dev/null
kube -n "$RAT_NS" set image deployment/rum-rate-anything "rate-anything=$rat_web_ready" >/dev/null

kube -n "$RUM_NS" patch configmap "$RUM_CONFIGMAP" --type=merge -p "$(python3 - "$RUM_HOST" <<'PY'
import json,sys
print(json.dumps({'data': {
    'FEATURE_RAT_PREVIEW_ACCOUNTS': None,
    'SANCTUM_STATEFUL_DOMAINS': sys.argv[1],
}}))
PY
)" >/dev/null
kube -n "$RUM_NS" delete networkpolicy "$RUM_POLICY" --ignore-not-found >/dev/null
kube -n "$RAT_NS" delete networkpolicy "$RAT_POLICY" --ignore-not-found >/dev/null

kube -n "$RUM_NS" rollout status deployment/rum-api --timeout=180s
kube -n "$RUM_NS" rollout status deployment/rum-web --timeout=180s
kube -n "$RAT_NS" rollout status deployment/rum-api --timeout=180s
kube -n "$RAT_NS" rollout status deployment/rum-rate-anything --timeout=180s

[[ "$(deployment_image "$RUM_NS" rum-api php-fpm)" == "$rum_api_ready" ]]
[[ "$(deployment_image "$RUM_NS" rum-web web)" == "$rum_web_ready" ]]
[[ "$(deployment_image "$RAT_NS" rum-api php-fpm)" == "$rat_api_ready" ]]
[[ "$(deployment_image "$RAT_NS" rum-rate-anything rate-anything)" == "$rat_web_ready" ]]
[[ -z "$(kube -n "$RUM_NS" get configmap "$RUM_CONFIGMAP" -o jsonpath='{.data.FEATURE_RAT_PREVIEW_ACCOUNTS}' 2>/dev/null || true)" ]]
[[ "$(kube -n "$RUM_NS" get configmap "$RUM_CONFIGMAP" -o jsonpath='{.data.SANCTUM_STATEFUL_DOMAINS}')" == "$RUM_HOST" ]]
! kube -n "$RUM_NS" get networkpolicy "$RUM_POLICY" >/dev/null 2>&1
! kube -n "$RAT_NS" get networkpolicy "$RAT_POLICY" >/dev/null 2>&1

[[ "$(host_set "$PUBLIC_NS")" == "$public_hosts" ]]
[[ "$(deployment_image "$PUBLIC_NS" rum-web web)" == "$public_web_before" ]]
[[ "$(deployment_image "$PUBLIC_NS" rum-api php-fpm)" == "$public_api_before" ]]
[[ "$(deployment_image "$PUBLIC_NS" rum-worker worker)" == "$public_worker_before" ]]

printf 'RUM160_INTERRUPTED_ISOLATED_STATE_RESTORED=true\n'
printf 'PUBLIC_RUM_UNCHANGED=true\n'
