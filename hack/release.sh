#!/bin/bash

set -euo pipefail

PROJECT_ROOT="$(realpath $(dirname $0)/..)"

echo "Landscaper release: updating go.mod files"

# update go.mod's internal dependency to local module so that it can be used by other repositories
VERSION=$(cat ${PROJECT_ROOT}/VERSION)

# 0,/)/ only replaces the first ocurrence until the first dep block with ')' is reached
sed -i -e "0,/)/{s@github.com/openmcp-project/landscaper/apis .*@github.com/openmcp-project/landscaper/apis ${VERSION}@}" \
  ${PROJECT_ROOT}/go.mod

sed -i -e "0,/)/{s@github.com/openmcp-project/landscaper/controller-utils .*@github.com/openmcp-project/landscaper/controller-utils ${VERSION}@}" \
  ${PROJECT_ROOT}/go.mod

sed -i -e "0,/)/{s@github.com/openmcp-project/landscaper/apis .*@github.com/openmcp-project/landscaper/apis ${VERSION}@}" \
  ${PROJECT_ROOT}/controller-utils/go.mod

echo "Landscaper release: starting revendor"

(
  cd $PROJECT_ROOT
  make revendor
)

echo "Landscaper release: finished revendor"

# the helm chart versions need to be updated in the release step to reflect the change in the Git repository
${PROJECT_ROOT}/hack/update-helm-chart-version.sh

echo "Landscaper release: finished"
