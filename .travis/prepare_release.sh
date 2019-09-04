#!/bin/bash
BASEPATH=$(dirname $(dirname $0))

# Split the IMAGE variable into repository and tag parts
# e.g. kabanero/kabanero-operator:0.1.1 -> kabanero/kabanero-operator
IFS=’:’ read -ra REPOSITORY <<< "$IMAGE"

# Levearge the gen_operator_deployment.sh script to generate deploy/kabanero-operators.yaml
$BASEPATH/contrib/gen_operator_deployment.sh $REPOSITORY:$TRAVIS_TAG