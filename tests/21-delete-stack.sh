#!/bin/bash

# Delete a stack and ensure the operator reconciler recreates it

set -Eeuox pipefail

namespace=kabanero

# Check stack exists
if ! oc -n ${namespace} get stack quarkus
then
  echo "Missing ${namespace} stack quarkus"
  exit 1
fi

# Delete stack
oc -n ${namespace} delete stack quarkus --ignore-not-found


echo "Waiting for quarkus stack to be recreated by reconciler...."
LOOP_COUNT=0
until oc -n ${namespace} get stack quarkus
do
  sleep 5
  LOOP_COUNT=`expr $LOOP_COUNT + 1`
  if [ $LOOP_COUNT -gt 10 ] ; then
    echo "Timed out waiting for quarkus stack to be recreated by reconciler"
  exit 1
 fi
done
