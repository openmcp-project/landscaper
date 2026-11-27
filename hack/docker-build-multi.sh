#!/bin/bash
#
# SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# Call with PLATFORMS="os/arch,os/arch" to specify target platforms, for example: linux/amd64,linux/arm64
# Call optionally with NO_LATEST_TAG=1 to skip tagging the images with "latest"
# Call with PUSH_IMAGES=true to push one multi-platform image per controller

echo "PLATFORMS=${PLATFORMS}"
echo "NO_LATEST_TAG=${NO_LATEST_TAG:-not set}"

PROJECT_ROOT="$(realpath $(dirname $0)/..)"
if [[ -z ${EFFECTIVE_VERSION:-} ]]; then
  EFFECTIVE_VERSION=$("$PROJECT_ROOT/hack/get-version.sh")
fi

DOCKER_BUILDER_NAME="ls-multiarch-builder"
if ! docker buildx ls | grep "$DOCKER_BUILDER_NAME" >/dev/null; then
 docker buildx create --name "$DOCKER_BUILDER_NAME"
fi

if [[ "${PUSH_IMAGES:-false}" == "true" ]]; then
  # Local builds must not push, so this is opt-in.
  for img in landscaper-controller landscaper-webhooks-server container-deployer-controller container-deployer-init container-deployer-wait helm-deployer-controller manifest-deployer-controller mock-deployer-controller; do
    ref="${IMAGE_REGISTRY:-ghcr.io/openmcp-project/images}/${img}:${EFFECTIVE_VERSION}"
    if [[ "${OVERWRITE_COMPONENTS:-false}" != "true" ]] && docker buildx imagetools inspect "$ref" >/dev/null 2>&1; then
      echo "$ref already exists. Set OVERWRITE_COMPONENTS to 'true' to overwrite it." >&2
      exit 1
    fi
    docker buildx build --builder ${DOCKER_BUILDER_NAME} --push --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${PLATFORMS} -t "$ref" -f Dockerfile --target ${img} "${PROJECT_ROOT}"
  done
  docker buildx rm "$DOCKER_BUILDER_NAME"
  exit 0
fi

for pf in ${PLATFORMS//,/ }; do
  echo "> Building docker images for $pf in version $EFFECTIVE_VERSION ..."
  os=${pf%/*}
  arch=${pf#*/}
  for img in landscaper-controller landscaper-webhooks-server container-deployer-controller container-deployer-init container-deployer-wait helm-deployer-controller manifest-deployer-controller mock-deployer-controller; do
    tags="-t ${img}:${EFFECTIVE_VERSION}-${os}-${arch}"
    if [[ -z "${NO_LATEST_TAG:-}" ]]; then
      tags="$tags -t ${img}:latest"
    fi
    docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${pf} $tags -f Dockerfile --target ${img} "${PROJECT_ROOT}"
  done
done

docker buildx rm "$DOCKER_BUILDER_NAME"
