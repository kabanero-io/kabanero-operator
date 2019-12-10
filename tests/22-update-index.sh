#!/bin/bash

# Update the kabanero index URL and ensure the URL is updated for the collection

set -Eeox pipefail

namespace=kabanero

ORIGYAML=$(oc get -n ${namespace} kabanero kabanero --export -o=json)

# Update kabanero collection url
oc patch -n ${namespace} kabanero kabanero --type merge --patch "$(cat 22-merge.yaml)"


echo "Waiting for java-microprofile collection URL to update"
LOOP_COUNT=0
until [ "$URL" == "https://github.com/kabanero-io/collections/releases/download/0.3.1/kabanero-index.yaml" ] 
do
  URL=$(oc -n ${namespace} get collection java-microprofile -o jsonpath='{.status.activeLocation}')
  sleep 5
  LOOP_COUNT=`expr $LOOP_COUNT + 1`
  if [ $LOOP_COUNT -gt 10 ] ; then
    echo "Timed out waiting for java-microprofile collection URL to update"
  exit 1
 fi
done

# Reset 
echo $ORIGYAML | oc apply -f -