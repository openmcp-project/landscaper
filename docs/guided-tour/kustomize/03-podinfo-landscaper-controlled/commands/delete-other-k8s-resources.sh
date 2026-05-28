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

echo "deleting context"
kubectl delete context "landscaper-examples" -n "${NAMESPACE}"

echo "deleting targets"
kubectl delete target "my-cluster" -n "${NAMESPACE}"
kubectl delete target "my-cluster-2" -n "${NAMESPACE}"
