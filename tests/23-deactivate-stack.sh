#!/bin/bash

# Delete a stack and ensure the operator reconciler does not reactivate it

set -Eeox pipefail

namespace=kabanero

ORIGYAML=$(oc get -n ${namespace} kabanero kabanero --export -o=json)

# Update kabanero
oc patch -n ${namespace} kabanero kabanero --type merge --patch "$(cat $(dirname "$0")/23-merge.yaml)"

sleep 5

# Delete stack
oc delete -n ${namespace} stack java-microprofile


echo "Checking stack java-microprofile is created but not activated by reconciler"
LOOP_COUNT=0
until oc get -n ${namespace} stack java-microprofile
do
  sleep 5
  LOOP_COUNT=`expr $LOOP_COUNT + 1`
  if [ $LOOP_COUNT -gt 10 ] ; then
    echo "stack java-microprofile was not recreated in time"
    exit 1
  fi
done


LOOP_COUNT=0
until [ "$STATUS" == "inactive" ]
do
  sleep 5
  LOOP_COUNT=`expr $LOOP_COUNT + 1`
  if [ $LOOP_COUNT -gt 10 ] ; then
    echo "stack java-microprofile did not reconcile to inactive state"
    exit 1
  fi
  STATUS=$(oc -n ${namespace} get stack java-microprofile -o jsonpath='{.status.status}')
done


if oc -n ${namespace} get pipeline java-microprofile-build-pl; then
  echo "inactive stack java-microprofile should not have active pipeline java-microprofile-build-pl"
  exit 1
fi


# Reset 
echo $ORIGYAML | oc apply -f -