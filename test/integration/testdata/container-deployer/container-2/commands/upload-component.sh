#!/bin/bash
#
# SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
#
# SPDX-License-Identifier: Apache-2.0

COMMAND_DIR="$(dirname $0)"
HACK_DIR="${COMMAND_DIR}/../../../hack"

source "${HACK_DIR}/settings"
"${HACK_DIR}/upload-component.sh" "${COMMAND_DIR}/component-constructor-1.yaml" "$REPO_BASE_URL_INTEGRATION_TESTS"
"${HACK_DIR}/upload-component.sh" "${COMMAND_DIR}/component-constructor-2.yaml" "$REPO_BASE_URL_INTEGRATION_TESTS"
