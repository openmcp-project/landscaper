#!/bin/bash
#
# SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit

COMPONENT_DIR="$(dirname $0)/.."
cd "${COMPONENT_DIR}"
COMPONENT_DIR="$(pwd)"
echo "COMPONENT_DIR: ${COMPONENT_DIR}"

source "${COMPONENT_DIR}/commands/settings"

echo "deleting dataobject my-release"
kubectl delete dataobject "my-release" -n "${NAMESPACE}"

echo "deleting dataobject my-values"
kubectl delete dataobject "my-values" -n "${NAMESPACE}"

echo "deleting context"
kubectl delete context "landscaper-examples" -n "${NAMESPACE}"

echo "deleting target"
kubectl delete target "my-cluster" -n "${NAMESPACE}"
