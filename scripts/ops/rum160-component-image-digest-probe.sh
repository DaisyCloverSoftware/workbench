#!/usr/bin/env bash
set -euo pipefail
umask 077

SOURCE_SHA="${1:-}"
COMPONENT="${2:-}"

if ! [[ "$SOURCE_SHA" =~ ^[0-9a-f]{40}$ ]]; then
  echo "usage: $0 <40-char-source-sha> <api|web|rate-anything>" >&2
  exit 64
fi

case "$COMPONENT" in
  api)
    IMAGE_REPOSITORY="ghcr.io/daisycloversoftware/rum-api"
    TARGET_NS="rum-dev-isolated"
    TARGET_DEPLOYMENT="rum-api"
    TARGET_CONTAINER="php-fpm"
    PROBE_COMMAND='["php","-r","sleep(300);"]'
    ;;
  web)
    IMAGE_REPOSITORY="ghcr.io/daisycloversoftware/rum-web"
    TARGET_NS="rum-dev-isolated"
    TARGET_DEPLOYMENT="rum-web"
    TARGET_CONTAINER="web"
    PROBE_COMMAND='["sh","-c","sleep 300"]'
    ;;
  rate-anything)
    IMAGE_REPOSITORY="ghcr.io/daisycloversoftware/rum-rate-anything"
    TARGET_NS="rum-rate-anything-preview"
    TARGET_DEPLOYMENT="rum-rate-anything"
    TARGET_CONTAINER="rate-anything"
    PROBE_COMMAND='["sh","-c","sleep 300"]'
    ;;
  *)
    echo "usage: $0 <40-char-source-sha> <api|web|rate-anything>" >&2
    exit 64
    ;;
esac

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
kube() { "${KUBECTL[@]}" "$@"; }

for ns in rum-dev-isolated rum-rate-anything-preview rum-dev rum-prod; do
  kube get namespace "$ns" >/dev/null
done
kube -n "$TARGET_NS" get deployment "$TARGET_DEPLOYMENT" >/dev/null

namespace_images() {
  local ns="$1"
  kube -n "$ns" get deployments -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{range .spec.template.spec.containers[*]}{.name}{"="}{.image}{";"}{end}{"\n"}{end}' | LC_ALL=C sort
}

rum_before="$(namespace_images rum-dev-isolated)"
rat_before="$(namespace_images rum-rate-anything-preview)"
public_before="$(namespace_images rum-dev)"
live_before="$(namespace_images rum-prod)"

created=()
cleanup() {
  local pod
  for pod in "${created[@]:-}"; do
    [[ -n "$pod" ]] && kube -n "$TARGET_NS" delete pod "$pod" --ignore-not-found --wait=false >/dev/null 2>&1 || true
  done
}
trap cleanup EXIT HUP INT TERM

probe_alias() {
  local tag="$1" suffix="$2" pod image_id waiting phase digest source_json manifest existing_image existing_id
  pod="wb-rum160-${COMPONENT//[^a-z0-9]/-}-${suffix}-${SOURCE_SHA:0:8}"

  if kube -n "$TARGET_NS" get pod "$pod" >/dev/null 2>&1; then
    existing_image="$(kube -n "$TARGET_NS" get pod "$pod" -o jsonpath='{.spec.containers[0].image}')"
    existing_id="$(kube -n "$TARGET_NS" get pod "$pod" -o jsonpath='{.status.containerStatuses[0].imageID}' 2>/dev/null || true)"
    if [[ "$existing_image" == "$IMAGE_REPOSITORY:$tag" && "$existing_id" == *@sha256:* ]]; then
      digest="${existing_id##*@}"
      [[ "$digest" =~ ^sha256:[0-9a-f]{64}$ ]] || return 7
      printf '%s\t%s\n' "$existing_id" "$digest"
      return 0
    fi
    echo "ERROR: existing probe pod does not contain reusable exact alias evidence: $TARGET_NS/$pod" >&2
    return 4
  fi

  source_json="$(mktemp)"
  manifest="$(mktemp)"
  kube -n "$TARGET_NS" get deployment "$TARGET_DEPLOYMENT" -o json >"$source_json"
  python3 - "$source_json" "$manifest" "$pod" "$TARGET_NS" "$TARGET_CONTAINER" "$IMAGE_REPOSITORY:$tag" "$PROBE_COMMAND" <<'PY'
import copy,json,sys
src,out,pod_name,namespace,container_name,image,command_json=sys.argv[1:]
with open(src,encoding='utf-8') as fh:
    deployment=json.load(fh)
template=deployment['spec']['template']['spec']
source=next(c for c in template.get('containers',[]) if c.get('name')==container_name)
pod_spec={
    'restartPolicy':'Never',
    'automountServiceAccountToken':False,
    'containers':[{
        'name':'probe',
        'image':image,
        'imagePullPolicy':'Always',
        'command':json.loads(command_json),
    }],
}
for key in ('imagePullSecrets','nodeSelector','tolerations','affinity','securityContext','runtimeClassName'):
    if key in template:
        pod_spec[key]=copy.deepcopy(template[key])
if 'securityContext' in source:
    pod_spec['containers'][0]['securityContext']=copy.deepcopy(source['securityContext'])
pod={
    'apiVersion':'v1','kind':'Pod',
    'metadata':{'name':pod_name,'namespace':namespace,'labels':{
        'app.kubernetes.io/name':'rum',
        'app.kubernetes.io/component':'rum160-image-probe',
        'app.kubernetes.io/managed-by':'workbench',
    }},
    'spec':pod_spec,
}
with open(out,'w',encoding='utf-8') as fh:
    json.dump(pod,fh)
PY
  rm -f "$source_json"
  kube create -f "$manifest" >/dev/null
  rm -f "$manifest"
  created+=("$pod")

  image_id=""
  for ((attempt=0; attempt<90; attempt++)); do
    image_id="$(kube -n "$TARGET_NS" get pod "$pod" -o jsonpath='{.status.containerStatuses[0].imageID}' 2>/dev/null || true)"
    [[ "$image_id" == *@sha256:* ]] && break
    waiting="$(kube -n "$TARGET_NS" get pod "$pod" -o jsonpath='{.status.containerStatuses[0].state.waiting.reason}' 2>/dev/null || true)"
    phase="$(kube -n "$TARGET_NS" get pod "$pod" -o jsonpath='{.status.phase}' 2>/dev/null || true)"
    case "$waiting" in
      ErrImagePull|ImagePullBackOff|InvalidImageName)
        echo "ERROR: image probe failed for $IMAGE_REPOSITORY:$tag: $waiting" >&2
        kube -n "$TARGET_NS" describe pod "$pod" >&2 || true
        return 5
        ;;
    esac
    [[ "$phase" == Failed ]] && { echo "ERROR: probe pod failed before imageID was observed" >&2; return 6; }
    sleep 2
  done
  digest="${image_id##*@}"
  [[ "$digest" =~ ^sha256:[0-9a-f]{64}$ ]] || { echo "ERROR: unexpected imageID: $image_id" >&2; return 7; }
  kube -n "$TARGET_NS" delete pod "$pod" --wait=true --timeout=30s >/dev/null
  printf '%s\t%s\n' "$image_id" "$digest"
}

candidate_tag="rat-candidate-${SOURCE_SHA:0:12}"
full_tag="sha-$SOURCE_SHA"
candidate="$(probe_alias "$candidate_tag" cand)"
full="$(probe_alias "$full_tag" full)"
candidate_id="${candidate%%$'\t'*}"
candidate_digest="${candidate##*$'\t'}"
full_id="${full%%$'\t'*}"
full_digest="${full##*$'\t'}"
[[ "$candidate_digest" == "$full_digest" ]] || { echo "ERROR: exact-head aliases resolve to different platform manifests" >&2; exit 8; }

[[ "$rum_before" == "$(namespace_images rum-dev-isolated)" ]] || { echo "ERROR: isolated RUM deployment images changed during probe" >&2; exit 9; }
[[ "$rat_before" == "$(namespace_images rum-rate-anything-preview)" ]] || { echo "ERROR: isolated RAT deployment images changed during probe" >&2; exit 10; }
[[ "$public_before" == "$(namespace_images rum-dev)" ]] || { echo "ERROR: public deployment images changed during probe" >&2; exit 11; }
[[ "$live_before" == "$(namespace_images rum-prod)" ]] || { echo "ERROR: LIVE deployment images changed during probe" >&2; exit 12; }

printf 'RUM160_SOURCE_SHA=%s\n' "$SOURCE_SHA"
printf 'RUM160_COMPONENT=%s\n' "$COMPONENT"
printf 'RUM160_CANDIDATE_TAG=%s:%s\n' "$IMAGE_REPOSITORY" "$candidate_tag"
printf 'RUM160_CANDIDATE_IMAGE_ID=%s\n' "$candidate_id"
printf 'RUM160_FULL_SHA_IMAGE_ID=%s\n' "$full_id"
printf 'RUM160_PLATFORM_DIGEST=%s\n' "$candidate_digest"
printf 'RUM160_IMMUTABLE_IMAGE=%s@%s\n' "$IMAGE_REPOSITORY" "$candidate_digest"
printf 'RUM160_ALIASES_MATCH=true\n'
printf 'RUM160_DEPLOYMENT_IMAGES_UNCHANGED=true\n'
printf 'RUM160_COMPONENT_IMAGE_DIGEST_PROBE_PASS=true\n'
