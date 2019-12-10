#!/bin/bash

set -Eeox pipefail

# require jq. kubectl can not handle multiple filter jsonpath
if ! which jq; then
  "jq is required, please install jq in your PATH."
  exit 1
fi

if ! oc whoami; then
  "Not logged in. Please login as a cluster-admin."
  exit 1
fi

# if the user is not kube:admin, or the user is not a cluster-admin, exit
USER="$(oc whoami)"
if [ "$USER" != "kube:admin" ]; then
  if [ "$USER" != "$(oc get clusterrolebinding -o json | jq -r '.items[] | select(.roleRef.name=="cluster-admin") | select(.subjects[].name=="'$USER'") | .subjects[0].name')" ]; then
    echo "$USER does not have a clusterrolebinding of cluster-admin. Please login as a cluster-admin."
    exit 1
  fi
fi

HOST=$(oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}')

if [ "$USER" == "kube:admin" ]; then
  USER=kubeadmin
fi

docker login -u $USER -p $(oc whoami -t) ${HOST}

