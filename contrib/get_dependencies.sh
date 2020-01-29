#!/bin/bash
set -Eeuox pipefail

$(dirname $0)/get_knative_serving_operator.sh
$(dirname $0)/get_knative_contrib_operator.sh
$(dirname $0)/get_tekton_operator.sh

DEST=dependencies.yaml
cat knative-serving.yaml >> $DEST; echo "---" >> $DEST
cat knative-contrib.yaml >> $DEST; echo "---" >> $DEST
cat tekton.yaml >> $DEST; echo "---" >> $DEST

rm knative-serving.yaml knative-contrib.yaml tekton.yaml



#Errata to knative ClusterRoles for OKD 3.11.0
cat << EOF >> $DEST
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: knative-serving-operator
rules:
- apiGroups:
  - '*'
  attributeRestrictions: null
  resources:
  - '*'
  verbs:
  - '*'
- apiGroups: null
  attributeRestrictions: null
  nonResourceURLs:
  - '*'
  resources: []
  verbs:
  - '*'
---
EOF