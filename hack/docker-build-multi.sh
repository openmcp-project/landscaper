#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# Call with PLATFORMS="os/arch,os/arch" to specify target platforms, for example: linux/amd64,linux/arm64
# Call optionally with NO_LATEST_TAG=1 to skip tagging the images with "latest"

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

for pf in ${PLATFORMS//,/ }; do
  echo "> Building docker images for $pf in version $EFFECTIVE_VERSION ..."
  os=${pf%/*}
  arch=${pf#*/}
  for img in landscaper-controller landscaper-webhooks-server container-deployer-controller container-deployer-init container-deployer-wait helm-deployer-controller manifest-deployer-controller mock-deployer-controller; do
    tags="-t ${img}:${EFFECTIVE_VERSION}.${os}-${arch}"
    if [[ -z "${NO_LATEST_TAG:-}" ]]; then
      tags="$tags -t ${img}:latest"
    fi
    docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg EFFECTIVE_VERSION=${EFFECTIVE_VERSION} --platform ${pf} $tags -f Dockerfile --target ${img} "${PROJECT_ROOT}"
  done
done

docker buildx rm "$DOCKER_BUILDER_NAME"