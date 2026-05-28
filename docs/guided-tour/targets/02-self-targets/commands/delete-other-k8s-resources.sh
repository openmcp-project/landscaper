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

echo "deleting target"
kubectl delete target "self-target" -n "${NAMESPACE}" --kubeconfig="${RESOURCE_CLUSTER_KUBECONFIG_PATH}"

echo "deleting clusterrolebinding"
kubectl delete clusterrolebinding "landscaper:guided-tour:self" --kubeconfig="${RESOURCE_CLUSTER_KUBECONFIG_PATH}"

echo "deleting serviceaccount"
kubectl delete serviceaccount "self-serviceaccount" -n "${NAMESPACE}" --kubeconfig="${RESOURCE_CLUSTER_KUBECONFIG_PATH}"
