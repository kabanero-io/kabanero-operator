#!/bin/bash

# Delete a pipeline and ensure the operator reconciler recreates it

set -Eeuox pipefail

namespace=kabanero

# Check pipeline exists
if ! oc -n ${namespace} get pipeline quarkus-build-pl
then
  echo "Missing ${namespace} pipeline quarkus-build-pl"
  exit 1
fi

# Delete pipeline
oc -n ${namespace} delete pipeline quarkus-build-pl --ignore-not-found


echo "Waiting for quarkus-build-pl to be recreated by reconciler...."
LOOP_COUNT=0
until oc -n ${namespace} get pipeline quarkus-build-pl
do
  sleep 5
  LOOP_COUNT=`expr $LOOP_COUNT + 1`
  if [ $LOOP_COUNT -gt 10 ] ; then
    echo "Timed out waiting for quarkus-build-pl to be recreated by reconciler"
  exit 1
 fi
done
