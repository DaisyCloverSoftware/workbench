#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ $# -ne 0 ]]; then
  echo "usage: $0" >&2
  exit 64
fi

SOURCE_SHA="226f898fa0508b9f5b41b8d0651bab315b7dd204"
TAG="ghcr.io/daisycloversoftware/rum-rate-anything:sha-$SOURCE_SHA"
NS="rum-rate-anything-preview"
POD="workbench-rum160-rat-tag-${SOURCE_SHA:0:8}"

if sudo -n kubectl version --client >/dev/null 2>&1; then
  K=(sudo -n kubectl)
elif sudo -n k3s kubectl version --client >/dev/null 2>&1; then
  K=(sudo -n k3s kubectl)
else
  echo "ERROR: no sanctioned Kubernetes client" >&2; exit 3
fi
kube(){ "${K[@]}" "$@"; }

kube get namespace "$NS" >/dev/null
kube -n "$NS" get serviceaccount rum >/dev/null
kube -n "$NS" get secret ghcr-pull -o name >/dev/null
kube -n "$NS" delete pod "$POD" --ignore-not-found --wait=false >/dev/null 2>&1 || true
cleanup(){ kube -n "$NS" delete pod "$POD" --ignore-not-found --wait=false >/dev/null 2>&1 || true; }
trap cleanup EXIT HUP INT TERM

cat <<EOF | kube apply -f - >/dev/null
apiVersion: v1
kind: Pod
metadata:
  name: $POD
  namespace: $NS
  labels:
    app.kubernetes.io/component: rat-tag-probe
    app.kubernetes.io/managed-by: workbench
spec:
  restartPolicy: Never
  serviceAccountName: rum
  imagePullSecrets:
    - name: ghcr-pull
  securityContext:
    runAsNonRoot: true
    runAsUser: 101
    runAsGroup: 101
    seccompProfile:
      type: RuntimeDefault
  containers:
    - name: probe
      image: $TAG
      imagePullPolicy: Always
      command: ["/bin/sh", "-c", "grep -F '$SOURCE_SHA' /usr/share/nginx/html/VERSION >/dev/null"]
      securityContext:
        allowPrivilegeEscalation: false
        capabilities:
          drop: ["ALL"]
EOF

phase=""; reason=""
for ((i=0; i<90; i++)); do
  phase="$(kube -n "$NS" get pod "$POD" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
  reason="$(kube -n "$NS" get pod "$POD" -o jsonpath='{.status.containerStatuses[0].state.waiting.reason}' 2>/dev/null || true)"
  [[ "$phase" == "Succeeded" ]] && break
  case "$reason" in ErrImagePull|ImagePullBackOff|InvalidImageName) kube -n "$NS" describe pod "$POD" >&2 || true; exit 5;; esac
  [[ "$phase" == "Failed" ]] && { kube -n "$NS" describe pod "$POD" >&2 || true; exit 6; }
  sleep 2
done
[[ "$phase" == "Succeeded" ]] || { kube -n "$NS" describe pod "$POD" >&2 || true; exit 7; }

image_id="$(kube -n "$NS" get pod "$POD" -o jsonpath='{.status.containerStatuses[0].imageID}')"
[[ "$image_id" =~ ^ghcr\.io/daisycloversoftware/rum-rate-anything@sha256:[0-9a-f]{64}$ ]] || { echo "ERROR: unexpected imageID: $image_id" >&2; exit 8; }

kube -n "$NS" delete pod "$POD" --wait=true --timeout=30s >/dev/null
trap - EXIT HUP INT TERM
printf 'RUM160_RAT_TAG_SOURCE_SHA=%s\n' "$SOURCE_SHA"
printf 'RUM160_RAT_TAG=%s\n' "$TAG"
printf 'RUM160_RAT_RESOLVED_IMAGE_ID=%s\n' "$image_id"
printf 'RUM160_RAT_TAG_PULL_PASS=true\n'
