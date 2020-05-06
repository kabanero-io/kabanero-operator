#!/bin/bash

set -Eeox pipefail

KABANERO_SUBSCRIPTIONS_YAML=https://raw.githubusercontent.com/kabanero-io/kabanero-operator/master/deploy/kabanero-subscriptions.yaml \
KABANERO_CUSTOMRESOURCES_YAML=https://raw.githubusercontent.com/kabanero-io/kabanero-operator/master/deploy/kabanero-customresources.yaml \
SAMPLE_KAB_INSTANCE_URL=https://raw.githubusercontent.com/kabanero-io/kabanero-operator/master/config/samples/default.yaml \
./install.sh