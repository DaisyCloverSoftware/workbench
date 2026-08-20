#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 4 ]]; then
  echo "usage: $0 <node> <namespace> <scale-set-name> <source-secret-namespace>" >&2
  exit 2
fi

NODE="$1"
NS="$2"
SET="$3"
SOURCE_NS="$4"
ARC_VERSION="0.14.2"
RUNNER_VERSION="2.336.0"
DOCKER_CLI_VERSION="28.3.3"
K=(sudo -n k3s kubectl)
H=(sudo -n env KUBECONFIG=/etc/rancher/k3s/k3s.yaml helm)

for x in sudo k3s helm jq mktemp; do command -v "$x" >/dev/null || { echo "missing $x" >&2; exit 1; }; done
sudo -n true

ready="$("${K[@]}" get node "$NODE" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}')"
[[ "$ready" == "True" ]] || { echo "$NODE is not Ready" >&2; exit 1; }

"${K[@]}" label node "$NODE" daisyclover.io/ci=true daisyclover.io/workload=ci --overwrite >/dev/null
"${K[@]}" taint node "$NODE" daisyclover.io/ci-only=true:NoSchedule --overwrite >/dev/null

if "${K[@]}" -n longhorn-system get nodes.longhorn.io "$NODE" >/dev/null 2>&1; then
  "${K[@]}" -n longhorn-system patch nodes.longhorn.io "$NODE" --type=merge -p '{"spec":{"allowScheduling":false}}' >/dev/null
fi

"${K[@]}" create namespace "$NS" --dry-run=client -o yaml | "${K[@]}" apply -f - >/dev/null
if ! "${K[@]}" -n "$NS" get secret arc-github-config >/dev/null 2>&1; then
  "${K[@]}" -n "$SOURCE_NS" get secret arc-github-config -o json \
    | jq --arg ns "$NS" '.metadata.namespace=$ns | del(.metadata.resourceVersion,.metadata.uid,.metadata.creationTimestamp,.metadata.managedFields)' \
    | "${K[@]}" apply -f - >/dev/null
fi

VALUES="$(mktemp)"
trap 'rm -f "$VALUES"' EXIT
cat >"$VALUES" <<EOF
githubConfigUrl: "https://github.com/DaisyCloverSoftware"
githubConfigSecret: arc-github-config
runnerGroup: daisyclover-private-ci
runnerScaleSetName: ${SET}
scaleSetLabels:
  - self-hosted
  - linux
  - x64
  - arc
  - ${NODE}
minRunners: 0
maxRunners: 1
template:
  spec:
    nodeSelector:
      kubernetes.io/hostname: ${NODE}
    tolerations:
      - key: daisyclover.io/ci-only
        operator: Equal
        value: "true"
        effect: NoSchedule
    terminationGracePeriodSeconds: 30
    initContainers:
      - name: init-dind-externals
        image: ghcr.io/actions/actions-runner:${RUNNER_VERSION}
        command: ["cp", "-r", "/home/runner/externals/.", "/home/runner/tmpDir/"]
        resources:
          requests: {cpu: 25m, memory: 64Mi}
          limits: {cpu: 250m, memory: 256Mi}
        volumeMounts:
          - {name: dind-externals, mountPath: /home/runner/tmpDir}
      - name: init-docker-cli-plugins
        image: docker:${DOCKER_CLI_VERSION}-cli
        command: ["sh", "-ec"]
        args:
          - |
            cp -a /usr/local/libexec/docker/cli-plugins/. /plugins/
            test -x /plugins/docker-compose
            test -x /plugins/docker-buildx
        resources:
          requests: {cpu: 25m, memory: 64Mi}
          limits: {cpu: 100m, memory: 128Mi}
        volumeMounts:
          - {name: docker-cli-plugins, mountPath: /plugins}
      - name: dind
        image: docker:${DOCKER_CLI_VERSION}-dind
        args: [dockerd, --host=unix:///var/run/docker.sock, '--group=$(DOCKER_GROUP_GID)']
        env:
          - {name: DOCKER_GROUP_GID, value: "123"}
        securityContext: {privileged: true}
        restartPolicy: Always
        startupProbe:
          exec: {command: [docker, info]}
          timeoutSeconds: 5
          failureThreshold: 24
          periodSeconds: 5
        resources:
          requests: {cpu: 250m, memory: 512Mi}
          limits: {cpu: "1", memory: 2Gi}
        volumeMounts:
          - {name: work, mountPath: /home/runner/_work}
          - {name: dind-sock, mountPath: /var/run}
          - {name: dind-externals, mountPath: /home/runner/externals}
    containers:
      - name: runner
        image: ghcr.io/actions/actions-runner:${RUNNER_VERSION}
        command: ["/home/runner/run.sh"]
        env:
          - {name: DOCKER_HOST, value: unix:///var/run/docker.sock}
          - {name: RUNNER_WAIT_FOR_DOCKER_IN_SECONDS, value: "120"}
        resources:
          requests: {cpu: 250m, memory: 512Mi}
          limits: {cpu: "1", memory: 2Gi}
        volumeMounts:
          - {name: work, mountPath: /home/runner/_work}
          - {name: dind-sock, mountPath: /var/run}
          - {name: docker-cli-plugins, mountPath: /usr/local/libexec/docker/cli-plugins, readOnly: true}
    volumes:
      - name: work
        emptyDir: {sizeLimit: 12Gi}
      - {name: dind-sock, emptyDir: {}}
      - {name: dind-externals, emptyDir: {}}
      - {name: docker-cli-plugins, emptyDir: {}}
EOF

"${H[@]}" upgrade --install "$SET" --namespace "$NS" --version "$ARC_VERSION" -f "$VALUES" oci://ghcr.io/actions/actions-runner-controller-charts/gha-runner-scale-set --wait --timeout 5m

"${K[@]}" get node "$NODE" -o wide
"${K[@]}" get node "$NODE" --show-labels
"${K[@]}" get autoscalingrunnersets.actions.github.com -n "$NS" -o wide
"${K[@]}" get pods -n "$NS" -o wide
