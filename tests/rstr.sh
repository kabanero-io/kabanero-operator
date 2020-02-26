#!/bin/bash

# Avoid x509 error when using private registry
# https://github.com/knative/serving/issues/5126

set -Eeuox pipefail


DATA=$(oc -n knative-serving get configmap config-deployment -o=jsonpath={.data.registriesSkippingTagResolving})
if echo ${DATA} | grep image-registry.openshift-image-registry.svc:5000
then
  echo "knative-serving configmap config-deployment already skips image-registry.openshift-image-registry.svc:5000"
elif [ -z "${DATA}" ]
then
  RSTR="image-registry.openshift-image-registry.svc:5000,dev.local,ko.local"
  oc -n knative-serving patch configmap config-deployment -p '{"data":{"registriesSkippingTagResolving":"'${RSTR}'"}}'
else
  NEWDATA="${DATA},image-registry.openshift-image-registry.svc:5000"
  oc -n knative-serving patch configmap config-deployment -p '{"data":{"registriesSkippingTagResolving":"'${NEWDATA}'"}}'
fi


