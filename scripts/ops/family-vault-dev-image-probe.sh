#!/usr/bin/env bash
set -euo pipefail
umask 077

DEV_NS="family-vault-dev"
LIVE_NS="family-vault-live"
DEPLOYMENT="family-vault-web"
IMAGE_REPOSITORY="ghcr.io/daisycloversoftware/family-vault-web"
PULL_SECRET="ghcr-pull-secret"
SOURCE_SHA="${1:-}"

if ! [[ "$SOURCE_SHA" =~ ^[0-9a-f]{40}$ ]]; then
  echo "Usage: family-vault-dev-image-probe.sh <40-hex-source-sha>" >&2
  exit 2
fi

for command in sleep mktemp; do
  command -v "$command" >/dev/null 2>&1 || {
    echo "ERROR: required command is unavailable: $command" >&2
    exit 3
  }
done

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

kube get namespace "$DEV_NS" >/dev/null
kube get namespace "$LIVE_NS" >/dev/null
kube -n "$DEV_NS" get secret "$PULL_SECRET" >/dev/null
kube -n "$DEV_NS" get deployment "$DEPLOYMENT" >/dev/null
kube -n "$LIVE_NS" get deployment "$DEPLOYMENT" >/dev/null

dev_image_before="$(kube -n "$DEV_NS" get deployment "$DEPLOYMENT" -o jsonpath='{.spec.template.spec.containers[0].image}')"
live_image_before="$(kube -n "$LIVE_NS" get deployment "$DEPLOYMENT" -o jsonpath='{.spec.template.spec.containers[0].image}')"

probe_pod="workbench-fv-probe-${SOURCE_SHA:0:8}"
manifest="$(mktemp)"
created=false
cleanup() {
  if [ "$created" = true ]; then
    kube -n "$DEV_NS" delete pod "$probe_pod" --ignore-not-found --wait=false >/dev/null 2>&1 || true
  fi
  rm -f "$manifest"
}
trap cleanup EXIT HUP INT TERM

if kube -n "$DEV_NS" get pod "$probe_pod" >/dev/null 2>&1; then
  echo "ERROR: probe pod already exists; refusing to replace it: $probe_pod" >&2
  exit 4
fi

cat > "$manifest" <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: $probe_pod
  namespace: $DEV_NS
  labels:
    app.kubernetes.io/name: family-vault
    app.kubernetes.io/component: image-probe
    app.kubernetes.io/managed-by: workbench
spec:
  restartPolicy: Never
  nodeSelector:
    daisyclover.io/apps: "true"
  imagePullSecrets:
    - name: $PULL_SECRET
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    runAsGroup: 1000
    seccompProfile:
      type: RuntimeDefault
  containers:
    - name: probe
      image: $IMAGE_REPOSITORY:$SOURCE_SHA
      imagePullPolicy: Always
      command: ["node", "-e", "process.exit(0)"]
      securityContext:
        allowPrivilegeEscalation: false
        capabilities:
          drop: ["ALL"]
EOF

kube apply -f "$manifest" >/dev/null
created=true

phase=""
waiting_reason=""
for ((attempt = 0; attempt < 90; attempt++)); do
  phase="$(kube -n "$DEV_NS" get pod "$probe_pod" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
  waiting_reason="$(kube -n "$DEV_NS" get pod "$probe_pod" -o jsonpath='{.status.containerStatuses[0].state.waiting.reason}' 2>/dev/null || true)"

  if [ "$phase" = "Succeeded" ]; then
    break
  fi

  case "$waiting_reason" in
    ErrImagePull|ImagePullBackOff|InvalidImageName)
      echo "ERROR: image probe failed: $waiting_reason" >&2
      kube -n "$DEV_NS" describe pod "$probe_pod" >&2 || true
      exit 5
      ;;
  esac

  if [ "$phase" = "Failed" ]; then
    echo "ERROR: image probe container failed after image resolution" >&2
    kube -n "$DEV_NS" describe pod "$probe_pod" >&2 || true
    exit 6
  fi

  sleep 2
done

if [ "$phase" != "Succeeded" ]; then
  echo "ERROR: image probe did not complete within the bounded wait" >&2
  kube -n "$DEV_NS" describe pod "$probe_pod" >&2 || true
  exit 7
fi

image_id="$(kube -n "$DEV_NS" get pod "$probe_pod" -o jsonpath='{.status.containerStatuses[0].imageID}')"
digest="${image_id##*@}"
if ! [[ "$digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
  echo "ERROR: image probe returned an unexpected imageID: $image_id" >&2
  exit 8
fi

kube -n "$DEV_NS" delete pod "$probe_pod" --wait=true --timeout=30s >/dev/null
created=false

dev_image_after="$(kube -n "$DEV_NS" get deployment "$DEPLOYMENT" -o jsonpath='{.spec.template.spec.containers[0].image}')"
live_image_after="$(kube -n "$LIVE_NS" get deployment "$DEPLOYMENT" -o jsonpath='{.spec.template.spec.containers[0].image}')"

[ "$dev_image_before" = "$dev_image_after" ] || {
  echo "ERROR: DEV application deployment changed during the probe" >&2
  exit 9
}
[ "$live_image_before" = "$live_image_after" ] || {
  echo "ERROR: LIVE application deployment changed during the probe" >&2
  exit 10
}

echo "FAMILY_VAULT_DEV_IMAGE_BEFORE=$dev_image_before"
echo "FAMILY_VAULT_LIVE_IMAGE_BEFORE=$live_image_before"
echo "FAMILY_VAULT_PROBED_IMAGE=$IMAGE_REPOSITORY:$SOURCE_SHA"
echo "FAMILY_VAULT_PROBED_IMAGE_ID=$image_id"
echo "FAMILY_VAULT_PROBED_DIGEST=$digest"
echo "FAMILY_VAULT_DEV_DEPLOYMENT_UNCHANGED=true"
echo "FAMILY_VAULT_LIVE_DEPLOYMENT_UNCHANGED=true"
echo "FAMILY_VAULT_DEV_IMAGE_PROBE_PASS=true"
