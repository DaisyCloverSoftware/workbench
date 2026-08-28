#!/usr/bin/env bash
set -euo pipefail

EXPECTED_SHA="27a195f035234d52125cc40c1cca7d7faa0d2d12"
EXPECTED_BRANCH="sprint-1-rat-foundation-search-explorer-20260826"
SOURCE_REPO="https://github.com/DaisyCloverSoftware/rum.git"
IMAGE_TAG="ghcr.io/daisycloversoftware/rum-rate-anything:rat-s1-27a195f03523-wb"
APP_VERSION="rat-candidate+${EXPECTED_SHA}"

tmp="$(mktemp -d)"
cleanup() {
  if [[ -n "${cid:-}" ]]; then docker rm -f "$cid" >/dev/null 2>&1 || true; fi
  rm -rf "$tmp"
}
trap cleanup EXIT

repo="$tmp/rum"
echo "RAT_FRONTEND_BUILD_START"
echo "expected_sha=$EXPECTED_SHA"
echo "image_tag=$IMAGE_TAG"

git clone --quiet --filter=blob:none --no-checkout --single-branch --branch "$EXPECTED_BRANCH" "$SOURCE_REPO" "$repo"
remote_head="$(git -C "$repo" rev-parse "refs/remotes/origin/$EXPECTED_BRANCH")"
if [[ "$remote_head" != "$EXPECTED_SHA" ]]; then
  echo "ERROR: branch head moved: expected $EXPECTED_SHA got $remote_head" >&2
  exit 31
fi

git -C "$repo" checkout --quiet --detach "$EXPECTED_SHA"
actual_sha="$(git -C "$repo" rev-parse HEAD)"
[[ "$actual_sha" == "$EXPECTED_SHA" ]] || { echo "ERROR: checkout mismatch" >&2; exit 32; }
[[ -z "$(git -C "$repo" status --porcelain)" ]] || { echo "ERROR: checkout is dirty" >&2; exit 33; }

docker build --pull \
  --label "org.opencontainers.image.revision=$EXPECTED_SHA" \
  --build-arg "APP_VERSION=$APP_VERSION" \
  -t "$IMAGE_TAG" \
  "$repo/apps/rate-anything"

cid="$(docker create "$IMAGE_TAG")"
docker cp "$cid:/usr/share/nginx/html/VERSION" "$tmp/VERSION"
docker cp "$cid:/usr/share/nginx/html/brand/rat-pirate-mascot.webp" "$tmp/rat-pirate-mascot.webp"
version="$(tr -d '\r\n' < "$tmp/VERSION")"
[[ "$version" == "$APP_VERSION" ]] || { echo "ERROR: VERSION mismatch: $version" >&2; exit 34; }
magic="$(xxd -p -l 12 "$tmp/rat-pirate-mascot.webp" | tr -d '\n')"
if [[ "${magic:0:8}" != "52494646" || "${magic:16:8}" != "57454250" ]]; then
  echo "ERROR: mascot is not RIFF/WEBP; magic12=$magic" >&2
  exit 35
fi
mascot_sha="$(sha256sum "$tmp/rat-pirate-mascot.webp" | awk '{print $1}')"
mascot_mime="$(file -b --mime-type "$tmp/rat-pirate-mascot.webp")"
docker rm "$cid" >/dev/null
cid=""

push_output="$tmp/push.txt"
docker push "$IMAGE_TAG" | tee "$push_output"
digest="$(sed -nE 's/^.*digest: (sha256:[0-9a-f]{64}).*$/\1/p' "$push_output" | tail -n1)"
if [[ ! "$digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
  repo_digest="$(docker image inspect "$IMAGE_TAG" --format '{{join .RepoDigests "\n"}}' | grep '^ghcr.io/daisycloversoftware/rum-rate-anything@sha256:' | tail -n1 || true)"
  digest="${repo_digest##*@}"
fi
[[ "$digest" =~ ^sha256:[0-9a-f]{64}$ ]] || { echo "ERROR: immutable digest not resolved" >&2; exit 36; }

echo "RAT_FRONTEND_BUILD_COMPLETE"
echo "source_sha=$actual_sha"
echo "image_tag=$IMAGE_TAG"
echo "image_digest=$digest"
echo "image_ref=ghcr.io/daisycloversoftware/rum-rate-anything@$digest"
echo "version=$version"
echo "mascot_mime=$mascot_mime"
echo "mascot_sha256=$mascot_sha"
echo "mascot_magic12=$magic"
