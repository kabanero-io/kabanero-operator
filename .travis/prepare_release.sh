#!/bin/bash
BASEPATH=$(dirname $(dirname $0))
DEST=$BASEPATH/deploy/kabanero-operators.yaml

# Find we are running on MAC.
MAC_EXEC=false
macos=Darwin
if [ "$(uname)" == "Darwin" ]; then
   MAC_EXEC=true
fi

# Prepare operator deployment file
cat << EOF >> $DEST
apiVersion: v1
kind: Namespace
metadata:
  name: kabanero
---
EOF

cat $BASEPATH/deploy/dependencies.yaml >> $DEST; echo "---" >> $DEST
cat $BASEPATH/deploy/operator.yaml >> $DEST; echo "---" >> $DEST
cat $BASEPATH/deploy/role.yaml >> $DEST; echo "---" >> $DEST
cat $BASEPATH/deploy/role_binding.yaml >> $DEST; echo "---" >> $DEST
cat $BASEPATH/deploy/service_account.yaml >> $DEST; echo "---" >> $DEST

# Update the operator deployment image entry if TRAVIS_TAG is set
if [ ! -z "$TRAVIS_TAG" ]; then
   if [[ $MAC_EXEC == true ]]; then
      sed -i '' -e "s!image: kabanero-operator:latest!image: kabanero-operator:$TRAVIS_TAG!g" $DEST
   else
      sed -i "s!image: kabanero-operator:latest!image: kabanero-operator:$TRAVIS_TAG!g" $DEST
   fi
fi
