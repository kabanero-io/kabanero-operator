#!/bin/bash
set -Eeuox pipefail

$(dirname $0)/get_knative_eventing_operator.sh
$(dirname $0)/get_knative_serving_operator.sh
$(dirname $0)/get_tekton_operator.sh

DEST=dependencies.yaml
cat knative-eventing.yaml >> $DEST; echo "---" >> $DEST
cat knative-serving.yaml >> $DEST; echo "---" >> $DEST
cat tekton.yaml >> $DEST; echo "---" >> $DEST

rm knative-eventing.yaml knative-serving.yaml tekton.yaml



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
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: knative-eventing-operator
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