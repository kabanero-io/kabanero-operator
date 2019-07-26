#!/bin/bash
BASEPATH=$(dirname $(dirname $0))
DEST=$BASEPATH/deploy/kabanero-operators.yaml

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