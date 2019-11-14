#!/bin/bash

set -Eeox pipefail

RELEASE="${RELEASE:-0.3.0}"
KABANERO_SUBSCRIPTIONS_YAML="${KABANERO_SUBSCRIPTIONS_YAML:-https://github.com/kabanero-io/kabanero-operator/releases/download/$RELEASE/kabanero-subscriptions.yaml}"
KABANERO_CUSTOMRESOURCES_YAML="${KABANERO_CUSTOMRESOURCES_YAML:-https://raw.githubusercontent.com/kabanero-io/kabanero-operator/pipeline-sa/deploy/kabanero-customresources.yaml}"
SLEEP_LONG="${SLEEP_LONG:-5}"
SLEEP_SHORT="${SLEEP_SHORT:-2}"

# Optional components (yes/no)
ENABLE_KAPPNAV="${ENABLE_KAPPNAV:-no}"

# Create service account to used by pipelines
oc apply -f $KABANERO_CUSTOMRESOURCES_YAML --selector 24-pipelines-sa
