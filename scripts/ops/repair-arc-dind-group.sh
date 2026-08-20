#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <namespace> <release>" >&2
  exit 2
fi

NS="$1"
RELEASE="$2"
ARC_VERSION="0.14.2"
KUBECONFIG_PATH="/etc/rancher/k3s/k3s.yaml"
VALUES="$(mktemp)"
trap 'rm -f "$VALUES"' EXIT

sudo -n env KUBECONFIG="$KUBECONFIG_PATH" helm get values "$RELEASE" -n "$NS" -o yaml >"$VALUES"
if grep -q -- '--group=$' "$VALUES"; then
  sed -i 's#--group=$#--group=$(DOCKER_GROUP_GID)#' "$VALUES"
fi
grep -qF -- '--group=$(DOCKER_GROUP_GID)' "$VALUES" || { echo "expected DinD group argument not found" >&2; exit 1; }

sudo -n env KUBECONFIG="$KUBECONFIG_PATH" helm upgrade "$RELEASE" \
  --namespace "$NS" \
  --version "$ARC_VERSION" \
  -f "$VALUES" \
  oci://ghcr.io/actions/actions-runner-controller-charts/gha-runner-scale-set \
  --wait --timeout 5m

sudo -n k3s kubectl get autoscalingrunnersets.actions.github.com -n "$NS" -o wide
