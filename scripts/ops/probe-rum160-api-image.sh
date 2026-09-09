#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 0 ]]; then
  echo "usage: $0" >&2
  exit 64
fi

SOURCE_SHA="226f898fa0508b9f5b41b8d0651bab315b7dd204"
IMAGE_REPOSITORY="ghcr.io/daisycloversoftware/rum-api"
IMAGE="$IMAGE_REPOSITORY:sha-$SOURCE_SHA"
RUM_NS="rum-dev-isolated"
RAT_NS="rum-rate-anything-preview"
PULL_SECRET="ghcr-pull"
SERVICE_ACCOUNT="rum"
PROBE_POD="workbench-rum160-api-probe-${SOURCE_SHA:0:8}"

if sudo -n kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n kubectl)
elif sudo -n k3s kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n k3s kubectl)
else
  echo "ERROR: no sanctioned non-interactive Kubernetes client is available" >&2
  exit 3
fi

kube() {
  "${KUBECTL[@]}" "$@"
}

for ns in "$RUM_NS" "$RAT_NS"; do
  kube get namespace "$ns" >/dev/null
done
kube -n "$RUM_NS" get serviceaccount "$SERVICE_ACCOUNT" >/dev/null
kube -n "$RUM_NS" get secret "$PULL_SECRET" -o name >/dev/null

image_of() {
  local ns="$1"
  kube -n "$ns" get deployment rum-api -o 'jsonpath={.spec.template.spec.containers[?(@.name=="php-fpm")].image}'
}

rum_api_before="$(image_of "$RUM_NS")"
rat_api_before="$(image_of "$RAT_NS")"

created=false
cleanup() {
  if [ "$created" = true ]; then
    kube -n "$RUM_NS" delete pod "$PROBE_POD" --ignore-not-found --wait=false >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT HUP INT TERM

if kube -n "$RUM_NS" get pod "$PROBE_POD" >/dev/null 2>&1; then
  echo "ERROR: probe pod already exists; refusing to replace it: $PROBE_POD" >&2
  exit 4
fi

cat <<EOF | kube apply -f - >/dev/null
apiVersion: v1
kind: Pod
metadata:
  name: $PROBE_POD
  namespace: $RUM_NS
  labels:
    app.kubernetes.io/name: rum
    app.kubernetes.io/component: api-image-probe
    app.kubernetes.io/managed-by: workbench
    rum.daisycloversoftware.uk/recovery: pr160
spec:
  restartPolicy: Never
  serviceAccountName: $SERVICE_ACCOUNT
  imagePullSecrets:
    - name: $PULL_SECRET
  securityContext:
    runAsNonRoot: true
    runAsUser: 10001
    runAsGroup: 10001
    seccompProfile:
      type: RuntimeDefault
  containers:
    - name: probe
      image: $IMAGE
      imagePullPolicy: Always
      command: ["php", "-r", "exit(0);"]
      securityContext:
        allowPrivilegeEscalation: false
        capabilities:
          drop: ["ALL"]
EOF
created=true

phase=""
waiting_reason=""
for ((attempt = 0; attempt < 90; attempt++)); do
  phase="$(kube -n "$RUM_NS" get pod "$PROBE_POD" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
  waiting_reason="$(kube -n "$RUM_NS" get pod "$PROBE_POD" -o jsonpath='{.status.containerStatuses[0].state.waiting.reason}' 2>/dev/null || true)"

  if [ "$phase" = "Succeeded" ]; then
    break
  fi

  case "$waiting_reason" in
    ErrImagePull|ImagePullBackOff|InvalidImageName)
      echo "ERROR: image probe failed: $waiting_reason" >&2
      kube -n "$RUM_NS" describe pod "$PROBE_POD" >&2 || true
      exit 5
      ;;
  esac

  if [ "$phase" = "Failed" ]; then
    echo "ERROR: probe container failed after image resolution" >&2
    kube -n "$RUM_NS" describe pod "$PROBE_POD" >&2 || true
    exit 6
  fi

  sleep 2
done

if [ "$phase" != "Succeeded" ]; then
  echo "ERROR: image probe did not complete within the bounded wait" >&2
  kube -n "$RUM_NS" describe pod "$PROBE_POD" >&2 || true
  exit 7
fi

image_id="$(kube -n "$RUM_NS" get pod "$PROBE_POD" -o jsonpath='{.status.containerStatuses[0].imageID}')"
digest="${image_id##*@}"
if ! [[ "$digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
  echo "ERROR: image probe returned an unexpected imageID: $image_id" >&2
  exit 8
fi

kube -n "$RUM_NS" delete pod "$PROBE_POD" --wait=true --timeout=30s >/dev/null
created=false

rum_api_after="$(image_of "$RUM_NS")"
rat_api_after="$(image_of "$RAT_NS")"
[ "$rum_api_before" = "$rum_api_after" ] || {
  echo "ERROR: isolated RUM API deployment changed during the probe" >&2
  exit 9
}
[ "$rat_api_before" = "$rat_api_after" ] || {
  echo "ERROR: isolated RAT API deployment changed during the probe" >&2
  exit 10
}

printf 'RUM160_API_PROBE_SOURCE_SHA=%s\n' "$SOURCE_SHA"
printf 'RUM160_API_PROBED_TAG=%s\n' "$IMAGE"
printf 'RUM160_API_PROBED_IMAGE_ID=%s\n' "$image_id"
printf 'RUM160_API_PROBED_DIGEST=%s\n' "$digest"
printf 'RUM160_ISOLATED_RUM_API_UNCHANGED=%s\n' "$rum_api_after"
printf 'RUM160_ISOLATED_RAT_API_UNCHANGED=%s\n' "$rat_api_after"
printf 'RUM160_API_IMAGE_PROBE_PASS=true\n'
