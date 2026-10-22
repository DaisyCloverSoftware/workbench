#!/usr/bin/env bash
set -euo pipefail
umask 077

SOURCE_SHA="${1:-}"
IMAGE_REPOSITORY="ghcr.io/daisycloversoftware/rum-api"
TARGET_NS="rum-dev-isolated"
TARGET_DEPLOYMENT="rum-api"
TARGET_CONTAINER="php-fpm"

if ! [[ "$SOURCE_SHA" =~ ^[0-9a-f]{40}$ ]]; then
  echo "usage: rum160-api-image-digest-probe.sh <40-hex-source-sha>" >&2
  exit 64
fi

for command in python3 sleep sort mktemp; do
  command -v "$command" >/dev/null 2>&1 || {
    echo "ERROR: required command is unavailable: $command" >&2
    exit 3
  }
done

if command -v k3s >/dev/null 2>&1 && sudo -n k3s kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n k3s kubectl)
elif sudo -n kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(sudo -n kubectl)
elif kubectl version --client >/dev/null 2>&1; then
  KUBECTL=(kubectl)
else
  echo "ERROR: no sanctioned non-interactive Kubernetes client is available" >&2
  exit 3
fi

kube() {
  "${KUBECTL[@]}" "$@"
}

for ns in rum-dev-isolated rum-rate-anything-preview rum-dev rum-prod; do
  kube get namespace "$ns" >/dev/null
 done
kube -n "$TARGET_NS" get deployment "$TARGET_DEPLOYMENT" >/dev/null

namespace_deployment_images() {
  local ns="$1"
  kube -n "$ns" get deployments -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{range .spec.template.spec.containers[*]}{.name}{"="}{.image}{";"}{end}{"\n"}{end}' | LC_ALL=C sort
}

isolated_rum_before="$(namespace_deployment_images rum-dev-isolated)"
isolated_rat_before="$(namespace_deployment_images rum-rate-anything-preview)"
public_before="$(namespace_deployment_images rum-dev)"
live_before="$(namespace_deployment_images rum-prod)"

created_pods=()
cleanup() {
  local pod
  for pod in "${created_pods[@]:-}"; do
    if [ -n "$pod" ]; then
      kube -n "$TARGET_NS" delete pod "$pod" --ignore-not-found --wait=false >/dev/null 2>&1 || true
    fi
  done
}
trap cleanup EXIT HUP INT TERM

probe_tag() {
  local tag="$1"
  local pod="$2"
  local deployment_json manifest image_id waiting_reason phase digest

  if kube -n "$TARGET_NS" get pod "$pod" >/dev/null 2>&1; then
    echo "ERROR: probe pod already exists; refusing to replace it: $pod" >&2
    return 4
  fi

  deployment_json="$(mktemp)"
  manifest="$(mktemp)"
  kube -n "$TARGET_NS" get deployment "$TARGET_DEPLOYMENT" -o json >"$deployment_json"

  python3 - "$deployment_json" "$manifest" "$pod" "$TARGET_NS" "$TARGET_CONTAINER" "$IMAGE_REPOSITORY:$tag" <<'PY'
import copy
import json
import sys

src_path, out_path, pod_name, namespace, container_name, image = sys.argv[1:]
with open(src_path, "r", encoding="utf-8") as fh:
    deployment = json.load(fh)

template_spec = deployment["spec"]["template"]["spec"]
source_container = next(
    c for c in template_spec.get("containers", []) if c.get("name") == container_name
)

pod_spec = {
    "restartPolicy": "Never",
    "automountServiceAccountToken": False,
    "containers": [
        {
            "name": "probe",
            "image": image,
            "imagePullPolicy": "Always",
            "command": ["php", "-r", "sleep(300);"],
        }
    ],
}

for key in ("imagePullSecrets", "nodeSelector", "tolerations", "affinity", "securityContext", "runtimeClassName"):
    if key in template_spec:
        pod_spec[key] = copy.deepcopy(template_spec[key])

if "securityContext" in source_container:
    pod_spec["containers"][0]["securityContext"] = copy.deepcopy(source_container["securityContext"])

pod = {
    "apiVersion": "v1",
    "kind": "Pod",
    "metadata": {
        "name": pod_name,
        "namespace": namespace,
        "labels": {
            "app.kubernetes.io/name": "rum",
            "app.kubernetes.io/component": "api-image-digest-probe",
            "app.kubernetes.io/managed-by": "workbench",
        },
    },
    "spec": pod_spec,
}

with open(out_path, "w", encoding="utf-8") as fh:
    json.dump(pod, fh)
PY

  rm -f "$deployment_json"
  kube create -f "$manifest" >/dev/null
  rm -f "$manifest"
  created_pods+=("$pod")

  image_id=""
  waiting_reason=""
  phase=""
  for ((attempt = 0; attempt < 90; attempt++)); do
    image_id="$(kube -n "$TARGET_NS" get pod "$pod" -o jsonpath='{.status.containerStatuses[0].imageID}' 2>/dev/null || true)"
    if [[ "$image_id" == *@sha256:* ]]; then
      break
    fi

    waiting_reason="$(kube -n "$TARGET_NS" get pod "$pod" -o jsonpath='{.status.containerStatuses[0].state.waiting.reason}' 2>/dev/null || true)"
    phase="$(kube -n "$TARGET_NS" get pod "$pod" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
    case "$waiting_reason" in
      ErrImagePull|ImagePullBackOff|InvalidImageName)
        echo "ERROR: image probe failed for $IMAGE_REPOSITORY:$tag: $waiting_reason" >&2
        kube -n "$TARGET_NS" describe pod "$pod" >&2 || true
        return 5
        ;;
    esac
    if [ "$phase" = "Failed" ] && [ -z "$image_id" ]; then
      echo "ERROR: probe pod failed before an immutable image ID was observed for $IMAGE_REPOSITORY:$tag" >&2
      kube -n "$TARGET_NS" describe pod "$pod" >&2 || true
      return 6
    fi
    sleep 2
  done

  digest="${image_id##*@}"
  if ! [[ "$digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
    echo "ERROR: image probe returned an unexpected imageID for $IMAGE_REPOSITORY:$tag: $image_id" >&2
    return 7
  fi

  kube -n "$TARGET_NS" delete pod "$pod" --wait=true --timeout=30s >/dev/null
  printf '%s\t%s\n' "$image_id" "$digest"
}

candidate_tag="rat-candidate-${SOURCE_SHA:0:12}"
full_sha_tag="sha-$SOURCE_SHA"
candidate_pod="wb-rum-api-cand-${SOURCE_SHA:0:10}"
full_sha_pod="wb-rum-api-full-${SOURCE_SHA:0:10}"

candidate_result="$(probe_tag "$candidate_tag" "$candidate_pod")"
full_sha_result="$(probe_tag "$full_sha_tag" "$full_sha_pod")"

candidate_image_id="${candidate_result%%$'\t'*}"
candidate_digest="${candidate_result##*$'\t'}"
full_sha_image_id="${full_sha_result%%$'\t'*}"
full_sha_digest="${full_sha_result##*$'\t'}"

if [ "$candidate_digest" != "$full_sha_digest" ]; then
  echo "ERROR: exact-head API aliases resolve to different immutable digests" >&2
  echo "candidate_digest=$candidate_digest" >&2
  echo "full_sha_digest=$full_sha_digest" >&2
  exit 8
fi

isolated_rum_after="$(namespace_deployment_images rum-dev-isolated)"
isolated_rat_after="$(namespace_deployment_images rum-rate-anything-preview)"
public_after="$(namespace_deployment_images rum-dev)"
live_after="$(namespace_deployment_images rum-prod)"

[ "$isolated_rum_before" = "$isolated_rum_after" ] || { echo "ERROR: isolated RUM deployment images changed during the probe" >&2; exit 9; }
[ "$isolated_rat_before" = "$isolated_rat_after" ] || { echo "ERROR: isolated RAT deployment images changed during the probe" >&2; exit 10; }
[ "$public_before" = "$public_after" ] || { echo "ERROR: public RUM deployment images changed during the probe" >&2; exit 11; }
[ "$live_before" = "$live_after" ] || { echo "ERROR: LIVE RUM deployment images changed during the probe" >&2; exit 12; }

printf 'RUM160_SOURCE_SHA=%s\n' "$SOURCE_SHA"
printf 'RUM160_API_CANDIDATE_TAG=%s\n' "$candidate_tag"
printf 'RUM160_API_CANDIDATE_IMAGE_ID=%s\n' "$candidate_image_id"
printf 'RUM160_API_FULL_SHA_TAG=%s\n' "$full_sha_tag"
printf 'RUM160_API_FULL_SHA_IMAGE_ID=%s\n' "$full_sha_image_id"
printf 'RUM160_API_DIGEST=%s\n' "$candidate_digest"
printf 'RUM160_API_IMMUTABLE_IMAGE=%s@%s\n' "$IMAGE_REPOSITORY" "$candidate_digest"
printf 'RUM160_API_ALIASES_MATCH=true\n'
printf 'RUM160_DEPLOYMENT_IMAGES_UNCHANGED=true\n'
printf 'RUM160_API_IMAGE_DIGEST_PROBE_PASS=true\n'
