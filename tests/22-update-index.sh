#!/bin/bash

# Update the kabanero index URL and ensure the URL is updated for the stack

set -Eeox pipefail

namespace=kabanero

ORIGYAML=$(oc get -n ${namespace} kabanero kabanero --export -o=json)

# Update kabanero stack url
oc patch -n ${namespace} kabanero kabanero --type merge --patch "$(cat $(dirname "$0")/22-merge.yaml)"


echo "Waiting for java-microprofile stack URL to update"
LOOP_COUNT=0
until [ "$URL" == "https://github.com/kabanero-io/stacks/releases/download/0.3.1/kabanero-index.yaml" ] 
do
  URL=$(oc -n ${namespace} get stack java-microprofile -o jsonpath='{.status.activeLocation}')
  sleep 5
  LOOP_COUNT=`expr $LOOP_COUNT + 1`
  if [ $LOOP_COUNT -gt 10 ] ; then
    echo "Timed out waiting for java-microprofile stack URL to update"
  exit 1
 fi
done

# Reset 
echo $ORIGYAML | oc apply -f -