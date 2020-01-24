#!/bin/bash

# Delete a pipeline and ensure the operator reconciler recreates it

set -Eeuox pipefail

namespace=kabanero

# Check pipeline exists
if ! oc -n ${namespace} get pipeline java-microprofile-build-pl
then
  echo "Missing ${namespace} pipeline java-microprofile-build-pl"
  exit 1
fi

# Delete pipeline
oc -n ${namespace} delete pipeline java-microprofile-build-pl --ignore-not-found


echo "Waiting for java-microprofile-build-pl to be recreated by reconciler...."
LOOP_COUNT=0
until oc -n ${namespace} get pipeline java-microprofile-build-pl
do
  sleep 5
  LOOP_COUNT=`expr $LOOP_COUNT + 1`
  if [ $LOOP_COUNT -gt 10 ] ; then
    echo "Timed out waiting for java-microprofile-build-pl to be recreated by reconciler"
  exit 1
 fi
done
