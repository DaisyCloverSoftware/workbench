#!/usr/bin/env bash
set -euo pipefail

# Narrow, review-gated infrastructure operation for RAT PR170 DEV acceptance.
# No application image, config, mail transport, datastore, LIVE, or ingress changes.
# HTTP/API listens INSIDE the capture pod's loopback only. No UI/HTTP Service.
# Usage: install <reviewed immutable image> | remove <same immutable image>
[[ $# -eq 2 ]] || { echo 'usage: rat170-isolated-dev-mailcapture.sh install|remove axllent/mailpit@sha256:<64hex>' >&2; exit 64; }
mode="$1"
image="$2"
[[ "$mode" == 'install' || "$mode" == 'remove' ]] || exit 64
[[ "$image" =~ ^axllent/mailpit@sha256:[0-9a-f]{64}$ ]] || { echo 'requires official immutable Mailpit digest' >&2; exit 64; }

dev='rum-dev-isolated'
rat='rum-rate-anything-preview'
live='rum-dev'
name='rat170-mailcapture'
api_expected='ghcr.io/daisycloversoftware/rum-api@sha256:e0f3de7f64cc0e54d7a0a812b47eccfbd9f2c8c97f623ac2aa72749665be9983'
rat_expected='ghcr.io/daisycloversoftware/rum-rate-anything@sha256:3860d5418e2a20929ef3d816e1cd4e783a8bf26cdcd1a59f1a6fd682e3bee63e'
live_api_expected='ghcr.io/daisycloversoftware/rum-api:sha-54d31b8'
live_web_expected='ghcr.io/daisycloversoftware/rum-web:sha-8106675'
kctl(){ if command -v k3s >/dev/null 2>&1; then sudo -n k3s kubectl "$@"; else kubectl "$@"; fi; }
image_of(){ kctl -n "$1" get deployment "$2" -o "jsonpath={.spec.template.spec.containers[?(@.name==\"$3\")].image}"; }
hosts(){ kctl -n "$1" get ingress -o jsonpath='{range .items[*].spec.rules[*]}{.host}{"\n"}{end}' | sort -u; }
for ns in "$dev" "$rat" "$live"; do kctl get namespace "$ns" >/dev/null; done
[[ "$(hosts "$dev")" == 'dev-rum.daisycloversoftware.uk' ]] || { echo 'DEV host guard failed' >&2; exit 78; }
[[ "$(hosts "$rat")" == 'dev-rum-ra.daisycloversoftware.uk' ]] || { echo 'RAT host guard failed' >&2; exit 78; }
[[ "$(hosts "$live")" == 'rateurmate.online' ]] || { echo 'LIVE host guard failed' >&2; exit 78; }
[[ "$(image_of "$dev" rum-api php-fpm)" == "$api_expected" ]] || { echo 'DEV API candidate changed' >&2; exit 78; }
[[ "$(image_of "$rat" rum-rate-anything rate-anything)" == "$rat_expected" ]] || { echo 'RAT frontend candidate changed' >&2; exit 78; }
[[ "$(image_of "$live" rum-api php-fpm)" == "$live_api_expected" ]] || { echo 'LIVE API changed' >&2; exit 78; }
[[ "$(image_of "$live" rum-web web)" == "$live_web_expected" ]] || { echo 'LIVE web changed' >&2; exit 78; }
[[ "$(image_of "$live" rum-worker worker)" == "$live_api_expected" ]] || { echo 'LIVE worker changed' >&2; exit 78; }

# Only these exact three named resources may ever be touched.
objects(){ kctl -n "$dev" get deployment,service,networkpolicy -o name | grep -E '(^deployment.apps/rat170-mailcapture$|^service/rat170-mailcapture$|^networkpolicy.networking.k8s.io/rat170-mailcapture-ingress$)' || true; }
count_objects(){ objects | wc -l | tr -d ' '; }
if [[ "$mode" == 'remove' ]]; then
  # Never tear down resources if a different image has taken over this named deployment.
  current="$(kctl -n "$dev" get deployment "$name" -o 'jsonpath={.spec.template.spec.containers[0].image}' --ignore-not-found)"
  [[ -z "$current" || "$current" == "$image" ]] || { echo 'capture deployment superseded; refusing deletion' >&2; exit 78; }
  if [[ -n "$current" ]]; then
    owner="$(kctl -n "$dev" get deployment "$name" -o 'jsonpath={.metadata.labels.rat170\.mailcapture/managed}')"
    [[ "$owner" == 'controlled' ]] || { echo 'unexpected capture owner; refusing deletion' >&2; exit 78; }
  fi
  kctl -n "$dev" delete deployment "$name" --ignore-not-found --wait=true >/dev/null
  kctl -n "$dev" delete service "$name" --ignore-not-found --wait=true >/dev/null
  kctl -n "$dev" delete networkpolicy rat170-mailcapture-ingress --ignore-not-found --wait=true >/dev/null
  [[ "$(count_objects)" == '0' ]] || { echo 'capture removal incomplete' >&2; exit 1; }
  echo 'RAT170_DEV_MAILCAPTURE_REMOVED=true'
  exit 0
fi

[[ "$(count_objects)" == '0' ]] || { echo 'capture resources already exist; inspect before any retry' >&2; exit 78; }

# Apply the NetworkPolicy BEFORE starting a pod or exposing its SMTP Service.
# Mailpit has no relay/forwarding configuration, no persistent volume and no external egress.
# Only the exact RUM DEV API pod is allowed into SMTP port 1025.
# The HTTP/UI/API listener is bound to 127.0.0.1:8025 INSIDE the container.
# Only authorised kubectl exec/loopback access can read a captured message.
changed=false
rollback(){
  local code=$?
  trap - ERR
  if [[ "$changed" == true ]]; then
    kctl -n "$dev" delete deployment "$name" --ignore-not-found --wait=true >/dev/null 2>&1 || true
    kctl -n "$dev" delete service "$name" --ignore-not-found --wait=true >/dev/null 2>&1 || true
    kctl -n "$dev" delete networkpolicy rat170-mailcapture-ingress --ignore-not-found --wait=true >/dev/null 2>&1 || true
  fi
  echo 'RAT170_MAILCAPTURE_INSTALL_FAILED; cleanup attempted' >&2
  exit "$code"
}
trap rollback ERR
changed=true
kctl -n "$dev" apply -f - >/dev/null <<'POLICY'
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: rat170-mailcapture-ingress
  labels:
    rat170.mailcapture/managed: controlled
spec:
  podSelector:
    matchLabels:
      rat170.mailcapture/pod: controlled
  policyTypes: [Ingress, Egress]
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app.kubernetes.io/name: rum
          app.kubernetes.io/instance: rum
          app.kubernetes.io/component: api
    ports:
    - protocol: TCP
      port: 1025
  egress: []
POLICY

kctl -n "$dev" apply -f - >/dev/null <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: $name
  labels:
    rat170.mailcapture/managed: controlled
spec:
  replicas: 1
  revisionHistoryLimit: 1
  selector:
    matchLabels:
      rat170.mailcapture/pod: controlled
  template:
    metadata:
      labels:
        rat170.mailcapture/pod: controlled
        rum.internal/access: allowed
    spec:
      terminationGracePeriodSeconds: 10
      automountServiceAccountToken: false
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        runAsGroup: 1000
        seccompProfile:
          type: RuntimeDefault
      containers:
      - name: mailpit
        image: $image
        imagePullPolicy: IfNotPresent
        env:
        - name: MP_UI_BIND_ADDR
          value: '127.0.0.1:8025'
        - name: MP_SMTP_BIND_ADDR
          value: '0.0.0.0:1025'
        - name: MP_ALLOWED_HOSTS
          value: 'localhost,127.0.0.1'
        - name: MP_SMTP_ALLOWED_RECIPIENTS
          value: '^rat169qa[a-z0-9]{6,24}@example[.]com$'
        - name: MP_SMTP_DISABLE_RDNS
          value: 'true'
        - name: MP_DISABLE_VERSION_CHECK
          value: 'true'
        - name: MP_MAX_MESSAGES
          value: '10'
        - name: MP_MAX_AGE
          value: '1h'
        - name: MP_DATABASE
          value: '/tmp/mailpit.db'
        - name: MP_QUIET
          value: 'true'
        ports:
        - name: smtp
          containerPort: 1025
        - name: local-ui
          containerPort: 8025
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop: [ALL]
        resources:
          requests:
            cpu: 10m
            memory: 32Mi
          limits:
            cpu: 250m
            memory: 128Mi
        volumeMounts:
        - name: volatile-mail
          mountPath: /tmp
      volumes:
      - name: volatile-mail
        emptyDir:
          medium: Memory
          sizeLimit: 32Mi
---
apiVersion: v1
kind: Service
metadata:
  name: $name
  labels:
    rat170.mailcapture/managed: controlled
spec:
  type: ClusterIP
  selector:
    rat170.mailcapture/pod: controlled
  ports:
  - name: smtp
    port: 1025
    protocol: TCP
    targetPort: smtp
EOF
kctl -n "$dev" rollout status deployment/"$name" --timeout=240s >/dev/null
[[ "$(kctl -n "$dev" get deployment "$name" -o 'jsonpath={.spec.template.spec.containers[0].image}')" == "$image" ]]
[[ "$(count_objects)" == '3' ]]
# No access to the HTTP port through a Service or Ingress.
[[ "$(kctl -n "$dev" get service "$name" -o 'jsonpath={.spec.ports[*].port}')" == '1025' ]]
[[ "$(hosts "$dev")" == 'dev-rum.daisycloversoftware.uk' ]]
[[ "$(image_of "$dev" rum-api php-fpm)" == "$api_expected" ]]
[[ "$(image_of "$rat" rum-rate-anything rate-anything)" == "$rat_expected" ]]
[[ "$(image_of "$live" rum-api php-fpm)" == "$live_api_expected" ]]
[[ "$(image_of "$live" rum-web web)" == "$live_web_expected" ]]
trap - ERR
printf 'RAT170_DEV_MAILCAPTURE_INSTALLED=true\nimage=%s\nnamespace=%s\nsmtp_cluster_only=true\nhttp_loopback_only=true\nallowlisted_synthetic_recipients=true\nno_application_deployment_changed=true\nlive_unchanged=true\n' "$image" "$dev"
