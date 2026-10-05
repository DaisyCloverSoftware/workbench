#!/usr/bin/env bash
set -euo pipefail

source_sha="${1:-}"
baseline_sha="${2:-}"
if ! [[ "$source_sha" =~ ^[0-9a-f]{40}$ ]] || ! [[ "$baseline_sha" =~ ^[0-9a-f]{40}$ ]]; then
  echo "Usage: $0 <exact-source-sha> <approved-dev-baseline-sha>" >&2
  exit 2
fi

repo='ghcr.io/daisycloversoftware/simlab-admin-academy'
tag="$repo:dev-$source_sha"
engine="${SIMLAB_OCI_ENGINE:-}"
if [ -z "$engine" ]; then
  for candidate in podman docker; do
    if command -v "$candidate" >/dev/null 2>&1; then engine="$candidate"; break; fi
  done
fi
case "$(basename "$engine")" in podman|docker) ;; *) echo 'Podman or Docker required' >&2; exit 3;; esac
command -v skopeo >/dev/null 2>&1 || { echo 'skopeo unavailable' >&2; exit 4; }

inspection="$("$engine" image inspect "$tag")" || { echo "LOCAL_EXACT_SOURCE_IMAGE_PRESENT=false" >&2; exit 5; }
node -e '
const rows=JSON.parse(process.argv[1]);
const image=Array.isArray(rows)?rows[0]:rows;
const cfg=image&&(image.Config||image.config)||{};
const labels=cfg.Labels||cfg.labels||{};
if(labels["org.opencontainers.image.revision"]!==process.argv[2]) throw new Error("source provenance label mismatch");
if(labels["io.daisyclover.simlab.baseline-source-sha"]!==process.argv[3]) throw new Error("DEV baseline provenance label mismatch");
if(labels["io.daisyclover.simlab.lineage-verified"]!=="true") throw new Error("lineage verification label missing");
if(cfg.User!=="10001:10001") throw new Error("runtime user mismatch");
const hc=cfg.Healthcheck||cfg.healthcheck||image.Healthcheck||image.healthcheck;
if(!hc||!JSON.stringify(hc).includes("api/health")) throw new Error("healthcheck metadata mismatch");
' "$inspection" "$source_sha" "$baseline_sha"

"$engine" push "$tag" >/dev/null
digest="$(skopeo inspect --format '{{.Digest}}' "docker://$tag" | tr -d '\r\n ')"
if ! [[ "$digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
  echo "Registry did not return a valid immutable digest" >&2
  exit 6
fi
immutable="$repo@$digest"
"$engine" pull "$immutable" >/dev/null

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
if [ "$(basename "$engine")" = 'podman' ]; then
  "$engine" save --format docker-archive --output "$tmp/simlab.tar" "$immutable"
else
  "$engine" save --output "$tmp/simlab.tar" "$immutable"
fi
GRYPE_IMAGE='anchore/grype@sha256:fd4ab4d1042b522c896e73bdf09ab8bf384fa417df99d6dd0d6e1008c7e7c821'
"$engine" run --rm -v "$tmp:/scan" "$GRYPE_IMAGE" docker-archive:/scan/simlab.tar --fail-on critical --output json --file /scan/grype.json >/dev/null

echo "LOCAL_EXACT_SOURCE_IMAGE_PRESENT=true"
echo "SOURCE_SHA=$source_sha"
echo "DEV_BASELINE_SOURCE_SHA=$baseline_sha"
echo "IMAGE_DIGEST=$digest"
echo "IMAGE_REF=$immutable"
echo "IMAGE_USER=10001:10001"
echo "IMAGE_HEALTHCHECK=PASS"
echo "EXACT_BUILT_IMAGE_CRITICAL_VULNERABILITY_GATE=PASS"
echo "SIMLAB_DEV_IMAGE_SALVAGE_PASS=true"
