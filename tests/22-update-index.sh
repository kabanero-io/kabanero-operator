#!/bin/bash

# Update the kabanero index URL & pipelines URL and check version is updated
# Uses the release-0.6 artifacts

set -Eeox pipefail

namespace=kabanero

ORIGYAML=$(oc get -n ${namespace} kabanero kabanero --export -o=json)

# Update kabanero stack url
oc patch -n ${namespace} kabanero kabanero --type merge --patch "$(cat $(dirname "$0")/22-merge.yaml)"


echo "Waiting for quarkus stack version to update"
LOOP_COUNT=0
until [ "$VER" == "0.2.26" ] 
do
  VER=$(oc -n ${namespace} get stack quarkus -o jsonpath='{.status.versions[0].version}')
  sleep 5
  LOOP_COUNT=`expr $LOOP_COUNT + 1`
  if [ $LOOP_COUNT -gt 10 ] ; then
    echo "Timed out waiting for quarkus stack version to update"
  exit 1
 fi
done

echo "Waiting for quarkus pipelines url to update"
LOOP_COUNT=0
until [ "$URL" == "https://github.com/kabanero-io/kabanero-pipelines/releases/download/0.6.1/default-kabanero-pipelines.tar.gz" ] 
do
  URL=$(oc -n ${namespace} get stack quarkus -o jsonpath='{.status.versions[0].pipelines[0].url}')
  sleep 5
  LOOP_COUNT=`expr $LOOP_COUNT + 1`
  if [ $LOOP_COUNT -gt 10 ] ; then
    echo "Timed out waiting for quarkus stack URL to update"
  exit 1
 fi
done

# Reset 
echo $ORIGYAML | oc apply -f -