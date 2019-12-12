#!/bin/bash

# Delete a collection and ensure the operator reconciler recreates it

set -Eeuox pipefail

namespace=kabanero

# Check collection exists
if ! oc -n ${namespace} get collection java-microprofile
then
  echo "Missing ${namespace} collection java-microprofile"
  exit 1
fi

# Delete collection
oc -n ${namespace} delete collection java-microprofile --ignore-not-found


echo "Waiting for java-microprofile collection to be recreated by reconciler...."
LOOP_COUNT=0
until oc -n ${namespace} get collection java-microprofile
do
  sleep 5
  LOOP_COUNT=`expr $LOOP_COUNT + 1`
  if [ $LOOP_COUNT -gt 10 ] ; then
    echo "Timed out waiting for java-microprofile collection to be recreated by reconciler"
  exit 1
 fi
done
