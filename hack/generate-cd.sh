#!/bin/bash
#
# SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
#
# SPDX-License-Identifier: Apache-2.0

# Builds and publishes the component versions of landscaper and its deployers.
#
# Environment:
#   OCM                   OCM CLI binary
#   OVERWRITE_COMPONENTS  "true" replaces existing component versions

set -euo pipefail

die() { echo "$@" >&2; exit 1; }

if [[ -z ${1:-} ]]; then
  die "provider argument is required"
fi

PROJECT_ROOT="$(realpath "$(dirname "$0")/..")"
TARGET="$1"
COMPONENTS=(
  github.com/openmcp-project/landscaper
  github.com/openmcp-project/landscaper/helm-deployer
  github.com/openmcp-project/landscaper/manifest-deployer
  github.com/openmcp-project/landscaper/container-deployer
  github.com/openmcp-project/landscaper/mock-deployer
)

export VERSION="$("$PROJECT_ROOT/hack/get-version.sh")"
export PROVIDER="${PROVIDER:-$TARGET}"
export IMAGE_REGISTRY="${IMAGE_REGISTRY:-ghcr.io/openmcp-project/images}"
export HELM_DEPLOYER_CHART_PATH=./charts/helm-deployer
export MANIFEST_DEPLOYER_CHART_PATH=./charts/manifest-deployer
export CONTAINER_DEPLOYER_CHART_PATH=./charts/container-deployer
export MOCK_DEPLOYER_CHART_PATH=./charts/mock-deployer
export LANDSCAPER_CHART_PATH=./charts/landscaper
export LANDSCAPER_CONTROLLER_RBAC_CHART_PATH=./charts/landscaper/charts/rbac
export LANDSCAPER_CONTROLLER_DEPLOYMENT_CHART_PATH=./charts/landscaper/charts/landscaper

# ocm replaces existing versions and has no flag to prevent it.
for component in "${COMPONENTS[@]}"; do
  target="$TARGET//$component:$VERSION"
  if output="$("$OCM" get cv "$target" 2>&1)"; then
    [[ "${OVERWRITE_COMPONENTS:-false}" == "true" ]] || die "$target already exists. Set OVERWRITE_COMPONENTS to 'true' to overwrite it."
  elif [[ "$output" != *"not found"* ]]; then
    echo "$output" >&2
    die "Cannot check whether $target exists."
  fi
done

echo -n "> Updating helm chart version"
"$PROJECT_ROOT/hack/update-helm-chart-version.sh"

echo "> Create Component Version ${VERSION}"

COMPONENT_ARCHIVE_PATH="$(mktemp -d)/ctf"

"$OCM" add cv \
  --constructor "$PROJECT_ROOT/.landscaper/components.yaml" \
  --working-directory "$PROJECT_ROOT" \
  --repository "ctf::$COMPONENT_ARCHIVE_PATH"

echo "> Transfer Component version ${VERSION} to ${TARGET}"
for component in "${COMPONENTS[@]}"; do
  "$OCM" transfer cv "ctf::$COMPONENT_ARCHIVE_PATH//$component:$VERSION" "$TARGET"
done

echo "> Remote Component Version Landscaper"
"$OCM" get cv "$TARGET//github.com/openmcp-project/landscaper:$VERSION" -o yaml
